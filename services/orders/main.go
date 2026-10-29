package main

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/reflection"
	"google.golang.org/grpc/status"

	pb "github.com/franklynux/istio-meshguard/services/orders/pb"
)

const (
	listenAddr   = ":50050"
	orderTimeout = 10 * time.Second
)

// server adapts the Orchestrator to the gRPC interface generated from orders.proto.
type server struct {
	pb.UnimplementedOrdersServer
	orch *Orchestrator
}

func (s *server) CreateOrder(ctx context.Context, req *pb.CreateOrderRequest) (*pb.CreateOrderResponse, error) {
	setVersionHeader(ctx)
	ctx, cancel := context.WithTimeout(ctx, orderTimeout)
	defer cancel()
	ctx = propagateTraceHeaders(ctx)

	items := make([]OrderItem, 0, len(req.GetItems()))
	for _, it := range req.GetItems() {
		items = append(items, OrderItem{SKU: it.GetSku(), Quantity: it.GetQuantity()})
	}

	res, err := s.orch.CreateOrder(ctx, items, req.GetEmail(), req.GetAmount())
	if err != nil {
		slog.Error("create order failed", "order_id", res.OrderID, "error", err)
		// No order id means nothing was charged, so a retry is safe. With an id,
		// money may have moved: report Internal so callers do not blindly retry.
		if res.OrderID == "" {
			return nil, status.Error(codes.Unavailable, "order could not be completed: a downstream service is unavailable")
		}
		return nil, status.Errorf(codes.Internal, "order %s may have been charged but could not be completed", res.OrderID)
	}

	return &pb.CreateOrderResponse{OrderId: res.OrderID, Status: res.Status, Error: res.Reason}, nil
}

func main() {
	if err := run(); err != nil {
		slog.Error("orders exited", "error", err)
		os.Exit(1)
	}
}

// run holds the real logic so deferred cleanup runs before main exits.
func run() error {
	clients, err := NewClients()
	if err != nil {
		return err
	}
	defer clients.Close()

	lis, err := net.Listen("tcp", listenAddr)
	if err != nil {
		return fmt.Errorf("listen: %w", err)
	}

	grpcServer := grpc.NewServer()
	orch := NewOrchestrator(clients.Inventory, clients.Payments, clients.Notifications)
	pb.RegisterOrdersServer(grpcServer, &server{orch: orch})
	reflection.Register(grpcServer)

	// Kubernetes sends SIGTERM when stopping a pod: finish in-flight orders first.
	go func() {
		stop := make(chan os.Signal, 1)
		signal.Notify(stop, syscall.SIGTERM, syscall.SIGINT)
		<-stop
		slog.Info("shutting down")
		grpcServer.GracefulStop()
	}()

	slog.Info("orders listening", "addr", listenAddr)
	return grpcServer.Serve(lis)
}