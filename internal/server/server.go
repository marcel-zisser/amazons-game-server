package server

import (
	"context"
	"fmt"
	"log"

	pb "github.com/marcel-zisser/amazons-game-server/api/proto/gen"
	"google.golang.org/grpc"
)

// GameServer implements the GameService
type GameServer struct {
	pb.UnimplementedGameServiceServer
}

// NewGameServer creates a new GameServer instance
func NewGameServer() *GameServer {
	return &GameServer{}
}

// Register registers the GameServer with a gRPC server
func (s *GameServer) Register(grpcServer *grpc.Server) {
	pb.RegisterGameServiceServer(grpcServer, s)
}

// Echo implements the Echo RPC method
func (s *GameServer) Echo(ctx context.Context, req *pb.EchoRequest) (*pb.EchoResponse, error) {
	log.Printf("Received echo with message: %s", req.Message)
	return &pb.EchoResponse{
		Message: fmt.Sprintf("Echo: %s", req.Message),
	}, nil
}
