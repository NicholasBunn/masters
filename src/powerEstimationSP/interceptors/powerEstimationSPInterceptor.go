package interceptors

import (
	"context"

	"google.golang.org/grpc"
)

type OutboundCallCounter struct {
	calls int
}

func (occ *OutboundCallCounter) ClientCallCounter(ctx context.Context, method string, req interface{}, reply interface{}, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
	occ.calls++
	// Run gRPC call here
	return invoker(ctx, method, req, reply, cc, opts...)
}
