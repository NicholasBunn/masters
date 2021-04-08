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
	var Tests = []struct {
		inputCompiler  []string
		inputDirectory []string
		inputFilename  []string
		ports          []string
		expectedOutput bool
	}{
		{[]string{"python3"}, []string{"/home/nic/go/src/github.com/nicholasbunn/masters/src/fetchDataService/"}, []string{"fetchServer.py"}, []string{":50051"}, true}, // Tests for core functionality of single service (starting of Python service)
		// {[]string{"python3", "python3"}, []string{"./src/fetchDataService/", "./src/estimateService/"}, []string{"fetchServer.py"}, {":50051", ":50052"}, false},                     // Tests for missing inputs (length mismatch of directories and filenames)
		{[]string{"python3", "python3"}, []string{"home/nic/go/src/github.com/nicholasbunn/masters/src/fetchDataService/", "home/nic/go/src/github.com/nicholasbunn/masters/src/estimateService/"}, []string{"fetchServer.py", "estimateServer.py"}, []string{":50051", ":50052"}, true}, // Tests for core functionality of multiple services
	}

	t.Run("Testing for core functionality of single Python service", func(t *testing.T) {
		// Tests for core functionality of single service (starting of Python service)
		testCaseOne := Tests[0]

		fmt.Println("Starting spin up service test with inputs: ", testCaseOne.inputCompiler, ", ", testCaseOne.inputDirectory, ", ", testCaseOne.inputFilename, ". Service is starting on port ", testCaseOne.ports)
		result := SpinUpServices(testCaseOne.inputCompiler, testCaseOne.inputDirectory, testCaseOne.inputFilename)
		time.Sleep(800 * time.Millisecond) // Bad practice, but I need to wait for the server to go online before checking for it. Maybe add a check in the SpinUpServiceFunction instead

		if !result {
			t.Error("Spin up service function failed to execute.")
		} else {
			output := SpinUpCheck(testCaseOne.inputCompiler, testCaseOne.inputDirectory, testCaseOne.inputFilename, testCaseOne.ports)

			if output != testCaseOne.expectedOutput {
				t.Error("Spin up service test failed with inputs: ", testCaseOne.inputCompiler, ", ", testCaseOne.inputDirectory, ", ", testCaseOne.inputFilename, ".\n Expected ", testCaseOne.expectedOutput, ", received ", output)
			} else {
				fmt.Println("Passed")
			}
		}
	})

	t.Run("Testing for core functionality of multiple Python services", func(t *testing.T) {
		// Tests for core functionality of single service (starting of Python service)
		testCaseTwo := Tests[1]

		fmt.Println("Starting spin up service test with inputs: ", testCaseTwo.inputCompiler, ", ", testCaseTwo.inputDirectory, ", ", testCaseTwo.inputFilename, ". Service is starting on port ", testCaseTwo.ports)
		result := SpinUpServices(testCaseTwo.inputCompiler, testCaseTwo.inputDirectory, testCaseTwo.inputFilename)
		time.Sleep(800 * time.Millisecond) // Bad practice, but I need to wait for the server to go online before checking for it. Maybe add a check in the SpinUpServiceFunction instead

		if !result {
			t.Error("Spin up service function failed to execute.")
		} else {
			output := SpinUpCheck(testCaseTwo.inputCompiler, testCaseTwo.inputDirectory, testCaseTwo.inputFilename, testCaseTwo.ports)

			if output != testCaseTwo.expectedOutput {
				t.Error("Spin up service test failed with inputs: ", testCaseTwo.inputCompiler, ", ", testCaseTwo.inputDirectory, ", ", testCaseTwo.inputFilename, ".\n Expected ", testCaseTwo.expectedOutput, ", received ", output)
			} else {
				fmt.Println("Passed")
			}
		}
	})

	// {[]string{"python3", "python3"}, []string{"./src/fetchDataService/", "./src/estimateService/"}, []string{"fetchServer.py"}, {":50051", ":50052"}, false},                     // Tests for missing inputs (length mismatch of directories and filenames)

}

// func TestCreatePythonServerConnection() {
// 	// Still needs to be implemented
// }
