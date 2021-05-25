package main

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io/ioutil"
	"log"
	"time"

	grpc_middleware "github.com/grpc-ecosystem/go-grpc-middleware"
	grpc_retry "github.com/grpc-ecosystem/go-grpc-middleware/retry"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"

	desktopPB "github.com/nicholasbunn/masters/src/desktopGateway/proto"
	"github.com/nicholasbunn/masters/src/frontend/interceptors"
)

const (
	addrMyself         = "localhost:50301"
	addrDesktopGateway = "localhost:50201"

	// Timeouts (to be passed in a config file)
	timeoutDuration     = 5 // The time, in seconds, that the client should wait when dialing (connecting to) the server before throwing an error
	callTimeoutDuration = 15 * time.Second

	// Input parameters (To be passed through the frontend)
	INPUTfilename = "TestData/CMU_2019_2020_openWater.xlsx" // MEEP Need to pass a path relative to the execution directory
	MODELTYPE     = "OPENWATER"
)

func main() {

	fmt.Println("Started frontend")

	// Load in TLS credentials
	creds, err := loadTLSCredentials()
	if err != nil {
		fmt.Printf("Error loading TLS credentials")
	} else {
		fmt.Println("Succesfully loaded TLS certificates")
	}

	metricInterceptor := interceptors.NewClientMetrics()
	authInterceptor := interceptors.ClientAuthStruct{}
	retryOptions := []grpc_retry.CallOption{
		grpc_retry.WithBackoff(grpc_retry.BackoffExponential(100 * time.Millisecond)),
		grpc_retry.WithMax(5),
	}
	interceptorChain := grpc_middleware.ChainUnaryClient(
		metricInterceptor.ClientMetricInterceptor,
		authInterceptor.ClientAuthInterceptor,
		grpc_retry.UnaryClientInterceptor(retryOptions...),
	)

	connDesktopGateway, err := createSecureServerConnection(addrDesktopGateway, creds, timeoutDuration, interceptorChain)
	if err != nil {
		log.Fatal(err)
	}

	clientLoginDesktopGateway := desktopPB.NewLoginServiceClient(connDesktopGateway)

	desktopContext, _ := context.WithTimeout(context.Background(), callTimeoutDuration)

	loginRequest := desktopPB.LoginRequest{
		Username: "admin",
		Password: "myPassword",
	}

	newResponse, newErr := clientLoginDesktopGateway.Login(desktopContext, &loginRequest)
	if newErr != nil {
		fmt.Println("Login failed")
	} else {
		fmt.Println(newResponse)
	}

	authInterceptor.AccessToken = newResponse.AccessToken

	requestMessage := desktopPB.EstimationRequest{
		Bla: "blank",
	}

	clientDesktopGateway := desktopPB.NewPowerEstimationServicesClient(connDesktopGateway)

	response, err := clientDesktopGateway.PowerEstimationSP(desktopContext, &requestMessage)
	if err != nil {
		fmt.Println(err)
	} else {
		fmt.Println(response.PowerEstimate[1])
	}
}

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

func createInsecureServerConnection(port string, timeout int, interceptor grpc.UnaryClientInterceptor) (*grpc.ClientConn, error) {
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
		fmt.Println("Failed to create connection to the server on port: " + port)
		return nil, err
	} else {
		fmt.Println("Succesfully created connection to the server on port: " + port)
		return conn, nil
	}
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
		fmt.Println("Failed to create connection to the server on port: " + port)
		return nil, err
	}

	fmt.Println("Succesfully created connection to the server on port: " + port)
	return conn, nil
}
