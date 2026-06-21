package main

import (
	"fmt"
	"github.com/joho/godotenv"
	"github.com/marcel-zisser/amazons-game-server/internal/matchmaking"
	"github.com/marcel-zisser/amazons-game-server/internal/server"
	"google.golang.org/grpc"
	"log"
	"net"
	"os"
)

func main() {
	fmt.Println("🚀 Amazons Game Server initialization started...")

	cwd, _ := os.Getwd()
	fmt.Printf("🔍 Go is looking for .env inside: %s\n", cwd)

	err := godotenv.Load()
	if err != nil {
		log.Println("ℹ️ No .env file found, relying on system environment variables")
	}

	// 2. Fetch your signing key securely from the environment
	signingKey := os.Getenv("JWT_SIGNING_KEY")
	if signingKey == "" {
		log.Fatal("❌ CRITICAL: JWT_SIGNING_KEY environment variable is not set!")
	}

	// 3. Fetch the port from the environment
	port := os.Getenv("PORT")
	if port == "" {
		port = "50051" // Default port
	}

	// Now you safely pass `[]byte(signingKey)` into your JWT generator/verifier
	log.Printf("Successfully loaded configuration. Server starting...")

	// Create a listener on the specified port
	listener, err := net.Listen("tcp", ":"+port)
	if err != nil {
		log.Fatalf("Failed to listen on %s: %v", port, err)
	}
	defer listener.Close()

	// Create a new gRPC server
	grpcServer := grpc.NewServer(
		grpc.UnaryInterceptor(server.AuthUnaryInterceptor),
		grpc.StreamInterceptor(server.AuthStreamInterceptor),
	)
	defer grpcServer.Stop()

	// Create matchmaking service
	matchmaker := matchmaking.NewMatchmakingService()
	log.Println("✅ Matchmaking service created")

	// Create and register the GameService with matchmaker
	gameServer := server.NewGameServer(matchmaker)
	gameServer.Register(grpcServer)

	log.Printf("🎮 Game Server listening on %s", port)

	// Start the server
	if err := grpcServer.Serve(listener); err != nil {
		log.Fatalf("Failed to serve: %v", err)
	}
}
