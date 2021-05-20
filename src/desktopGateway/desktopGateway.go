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
	"strings"
	"time"

	// My packages
	authentication "github.com/nicholasbunn/masters/src/authenticationStuff"

	// gRPC packages
	grpc_middleware "github.com/grpc-ecosystem/go-grpc-middleware"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/metadata"

	// Proto packages
	authenticationPB "github.com/nicholasbunn/masters/src/authenticationService/proto"
	serverPB "github.com/nicholasbunn/masters/src/desktopGateway/proto"
	estimationPB "github.com/nicholasbunn/masters/src/powerEstimationSP/proto"

	// Interceptors
	"github.com/nicholasbunn/masters/src/desktopGateway/interceptors"
)

var (
	// Addresses (To be passed in a config file)
	addrMyself                = os.Getenv("DESKTOPGATEWAYHOST") + ":50201"
	addrEstimationSP          = os.Getenv("POWERESTIMATIONHOST") + ":50101"
	addrAuthenticationService = "localhost:50401"

	// Logging stuff
	DebugLogger   *log.Logger
	InfoLogger    *log.Logger
	WarningLogger *log.Logger
	ErrorLogger   *log.Logger
)

const (
	// Timeouts (to be passed in a config file)
	timeoutDuration     = 5 // The time, in seconds, that the client should wait when dialing (connecting to) the server before throwing an error
	callTimeoutDuration = 15 * time.Second

	// Input parameters (To be passed through the frontend)
	INPUTfilename = "TestData/CMU_2019_2020_openWater.xlsx" // MEEP Need to pass a path relative to the execution directory
	MODELTYPE     = "OPENWATER"

	// JWT stuff, load this in from config
	secretkey     = "secret"
	tokenduration = 15 * time.Minute
)

func init() {
	// Set up logger
	// If the file doesn't exist, create it, otherwise append to the file
	pathSlice := strings.Split(os.Args[0], "/") // This just extracts the services name (filename)
	file, err := os.OpenFile("program logs/"+pathSlice[len(pathSlice)-1]+".log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0666)
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
	// Load in TLS credentials
	// creds, err := loadTLSCredentials()
	// if err != nil {
	// 	ErrorLogger.Printf("Error loading TLS credentials")
	// } else {
	// 	DebugLogger.Println("Succesfully loaded TLS certificates")
	// }

	// Create a listener on the specified tcp port
	listener, err := net.Listen("tcp", addrMyself)
	if err != nil {
		ErrorLogger.Fatalf("Failed to listen on port %v: \n%v", addrMyself, err)
	}
	InfoLogger.Println("Listening on port: ", addrMyself)

	metricInterceptor := interceptors.ServerMetricStruct{}
	authInterceptor := interceptors.ServerAuthStruct{
		JwtManager:           authentication.NewJWTManager(secretkey, tokenduration),
		AuthenticatedMethods: accessibleRoles(),
	}
	interceptorChain := grpc_middleware.ChainUnaryServer(
		metricInterceptor.ServerMetricInterceptor,
		authInterceptor.ServerAuthInterceptor,
	)

	// Create a gRPC server object
	gatewayServer := grpc.NewServer(
		grpc.UnaryInterceptor(interceptorChain),
		// grpc.Creds(creds),
	)

	// Attach the Login service offering to the server
	serverPB.RegisterLoginServiceServer(gatewayServer, &loginServer{})
	DebugLogger.Println("Succesfully registered Login Service to the server")
	// Attach the power estimation service package offering to the server
	serverPB.RegisterPowerEstimationServicesServer(gatewayServer, &estimationServer{})
	DebugLogger.Println("Succesfully registered Power Estimation Services to the server")
	// Start the server
	if err := gatewayServer.Serve(listener); err != nil {
		ErrorLogger.Fatalf("Failed to expose service: \n%v", err)
	}
}

func accessibleRoles() map[string][]string {
	return map[string][]string{
		"/src/fetchDataService":                      {"admin"},
		"/src/prepareDataService":                    {"admin"},
		"/src/estimateService":                       {"admin"},
		"/PowerEstimationServices/PowerEstimationSP": {"admin"},
	}
}

func authMethods() map[string]bool {
	return map[string]bool{
		"/PowerEstimationServicePackage/PowerEstimatorService": true,
	}
}

// Use this to implement the login service routing
type loginServer struct {
	serverPB.UnimplementedLoginServiceServer
}

// Use this to implement the power estimation service routing
type estimationServer struct {
	serverPB.UnimplementedPowerEstimationServicesServer
	serverPB.UnimplementedLoginServiceServer
}

func (s *loginServer) Login(ctx context.Context, request *serverPB.LoginRequest) (*serverPB.LoginResponse, error) {
	InfoLogger.Println("Received Login service call")
	// ________CONNECT TO USER DATABASE________

	// ________SEARCH FOR/VERIFY USER WITH PROVIDED CREDENTIALS_______

	// Create a secure connection to the server
	metricInterceptor := interceptors.ClientMetricStruct{}
	interceptorChain := grpc_middleware.ChainUnaryClient(
		metricInterceptor.ClientMetricInterceptor,
	)

	connAuthenticationService, err := CreateInsecureServerConnection(addrAuthenticationService, timeoutDuration, interceptorChain)
	if err != nil {
		return nil, err
	}

	/* Create the client and pass the connection made above to it. After the client
	has been created, we create the gRPC requests */
	InfoLogger.Println("Creating clients")
	clientAuthenticationPB := authenticationPB.NewAuthenticationServiceClient(connAuthenticationService)
	DebugLogger.Println("Succesfully created the client")

	// Create the request message for the power estimation service package
	requestMessageAuthenticationService := authenticationPB.LoginAuthRequest{
		Username: request.Username,
		Password: request.Password,
	}

	// Make the gRPC service call
	InfoLogger.Println("Making Login service call")
	loginContext, _ := context.WithTimeout(context.Background(), callTimeoutDuration)
	// Invoke the power estimation service package
	responseLogin, err := clientAuthenticationPB.LoginAuth(loginContext, &requestMessageAuthenticationService)
	// Handle errors, if any, otherwise, close the connection to the auth service
	if err != nil {
		ErrorLogger.Println("Failed to make the login service call: ", err)
		return nil, err
	} else {
		DebugLogger.Println("Succesfully made service call to authentication service.")
		connAuthenticationService.Close()
	}

	// ________RETURN PERMISSIONS/RESPONSE________
	responseMessage := serverPB.LoginResponse{
		AccessToken: responseLogin.AccessToken,
		Permissions: responseLogin.Permissions,
	}

	return &responseMessage, nil
}

func (s *estimationServer) CostEstimationSP(ctx context.Context, request *serverPB.EstimationRequest) (*serverPB.CostEstimationRespose, error) {
	responseMessage := serverPB.CostEstimationRespose{
		Blabla: "pass",
	}

	return &responseMessage, nil
}

func (s *estimationServer) PowerEstimationSP(ctx context.Context, request *serverPB.EstimationRequest) (*serverPB.PowerEstimationResponse, error) {
	InfoLogger.Println("Received Power Estimator service call")
	// Load in credentials for the servers
	creds, err := loadTLSCredentials()
	if err != nil {
		ErrorLogger.Printf("Error loading TLS credentials")
		return nil, err
	} else {
		DebugLogger.Println("Succesfully loaded TLS certificates")
	}

	// Extract the user's JWT from the incoming request. Can ignore the ok output as ths has already been checked.
	md, _ := metadata.FromIncomingContext(ctx)

	// Create a secure connection to the server
	metricInterceptor := interceptors.ClientMetricStruct{}
	authInterceptor := interceptors.ClientAuthStruct{
		AccessToken:          md["authorisation"][0], // Pass the user's JWT to the outgoing request
		AuthenticatedMethods: authMethods(),
	}
	interceptorChain := grpc_middleware.ChainUnaryClient(
		metricInterceptor.ClientMetricInterceptor,
		authInterceptor.ClientAuthInterceptor,
	)

	connEstimationSP, err := CreateSecureServerConnection(addrEstimationSP, creds, timeoutDuration, interceptorChain)
	if err != nil {
		return nil, err
	}

	/* Create the client and pass the connection made above to it. After the client
	has been created, we create the gRPC requests */
	InfoLogger.Println("Creating clients")
	clientEstimationSP := estimationPB.NewPowerEstimationServicePackageClient(connEstimationSP)
	DebugLogger.Println("Succesfully created the client")

	// Create the request message for the power estimation service package
	requestMessageEstimationSP := estimationPB.ServicePackageRequestMessage{
		InputFile: INPUTfilename,
		ModelType: estimationPB.ModelTypeEnum_OPENWATER,
	}

	// Make the gRPC service call
	InfoLogger.Println("Making PowerEstimationSP service call")
	estimationContext, _ := context.WithTimeout(context.Background(), callTimeoutDuration)
	// Invoke the power estimation service package
	responseEstimationSP, err := clientEstimationSP.PowerEstimatorService(estimationContext, &requestMessageEstimationSP)
	// Handle errors, if any, otherwise, close the connection
	if err != nil {
		ErrorLogger.Println("Failed to make the power estimation SP service call: ")
		return nil, err
	} else {
		DebugLogger.Println("Succesfully made service call to estimation SP.")
		connEstimationSP.Close()
	}

	// Create and populate the response message for the request being served
	responseMessage := serverPB.PowerEstimationResponse{
		PowerEstimate: responseEstimationSP.PowerEstimate,
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
