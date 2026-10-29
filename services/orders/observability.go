package main

import (
	"context"
	"os"

	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

var traceHeaders = []string{
	"x-request-id",
	"x-b3-traceid",
	"x-b3-spanid",
	"x-b3-parentspanid",
	"x-b3-sampled",
	"x-b3-flags",
	"b3",
	"traceparent",
	"tracestate",
}

func propagateTraceHeaders(ctx context.Context) context.Context {
	in, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return ctx
	}
	out := metadata.MD{}
	for _, h := range traceHeaders {
		if v := in.Get(h); len(v) > 0 {
			out.Set(h, v...)
		}
	}
	if len(out) == 0 {
		return ctx
	}
	return metadata.NewOutgoingContext(ctx, out)
}

// setVersionHeader tells the caller which orders version served the request.
func setVersionHeader(ctx context.Context) {
	if v := os.Getenv("ORDERS_VERSION"); v != "" {
		_ = grpc.SetHeader(ctx, metadata.Pairs("x-orders-version", v))
	}
}