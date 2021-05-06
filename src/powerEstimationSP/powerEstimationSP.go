package main

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"

	estimateServicePB "github.com/nicholasbunn/masters/src/estimateService/proto"
	fetchDataServicePB "github.com/nicholasbunn/masters/src/fetchDataService/proto"
	prepareDataServicePB "github.com/nicholasbunn/masters/src/prepareDataService/proto"

	"github.com/nicholasbunn/masters/src/powerEstimationSP/interceptors"
)

var (
	addrFS              = "127.0.0.1:50051"
	addrPS              = "localhost:50052"
	addrES              = "localhost:50053"
	timeoutDuration     = 5 // The time, in seconds, that the client should wait when dialing (connecting to) the server before throwing an error
	callTimeoutDuration = 15 * time.Second
	INPUTfilename       = "TestData/CMU_2019_2020_openWater.xlsx" // MEEP Need to pass a path relative to the execution directory
	MODELTYPE           = "OPENWATER"

	// Logging stuff
	DebugLogger   *log.Logger
	InfoLogger    *log.Logger
	WarningLogger *log.Logger
	ErrorLogger   *log.Logger
)

// MEEP Implement switch case to deal with user input for model type
// MEEP Figure out how to shutdown python server and possibly pass number of service calls as arguments
// MEEP Should errors be fatal, or should the program run regardless? Reconsider error handling on a case-by-case basis
// MEEP Implement secure connections
// MEEP Set service call timeout values
// MEEP Set timeout values for gRPC Dial
// MEEP Try to get the information to be streamed

func init() {
	// Set up logger
	// If the file doesn't exist, create it or append to the file
	file, err := os.OpenFile("program logs/"+"logs.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0666)
	if err != nil {
		log.Fatal(err)
	}

	log.SetOutput(file)

	DebugLogger = log.New(file, "DEBUG: ", log.Ldate|log.Ltime|log.Lmicroseconds|log.Lshortfile)
	InfoLogger = log.New(file, "INFO: ", log.Ldate|log.Ltime|log.Lmicroseconds|log.Lshortfile)
	WarningLogger = log.New(file, "WARNING: ", log.Ldate|log.Ltime|log.Lmicroseconds|log.Lshortfile)
	ErrorLogger = log.New(file, "ERROR: ", log.Ldate|log.Ltime|log.Lmicroseconds|log.Lshortfile)
}

func main() {
	InfoLogger.Println("Started GoLang Aggregator")

	// // Create a metrics registry.
	// reg := prometheus.NewRegistry()
	// // Create some standard client metrics.
	// grpcMetrics := grpc_prometheus.NewClientMetrics()
	// // Register client metrics to registry.
	// reg.MustRegister(grpcMetrics)

	// // Create a HTTP server for prometheus.
	// httpServer := &http.Server{Handler: promhttp.HandlerFor(reg, promhttp.HandlerOpts{}), Addr: log.Print("0.0.0.0:%d", 9092)}

	// // Start your http server for prometheus.
	// go func() {
	// 	if err := httpServer.ListenAndServe(); err != nil {
	// 		log.Fatal("Unable to start a http server.")
	// 	}
	// }()

	creds, err := loadTLSCredentials()
	if err != nil {
		ErrorLogger.Printf("Error loading TLS credentials")
	}
	DebugLogger.Println("Succesfully loaded TLS certificates")

	callCounterFS := interceptors.ClientMetricStruct{}
	connFS := CreatePythonServerConnection(addrFS, creds, timeoutDuration, callCounterFS.ClientMetrics)

	callCounterPS := interceptors.ClientMetricStruct{}
	connPS := CreatePythonServerConnection(addrPS, creds, timeoutDuration, callCounterPS.ClientMetrics)

	callCounterES := interceptors.ClientMetricStruct{}
	connES := CreatePythonServerConnection(addrES, creds, timeoutDuration, callCounterES.ClientMetrics)

	/* Create the client and pass the connection made above to it. After the client has been
	created, we create the gRPC request */
	InfoLogger.Println("Creating GoLang Clients")
	clientFS := fetchDataServicePB.NewFetchDataClient(connFS)
	clientPS := prepareDataServicePB.NewPrepareDataClient(connPS)
	clientES := estimateServicePB.NewEstimatePowerClient(connES)
	DebugLogger.Println("Succesfully created the GoLang clients")

	requestMessageFS := fetchDataServicePB.FetchDataRequestMessage{
		InputFile: INPUTfilename,
	}
	DebugLogger.Println("Succesfully created a FetchDataRequestMessage")

	// Create header to read the metadata that the response carries
	// var headerFS, trailerFS metadata.MD // MEEP: Header has no information in it yet, this is filled by the server

	// Make the gRPC service call
	InfoLogger.Println("Making FetchData service call.")
	fetchDataContext, _ := context.WithTimeout(context.Background(), callTimeoutDuration)
	responseMessageFS, errFS := clientFS.FetchDataService(fetchDataContext, &requestMessageFS) // The responseMessageFS is a RawDataMessage
	if errFS != nil {
		ErrorLogger.Println("Failed to make FetchData service call: ")
		ErrorLogger.Println(errFS)
		// ErrorLogger.Fatal(errFS)
	} else {
		DebugLogger.Println("Succesfully made service call to Python fetchDataServer.")
		connFS.Close()
	}

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

	InfoLogger.Println("Making PrepareEstimateData service call.")
	prepareDataContext, _ := context.WithTimeout(context.Background(), callTimeoutDuration) // MEEP could still use the cancelFunc, come back to this
	// Invoke prepareserver and pass fetchserver outputs as arguments
	responseMessagePS, errPS := clientPS.PrepareEstimateDataService(prepareDataContext, &requestMessagePS)

	if errPS != nil {
		ErrorLogger.Println("Failed to make PrepareData service call: ")
		ErrorLogger.Println(errPS)
		// ErrorLogger.Fatal(errPS)
	}
	DebugLogger.Println("Succesfully made service call to python prepareDataServer.")
	connPS.Close()

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

	switch MODELTYPE {
	case "OPENWATER":
		requestMessageES.ModelType = estimateServicePB.ModelTypeEnum_OPENWATER
	case "ICE":
		requestMessageES.ModelType = estimateServicePB.ModelTypeEnum_ICE
	case "UNKNOWN":
		requestMessageES.ModelType = estimateServicePB.ModelTypeEnum_OPENWATER
	default:
		requestMessageES.ModelType = estimateServicePB.ModelTypeEnum_OPENWATER
	}

	InfoLogger.Println("Making EstimateRequestMessage service call.")
	// Invoke estimateserver and pass prepareserver outputs as arguements
	estimateContext, _ := context.WithTimeout(context.Background(), callTimeoutDuration) // MEEP could still use the cancelFunc, come back to this
	responseMessageES, errES := clientES.EstimatePowerService(estimateContext, &requestMessageES)
	if errES != nil {
		ErrorLogger.Println("Failed to make Estimate service call: ")
		ErrorLogger.Println(errES)
		// ErrorLogger.Fatal(errES)
	}
	DebugLogger.Println("Succesfully made service call to Python estimateServer.")
	connPS.Close()

	fmt.Println(responseMessageES.PowerEstimate[1]) // MEEP remove once you've done something with responseMEssageFS
}

// DEPRECATED
func SpinUpServices(interpreter []string, directories []string, filenames []string) bool {
	// Check that the 'directories' and 'filenames' are of the same length before iterating through them
	if len(directories) != len(filenames) {
		log.Println("The 'directories' and 'filenames' slices passed into the 'SpinUpSerivces' function are not of equal lengths")
		log.Fatal()
		return false // These are here for error handling when I get around to it, won't execute at the moment
	} else {
		// Reusable variables
		fileLocation := ""
		var cmd *exec.Cmd
		var err error

		// Iterate through the required services and start them up
		for i := range directories {
			log.Println("Invoking " + interpreter[i] + " service: " + filenames[i])
			fileLocation = directories[i] + filenames[i]
			cmd = exec.Command(interpreter[i], fileLocation)
			err = cmd.Start()
			if err != nil {
				log.Println("Failed to invoke {}", filenames[i])
				log.Fatal(err)
				return false
			}
		}

		return true
	}
}

func CreatePythonServerConnection(port string, credentials credentials.TransportCredentials, timeout int, interceptor grpc.UnaryClientInterceptor) *grpc.ClientConn {
	// This function takes a port address, credentials object, timeout, and an interceptor as an input, creates a connection to the server at the port adress and returns
	// a secure gRPC connection with the specified interceptor

	conn, err := grpc.Dial(port, grpc.WithTransportCredentials(credentials), grpc.WithBlock(), grpc.WithTimeout(time.Duration(timeoutDuration)*time.Second), grpc.WithUnaryInterceptor(interceptor))
	if err != nil {
		ErrorLogger.Println("Failed to create connection to Python server on port: " + port)
		ErrorLogger.Println(err)
		// ErrorLogger.Fatal(err)
	}
	InfoLogger.Println("Succesfully created connection to the Python server on port: " + port)

	return conn
}

func loadTLSCredentials() (credentials.TransportCredentials, error) {
	// This function loads TLS credentials for both the client and server,
	// enabling mutual TLS authentication between the client and server

	// Load certificate of the CA who signed server's certificate
	pemServerCA, err := ioutil.ReadFile("certification/ca-cert.pem")
	if err != nil {
		return nil, err
	}

	// Load the server CA's certificates
	certificatePool := x509.NewCertPool()
	if !certificatePool.AppendCertsFromPEM(pemServerCA) {
		return nil, fmt.Errorf("Failed to add the server CA's certificate")
	}

	// Load the client's certificate and private key
	clientCertificate, err := tls.LoadX509KeyPair("certification/client-cert.pem", "certification/client-key.pem")
	if err != nil {
		return nil, err
	}

	// Create the credentials and return it
	config := &tls.Config{
		Certificates: []tls.Certificate{clientCertificate},
		RootCAs:      certificatePool,
	}

	return credentials.NewTLS(config), nil
}
