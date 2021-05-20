package main

import (
	"context"
	"fmt"
	"time"

	grpc_middleware "github.com/grpc-ecosystem/go-grpc-middleware"
	"google.golang.org/grpc"

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
	metricInterceptor := interceptors.ClientMetricStruct{}
	authInterceptor := interceptors.ClientAuthStruct{}
	interceptorChain := grpc_middleware.ChainUnaryClient(
		metricInterceptor.ClientMetricInterceptor,
		authInterceptor.ClientAuthInterceptor,
	)

	connDesktopGateway, err := CreateInsecureServerConnection(addrDesktopGateway, timeoutDuration, interceptorChain)

	clientLoginDesktopGateway := desktopPB.NewLoginServiceClient(connDesktopGateway)

	desktopContext, _ := context.WithTimeout(context.Background(), callTimeoutDuration)

	loginRequest := desktopPB.LoginRequest{
		Username: "guest1",
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
		fmt.Println("Failed")
	} else {
		fmt.Println(response.PowerEstimate[1])
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
		fmt.Println("Failed to create connection to the server on port: " + port)
		return nil, err
	} else {
		fmt.Println("Succesfully created connection to the server on port: " + port)
		return conn, nil
	}
}
