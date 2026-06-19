package main

import (
	"fmt"
	"log"
	"net"

	"github.com/marcel-zisser/amazons-game-server/internal/server"
	"google.golang.org/grpc"
)

const (
	port = ":50051"
)

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

	// Create and register the GameService
	gameServer := server.NewGameServer()
	gameServer.Register(grpcServer)

	log.Printf("🎮 Game Server listening on %s", port)

	// Start the server
	if err := grpcServer.Serve(listener); err != nil {
		log.Fatalf("Failed to serve: %v", err)
	}
}
