package interceptors

import (
	// Native packages

	"context"
	"log"
	"os"
	"strings"

	// gRPC packages
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	// Personal packages
	authentication "github.com/nicholasbunn/mastersSandbox/src/authenticationStuff"
)

var (
	// Logging stuff
	DebugLogger   *log.Logger
	InfoLogger    *log.Logger
	WarningLogger *log.Logger
	ErrorLogger   *log.Logger
)

func init() {
	/* The init functin is used to set up the logger and metric interceptors whenever the service is started
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

type ClientAuthStruct struct {
	AccessToken          string
	AuthenticatedMethods map[string]bool
}

type ServerAuthStruct struct {
	JwtManager           *authentication.JWTManager
	AuthenticatedMethods map[string][]string
}

func (interceptor *ClientAuthStruct) ClientAuthInterceptor(ctx context.Context, method string, req interface{}, reply interface{}, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
	InfoLogger.Println("Starting client-side authentication interceptor")
	log.Println(method)

	// Always inject JWT, even if the requested service is publically available. This removes the need for the frontend to know of what calls are on offer
	InfoLogger.Println("Injecting JWT into metadata")
	return invoker(interceptor.attachToken(ctx), method, req, reply, cc, opts...)

	// InfoLogger.Println("Requested method is publically available")
	// return invoker(ctx, method, req, reply, cc, opts...)
}

func (interceptor *ServerAuthStruct) ServerAuthInterceptor(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
	InfoLogger.Println("Starting server-side authentication interceptor")

	err := interceptor.authorise(ctx, info.FullMethod)
	if err != nil {
		return nil, err
	}

	return handler(ctx, req)
}

func (interceptor *ClientAuthStruct) attachToken(ctx context.Context) context.Context {
	return metadata.AppendToOutgoingContext(ctx, "authorisation", interceptor.AccessToken)
}

func (interceptor *ServerAuthStruct) authorise(ctx context.Context, method string) error {
	/* This (unexported) function goes through a series of checks to verify that the user making a request is properly
	authenticated for that request */

	// Check if the method requires authentication
	accessibleRoles, ok := interceptor.AuthenticatedMethods[method]
	if !ok {
		// If the method is not in the map then it means that the method is publicly accessible
		InfoLogger.Println("Authentication is not required for ", method)
		return nil
	}

	// Check if the request has metadata attached to it
	md, ok := metadata.FromIncomingContext(ctx)

	if !ok {
		DebugLogger.Println("Failed to authenticate: metadata is not provided")
		return status.Errorf(codes.PermissionDenied, "metadata is not provided")
	}

	// Check if a JWT has been included in the metadata
	values := md["authorisation"]
	if len(values) == 0 {
		DebugLogger.Println("Failed to authenticate: JWT has not been provided")
		return status.Errorf(codes.PermissionDenied, "authentication token has not been provided")
	}

	// Check that the provided JWT is valid
	accessToken := values[0]
	claims, err := interceptor.JwtManager.VerifyJWT(accessToken)
	if err != nil {
		DebugLogger.Println("Failed to authenticate: Provided JWT is invalid")
		return status.Errorf(codes.PermissionDenied, "access token is invalid: %v", err)
	}

	// Check that the role of the user making the service call authenticates them for the service being called
	for _, role := range accessibleRoles {
		if role == claims.Role {
			DebugLogger.Println("Succesfully authenticated request for ", method)
			return nil
		}
	}

	DebugLogger.Println("Failed to authenticate: the user does not have permission to access the requested service")
	return status.Error(codes.PermissionDenied, "user does not have permission to access this RPC")
}
