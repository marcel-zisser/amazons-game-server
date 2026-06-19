package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"time"

	pb "github.com/marcel-zisser/amazons-game-server/api/proto/gen"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func main() {
	addr := flag.String("addr", "localhost:50051", "the address to connect to")
	message := flag.String("msg", "Hello from client", "the message to send")
	flag.Parse()

	// Create a connection to the server
	conn, err := grpc.NewClient(*addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("Failed to connect: %v", err)
	}
	defer conn.Close()

	// Create a client
	client := pb.NewGameServiceClient(conn)

	// Create a context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	// Call the Ping RPC
	response, err := client.Ping(ctx, &pb.PingRequest{Message: *message})
	if err != nil {
		log.Fatalf("Failed to call Ping: %v", err)
	}

	fmt.Printf("✅ Ping response: %s\n", response.Message)
}
