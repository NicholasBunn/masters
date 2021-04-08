package main

import (
	"context"
	"fmt"
	"log"
	"os/exec"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"

	estimateServicePB "github.com/nicholasbunn/masters/src/estimateService/proto"
	fetchDataServicePB "github.com/nicholasbunn/masters/src/fetchDataService/proto"
	prepareDataServicePB "github.com/nicholasbunn/masters/src/prepareDataService/proto"

	"github.com/nicholasbunn/masters/src/powerEstimationSP/interceptors"
)

var (
	addrFS          = "localhost:50051"
	addrPS          = "localhost:50052"
	addrES          = "localhost:50053"
	timeoutDuration = 5                                       // The time, in seconds, that the client should wait when dialing (connecting to) the server before throwing an error
	INPUTfilename   = "TestData/CMU_2019_2020_openWater.xlsx" // MEEP Need to pass a path relative to the execution directory
	MODELTYPE       = "OPENWATER"
)

// MEEP Implement switch case to deal with user input for model type
// MEEP Figure out how to shutdown python server and possibly pass number of service calls as arguments
// MEEP Should errors be fatal, or should the program run regardless? Reconsider error handling on a case-by-case basis
// MEEP Implement secure connections
// MEEP Set service call timeout values
// MEEP Set timeout values for gRPC Dial
// MEEP Try to get the information to be streamed

func main() {
	fmt.Println("Started GoLang Aggregator")

	// Spin up low-level services
	interpretersSlice := []string{"python3", "python3", "python3"}
	directoriesSlice := []string{"./src/fetchDataService/", "./src/prepareDataService/", "./src/estimateService/"}
	filenamesSlice := []string{"fetchServer.py", "prepareServer.py", "estimateServer.py"}

	_ = SpinUpServices(interpretersSlice, directoriesSlice, filenamesSlice)

	// First invoke fetchserver
	/* Create connection to the Python server. Here you need to use the WithInsecure option because
	the Python server doesn't support secure connections. */

	callCounterFS := interceptors.OutboundCallCounter{}
	connFS := CreatePythonServerConnection(addrFS, timeoutDuration, callCounterFS.ClientCallCounter)

	callCounterPS := interceptors.OutboundCallCounter{}
	connPS := CreatePythonServerConnection(addrPS, timeoutDuration, callCounterPS.ClientCallCounter)

	callCounterES := interceptors.OutboundCallCounter{}
	connES := CreatePythonServerConnection(addrES, timeoutDuration, callCounterES.ClientCallCounter)

	/* Create the client and pass the connection made above to it. After the client has been
	created, we create the gRPC request */
	clientFS := fetchDataServicePB.NewFetchDataClient(connFS)
	clientPS := prepareDataServicePB.NewPrepareDataClient(connPS)
	clientES := estimateServicePB.NewEstimatePowerClient(connES)
	fmt.Println("Succesfully created the GoLang clients")

	requestMessageFS := fetchDataServicePB.FetchDataRequestMessage{
		InputFile: INPUTfilename,
	}
	fmt.Println("Succesfully created a FetchDataRequestMessage")

	// Create header to read the metadat that the response carries
	var headerFS metadata.MD // MEEP: Header has no information in it yet, this is filled by the server

	// Make the gRPC service call
	fetchDataContext, _ := context.WithTimeout(context.Background(), 5*time.Second)
	responseMessageFS, errFS := clientFS.FetchDataService(fetchDataContext, &requestMessageFS, grpc.Header(&headerFS)) // The responseMessageFS is a RawDataMessage
	if errFS != nil {
		fmt.Println("Failed to make FetchData service call: ")
		log.Fatal(errFS)
	}
	fmt.Println("Succesfully made service call to Python fetchDataServer.")
	fmt.Println("The fetch service client has performed ", callCounterFS, " calls.")
	connFS.Close()

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

	prepareDataContext, _ := context.WithTimeout(context.Background(), 5*time.Second) // MEEP could still use the cancelFunc, come back to this
	// Invoke prepareserver and pass fetchserver outputs as arguements

	responseMessagePS, errPS := clientPS.PrepareEstimateDataService(prepareDataContext, &requestMessagePS)

	if errPS != nil {
		fmt.Println("Failed to make PrepareData service call: ")
		log.Fatal(errPS)
	}
	fmt.Println("Succesfully made service call to python prepareDataServer.")
	connPS.Close()

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

	switch MODELTYPE {
	case "OPENWATER":
		requestMessageES.ModelType = estimateServicePB.ModelTypeEnum_OPENWATER
	case "ICE":
		requestMessageES.ModelType = estimateServicePB.ModelTypeEnum_ICE
	case "UNKNOWN":
		requestMessageES.ModelType = estimateServicePB.ModelTypeEnum_OPENWATER
	default:
		requestMessageES.ModelType = estimateServicePB.ModelTypeEnum_OPENWATER
	}

	fmt.Println("Succesfully created an EstimateRequestMessage")

	// Invoke estimateserver and pass prepareserver outputs as arguements
	estimateContext, _ := context.WithTimeout(context.Background(), 5*time.Second) // MEEP could still use the cancelFunc, come back to this
	responseMessageES, errES := clientES.EstimatePowerService(estimateContext, &requestMessageES)
	if errES != nil {
		fmt.Println("Failed to make Estimate service call: ")
		log.Fatal(errES)
	}
	fmt.Println("Succesfully made service call to Python estimateServer")
	connPS.Close()
	fmt.Println(responseMessageES.PowerEstimate[1]) // MEEP remove once you've done something with responseMEssageFS
}

func SpinUpServices(interpreter []string, directories []string, filenames []string) bool {
	// Check that the 'directories' and 'filenames' are of the same length before iterating through them
	if len(directories) != len(filenames) {
		fmt.Println("The 'directories' and 'filenames' slices passed into the 'SpinUpSerivces' function are not of equal lengths")
		log.Fatal()
		return false // These are here for error handling when I get around to it, won't execute at the moment
	} else {
		// Reusable variables
		fileLocation := ""
		var cmd *exec.Cmd
		var err error

		// Iterate through the required services and start them up
		for i := range directories {
			fmt.Println("Invoking " + interpreter[i] + " service: " + filenames[i])
			fileLocation = directories[i] + filenames[i]
			cmd = exec.Command(interpreter[i], fileLocation)
			err = cmd.Start()
			if err != nil {
				fmt.Println("Failed to invoke {}", filenames[i])
				log.Fatal(err)
				return false
			}
		}

		return true
	}
}

func CreatePythonServerConnection(port string, timeout int, interceptor grpc.UnaryClientInterceptor) *grpc.ClientConn {
	conn, err := grpc.Dial(port, grpc.WithInsecure(), grpc.WithBlock(), grpc.WithTimeout(time.Duration(timeoutDuration)*time.Second), grpc.WithUnaryInterceptor(interceptor))
	if err != nil {
		fmt.Println("Failed to create connection to Python server on port: " + port)
		log.Fatal(err)
	}
	fmt.Println("Succesfully created connection to the Python server on port: " + port)

	return conn
}
