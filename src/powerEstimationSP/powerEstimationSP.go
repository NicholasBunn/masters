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

	// gRPC packages
	"github.com/go-yaml/yaml"
	grpc_middleware "github.com/grpc-ecosystem/go-grpc-middleware"
	grpc_retry "github.com/grpc-ecosystem/go-grpc-middleware/retry"
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
	addrMyself string
	addrFS     string
	addrPS     string
	addrES     string

	timeoutDuration     int           // The time, in seconds, that the client should wait when dialing (connecting to) the server before throwing an error
	callTimeoutDuration time.Duration // The time, in seconds, that the client should wait when making a call to the server before throwing an error

	// Input parameters (To be passed through the frontend)
	INPUTfilename = "TestData/CMU_2019_2020_openWater.xlsx" // MEEP Need to pass a path relative to the execution directory
	MODELTYPE     = "OPENWATER"

	// JWT stuff, load this in from config
	secretkey     string
	tokenduration time.Duration

	accessibleRoles map[string][]string // This is a map of service calls with their required permission levels

	authMethods map[string]bool // This is a map of which service calls require authentication

	// Logging stuff
	DebugLogger   *log.Logger
	InfoLogger    *log.Logger
	WarningLogger *log.Logger
	ErrorLogger   *log.Logger

	// Metric interceptors
	clientMetricInterceptor *interceptors.ClientMetricStruct
	serverMetricInterceptor *interceptors.ServerMetricStruct
)

func init() {
	/* The init function is used to load in configuration variables, and set up the logger and metric interceptors whenever the service is started
	 */

	// ________CONFIGURATION________
	// Load YAML configurations into config struct
	config, _ := DecodeConfig("src/powerEstimationSP/configuration.yaml")

	addrMyself = os.Getenv("POWERESTIMATIONHOST") + ":" + config.Server.Port.Myself
	addrFS = os.Getenv("FETCHHOST") + ":" + config.Client.Port.FetchService
	addrPS = os.Getenv("PREPAREHOST") + ":" + config.Client.Port.PrepareService
	addrES = os.Getenv("ESTIMATEHOST") + ":" + config.Client.Port.EstimationService

	// Load timeouts from config
	timeoutDuration = config.Client.Timeout.Connection
	fmt.Println(timeoutDuration)
	callTimeoutDuration = time.Duration(config.Client.Timeout.Call) * time.Second
	fmt.Println(callTimeoutDuration)

	// Load JWT parameters from config
	secretkey = config.Server.Authentication.Jwt.SecretKey
	fmt.Println(secretkey)
	tokenduration = time.Duration(config.Server.Authentication.Jwt.TokenDuration) * (time.Minute)
	fmt.Println(tokenduration)

	accessibleRoles = map[string][]string{
		config.Server.Authentication.AccessLevel.Name.PowerEstimate:  config.Server.Authentication.AccessLevel.Role.PowerEstimate,
		config.Server.Authentication.AccessLevel.Name.PowerEvaluator: config.Server.Authentication.AccessLevel.Role.PowerEvaluator,
	}
	fmt.Println(accessibleRoles)

	authMethods = map[string]bool{
		config.Client.AuthenticatedMethods.Name.FetchDataService:   config.Client.AuthenticatedMethods.RequiresAuthentication.FetchDataService,
		config.Client.AuthenticatedMethods.Name.PrepareDataService: config.Client.AuthenticatedMethods.RequiresAuthentication.PrepareDataService,
		config.Client.AuthenticatedMethods.Name.EstimateService:    config.Client.AuthenticatedMethods.RequiresAuthentication.EstimateService,
	}
	fmt.Println(authMethods)

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

	// Metric interceptor
	clientMetricInterceptor = interceptors.NewClientMetrics() // Custom metric (Prometheus) interceptor
	serverMetricInterceptor = interceptors.NewServerMetrics() // Custom metric (Prometheus) interceptor
}

func main() {
	/* The main function sets up a server to listen on the specified port,
	encrypts the server connection with TLS, and registers the services on
	offer */

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
		grpc.UnaryInterceptor(serverMetricInterceptor.ServerMetricInterceptor), // Add the interceptor to this server
	)

	// Attach the power-train estimation service offering to the server
	serverPB.RegisterPowerEstimationServicePackageServer(estimationServer, &server{})
	DebugLogger.Println("Succesfully registered Power Estimation Service Package to the server")

	// Start the server
	if err := estimationServer.Serve(listener); err != nil {
		ErrorLogger.Fatalf("Failed to expose service: \n%v", err)
	}
}

// ________REQUIRED STRUCTS________

type Config struct {
	Server struct {
		Port struct {
			Myself string `yaml:"myself"`
		} `yaml:"port"`
		Authentication struct {
			Jwt struct {
				SecretKey     string `yaml:"secretKey"`
				TokenDuration int    `yaml:"tokenDuration"`
			} `yaml:"jwt"`
			AccessLevel struct {
				Name struct {
					PowerEstimate  string `yaml:"powerEstimate"`
					PowerEvaluator string `yaml:"powerEvaluator"`
				} `yaml:"name"`
				Role struct {
					PowerEstimate  []string `yaml:"powerEstimate"`
					PowerEvaluator []string `yaml:"powerEvaluator"`
				} `yaml:"role"`
			} `yaml:"accessLevel"`
		} `yaml:"authentication"`
	} `yaml:"server"`

	Client struct {
		Port struct {
			FetchService      string `yaml:"fetch"`
			PrepareService    string `yaml:"prepare"`
			EstimationService string `yaml:"estimation"`
		} `yaml:"port"`
		Timeout struct {
			Connection int `yaml:"connection"`
			Call       int `yaml:"call"`
		} `yaml:"timeout"`
		AuthenticatedMethods struct {
			Name struct {
				FetchDataService   string `yaml:"fetchDataService"`
				PrepareDataService string `yaml:"prepareDataService"`
				EstimateService    string `yaml:"estimateService"`
			} `yaml:"name"`
			RequiresAuthentication struct {
				FetchDataService   bool `yaml:"fetchDataService"`
				PrepareDataService bool `yaml:"prepareDataService"`
				EstimateService    bool `yaml:"estimateService"`
			} `yaml:"requiresAuthentication"`
		} `yaml:"authenticatedMethods"`
	} `yaml:"client"`
}

type server struct {
	// Use this to implement the power estimation service package

	serverPB.UnimplementedPowerEstimationServicePackageServer
}

// ________IMPLEMENT THE OFFERED SERVICES________

func (s *server) PowerEstimatorService(ctx context.Context, request *serverPB.ServicePackageRequestMessage) (*serverPB.EstimateResponseMessage, error) {
	/* This service invokes three microservices in order to create an estimation
	of the power required for the provided route. It first colelcts the required wave
	data, then sends it to a processing service which structures the data for a ML
	algorithm, before finally sending the structured data into the model for a
	prediction */

	InfoLogger.Println("Received Power Estimator service call")

	// Load in credentials for the servers
	creds, err := loadTLSCredentials()
	if err != nil {
		ErrorLogger.Printf("Error loading TLS credentials")
		return nil, err
	} else {
		DebugLogger.Println("Succesfully loaded TLS certificates")
	}

	// Create the retry options to specify how the client should retry connection interrupts
	retryOptions := []grpc_retry.CallOption{
		grpc_retry.WithBackoff(grpc_retry.BackoffExponential(100 * time.Millisecond)), // Use exponential backoff to progressively wait longer between retries
		grpc_retry.WithMax(5), // Set the maximum number of retries
	}

	// Create an interceptor chain with the above interceptors
	interceptorChain := grpc_middleware.ChainUnaryClient(
		clientMetricInterceptor.ClientMetricInterceptor,
		grpc_retry.UnaryClientInterceptor(retryOptions...),
	)

	// Create an secure connection to the fetch data server
	connFS, err := createSecureServerConnection(
		addrFS,           // Set the address of the server
		creds,            // Add the TLS credentials
		timeoutDuration,  // Set the duration the client will wait before timing out
		interceptorChain, // Add the interceptor to this server
	)
	if err != nil {
		return nil, err
	}

	// Create an secure connection to the prepare data server
	connPS, err := createSecureServerConnection(
		addrPS,           // Set the address of the server
		creds,            // Add the TLS credentials
		timeoutDuration,  // Set the duration the client will wait before timing out
		interceptorChain, // Add the interceptor to this server
	)
	if err != nil {
		return nil, err
	}

	// Create an secure connection to the estimation server
	connES, err := createSecureServerConnection(
		addrES,           // Set the address of the server
		creds,            // Add the TLS credentials
		timeoutDuration,  // Set the duration the client will wait before timing out
		interceptorChain, // Add the interceptor to this server
	)
	if err != nil {
		return nil, err
	}

	/* Create the clients and pass the connections made above to them. After the clients have been created, we create the gRPC requests */
	InfoLogger.Println("Creating Clients")
	clientFS := fetchDataServicePB.NewFetchDataClient(connFS)     // fetch data service client
	clientPS := prepareDataServicePB.NewPrepareDataClient(connPS) // prepare data service client
	clientES := estimateServicePB.NewEstimatePowerClient(connES)  // estimate service client
	DebugLogger.Println("Succesfully created the GoLang clients")

	// Create the request message for the fetch data service
	requestMessageFS := fetchDataServicePB.FetchDataRequestMessage{
		InputFile: request.InputFile,
	}
	DebugLogger.Println("Succesfully created a FetchDataRequestMessage")

	// Make the service call to the fetch data server
	InfoLogger.Println("Making FetchData service call")
	fetchDataContext, cancel := context.WithTimeout(context.Background(), callTimeoutDuration)
	defer cancel()
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

	// Make the service call to the prepare data server
	InfoLogger.Println("Making PrepareEstimateData service call.")
	prepareDataContext, cancel := context.WithTimeout(context.Background(), callTimeoutDuration)
	defer cancel()
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

	// Make the service call to the estimate server
	InfoLogger.Println("Making EstimateRequestMessage service call.")
	// Invoke the estimate service
	estimateContext, cancel := context.WithTimeout(context.Background(), callTimeoutDuration)
	defer cancel()
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

// ________SUPPORTING FUNCTIONS________

func DecodeConfig(configPath string) (*Config, error) {
	// Create a new config structure
	config := &Config{}

	// Open the config file
	file, err := os.Open(configPath)
	if err != nil {
		fmt.Println("Could not open config file")
		return nil, err
	}
	defer file.Close()

	// Initialise a new YAML decoder
	decoder := yaml.NewDecoder(file)

	// Start YAML decoding from file
	if err := decoder.Decode(&config); err != nil {
		fmt.Println("Could not decode config file: \n", err)
		return nil, err
	}

	return config, nil
}

func loadTLSCredentials() (credentials.TransportCredentials, error) {
	/* This (unexported) function loads TLS credentials for both the client and server,
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

func createSecureServerConnection(port string, credentials credentials.TransportCredentials, timeout int, interceptor grpc.UnaryClientInterceptor) (*grpc.ClientConn, error) {
	/* This (unexported) function takes a port address, gRPC TransportCredentials object, timeout,
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

func createInsecureServerConnection(port string, timeout int, interceptor grpc.UnaryClientInterceptor) (*grpc.ClientConn, error) {
	/* This (unexported) function takes a port address, timeout, and UnaryClientInterceptor
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
