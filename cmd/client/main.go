package main

import (
	"context"
	"flag"
	"fmt"
	"log"

	pb "github.com/marcel-zisser/amazons-game-server/api/proto/gen"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func main() {
	addr := flag.String("addr", "localhost:50051", "the address to connect to")
	flag.Parse()

	// Create a connection to the server
	conn, err := grpc.NewClient(*addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("Failed to connect: %v", err)
	}
	defer conn.Close()

	// Create a client
	client := pb.NewGameServiceClient(conn)

	var msg string

	for {
		_, error := fmt.Scanln(&msg)
		if error != nil {
			log.Fatal(error)
		}

		// Call the Echo RPC
		response, err := client.Echo(context.Background(), &pb.EchoRequest{Message: msg})
		if err != nil {
			log.Fatalf("Failed to call Echo: %v", err)
		}

		fmt.Printf("✅ Echo response: %s\n", response.Message)
	}
}
