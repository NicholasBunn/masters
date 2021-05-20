package main

import (
	// Native packages
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"os"
	"time"

	// gRPC packages
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"

	// Proto packages
	estimateServicePB "github.com/nicholasbunn/masters/src/estimateService/proto"
	fetchDataServicePB "github.com/nicholasbunn/masters/src/fetchDataService/proto"
	serverPB "github.com/nicholasbunn/masters/src/powerEstimationSP/proto"
	prepareDataServicePB "github.com/nicholasbunn/masters/src/prepareDataService/proto"

	// Interceptors
	"github.com/nicholasbunn/masters/src/powerEstimationSP/interceptors"
)

var (
	// Addresses (To be passed in a config file)
	addrMyself = os.Getenv("POWERESTIMATIONHOST") + ":50101"
	addrFS     = os.Getenv("FETCHHOST") + ":50051"
	addrPS     = os.Getenv("PREPAREHOST") + ":50052"
	addrES     = os.Getenv("ESTIMATEHOST") + ":50053"

	// Logging stuff
	DebugLogger   *log.Logger
	InfoLogger    *log.Logger
	WarningLogger *log.Logger
	ErrorLogger   *log.Logger
)

const (
	// Timeouts (To be passed in a config file)
	timeoutDuration     = 5 // The time, in seconds, that the client should wait when dialing (connecting to) the server before throwing an error
	callTimeoutDuration = 15 * time.Second
)

// MEEP Use use fatal error to trigger a restart
// MEEP Set service call timeout values
// MEEP Set timeout values for gRPC Dial
// MEEP Try to get the information to be streamed

func init() {
	// Set up logger
	// If the file doesn't exist, create it or append to the file
	file, err := os.OpenFile("program logs/"+"powerEstimationSP.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0666)
	if err != nil {
		fmt.Println("Unable to initialise log file, good luck :)")
		return
	}

	log.SetOutput(file)

	DebugLogger = log.New(file, "DEBUG: ", log.Ldate|log.Ltime|log.Lmicroseconds|log.Lshortfile)
	InfoLogger = log.New(file, "INFO: ", log.Ldate|log.Ltime|log.Lmicroseconds|log.Lshortfile)
	WarningLogger = log.New(file, "WARNING: ", log.Ldate|log.Ltime|log.Lmicroseconds|log.Lshortfile)
	ErrorLogger = log.New(file, "ERROR: ", log.Ldate|log.Ltime|log.Lmicroseconds|log.Lshortfile)
}

func main() {
	InfoLogger.Println("Started aggregator")
	// Load in TLS credentials
	creds, err := loadTLSCredentials()
	if err != nil {
		ErrorLogger.Fatalf("Failed to load TLS credentials")
	} else {
		DebugLogger.Println("Succesfully loaded TLS certificates")
	}

	// Create a listener on the specified tcp port
	listener, err := net.Listen("tcp", addrMyself)
	if err != nil {
		ErrorLogger.Fatalf("Failed to listen on port %v: \n%v", addrMyself, err)
	}
	InfoLogger.Println("Listening on port: ", addrMyself)

	// Create a gRPC server object
	estimationServer := grpc.NewServer(
		grpc.Creds(creds),
	)
	// Attach the power estimation service offering to the server
	serverPB.RegisterPowerEstimationServicePackageServer(estimationServer, &server{})
	DebugLogger.Println("Succesfully registered Power Estimation Service Package to the server")
	// Start the server
	if err := estimationServer.Serve(listener); err != nil {
		ErrorLogger.Fatalf("Failed to expose service: \n%v", err)
	}
}

// Use this to implement the power estimation service package
type server struct {
	serverPB.UnimplementedPowerEstimationServicePackageServer
}

func (s *server) PowerEstimatorService(ctx context.Context, request *serverPB.ServicePackageRequestMessage) (*serverPB.EstimateResponseMessage, error) {
	InfoLogger.Println("Received Power Estimator service call")
	// Load in credentials for the servers
	creds, err := loadTLSCredentials()
	if err != nil {
		ErrorLogger.Printf("Error loading TLS credentials")
		return nil, err
	} else {
		DebugLogger.Println("Succesfully loaded TLS certificates")
	}

	// Create secure connections to the servers
	callCounterFS := interceptors.ClientMetricStruct{}
	connFS, err := CreateSecureServerConnection(addrFS, creds, timeoutDuration, callCounterFS.ClientMetrics)
	if err != nil {
		return nil, err
	}

	callCounterPS := interceptors.ClientMetricStruct{}
	connPS, err := CreateSecureServerConnection(addrPS, creds, timeoutDuration, callCounterPS.ClientMetrics)
	if err != nil {
		return nil, err
	}

	callCounterES := interceptors.ClientMetricStruct{}
	connES, err := CreateSecureServerConnection(addrES, creds, timeoutDuration, callCounterES.ClientMetrics)
	if err != nil {
		return nil, err
	}

	/* Create the clients and pass the connections made above to them. After the clients have been created, we create the gRPC requests */
	InfoLogger.Println("Creating Clients")
	clientFS := fetchDataServicePB.NewFetchDataClient(connFS)
	clientPS := prepareDataServicePB.NewPrepareDataClient(connPS)
	clientES := estimateServicePB.NewEstimatePowerClient(connES)
	DebugLogger.Println("Succesfully created the GoLang clients")

	// Create the request message for the fetch data service
	requestMessageFS := fetchDataServicePB.FetchDataRequestMessage{
		InputFile: request.InputFile,
	}
	DebugLogger.Println("Succesfully created a FetchDataRequestMessage")

	// Make the gRPC service call
	InfoLogger.Println("Making FetchData service call")
	fetchDataContext, _ := context.WithTimeout(context.Background(), callTimeoutDuration)
	// Invoke the fetch data service
	responseMessageFS, err := clientFS.FetchDataService(fetchDataContext, &requestMessageFS) // The responseMessageFS is a RawDataMessage
	// Handle errors, if any, otherwise, close the connection
	if err != nil {
		ErrorLogger.Println("Failed to make the fetch data service call: ")
		return nil, err
	} else {
		DebugLogger.Println("Succesfully made service call to fetch data server.")
		connFS.Close()
	}

	/* Create the request message for the prepare data service with the response
	from the fetch data service */
	requestMessagePS := prepareDataServicePB.PrepareRequestMessage{
		IndexNumber:            responseMessageFS.IndexNumber,
		TimeAndDate:            responseMessageFS.TimeAndDate,
		PortPropMotorCurrent:   responseMessageFS.PortPropMotorCurrent,
		PortPropMotorPower:     responseMessageFS.PortPropMotorPower,
		PortPropMotorSpeed:     responseMessageFS.PortPropMotorSpeed,
		PortPropMotorVoltage:   responseMessageFS.PortPropMotorVoltage,
		StbdPropMotorCurrent:   responseMessageFS.StbdPropMotorCurrent,
		StbdPropMotorPower:     responseMessageFS.StbdPropMotorPower,
		StbdPropMotorSpeed:     responseMessageFS.StbdPropMotorSpeed,
		StbdPropMotorVoltage:   responseMessageFS.StbdPropMotorVoltage,
		RudderOrderPort:        responseMessageFS.RudderOrderPort,
		RudderOrderStbd:        responseMessageFS.RudderOrderStbd,
		RudderPositionPort:     responseMessageFS.RudderPositionPort,
		RudderPositionStbd:     responseMessageFS.RudderPositionStbd,
		PropellerPitchPort:     responseMessageFS.PropellerPitchPort,
		PropellerPitchStbd:     responseMessageFS.PropellerPitchStbd,
		ShaftRpmIndicationPort: responseMessageFS.ShaftRpmIndicationPort,
		ShaftRpmIndicationStbd: responseMessageFS.ShaftRpmIndicationStbd,
		NavTime:                responseMessageFS.NavTime,
		Latitude:               responseMessageFS.Latitude,
		Longitude:              responseMessageFS.Longitude,
		Sog:                    responseMessageFS.Sog,
		Cog:                    responseMessageFS.Cog,
		Hdt:                    responseMessageFS.Hdt,
		WindDirectionRelative:  responseMessageFS.WindDirectionRelative,
		WindSpeed:              responseMessageFS.WindSpeed,
		Depth:                  responseMessageFS.Depth,
		EpochTime:              responseMessageFS.EpochTime,
		BrashIce:               responseMessageFS.BrashIce,
		RammingCount:           responseMessageFS.RammingCount,
		IceConcentration:       responseMessageFS.IceConcentration,
		IceThickness:           responseMessageFS.IceThickness,
		FlowSize:               responseMessageFS.FlowSize,
		BeaufortNumber:         responseMessageFS.BeaufortNumber,
		WaveDirection:          responseMessageFS.WaveDirection,
		WaveHeightAve:          responseMessageFS.WaveHeightAve,
		MaxSwellHeight:         responseMessageFS.MaxSwellHeight,
		WaveLength:             responseMessageFS.WaveLength,
		WavePeriodAve:          responseMessageFS.WavePeriodAve,
		EncounterFrequencyAve:  responseMessageFS.EncounterFrequencyAve,
	}

	// Make the gRPC service call
	InfoLogger.Println("Making PrepareEstimateData service call.")
	prepareDataContext, _ := context.WithTimeout(context.Background(), callTimeoutDuration) // MEEP could still use the cancelFunc, come back to this
	// Invoke the prepare data service
	responseMessagePS, err := clientPS.PrepareEstimateDataService(prepareDataContext, &requestMessagePS)
	// Handle errors, if any, otherwise, close the connection
	if err != nil {
		ErrorLogger.Println("Failed to make PrepareData service call: ")
		return nil, err
	} else {
		DebugLogger.Println("Succesfully made service call to python prepareDataServer.")
		connPS.Close()
	}
	/* Create the request message for the estimate service with the response
	from both the fetch data and prepare data services */
	requestMessageES := estimateServicePB.EstimateRequestMessage{
		PortPropMotorSpeed:    responseMessagePS.PortPropMotorSpeed,
		StbdPropMotorSpeed:    responseMessagePS.StbdPropMotorSpeed,
		PropellerPitchPort:    responseMessagePS.PropellerPitchPort,
		PropellerPitchStbd:    responseMessagePS.PropellerPitchStbd,
		Sog:                   responseMessagePS.Sog,
		WindDirectionRelative: responseMessagePS.WindDirectionRelative,
		WindSpeed:             responseMessagePS.WindSpeed,
		BeaufortNumber:        responseMessagePS.BeaufortNumber,
		WaveDirection:         responseMessagePS.WaveDirection,
		WaveLength:            responseMessagePS.WaveLength,
		MotorPowerPort:        responseMessageFS.PortPropMotorPower,
		MotorPowerStbd:        responseMessageFS.StbdPropMotorPower,
		OriginalSog:           responseMessageFS.Sog,
	}

	// Set the model type enum based on the request being served
	switch request.ModelType {
	case 1: // OpenWater
		requestMessageES.ModelType = estimateServicePB.ModelTypeEnum_OPENWATER
	case 2: // Ice
		requestMessageES.ModelType = estimateServicePB.ModelTypeEnum_ICE
	case 0: // Unknown
		requestMessageES.ModelType = estimateServicePB.ModelTypeEnum_OPENWATER
	default: // Default
		requestMessageES.ModelType = estimateServicePB.ModelTypeEnum_OPENWATER
	}

	// Make the gRPC service call
	InfoLogger.Println("Making EstimateRequestMessage service call.")
	// Invoke the estimate service
	estimateContext, _ := context.WithTimeout(context.Background(), callTimeoutDuration) // MEEP could still use the cancelFunc, come back to this
	// Handle errors, if any, otherwise, close the connection
	responseMessageES, err := clientES.EstimatePowerService(estimateContext, &requestMessageES)
	if err != nil {
		ErrorLogger.Println("Failed to make Estimate service call: ")
		return nil, err
	} else {
		DebugLogger.Println("Succesfully made service call to Python estimateServer.")
		connPS.Close()
	}

	// Create and populate the response message for the request being served
	responseMessage := serverPB.EstimateResponseMessage{
		PowerEstimate: responseMessageES.PowerEstimate,
	}

	return &responseMessage, nil
}

func loadTLSCredentials() (credentials.TransportCredentials, error) {
	/* This function loads TLS credentials for both the client and server,
	enabling mutual TLS authentication between the client and server */

	// Load certificate of the CA who signed server's certificate
	pemServerCA, err := ioutil.ReadFile("certification/ca-cert.pem")
	if err != nil {
		return nil, err
	}

	// Load the server CA's certificates
	certificatePool := x509.NewCertPool()
	if !certificatePool.AppendCertsFromPEM(pemServerCA) {
		return nil, fmt.Errorf("failed to add the server CA's certificate")
	}

	// Load the client's certificate and private key
	clientCertificate, err := tls.LoadX509KeyPair("certification/client-cert.pem", "certification/client-key.pem")
	if err != nil {
		return nil, err
	}

	// Create the credentials object and return it
	config := &tls.Config{
		Certificates: []tls.Certificate{clientCertificate},
		RootCAs:      certificatePool,
	}

	return credentials.NewTLS(config), nil
}

func CreateSecureServerConnection(port string, credentials credentials.TransportCredentials, timeout int, interceptor grpc.UnaryClientInterceptor) (*grpc.ClientConn, error) {
	/* This function takes a port address, credentials object, timeout, and an interceptor as an input, creates a connection to the server at the port adress and
	returns a secure gRPC connection with the specified interceptor */

	ctx, cancel := context.WithTimeout(context.Background(), (time.Duration(timeoutDuration) * time.Second))
	defer cancel()

	conn, err := grpc.DialContext(
		ctx,
		port,
		grpc.WithTransportCredentials(credentials),
		grpc.WithBlock(),
		grpc.WithUnaryInterceptor(interceptor),
	)
	// conn, err := grpc.Dial(port, grpc.WithTransportCredentials(credentials), grpc.WithBlock(), grpc.WithTimeout(time.Duration(timeoutDuration)*time.Second), grpc.WithUnaryInterceptor(interceptor))
	if err != nil {
		ErrorLogger.Println("Failed to create connection to the server on port: " + port)
		return nil, err
	} else {
		InfoLogger.Println("Succesfully created connection to the server on port: " + port)
		return conn, nil
	}
}

func CreateInsecureServerConnection(port string, timeout int, interceptor grpc.UnaryClientInterceptor) (*grpc.ClientConn, error) {
	/* This function takes a port address, credentials object, timeout, and an interceptor as an input, creates a connection to the server at the port adress and
	returns a secure gRPC connection with the specified interceptor */

	ctx, cancel := context.WithTimeout(context.Background(), (time.Duration(timeoutDuration) * time.Second))
	defer cancel()

	conn, err := grpc.DialContext(
		ctx,
		port,
		grpc.WithBlock(),
		grpc.WithInsecure(),
		grpc.WithUnaryInterceptor(interceptor),
	)
	// conn, err := grpc.Dial(port, grpc.WithInsecure(), grpc.WithBlock(), grpc.WithTimeout(time.Duration(timeoutDuration)*time.Second), grpc.WithUnaryInterceptor(interceptor))
	if err != nil {
		ErrorLogger.Println("Failed to create connection to the server on port: " + port)
		return nil, err
	} else {
		InfoLogger.Println("Succesfully created connection to the server on port: " + port)
		return conn, nil
	}
}
