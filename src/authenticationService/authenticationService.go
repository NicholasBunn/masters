package main

import (
	// Native packages
	"context"
	"log"
	"net"
	"os"
	"strings"
	"time"

	// Required packages
	authentication "github.com/nicholasbunn/masters/src/authenticationStuff"

	// gRPC packages
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	// Proto packages
	serverPB "github.com/nicholasbunn/masters/src/authenticationService/proto"
)

var (
	// Addresses (To be passed in a config file)
	addrMyself = os.Getenv("AUTHENTICATIONHOST") + ":50401"

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

	// JWT token information (to be passed in a config file)
	secretKey     = "secret"
	tokenDuration = 15 * time.Minute
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

	InfoLogger.Println("Stated authentication service")

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

	// Create a gRPC server object
	authenticationServer := grpc.NewServer(
	// grpc.Creds(creds), // Add the TLS credentials to this server
	// grpc.UnaryInterceptor(interceptors.AuthenticationInterceptor), // Add the interceptor chain to this server
	)

	// Attach the authentication service offering to the server
	serverPB.RegisterAuthenticationServiceServer(authenticationServer, &authServer{})
	DebugLogger.Println("Succesfully registered Authentication Service to the server")

	// Start the server
	if err := authenticationServer.Serve(listener); err != nil {
		ErrorLogger.Fatalf("Failed to expose service: \n%v", err)
	}
}

// ________STRUCTS TO IMPLEMENT THE OFFERED SERVICES________

type authServer struct {
	// Use this to implement the authentication service

	serverPB.UnimplementedAuthenticationServiceServer
}

// ________IMPLEMENT THE OFFERED SERVICES________

func (s *authServer) LoginAuth(ctx context.Context, request *serverPB.LoginAuthRequest) (*serverPB.LoginAuthResponse, error) {
	/* This service logs the user in by checking the provided details against a user
	database. If the user exists, a JWT is generated and returned to them. */

	InfoLogger.Println("Received LoginAuth service call")
	// Find the user with the provided username, return a NotFound error if they don't exist
	user, err := find(request.GetUsername())
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "cannot find user: %v", err)
	}

	// Check if a user with the provided username and password combination exists, return a NotFound error if they don't
	if user == nil {
		return nil, status.Errorf(codes.NotFound, "the username you provided doesn't exist")
	} else if !user.CheckPassword(request.GetPassword()) {
		return nil, status.Errorf(codes.NotFound, "the password you provided is incorrect")
	}

	// Create a jwtManager object for the user
	jwtManager := authentication.JWTManager{
		SecretKey:     secretKey,
		TokenDuration: tokenDuration,
	}

	// Generate and return a JWT for the user
	token, err := jwtManager.GenerateManager(user)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "could not generate access token")
	}

	// Create and populate the response message for the request being served
	response := &serverPB.LoginAuthResponse{
		Permissions: user.Role,
		AccessToken: token,
	}

	return response, nil
}

// ________SUPPORTING FUNCTIONS________

func save(user *authentication.User) error {
	// Still need to implement
	return nil
}

func find(username string) (*authentication.User, error) {
	// Still need to implement
	if username == "admin" {
		user, err := authentication.CreateUser("admin", "myPassword", "admin")
		if err != nil {
			return nil, status.Errorf(codes.Internal, "could not create user")
		}
		return user, nil
	}

	user, err := authentication.CreateUser("guest", "myPassword", "guest")
	if err != nil {
		return nil, status.Errorf(codes.Internal, "could not create user")
	}

	return user, nil
}
