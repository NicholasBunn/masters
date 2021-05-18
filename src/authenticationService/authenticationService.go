package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"os"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	// Proto packages
	serverPB "github.com/nicholasbunn/masters/src/authenticationService/proto"

	// Personal packages
	authentication "github.com/nicholasbunn/masters/src/authenticationStuff"
)

var (
	// Addresses (To be passed in a config file)
	addrMyself = "localhost:50401"

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

	secretKey     = "secret"
	tokenDuration = 15 * time.Minute
)

func init() {
	// Set up logger
	// If the file doesn't exist, create it, otherwise append to the file
	file, err := os.OpenFile("program logs/"+"authenticationService.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0666)
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
	// Create a listener on the specified tcp port
	listener, err := net.Listen("tcp", addrMyself)
	if err != nil {
		ErrorLogger.Fatalf("Failed to listen on port %v: \n%v", addrMyself, err)
	}
	InfoLogger.Println("Listening on port: ", addrMyself)

	// Create a gRPC server object
	authenticationServer := grpc.NewServer(
	// grpc.UnaryInterceptor(interceptors.AuthenticationInterceptor),
	// grpc.Creds(creds),
	)

	serverPB.RegisterAuthenticationServiceServer(authenticationServer, &authServer{})
	DebugLogger.Println("Succesfully registered Power Estimation Services to the server")

	// Start the server
	if err := authenticationServer.Serve(listener); err != nil {
		ErrorLogger.Fatalf("Failed to expose service: \n%v", err)
	}
}

// Use this to implement the authentication service
type authServer struct {
	serverPB.UnimplementedAuthenticationServiceServer
}

func (s *authServer) LoginAuth(ctx context.Context, request *serverPB.LoginAuthRequest) (*serverPB.LoginAuthResponse, error) {
	// 1. Find the user with the provided username, return an error if they don't exist
	user, err := Find(request.GetUsername())
	if err != nil {
		return nil, status.Errorf(codes.Internal, "cannot find user: %v", err)
	}

	// 2. Test if the provided username or password are correct for the user, return an error if they aren't not
	if user == nil {
		return nil, status.Errorf(codes.NotFound, "the username you provided doesn't exist")
	} else if !user.CheckPassword(request.GetPassword()) {
		return nil, status.Errorf(codes.NotFound, "incorrect username/password")
	}

	// 3. Generate a JWT
	jwtManager := authentication.JWTManager{
		SecretKey:     secretKey,
		TokenDuration: tokenDuration,
	}

	token, err := jwtManager.GenerateManager(user)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "cannot generate access token")
	}

	// Return the access token
	response := &serverPB.LoginAuthResponse{AccessToken: token}
	return response, nil
}

// ________USER RELATED CODE________

// ________USER DB-RELATED CODE________
func Save(user *authentication.User) error {
	// Still need to implement
	return nil
}

func Find(username string) (*authentication.User, error) {
	// Still need to implement

	user, err := authentication.CreateUser("admin1", "secret", "guest")
	if err != nil {
		return nil, fmt.Errorf(codes.Internal.String(), "could not create user")
	}

	return user, nil
}

// ________JWT RELATED CODE________
