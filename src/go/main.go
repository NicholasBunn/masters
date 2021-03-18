package main

import (
	"context"
	"fmt"
	"log"
	"os/exec"
	"time"

	powerpb "github.com/nicholasbunn/SANAE60/src/go/proto"
	"google.golang.org/grpc"
)

var (
	addrFS          = "localhost:50051"
	addrPS          = "localhost:50052"
	addrES          = "localhost:50053"
	timeoutDuration = 5                                                  // The time, in seconds, that the client should wait when dialing (connecting to) the server before throwing an error
	INPUTFILENAME   = "src/python/estimate/CMU_2019_2020_openWater.xlsx" // MEEP Need to pass a path relative to the execution directory
	MODELTYPE       = "OPENWATER"
)

// MEEP Implement switch case to deal with user input for model type
// MEEP Figure out how to shutdown python server and possibly pass number of service calls as arguments
/* MEEP Use .Run() rather than .Start() and start each in a new GoRoutine as .Run() is a blocking call. Also, figure
out how to let this script know that the .Run() has completed succesfully. Then set complete flag (or incomplete flag
if it's done in error handling) to be checked before service calls are made. This will protect program lock if the service isn't running. */
// MEEP Should errors be fatal, or should the program run regardless?
// MEEP Implement secure connections
// MEEP Set service call timeout values
// MEEP Set timeout values for gRPC Dial
// MEEP Try to get the information to be streamed

func main() {
	// Spin up Python services
	fmt.Println("Invoking Python services")
	fetchServerCmd := exec.Command("python", "./src/python/estimate/fetchServer.py")
	errFSC := fetchServerCmd.Start() // MEEP use Run() or Start()?
	if errFSC != nil {
		/* MEEP Do I want a fatal error here, or should I rather block service calls requiring
		that services' information? This decision will effect how you handle errors when
		connecting to servers too */
		fmt.Println("Failed to invoke Python fetchServer: ")
		log.Fatal(errFSC)
	}

	prepareServerCmd := exec.Command("python", "./src/python/estimate/prepareServer.py")
	errPSC := prepareServerCmd.Start()
	if errPSC != nil {
		fmt.Println("Failed to invoke Python prepareServer")
		log.Fatal(errPSC)
	}

	estimateServerCmd := exec.Command("python", "./src/python/estimate/estimateServer.py")
	errESC := estimateServerCmd.Start()
	if errESC != nil {
		fmt.Println("Failed to invoke Python estimateServer")
		log.Fatal(errESC)
	}

	fmt.Println("Started GoLang Aggregator")

	// First invoke fetchserver
	/* Create connection to the Python server. Here you need to use the WithInsecure option because
	the Python server doesn't support secure connections. */
	connFS, err := grpc.Dial(addrFS, grpc.WithInsecure(), grpc.WithBlock(), grpc.WithTimeout(time.Duration(timeoutDuration)*time.Second)) // MEEP set timeout
	if err != nil {
		fmt.Println("Failed to create connection to the Python fetchServer: ")
		log.Fatal(err)
	}
	defer connFS.Close()
	fmt.Println("Succesfully created connection to the Python fetchServer.")

	connPS, err := grpc.Dial(addrPS, grpc.WithInsecure(), grpc.WithBlock(), grpc.WithTimeout(time.Duration(timeoutDuration)*time.Second)) // MEEP set timeout
	if err != nil {
		fmt.Println("Failed to create connection to the Python prepareServer: ")
		log.Fatal(err)
	}
	defer connFS.Close()
	fmt.Println("Succesfully created connection to the Python prepareServer.")

	connES, err := grpc.Dial(addrES, grpc.WithInsecure(), grpc.WithBlock(), grpc.WithTimeout(time.Duration(timeoutDuration)*time.Second)) // MEEP set timeout
	if err != nil {
		fmt.Println("Failed to create connection to the Python estimateServer: ")
		log.Fatal(err)
	}
	defer connFS.Close()
	fmt.Println("Succesfully created connection to the Python estimateServer.")

	/* Create the client and pass the connection made above to it. After the client has been
	created, we create the gRPC request */
	clientFS := powerpb.NewFetchDataClient(connFS)
	clientPS := powerpb.NewPrepareDataClient(connPS)
	clientES := powerpb.NewPowerEstimateClient(connES)
	fmt.Println("Succesfully created the GoLang client")

	requestMessageFS := powerpb.DataRequestMessage{
		InputFile: INPUTFILENAME,
	}
	fmt.Println("Succesfully created a DataRequestMessage")

	// Make the gRPC service call
	fetchDataContext, _ := context.WithTimeout(context.Background(), 5*time.Second)            // MEEP could still use the cancelFunc, come back to this
	responseMessageFS, errFS := clientFS.FetchDataService(fetchDataContext, &requestMessageFS) // The responseMessageFS is a RawDataMessage
	if errFS != nil {
		fmt.Println("Failed to make FetchData service call: ")
		log.Fatal(errFS)
	}
	fmt.Println("Succesfully made service call to Python fetchServer.")

	prepareDataContext, _ := context.WithTimeout(context.Background(), 5*time.Second) // MEEP could still use the cancelFunc, come back to this
	// Invoke prepareserver and pass fetchserver outputs as arguements
	responseMessagePS, errPS := clientPS.PrepareEstimateDataService(prepareDataContext, responseMessageFS)
	if errPS != nil {
		fmt.Println("Failed to make PrepareData service call: ")
		log.Fatal(errPS)
	}
	fmt.Println("Succesfully made service call to python prepareServer.")

	requestMessageES := powerpb.EstimateRequestMessage{
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
	switch MODELTYPE {
	case "OPENWATER":
		requestMessageES.ModelType = powerpb.ModelTypeEnum_OPENWATER
	case "ICE":
		requestMessageES.ModelType = powerpb.ModelTypeEnum_ICE
	case "UNKNOWN":
		requestMessageES.ModelType = powerpb.ModelTypeEnum_OPENWATER
	default:
		requestMessageES.ModelType = powerpb.ModelTypeEnum_OPENWATER
	}
	fmt.Println("Succesfully created an EstimateRequestMessage")

	// Invoke estimateserver and pass prepareserver outputs as arguements
	estimateContext, _ := context.WithTimeout(context.Background(), 5*time.Second) // MEEP could still use the cancelFunc, come back to this
	responseMessageES, errES := clientES.EstimateService(estimateContext, &requestMessageES)
	if errES != nil {
		fmt.Println("Failed to make Estimate service call: ")
		log.Fatal(errES)
	}
	fmt.Println("Succesfully made service call to Python estimateServer")
	fmt.Println(responseMessageES.PowerEstimate) // MEEP remove once you've done something with responseMEssageFS
}
