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
	matchmaker *matchmaking.MatchmakingService
	games      map[string]*game.GameEngine
	gamesMu    sync.Mutex
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
	matchCh, err := s.matchmaker.JoinQueue(ctx, playerName)
	if err != nil {
		return fmt.Errorf("failed to join matchmaking queue: %w", err)
	}

	// Wait for a match to be found
	select {
	case <-ctx.Done():
		// Player disconnected
		s.matchmaker.RemoveFromQueue(playerName)
		fmt.Printf("Player %s disconnected before match was found\n", playerName)
		return ctx.Err()

	case match := <-matchCh:
		// Create game engine for this match
		gameEngine := game.NewGameEngine(match.MatchID, match.Player1, match.Player2)

		s.gamesMu.Lock()
		s.games[match.MatchID] = gameEngine
		s.gamesMu.Unlock()

		// Determine if this player is Player1 or Player2
		var opponentName string
		var yourPlayerNumber int

		if match.Player1.Name == playerName {
			opponentName = match.Player2.Name
			yourPlayerNumber = 1
		} else {
			opponentName = match.Player1.Name
			yourPlayerNumber = 2
		}

		// Store the stream in the player object
		var currentPlayer *matchmaking.Player
		if match.Player1.Name == playerName {
			currentPlayer = match.Player1
		} else {
			currentPlayer = match.Player2
		}
		currentPlayer.Stream = stream

		// Send MATCH_FOUND event
		matchFoundEvent := &pb.GameEvent{
			Type:         pb.GameEvent_MATCH_FOUND,
			MatchId:      match.MatchID,
			OpponentName: opponentName,
			BoardState:   flattenBoard(gameEngine.Board),
		}

		if err := stream.Send(matchFoundEvent); err != nil {
			match.Logger.Printf("Error sending match found event: %v", err)
			return err
		}

		// Send initial YOUR_TURN event if this player is Player1
		if yourPlayerNumber == 1 {
			turnEvent := &pb.GameEvent{
				Type:          pb.GameEvent_YOUR_TURN,
				MatchId:       match.MatchID,
				BoardState:    flattenBoard(gameEngine.Board),
				CurrentPlayer: pb.GameEvent_PLAYER_WHITE,
			}
			if err := stream.Send(turnEvent); err != nil {
				match.Logger.Printf("Error sending initial turn event: %v", err)
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
	logger := s.matchmaker.GetMatch(req.MatchId).Logger

	s.gamesMu.Lock()
	gameEngine, exists := s.games[req.MatchId]
	s.gamesMu.Unlock()

	if !exists {
		return nil, fmt.Errorf("match not found")
	}

	var player, opponent *matchmaking.Player

	if gameEngine.CurrentPlayer == gameEngine.Player1.Color {
		logger.Printf("Move received for match %s by %s", req.MatchId, gameEngine.Player1.Name)
		player = gameEngine.Player1
		opponent = gameEngine.Player2
	} else {
		logger.Printf("Move received for match %s by %s", req.MatchId, gameEngine.Player2.Name)
		player = gameEngine.Player2
		opponent = gameEngine.Player1
	}

	// Make the move
	err := gameEngine.MakeMove(
		player.Name,
		req.FromPos.Row, req.FromPos.Col,
		req.ToPos.Row, req.ToPos.Col,
		req.ArrowPos.Row, req.ArrowPos.Col,
	)

	success := true
	errorMessage := ""

	if err != nil {
		success = false
		errorMessage = err.Error()

		gameEngine.GameOver = true
		gameEngine.Winner = opponent
	}

	// Check for game over
	if gameEngine.GameOver {
		logger.Printf("Game over for match %s. Winner: %s", req.MatchId, gameEngine.Winner.Name)
		turnEvent := &pb.GameEvent{
			Type:       pb.GameEvent_GAME_OVER,
			MatchId:    req.MatchId,
			BoardState: flattenBoard(gameEngine.Board),
			WinnerName: gameEngine.Winner.Name,
		}

		opponent.StreamMu.Lock()
		if err := opponent.Stream.Send(turnEvent); err != nil {
			logger.Printf("Error sending turn event to opponent: %v", err)
		}
		opponent.StreamMu.Unlock()

		player.StreamMu.Lock()
		if err := player.Stream.Send(turnEvent); err != nil {
			logger.Printf("Error sending turn event to player: %v", err)
		}
		player.StreamMu.Unlock()

		s.matchmaker.GetMatch(req.MatchId).CloseLogFile()
	} else {
		gameEngine.CurrentPlayer = opponent.Color

		// Send YOUR_TURN event to opponent
		if opponent.Stream != nil {
			turnEvent := &pb.GameEvent{
				Type:          pb.GameEvent_YOUR_TURN,
				MatchId:       req.MatchId,
				BoardState:    flattenBoard(gameEngine.Board),
				CurrentPlayer: gameEngine.CurrentPlayer,
			}

			opponent.StreamMu.Lock()
			if err := opponent.Stream.Send(turnEvent); err != nil {
				logger.Printf("Error sending turn event to opponent: %v", err)
			}
			opponent.StreamMu.Unlock()
		}
	}

	return &pb.MoveResponse{
		Success:      success,
		ErrorMessage: errorMessage,
	}, nil
}

// Echo implements the Echo RPC method
func (s *GameServer) Echo(ctx context.Context, req *pb.EchoRequest) (*pb.EchoResponse, error) {
	log.Printf("Received echo with message: %s", req.Message)
	return &pb.EchoResponse{
		Message: fmt.Sprintf("Echo: %s", req.Message),
	}, nil
}

func flattenBoard(board [][]pb.GameEvent_FieldState) []pb.GameEvent_FieldState {
	// Calculate total elements to pre-allocate memory
	totalElements := 0
	for _, row := range board {
		totalElements += len(row)
	}

	// Create the 1D slice with the exact required capacity
	flat := make([]pb.GameEvent_FieldState, 0, totalElements)

	// Flatten the 2D slice
	for _, row := range board {
		flat = append(flat, row...)
	}

	return flat
}
