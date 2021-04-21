package interceptors

import (
	"bytes"
	"context"
	"encoding/binary"
	"encoding/gob"
	"errors"
	"fmt"
	"strings"
	"time"

	prometheus "github.com/prometheus/client_golang/prometheus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"

	// "github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/push"
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
	fmt.Println("Golang Interceptor")

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

	if err := push.New("http://localhost:9091/", "PowerEstimationSP").
		Collector(requestSize).
		Collector(responseSize).
		Grouping("Service", strings.Split(method, "/")[2]).
		Push(); err != nil {
		fmt.Println("Could not push response message size to Pushgateway:", err)
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
