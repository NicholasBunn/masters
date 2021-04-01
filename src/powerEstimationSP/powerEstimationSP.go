package main

// Just putting this here to test a branch publish
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
	INPUTFILENAME   = "TestData/CMU_2019_2020_openWater.xlsx" // MEEP Need to pass a path relative to the execution directory
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
	SpinUpService("python3", "./src/fetchDataService/", "fetchServer.py")

	SpinUpService("python3", "./src/prepareDataService/", "prepareServer.py")

	SpinUpService("python3", "./src/estimateService/", "estimateServer.py")

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
		InputFile: INPUTFILENAME,
	}
	fmt.Println("Succesfully created a FetchDataRequestMessage")

	// Create header to read the metadat that the response carries
	var headerFS metadata.MD // MEEP: Header has no information in it yet, this is filled by the server

	// Make the gRPC service call
	fetchDataContext, _ := context.WithTimeout(context.Background(), 5*time.Second)                                    // MEEP could still use the cancelFunc, come back to this
	responseMessageFS, errFS := clientFS.FetchDataService(fetchDataContext, &requestMessageFS, grpc.Header(&headerFS)) // The responseMessageFS is a RawDataMessage
	if errFS != nil {
		fmt.Println("Failed to make FetchData service call: ")
		log.Fatal(errFS)
	}
	fmt.Println("Succesfully made service call to Python fetchDataServer.")
	fmt.Println("The fetch service client has performed %s calls.", callCounterFS)
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

func SpinUpService(interpreter string, directory string, fileName string) {
	fmt.Println("Invoking " + interpreter + " service: " + fileName)
	fileLocation := directory + fileName
	cmd := exec.Command(interpreter, fileLocation)
	err := cmd.Start()
	if err != nil {
		fmt.Println("Failed to invoke %s: ", fileName)
		log.Fatal(err)
	}
}

// MEEP: this should just accept a list of the services and iterate through it!
func CreatePythonServerConnection(port string, timeout int, interceptor grpc.UnaryClientInterceptor) *grpc.ClientConn {
	conn, err := grpc.Dial(port, grpc.WithInsecure(), grpc.WithBlock(), grpc.WithTimeout(time.Duration(timeoutDuration)*time.Second), grpc.WithUnaryInterceptor(interceptor))
	if err != nil {
		fmt.Println("Failed to create connection to Python server on port: " + port)
		log.Fatal(err)
	}
	fmt.Println("Succesfully created connection to the Python server on port: " + port)

	return conn
}
