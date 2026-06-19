package server

import (
	"context"
	"fmt"
	"log"
	"sync"

	pb "github.com/marcel-zisser/amazons-game-server/api/proto/gen"
	"github.com/marcel-zisser/amazons-game-server/internal/game"
	"github.com/marcel-zisser/amazons-game-server/internal/matchmaking"
	"google.golang.org/grpc"
)

// GameServer implements the GameService
type GameServer struct {
	pb.UnimplementedGameServiceServer
	matchmaker  *matchmaking.MatchmakingService
	games       map[string]*game.GameEngine
	gamesMu     sync.Mutex
}

// NewGameServer creates a new GameServer instance with matchmaking
func NewGameServer(matchmaker *matchmaking.MatchmakingService) *GameServer {
	return &GameServer{
		matchmaker: matchmaker,
		games:      make(map[string]*game.GameEngine),
	}
}

// Register registers the GameServer with a gRPC server
func (s *GameServer) Register(grpcServer *grpc.Server) {
	pb.RegisterGameServiceServer(grpcServer, s)
}

// PlayGame implements the PlayGame RPC - handles matchmaking and game streaming
func (s *GameServer) PlayGame(req *pb.PlayGameRequest, stream pb.GameService_PlayGameServer) error {
	ctx := stream.Context()
	playerName := req.BotName

	log.Printf("Player %s joined matchmaking", playerName)

	// Join the matchmaking queue
	matchCh := s.matchmaker.JoinQueue(ctx, playerName)

	// Wait for a match to be found
	select {
	case <-ctx.Done():
		// Player disconnected
		s.matchmaker.RemoveFromQueue(playerName)
		return ctx.Err()
	case match := <-matchCh:
		log.Printf("Match found: %s vs %s (Match ID: %s)", match.Player1.PlayerName, match.Player2.PlayerName, match.MatchID)

		// Create game engine for this match
		gameEngine := game.NewGameEngine(match.MatchID, match.Player1.PlayerName, match.Player2.PlayerName, match.GameState)

		s.gamesMu.Lock()
		s.games[match.MatchID] = gameEngine
		s.gamesMu.Unlock()

		// Determine if this player is Player1 or Player2
		var opponentName string
		var yourPlayerNumber int
		if match.Player1.PlayerName == playerName {
			opponentName = match.Player2.PlayerName
			yourPlayerNumber = 1
		} else {
			opponentName = match.Player1.PlayerName
			yourPlayerNumber = 2
		}

		// Send MATCH_FOUND event
		matchFoundEvent := &pb.GameEvent{
			Type:         pb.GameEvent_MATCH_FOUND,
			MatchId:      match.MatchID,
			OpponentName: opponentName,
			BoardState:   gameEngine.GetBoardState(),
		}

		if err := stream.Send(matchFoundEvent); err != nil {
			log.Printf("Error sending match found event: %v", err)
			return err
		}

		// Send initial YOUR_TURN event if this player is Player1
		if yourPlayerNumber == 1 {
			turnEvent := &pb.GameEvent{
				Type:       pb.GameEvent_YOUR_TURN,
				MatchId:    match.MatchID,
				BoardState: gameEngine.GetBoardState(),
			}
			if err := stream.Send(turnEvent); err != nil {
				log.Printf("Error sending initial turn event: %v", err)
				return err
			}
		}

		// Keep connection alive until game ends
		// In a real implementation, you'd subscribe to game updates
		<-ctx.Done()
		return ctx.Err()
	}
}

// SubmitMove implements the SubmitMove RPC - processes player moves
func (s *GameServer) SubmitMove(ctx context.Context, req *pb.MoveRequest) (*pb.MoveResponse, error) {
	log.Printf("Move received for match %s", req.MatchId)

	s.gamesMu.Lock()
	gameEngine, exists := s.games[req.MatchId]
	s.gamesMu.Unlock()

	if !exists {
		return nil, fmt.Errorf("match not found")
	}

	// Make the move
	err := gameEngine.MakeMove(
		"", // TODO: Identify player from context
		req.FromPos.Row, req.FromPos.Col,
		req.ToPos.Row, req.ToPos.Col,
		req.ArrowPos.Row, req.ArrowPos.Col,
	)

	if err != nil {
		return &pb.MoveResponse{
			Success:      false,
			ErrorMessage: err.Error(),
		}, nil
	}

	return &pb.MoveResponse{
		Success: true,
	}, nil
}

// Echo implements the Echo RPC method
func (s *GameServer) Echo(ctx context.Context, req *pb.EchoRequest) (*pb.EchoResponse, error) {
	log.Printf("Received echo with message: %s", req.Message)
	return &pb.EchoResponse{
		Message: fmt.Sprintf("Echo: %s", req.Message),
	}, nil
}
