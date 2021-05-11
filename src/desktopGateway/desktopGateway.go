package main

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"os"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"

	serverPB "github.com/nicholasbunn/masters/src/desktopGateway/proto"
	estimationPB "github.com/nicholasbunn/masters/src/powerEstimationSP/proto"

	"github.com/nicholasbunn/masters/src/desktopGateway/interceptors"
)

var (
	// Logging stuff
	DebugLogger   *log.Logger
	InfoLogger    *log.Logger
	WarningLogger *log.Logger
	ErrorLogger   *log.Logger
)

const (
	// Addresses (To be passed in a config file)
	addrMyself       = "localhost:50201"
	addrEstimationSP = "localhost:50101"

	// Timeouts (to be passed in a config file)
	timeoutDuration     = 5 // The time, in seconds, that the client should wait when dialing (connecting to) the server before throwing an error
	callTimeoutDuration = 15 * time.Second

	// Input parameters (To be passed through the frontend)
	INPUTfilename = "TestData/CMU_2019_2020_openWater.xlsx" // MEEP Need to pass a path relative to the execution directory
	MODELTYPE     = "OPENWATER"
)

func init() {
	// Set up logger
	// If the file doesn't exist, create it or append to the file
	file, err := os.OpenFile("program logs/"+"desktopGateway.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0666)
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
	// // Load in credentials
	// creds, err := loadTLSCredentials()
	// if err != nil {
	// 	ErrorLogger.Printf("Error loading TLS credentials")
	// } else {
	// 	DebugLogger.Println("Succesfully loaded TLS certificates")
	// }

	// Receive login request
	// Login to DB
	// Return permissions to the frontend

	// ________RECEIVE REQUEST________
	// Create a listener on the specified tcp port
	DebugLogger.Println("TP")
	listener, err := net.Listen("tcp", addrMyself)
	if err != nil {
		ErrorLogger.Fatalf("Failed to listen on port %v: \n%v", addrMyself, err)
	}
	InfoLogger.Println("Listening on port: ", addrMyself)

	// Create a grpc server object
	gatewayServer := grpc.NewServer()
	// Attach the power estimation service package offering to the server
	serverPB.RegisterPowerEstimationServicesServer(gatewayServer, &server{})
	// Start the server
	if err := gatewayServer.Serve(listener); err != nil {
		ErrorLogger.Fatalf("Failed to expose service: \n%v", err)
	}
}

// Use this to implement the power estimation service routing
type server struct {
	serverPB.UnimplementedPowerEstimationServicesServer
}

func (s *server) PowerEstimationSP(ctx context.Context, request *serverPB.EstimationRequest) (*serverPB.PowerEstimationResponse, error) {
	// Create a connection over the specified tcp port
	callCounter := interceptors.ClientMetricStruct{}
	connEstimationSP := CreateInsecureServerConnection(addrEstimationSP, timeoutDuration, callCounter.ClientMetrics)

	/* Create the client and pass the connection made above to it. After thje client has been
	created, we create the gRPC request */
	InfoLogger.Println("Creating clients")
	clientEstimationSP := estimationPB.NewPowerEstimationServicePackageClient(connEstimationSP)
	DebugLogger.Println("Succesfully created the clients")

	requestMessageEstimationSP := estimationPB.ServicePackageRequestMessage{
		InputFile: INPUTfilename,
		ModelType: estimationPB.ModelTypeEnum_OPENWATER,
	}

	estimationContext, _ := context.WithTimeout(context.Background(), callTimeoutDuration)
	responseEstimationSP, errEstimationSP := clientEstimationSP.PowerEstimatorService(estimationContext, &requestMessageEstimationSP)
	if errEstimationSP != nil {
		ErrorLogger.Println("Failed to make EstimationSP service call: ")
		ErrorLogger.Println(errEstimationSP)
		// ErrorLogger.Fatal(errFS)
	} else {
		DebugLogger.Println("Succesfully made service call to GoLang EstimationSP.")
		connEstimationSP.Close()
	}

	responseMessage := serverPB.PowerEstimationResponse{
		PowerEstimate: responseEstimationSP.PowerEstimate,
	}

	return &responseMessage, nil
}

func CreateInsecureServerConnection(port string, timeout int, interceptor grpc.UnaryClientInterceptor) *grpc.ClientConn {
	// This function takes a port address, credentials object, timeout, and an interceptor as an input, creates a connection to the server at the port adress and returns
	// a secure gRPC connection with the specified interceptor

	conn, err := grpc.Dial(port, grpc.WithInsecure(), grpc.WithBlock(), grpc.WithTimeout(time.Duration(timeoutDuration)*time.Second), grpc.WithUnaryInterceptor(interceptor))
	if err != nil {
		ErrorLogger.Println("Failed to create connection to server on port: " + port)
		ErrorLogger.Println(err)
		// ErrorLogger.Fatal(err)
	} else {
		InfoLogger.Println("Succesfully created connection to the server on port: " + port)
	}
	return conn
}

func CreateSecureServerConnection(port string, credentials credentials.TransportCredentials, timeout int, interceptor grpc.UnaryClientInterceptor) *grpc.ClientConn {
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
