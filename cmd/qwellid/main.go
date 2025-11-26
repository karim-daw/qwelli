package main

import (
	"log"
	"net"

	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"

	pb "github.com/karim-daw/qwelli/api/gen/go/qwelli/v1"
	"github.com/karim-daw/qwelli/internal/server"
)

func main() {
	lis, err := net.Listen("tcp", ":50051")
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	grpcServer := grpc.NewServer()

	pb.RegisterSearchServiceServer(grpcServer, server.NewSearchService())

	// Enable gRPC reflection for tools like grpcurl
	reflection.Register(grpcServer)

	log.Println("Qwelli gRPC server running on :50051")
	if err := grpcServer.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}
