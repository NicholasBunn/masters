package interceptors

import (
	"bytes"
	"context"
	"encoding/binary"
	"encoding/gob"
	"log"
	"os"
	"strings"

	prometheus "github.com/prometheus/client_golang/prometheus"
	"google.golang.org/grpc"

	// "github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/push"
)

var (
	// Logging stuff
	DebugLogger   *log.Logger
	InfoLogger    *log.Logger
	WarningLogger *log.Logger
	ErrorLogger   *log.Logger
)

// This isn't actually being used right now, reconsider how you're implementing the client-side interceptor
type ClientMetricStruct struct {
	/* This struct represents a collection of client-side metrics to be registered on a
	Prometheus metrics registry */
	clientRequestCounter      *prometheus.Counter
	clientResponseCounter     *prometheus.Counter
	clientRequestMessageSize  *prometheus.Histogram
	clientResponseMessageSize *prometheus.Histogram
}

type ServerMetricStruct struct {
	/* This struct represents a collection of server-side metrics to be reqistered on a
	Prometheus metrics registry */
	serverCallCounter    *prometheus.Counter
	serverLastCallTime   *prometheus.Gauge
	serverRequestLatency *prometheus.Histogram
}

func init() {
	// Logger setup
	file, err := os.OpenFile("program logs/"+"logs.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0666)
	if err != nil {
		log.Fatal(err)
	}

	log.SetOutput(file)

	DebugLogger = log.New(file, "DEBUG: ", log.Ldate|log.Ltime|log.Lmicroseconds|log.Lshortfile)
	InfoLogger = log.New(file, "INFO: ", log.Ldate|log.Ltime|log.Lmicroseconds|log.Lshortfile)
	WarningLogger = log.New(file, "WARNING: ", log.Ldate|log.Ltime|log.Lmicroseconds|log.Lshortfile)
	ErrorLogger = log.New(file, "ERROR: ", log.Ldate|log.Ltime|log.Lmicroseconds|log.Lshortfile)
}

func (metr *ClientMetricStruct) ClientMetricInterceptor(ctx context.Context, method string, req interface{}, reply interface{}, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
	// Client side interceptor, to be attached to all client connections
	InfoLogger.Println("Starting client interceptor method")

	// Run gRPC call here
	err := invoker(ctx, method, req, reply, cc, opts...)

	requestSize := prometheus.NewHistogram(prometheus.HistogramOpts{
		Name: "request_size",
		Help: "The size (in bytes) of the request",
	})
	responseSize := prometheus.NewHistogram(prometheus.HistogramOpts{
		Name: "response_size",
		Help: "The size (in bytes) of the server's response",
	})

	// Record reply size here
	size, _ := GetMessageSize(req)
	requestSize.Observe(float64(size))

	size, _ = GetMessageSize(reply)
	responseSize.Observe(float64(size))

	if err := push.New("http://127.0.0.1:9091/", "PowerEstimationSP").
		Collector(requestSize).
		Collector(responseSize).
		Grouping("Service", strings.Split(method, "/")[2]).
		Push(); err != nil {
		ErrorLogger.Println("Could not push response message size to Pushgateway: \n", err)
	} else {
		DebugLogger.Println("Succesfully pushed metrics")
	}

	return err
}

func (metr *ServerMetricStruct) ServerMetricInterceptor(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
	// Server-side interceptor, to be attached to all server connections
	InfoLogger.Println("Starting server interceptor method")

	// ________INCREMENT CALL COUNTER________

	// ________SET LAST CALL TIME________

	// ________SET START TIME________

	// Run gRPC call here
	return handler(ctx, req)

	// ________SET END TIME________
}

func GetMessageSize(val interface{}) (int, error) {
	// This function takes in an interface for a gRPC message and returns its
	// size in bytes.
	var buff bytes.Buffer

	encoder := gob.NewEncoder(&buff)
	err := encoder.Encode(val)
	if err != nil {
		// ToDo Log error
		return 0, err
	}

	return binary.Size(buff.Bytes()), nil
}
