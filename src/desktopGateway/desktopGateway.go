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

	// Required packages
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
	addrAuthenticationService = os.Getenv("AUTHENTICATIONHOST") + ":50401"

	// Logging stuff
	DebugLogger   *log.Logger
	InfoLogger    *log.Logger
	WarningLogger *log.Logger
	ErrorLogger   *log.Logger
)

const (
	// Timeouts (to be passed in a config file)
	timeoutDuration     = 5                // The time, in seconds, that the client should wait when dialing (connecting to) the server before throwing an error
	callTimeoutDuration = 15 * time.Second // The time, in seconds, that the client should wait when making a call to the server before throwing an error

	// Input parameters (To be passed through the frontend)
	INPUTfilename = "TestData/CMU_2019_2020_openWater.xlsx" // MEEP Need to pass a path relative to the execution directory
	MODELTYPE     = "OPENWATER"

	// JWT stuff, load this in from config
	secretkey     = "secret"
	tokenduration = 15 * time.Minute
)

func init() {
	/* The init functin is used to set up the logger whenever the service is started
	 */

	// If the file doesn't exist, create it, otherwise append to the file
	pathSlice := strings.Split(os.Args[0], "/") // This just extracts the services name (filename)
	file, err := os.OpenFile("program logs/"+pathSlice[len(pathSlice)-1]+".log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0666)
	if err != nil {
		// If opening the log file throws an error, continue to create the loggers but print to terminal instead
		log.Println("Unable to initialise log file, good luck :)")
	} else {
		log.SetOutput(file)
	}

	DebugLogger = log.New(file, "DEBUG: ", log.Ldate|log.Ltime|log.Lmicroseconds|log.Lshortfile)
	InfoLogger = log.New(file, "INFO: ", log.Ldate|log.Ltime|log.Lmicroseconds|log.Lshortfile)
	WarningLogger = log.New(file, "WARNING: ", log.Ldate|log.Ltime|log.Lmicroseconds|log.Lshortfile)
	ErrorLogger = log.New(file, "ERROR: ", log.Ldate|log.Ltime|log.Lmicroseconds|log.Lshortfile)
}

func main() {
	/* The main function sets up a server to listen on the specified port,
	encrypts the server connection with TLS, and registers the services on
	offer */

	InfoLogger.Println("Started gateway")

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

	// Create the interceptors required for this connection
	metricInterceptor := interceptors.ServerMetricStruct{} // Custom metric (Prometheus) interceptor
	authInterceptor := interceptors.ServerAuthStruct{      // Custom auth (JWT) interceptor
		JwtManager:           authentication.NewJWTManager(secretkey, tokenduration),
		AuthenticatedMethods: accessibleRoles(),
	}
	// Create an interceptor chain with the above interceptors
	interceptorChain := grpc_middleware.ChainUnaryServer(
		metricInterceptor.ServerMetricInterceptor,
		authInterceptor.ServerAuthInterceptor,
	)

	// Create a gRPC server object
	gatewayServer := grpc.NewServer(
		// grpc.Creds(creds), // Add the TLS credentials to this server
		grpc.UnaryInterceptor(interceptorChain), // Add the interceptor chain to this server
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
	/* This is a map of service calls with their required permission levels
	(to be passed in through config file) */

	return map[string][]string{
		"/src/fetchDataService":                      {"admin"},
		"/src/prepareDataService":                    {"admin"},
		"/src/estimateService":                       {"admin"},
		"/PowerEstimationServices/PowerEstimationSP": {"admin"},
	}
}

func authMethods() map[string]bool {
	/* This is a map of which service calls require authentication (to be passed in through config file) */

	return map[string]bool{
		"/PowerEstimationServicePackage/PowerEstimatorService": true,
	}
}

// ________STRUCTS TO IMPLEMENT THE OFFERED SERVICES________

type loginServer struct {
	// Use this to implement the login service routing

	serverPB.UnimplementedLoginServiceServer
}

type estimationServer struct {
	// Use this to implement the power estimation service routing

	serverPB.UnimplementedPowerEstimationServicesServer
	serverPB.UnimplementedLoginServiceServer
}

// ________IMPLEMENT THE OFFERED SERVICES________

func (s *loginServer) Login(ctx context.Context, request *serverPB.LoginRequest) (*serverPB.LoginResponse, error) {
	/* This service routes a login request to the authentication
	service to log in the user and provide them with a JWT. It
	then returns a list of available services to the user/frontend.*/

	InfoLogger.Println("Received Login service call")

	// Create the interceptors required for this connection
	metricInterceptor := interceptors.ClientMetricStruct{} // Custom auth (JWT) interceptor

	// Create an insecure connection to the server
	connAuthenticationService, err := CreateInsecureServerConnection(
		addrAuthenticationService,                 // Set the address of the server
		timeoutDuration,                           // Set the duration the client will wait before timing out
		metricInterceptor.ClientMetricInterceptor, // Add the interceptor chain to this server
	)
	if err != nil {
		return nil, err
	}

	/* Create the client and pass the connection made above to it. After the client
	has been created, we create the gRPC requests */
	InfoLogger.Println("Creating clients")
	clientAuthenticationPB := authenticationPB.NewAuthenticationServiceClient(connAuthenticationService)
	DebugLogger.Println("Succesfully created the client")

	// Create the request message for the authentication service
	requestMessageAuthenticationService := authenticationPB.LoginAuthRequest{
		Username: request.Username,
		Password: request.Password,
	}

	// Make the service call to the server
	InfoLogger.Println("Making Login service call")
	loginContext, _ := context.WithTimeout(context.Background(), callTimeoutDuration)
	// Invoke the login service
	responseLogin, err := clientAuthenticationPB.LoginAuth(loginContext, &requestMessageAuthenticationService)
	// Handle errors, if any, otherwise, close the connection to the auth service
	if err != nil {
		ErrorLogger.Println("Failed to make the login service call: ", err)
		return nil, err
	} else {
		DebugLogger.Println("Succesfully made service call to authentication service.")
		connAuthenticationService.Close()
	}

	// Create and populate the response message for the request being served
	responseMessage := serverPB.LoginResponse{
		AccessToken: responseLogin.AccessToken,
		Permissions: responseLogin.Permissions,
	}

	return &responseMessage, nil
}

func (s *estimationServer) CostEstimationSP(ctx context.Context, request *serverPB.EstimationRequest) (*serverPB.CostEstimationRespose, error) {
	/* This service routes a cost estimation request to the power-train estimation
	aggregator. This request generates an estimation of the cost for a provided route. */

	responseMessage := serverPB.CostEstimationRespose{
		Blabla: "pass",
	}

	return &responseMessage, nil
}

func (s *estimationServer) PowerEstimationSP(ctx context.Context, request *serverPB.EstimationRequest) (*serverPB.PowerEstimationResponse, error) {
	/* This service routes a power estimation request to the power-train estimation aggregator. This request generates an estimation of the power required for a provided route. */

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

	// Create the interceptors required for this connection
	metricInterceptor := interceptors.ClientMetricStruct{} // Custom metric (Prometheus) interceptor
	authInterceptor := interceptors.ClientAuthStruct{      // Custom auth (JWT) interceptor
		AccessToken:          md["authorisation"][0], // Pass the user's JWT to the outgoing request
		AuthenticatedMethods: authMethods(),
	}
	// Create an interceptor chain with the above interceptors
	interceptorChain := grpc_middleware.ChainUnaryClient(
		metricInterceptor.ClientMetricInterceptor,
		authInterceptor.ClientAuthInterceptor,
	)

	// Create an secure connection to the server
	connEstimationSP, err := CreateSecureServerConnection(
		addrEstimationSP, // Set the address of the server
		creds,            // Add the TLS credentials
		timeoutDuration,  // Set the duration the client will wait before timing out
		interceptorChain, // Add the interceptor chain to this server
	)
	if err != nil {
		return nil, err
	}

	/* Create the client and pass the connection made above to it. After the client
	has been created, we create the gRPC requests */
	InfoLogger.Println("Creating clients")
	clientEstimationSP := estimationPB.NewPowerEstimationServicePackageClient(connEstimationSP)
	DebugLogger.Println("Succesfully created the client")

	// Create the request message for the power-train estimation aggregator
	requestMessageEstimationSP := estimationPB.ServicePackageRequestMessage{
		InputFile: INPUTfilename,
		ModelType: estimationPB.ModelTypeEnum_OPENWATER,
	}

	// Make the service call to the server
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

// ________SUPPORTING FUNCTIONS________

func loadTLSCredentials() (credentials.TransportCredentials, error) {
	/* This function loads TLS credentials for both the client and server,
	enabling mutual TLS authentication between the client and server. It takes no inputs and returns a gRPC TransportCredentials object. */

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

	// Create and return the credentials object
	config := &tls.Config{
		Certificates: []tls.Certificate{clientCertificate},
		RootCAs:      certificatePool,
	}

	return credentials.NewTLS(config), nil
}

func CreateSecureServerConnection(port string, credentials credentials.TransportCredentials, timeout int, interceptor grpc.UnaryClientInterceptor) (*grpc.ClientConn, error) {
	/* This function takes a port address, gRPC TransportCredentials object, timeout,
	and UnaryClientInterceptor object as inputs. It creates a connection to the server
	at the port adress and returns a secure gRPC connection with the specified
	interceptor */

	// Create the context for the request
	ctx, cancel := context.WithTimeout(
		context.Background(),
		(time.Duration(timeoutDuration) * time.Second),
	)
	defer cancel()

	conn, err := grpc.DialContext(
		ctx,              // Add the created context to the connection
		port,             // Add the port that the server is listening on
		grpc.WithBlock(), // Make the dial a blocking call so that we can ensure the connection is indeed created
		grpc.WithTransportCredentials(credentials), // Add the TLS credentials
		grpc.WithUnaryInterceptor(interceptor),     // Add the provided interceptors to the connection
	)

	// Handle errors, if any
	if err != nil {
		ErrorLogger.Println("Failed to create connection to the server on port: " + port)
		return nil, err
	}

	InfoLogger.Println("Succesfully created connection to the server on port: " + port)
	return conn, nil
}

func CreateInsecureServerConnection(port string, timeout int, interceptor grpc.UnaryClientInterceptor) (*grpc.ClientConn, error) {
	/* This function takes a port address, timeout, and UnaryClientInterceptor
	object as inputs. It creates a connection to the server	at the port adress
	and returns an insecure gRPC connection with the specified interceptor */

	// Create the context for the request
	ctx, cancel := context.WithTimeout(
		context.Background(),
		(time.Duration(timeoutDuration) * time.Second),
	)
	defer cancel()

	conn, err := grpc.DialContext(
		ctx,                                    // Add the created context to the connection
		port,                                   // Add the port that the server is listening on
		grpc.WithBlock(),                       // Make the dial a blocking call so that we can ensure the connection is indeed created
		grpc.WithInsecure(),                    // Specify that the connection is insecure (no credentials/authorisation required)
		grpc.WithUnaryInterceptor(interceptor), // Add the provided interceptors to the connection
	)

	// Hamndle errors, if any
	if err != nil {
		ErrorLogger.Println("Failed to create connection to the server on port: " + port)
		return nil, err
	}

	InfoLogger.Println("Succesfully created connection to the server on port: " + port)
	return conn, nil
}
