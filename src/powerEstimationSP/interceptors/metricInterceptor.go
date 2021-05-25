package interceptors

import (
	"bytes"
	"context"
	"encoding/binary"
	"encoding/gob"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"google.golang.org/grpc"

	prometheus "github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/push"
)

var (
	// Logging stuff
	DebugLogger   *log.Logger
	InfoLogger    *log.Logger
	WarningLogger *log.Logger
	ErrorLogger   *log.Logger
)

func init() {
	// Logger setup
	pathSlice := strings.Split(os.Args[0], "/")
	file, err := os.OpenFile("program logs/"+pathSlice[len(pathSlice)-1]+".log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0666)
	if err != nil {
		log.Fatal(err)
	}

	log.SetOutput(file)

	DebugLogger = log.New(file, "DEBUG: ", log.Ldate|log.Ltime|log.Lmicroseconds|log.Lshortfile)
	InfoLogger = log.New(file, "INFO: ", log.Ldate|log.Ltime|log.Lmicroseconds|log.Lshortfile)
	WarningLogger = log.New(file, "WARNING: ", log.Ldate|log.Ltime|log.Lmicroseconds|log.Lshortfile)
	ErrorLogger = log.New(file, "ERROR: ", log.Ldate|log.Ltime|log.Lmicroseconds|log.Lshortfile)
}

// This isn't actually being used right now, reconsider how you're implementing the client-side interceptor
type ClientMetricStruct struct {
	/* This struct represents a collection of client-side metrics to be registered on a
	Prometheus metrics registry */
	clientRequestCounter      *prometheus.CounterVec   // Counts the number of call made by the client
	clientResponseCounter     *prometheus.CounterVec   // Counts the number of responses received by the client
	clientRequestMessageSize  *prometheus.HistogramVec // Records the size of the request message sent out
	clientResponseMessageSize *prometheus.HistogramVec // Records the size of the response message received
}

type ServerMetricStruct struct {
	/* This struct represents a collection of server-side metrics to be reqistered on a
	Prometheus metrics registry */
	serverRequestCounter  *prometheus.CounterVec   // Counts the number of requests received by the server
	serverResponseCounter *prometheus.CounterVec   // Counts the number of responses sent by the server
	serverLastCallTime    *prometheus.GaugeVec     // Records the lat time a call was made to the server
	serverRequestLatency  *prometheus.HistogramVec // Records the amount of time the server took to serve the call
}

func NewClientMetrics() *ClientMetricStruct {
	return &ClientMetricStruct{
		clientRequestCounter: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "client_request_counter",
				Help: "The number of requests made by the client",
			}, []string{"grpc_type", "grpc_service", "grpc_method"}),
		clientResponseCounter: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "client_response_counter",
				Help: "The number of responses received by the client",
			}, []string{"grpc_type", "grpc_service", "grpc_method"}),
		clientRequestMessageSize: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Name: "client_request_size",
				Help: "The size (in bytes) of the request sent by the client",
			}, []string{"grpc_type", "grpc_service", "grpc_method"}),
		clientResponseMessageSize: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Name: "client_response_size",
				Help: "The size (in bytes) of the response received by the client",
			}, []string{"grpc_type", "grpc_service", "grpc_method"}),
	}
}

func NewServerMetrics() *ServerMetricStruct {
	return &ServerMetricStruct{
		serverRequestCounter: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "server_request_counter",
				Help: "The number of requests made to the server",
			}, []string{"grpc_type", "grpc_service", "grpc_method"}),
		serverResponseCounter: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "server_response_counter",
				Help: "The number of response sent by the server",
			}, []string{"grpc_type", "grpc_service", "grpc_method"}),
		serverLastCallTime: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "server_last_call_time",
				Help: "The last time a call was made to the server",
			}, []string{"grpc_type", "grpc_service", "grpc_method"}),
		serverRequestLatency: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Name: "server_request_latency",
				Help: "The time it took for the server to serve the request",
			}, []string{"grpc_type", "grpc_service", "grpc_method"}),
	}
}

func (metr *ClientMetricStruct) ClientMetricInterceptor(ctx context.Context, method string, req interface{}, reply interface{}, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
	// Client side interceptor, to be attached to all client connections

	InfoLogger.Println("Starting client interceptor method")

	// Extract service and method names
	requesterInfo := strings.Split(method, "/")
	serviceName := requesterInfo[1]
	serviceMethod := requesterInfo[2]

	// Increment the request call counter
	metr.clientRequestCounter.With(prometheus.Labels{"grpc_type": "unary", "grpc_service": serviceName, "grpc_method": serviceMethod}).Inc()

	// Record request size here
	size, _ := getMessageSize(req)
	metr.clientRequestMessageSize.With(prometheus.Labels{"grpc_type": "unary", "grpc_service": serviceName, "grpc_method": serviceMethod}).Observe(float64(size))

	// Run gRPC call here
	err := invoker(ctx, method, req, reply, cc, opts...)

	// Increment the response call counter
	metr.clientResponseCounter.With(prometheus.Labels{"grpc_type": "unary", "grpc_service": serviceName, "grpc_method": serviceMethod}).Inc()

	// Record response size here
	size, _ = getMessageSize(reply)
	metr.clientResponseMessageSize.With(prometheus.Labels{"grpc_type": "unary", "grpc_service": serviceName, "grpc_method": serviceMethod}).Observe(float64(size))

	// Push metrics to the pushgateway
	err = pushClientMetrics(metr, serviceName)

	return err
}

func (metr *ServerMetricStruct) ServerMetricInterceptor(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
	// Server-side interceptor, to be attached to all server connections

	InfoLogger.Println("Starting server interceptor method")

	// Extract service and method names
	requesterInfo := strings.Split(info.FullMethod, "/")
	serviceName := requesterInfo[1]
	serviceMethod := requesterInfo[2]

	// Increment the request call counter
	metr.serverRequestCounter.With(prometheus.Labels{"grpc_type": "unary", "grpc_service": serviceName, "grpc_method": serviceMethod}).Inc()

	// Set the last call time
	metr.serverLastCallTime.With(prometheus.Labels{"grpc_type": "unary", "grpc_service": serviceName, "grpc_method": serviceMethod}).SetToCurrentTime()

	// Start the call timer
	start := time.Now()

	// Run gRPC call here
	h, err := handler(ctx, req)

	// Set the call latency (response time)
	metr.serverRequestLatency.With(prometheus.Labels{"grpc_type": "unary", "grpc_service": serviceName, "grpc_method": serviceMethod}).Observe(float64(time.Since(start).Seconds()))

	// Increment the response call counter
	metr.serverResponseCounter.With(prometheus.Labels{"grpc_type": "unary", "grpc_service": serviceName, "grpc_method": serviceMethod}).Inc()

	// Push metrics to the pushgateway
	err = pushServerMetrics(metr, serviceName)

	return h, err
}

func getMessageSize(val interface{}) (int, error) {
	// This function takes in an interface for a gRPC message and returns its
	// size in bytes.
	var buff bytes.Buffer

	encoder := gob.NewEncoder(&buff)
	err := encoder.Encode(val)
	if err != nil {
		// ToDo Log error
		return 0, fmt.Errorf("unable to get message size")
	}

	return binary.Size(buff.Bytes()), nil
}

func pushClientMetrics(metrics *ClientMetricStruct, service string) error {
	InfoLogger.Println("Pushing metrics to gateway")
	err := push.New(os.Getenv("PUSHGATEWAYHOST")+":9091", service+"Client").
		Collector(*metrics.clientRequestCounter).
		Collector(*metrics.clientRequestMessageSize).
		Collector(*metrics.clientResponseCounter).
		Collector(*metrics.clientResponseMessageSize).
		Push()

	if err != nil {
		ErrorLogger.Println("Could not push client metrics to endpoint: \n", err)
	} else {
		DebugLogger.Println("Succesfully pushed client metrics to endpoint")
	}

	return err
}

func pushServerMetrics(metrics *ServerMetricStruct, serviceName string) error {
	InfoLogger.Println("Pushing metrics to gateway")
	err := push.New(os.Getenv("PUSHGATEWAYHOST")+":9091", serviceName+"Server").
		Collector(*metrics.serverRequestCounter).
		Collector(*metrics.serverLastCallTime).
		Collector(*metrics.serverResponseCounter).
		Collector(*metrics.serverRequestLatency).
		Push()

	if err != nil {
		ErrorLogger.Println("Could not push server metrics to endpoint: \n", err)
	} else {
		DebugLogger.Println("Succesfully pushed server metrics to endpoint")
	}

	return err
}
