package interceptors

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

type OutboundCallCounter struct {
	calls int
}

// Client side
func (occ *OutboundCallCounter) ClientMetrics(ctx context.Context, method string, req interface{}, reply interface{}, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
	fmt.Println("Golang Interceptor")
	occ.calls++
	md := metadata.Pairs("call-count", fmt.Sprint(occ.calls), "call-time", strconv.FormatInt(time.Now().UTC().UnixNano(), 10))

	ctx = metadata.NewOutgoingContext(ctx, md)

	// Run gRPC call here
	return invoker(ctx, method, req, reply, cc, opts...)
}

// Server side
func (occ *OutboundCallCounter) ServerMetrics(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (response interface{}, err error) {
	occ.calls++

	meta, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return nil, errors.New("could not grab metadata from context")
	}

	// Set ping-counts into the current ping value
	meta.Set("call-count", string(occ.calls))

	meta.Set("start-time", time.Now().Format(time.UnixDate))
	// Metadata is sent on its own, so we need to send the header. There is also something called Trailer
	grpc.SendHeader(ctx, meta)

	// Run gRPC call here
	return handler(ctx, req)
}
