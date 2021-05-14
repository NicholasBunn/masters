package interceptors

import (
	"bytes"
	"context"
	"encoding/binary"
	"encoding/gob"
	"errors"
	"log"
	"os"
	"strings"
	"time"

	prometheus "github.com/prometheus/client_golang/prometheus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"

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
	// This struct represents a collection of metrics to be registered on a
	// Prometheus metrics registry.
	clientRequestCounter      *prometheus.Counter
	clientResponseCounter     *prometheus.Counter
	clientRequestMessageSize  *prometheus.Histogram
	clientResponseMessageSize *prometheus.Histogram
}

type ServerMetricStruct struct {
	serverCallCounter *prometheus.Counter
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

// Client side
func (metr *ClientMetricStruct) ClientMetrics(ctx context.Context, method string, req interface{}, reply interface{}, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
	InfoLogger.Println("Starting interceptor method")

	// Run gRPC call here
	err := invoker(ctx, method, req, reply, cc, opts...)

	requestSize := prometheus.NewHistogram(prometheus.HistogramOpts{
		Name: "request_size",
		Help: "The size (in bytes) of the response",
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
		ErrorLogger.Println("Could not push response message size to Pushgateway:", err)
	} else {
		DebugLogger.Println("Succesfully pushed metrics")
	}

	return err
}

// Server side
func (metr *ServerMetricStruct) ServerMetrics(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (response interface{}, err error) {
	// metr.clientCallCounter++

	meta, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return nil, errors.New("could not grab metadata from context")
	}

	// Set ping-counts into the current ping value
	// meta.Set("call-count", string(metr.clientCallCounter))

	meta.Set("start-time", time.Now().Format(time.UnixDate))
	// Metadata is sent on its own, so we need to send the header. There is also something called Trailer
	grpc.SendHeader(ctx, meta)

	// Run gRPC call here
	return handler(ctx, req)
}
