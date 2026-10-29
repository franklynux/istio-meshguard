package main

import (
	"context"
	"testing"

	"google.golang.org/grpc/metadata"
)

func TestPropagateTraceHeaders(t *testing.T) {
	in := metadata.Pairs(
		"traceparent", "00-abc-def-01",
		"x-request-id", "req-1",
		"authorization", "secret",
	)
	ctx := metadata.NewIncomingContext(context.Background(), in)

	out, ok := metadata.FromOutgoingContext(propagateTraceHeaders(ctx))
	if !ok {
		t.Fatal("expected outgoing metadata")
	}
	if got := out.Get("traceparent"); len(got) != 1 || got[0] != "00-abc-def-01" {
		t.Errorf("traceparent = %v, want [00-abc-def-01]", got)
	}
	if got := out.Get("x-request-id"); len(got) != 1 || got[0] != "req-1" {
		t.Errorf("x-request-id = %v, want [req-1]", got)
	}
	if got := out.Get("authorization"); len(got) != 0 {
		t.Errorf("authorization must not be forwarded, got %v", got)
	}
}

func TestPropagateTraceHeaders_NoHeaders(t *testing.T) {
	ctx := propagateTraceHeaders(context.Background())
	if _, ok := metadata.FromOutgoingContext(ctx); ok {
		t.Error("expected no outgoing metadata when nothing to forward")
	}
}