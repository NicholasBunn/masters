package main

import (
	"fmt"
	"net"
	"testing"
	"time"
)

// ToDo, check ports don't have connections before spinning up services?

func SpinUpCheck(compiler []string, directory []string, filename []string, ports []string) bool {
	// SpinUpServices(compiler, directory, filename)

	// Simply check that the server is up and can
	// accept connections.
	_, err := net.Listen("tcp", ports[0])
	if err != nil {
		fmt.Println("SpinUpServices succesfully started ", compiler, " service with inputs: ", directory, ", ", filename, ".")
		return true
	} else {
		return false
	}
}

func TestSpinUpService(t *testing.T) {
	// Spin up low-level services
	var tests = []struct {
		inputCompiler  []string
		inputDirectory []string
		inputFilename  []string
		ports          []string
		output         bool
	}{
		{[]string{"python3"}, []string{"/home/nic/go/src/github.com/nicholasbunn/masters/src/fetchDataService/"}, []string{"fetchServer.py"}, []string{":50051"}, true}, // Tests for core functionality of single service (starting of Python service)
		// {[]string{"python3", "python3"}, []string{"./src/fetchDataService/", "./src/estimateService/"}, []string{"fetchServer.py"}, {":50051", ":50052"}, false},                     // Tests for missing inputs (length mismatch of directories and filenames)
		{[]string{"python3", "python3"}, []string{"home/nic/go/src/github.com/nicholasbunn/masters/src/fetchDataService/", "home/nic/go/src/github.com/nicholasbunn/masters/src/estimateService/"}, []string{"fetchServer.py", "estimateServer.py"}, []string{":50051", ":50052"}, true}, // Tests for core functionality of multiple services
	}

	for _, test := range tests {
		fmt.Println("Starting spin up service test with inputs: ", test.inputCompiler, ", ", test.inputDirectory, ", ", test.inputFilename, ". Service is starting on port ", test.ports)
		result := SpinUpServices(test.inputCompiler, test.inputDirectory, test.inputFilename)
		time.Sleep(1 * time.Second) // Bad practice, but I need to wait for the server to go online before checking for it. Maybe add a check in the SpinUpServiceFunction instead

		if !result {
			t.Error("Spin up service function failed to execute.")
		} else {
			output := SpinUpCheck(test.inputCompiler, test.inputDirectory, test.inputFilename, test.ports)

			if output != test.output {
				t.Error("Spin up service test failed with inputs: ", test.inputCompiler, ", ", test.inputDirectory, ", ", test.inputFilename, ".\n Expected ", test.output, ", received ", output)
			} else {
				fmt.Println("Passed")
			}
		}
	}

}

// func TestCreatePythonServerConnection() {
// 	// Still needs to be implemented
// }
