package main

import (
	"context"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"

	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"

	pb "github.com/franklynux/istio-meshguard/services/inventory/pb"
)

// server adapts our Store to the gRPC interface generated from inventory.proto.
type server struct {
	pb.UnimplementedInventoryServer
	store *Store
}

func (s *server) CheckStock(_ context.Context, req *pb.CheckStockRequest) (*pb.CheckStockResponse, error) {
	available, found := s.store.CheckStock(req.GetSku(), req.GetQuantity())
	return &pb.CheckStockResponse{Available: available, Found: found}, nil
}

func (s *server) Decrement(_ context.Context, req *pb.DecrementRequest) (*pb.DecrementResponse, error) {
	ok, err := s.store.Decrement(req.GetSku(), req.GetQuantity())
	if err != nil {
		// A business rejection (unknown SKU, not enough stock) is a normal
		// answer, so it goes in the response's error field, not a gRPC error.
		return &pb.DecrementResponse{Success: false, Error: err.Error()}, nil
	}
	return &pb.DecrementResponse{Success: ok}, nil
}

func main() {
	lis, err := net.Listen("tcp", ":50051")
	if err != nil {
		log.Fatalf("listen: %v", err)
	}

	grpcServer := grpc.NewServer()
	pb.RegisterInventoryServer(grpcServer, &server{store: NewStore()})
	// Reflection lets tools like grpcurl discover the API without the .proto file.
	reflection.Register(grpcServer)

	// Kubernetes sends SIGTERM when it stops a pod. Finish in-flight requests first.
	go func() {
		stop := make(chan os.Signal, 1)
		signal.Notify(stop, syscall.SIGTERM, syscall.SIGINT)
		<-stop
		log.Println("shutting down")
		grpcServer.GracefulStop()
	}()

	log.Println("inventory listening on :50051")
	if err := grpcServer.Serve(lis); err != nil {
		log.Fatalf("serve: %v", err)
	}
}