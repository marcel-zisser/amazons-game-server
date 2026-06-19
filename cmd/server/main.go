package main

import (
	"context"
	"fmt"
	"log"
	"net"

	pb "github.com/marcel-zisser/amazons-game-server/api/proto/gen"
	"google.golang.org/grpc"
)

const (
	port = ":50051"
)

// Server implements the GameService
type gameServer struct {
	pb.UnimplementedGameServiceServer
}

// Ping implements the Ping RPC method
func (s *gameServer) Ping(ctx context.Context, req *pb.PingRequest) (*pb.PingResponse, error) {
	log.Printf("Received ping with message: %s", req.Message)
	return &pb.PingResponse{
		Message: fmt.Sprintf("Pong: %s", req.Message),
	}, nil
}

func main() {
	fmt.Println("🚀 Amazons Game Server initialization started...")

	// Create a listener on the specified port
	listener, err := net.Listen("tcp", port)
	if err != nil {
		log.Fatalf("Failed to listen on %s: %v", port, err)
	}
	defer listener.Close()

	// Create a new gRPC server
	grpcServer := grpc.NewServer()
	defer grpcServer.Stop()

	// Register the GameService
	pb.RegisterGameServiceServer(grpcServer, &gameServer{})

	log.Printf("🎮 Game Server listening on %s", port)

	// Start the server
	if err := grpcServer.Serve(listener); err != nil {
		log.Fatalf("Failed to serve: %v", err)
	}
}
