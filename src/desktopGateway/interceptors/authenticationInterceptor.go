package interceptors

import (
	"context"
	"log"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	// Personal packages
	authentication "github.com/nicholasbunn/masters/src/authenticationStuff"
)

func accessibleRoles() map[string][]string {
	return map[string][]string{
		"/src/fetchDataService":   {"admin"},
		"/src/prepareDataService": {"admin"},
		"/src/estimateService":    {"admin"},
	}
}

type ClientAuthStruct struct {
}

type ServerAuthStruct struct {
	jwtManager      *authentication.JWTManager
	accessibleRoles map[string][]string
}

func (interceptor *ClientAuthStruct) ClientAuthInterceptor(ctx context.Context, method string, req interface{}, reply interface{}, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
	// TO BE IMPLEMENTED: inject token into metadata
	return invoker(ctx, method, req, reply, cc, opts...)
}

func (interceptor *ServerAuthStruct) ServerAuthInterceptor(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
	log.Println("Starting authentication interceptor")

	err := interceptor.authorise(ctx, info.FullMethod)
	if err != nil {
		return nil, err
	}

	return handler(ctx, req)
}

func (interceptor *ServerAuthStruct) authorise(ctx context.Context, method string) error {
	/* This (unexported) function goes through a series of checks to verify that the user making a request is properly
	authenticated for that request */

	// Check if the method requires authentication
	accessibleRoles, ok := interceptor.accessibleRoles[method]
	if !ok {
		// If the method is not in the map then it means that it is publicly accessible
		log.Println("Authentication is not required for ", method)
		return nil
	}

	// Check if the request has metadata attached to it
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return status.Errorf(codes.Unauthenticated, "metadata is not provided")
	}

	// Check if a JWT has been included in the metadata
	values := md["authorisation"]
	if len(values) == 0 {
		return status.Errorf(codes.Unauthenticated, "authentication token has not been provided")
	}

	// Check that the provided JWT is valid
	accessToken := values[0]
	claims, err := interceptor.jwtManager.VerifyJWT(accessToken)
	if err != nil {
		return status.Errorf(codes.Unauthenticated, "access token is invalid: %v", err)
	}

	// Check that the role of the user making the service call authenticates them for the service being called
	for _, role := range accessibleRoles {
		if role == claims.Role {
			log.Printf("Succesfully authenticated request for ", method)
			return nil
		}
	}

	return status.Error(codes.PermissionDenied, "user does not have permission to access this RPC")
}
