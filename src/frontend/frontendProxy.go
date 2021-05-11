package main

import (
	"context"
	"fmt"
	"time"

	desktopPB "github.com/nicholasbunn/masters/src/desktopGateway/proto"
	"github.com/nicholasbunn/masters/src/frontend/interceptors"
	"google.golang.org/grpc"
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
	// loginRequest := desktopPB.LoginRequest{
	// 	Username:       "devUsername",
	// 	HashedPassword: "HashedDevPassword",
	// }

	callCounter := interceptors.ClientMetricStruct{}
	connDesktopGateway := CreateInsecureServerConnection(addrDesktopGateway, timeoutDuration, callCounter.ClientMetrics)

	clientDesktopGateway := desktopPB.NewPowerEstimationServicesClient(connDesktopGateway)

	requestMessage := desktopPB.EstimationRequest{
		Bla: "blank",
	}

	desktopContext, _ := context.WithTimeout(context.Background(), callTimeoutDuration)

	response, err := clientDesktopGateway.PowerEstimationSP(desktopContext, &requestMessage)
	if err != nil {
		fmt.Println("Failed")
	} else {
		fmt.Println(response.PowerEstimate[1])
	}

}

func CreateInsecureServerConnection(port string, timeout int, interceptor grpc.UnaryClientInterceptor) *grpc.ClientConn {
	// This function takes a port address, credentials object, timeout, and an interceptor as an input, creates a connection to the server at the port adress and returns
	// a secure gRPC connection with the specified interceptor

	conn, err := grpc.Dial(port, grpc.WithInsecure(), grpc.WithBlock(), grpc.WithTimeout(time.Duration(timeoutDuration)*time.Second), grpc.WithUnaryInterceptor(interceptor))
	if err != nil {
		// ErrorLogger.Println("Failed to create connection to server on port: " + port)
		// ErrorLogger.Println(err)
		// ErrorLogger.Fatal(err)
	} else {
		// InfoLogger.Println("Succesfully created connection to the server on port: " + port)
	}
	return conn
}
