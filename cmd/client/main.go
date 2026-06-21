package main

import (
	"context"
	"flag"
	"fmt"
	"log"

	pb "github.com/marcel-zisser/amazons-game-server/api/proto/gen"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
)

func mapTo2dBoard(board []pb.GameEvent_FieldState) [][]pb.GameEvent_FieldState {
	// This creates a 10x10 board initialized entirely to 0 (FIELD_EMPTY)
	// 1. Allocate the 10 rows
	mappedBoard := make([][]pb.GameEvent_FieldState, 10)

	// 2. Allocate 10 columns for each individual row
	for i := range mappedBoard {
		mappedBoard[i] = make([]pb.GameEvent_FieldState, 10)
	}

	for i := 0; i < 10; i++ {
		for j := 0; j < 10; j++ {
			mappedBoard[i][j] = board[i*10+j]
		}
	}

	return mappedBoard
}

func makeMove(gameState [][]pb.GameEvent_FieldState, color pb.GameEvent_PlayerColor, matchId string) *pb.MoveRequest {
	pieces := findPieces(gameState, color)

	first_piece := pieces[0]
	fromRow, fromCol := first_piece[0], first_piece[1]
	toRow, toCol := fromRow, fromCol+1

	arrowRow, arrowCol := toRow, toCol+1

	moveRequest := &pb.MoveRequest{
		FromPos:  &pb.Position{Row: int32(fromRow), Col: int32(fromCol)},
		ToPos:    &pb.Position{Row: int32(toRow), Col: int32(toCol)},
		ArrowPos: &pb.Position{Row: int32(arrowRow), Col: int32(arrowCol)},
		MatchId:  matchId,
	}

	return moveRequest
}

func findPieces(board [][]pb.GameEvent_FieldState, color pb.GameEvent_PlayerColor) [][2]int {
	var pieces [][2]int
	var pieceType pb.GameEvent_FieldState

	if color == pb.GameEvent_PLAYER_WHITE {
		pieceType = pb.GameEvent_FIELD_WHITE_AMAZON
	} else {
		pieceType = pb.GameEvent_FIELD_BLACK_AMAZON
	}

	for i := 0; i < 10; i++ {
		for j := 0; j < 10; j++ {
			if board[i][j] == pieceType {
				pieces = append(pieces, [2]int{i, j})
			}
		}
	}

	return pieces
}

func main() {
	addr := flag.String("addr", "localhost:50051", "the address to connect to")
	playerName := flag.String("player", "TestBot", "the name of the player")
	flag.Parse()

	// Create a connection to the server
	conn, err := grpc.NewClient(*addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("Failed to connect: %v", err)
	}

	ctx := context.Background()
	defer conn.Close()

	// Create a client
	client := pb.NewGameServiceClient(conn)
	token := "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJleHAiOjE3ODI2NTkyMjEsImlhdCI6MTc4MjA1NDQyMSwiaXNzIjoiYW1hem9ucy10b3VybmFtZW50LXNlcnZlciIsInRlYW0iOiJKRFMifQ.-kmNbwNHiHXQgtxO4TsAhfIEdCPU9LmhlM0YIRPMrKM"
	authenticatedCtx := metadata.AppendToOutgoingContext(ctx, "authorization", "Bearer "+token)

	// echo(client)
	stream, err := client.PlayGame(authenticatedCtx, &pb.PlayGameRequest{BotName: *playerName})
	if err != nil {
		log.Fatalf("Failed to call PlayGame: %v", err)
	}

	var event pb.GameEvent
	var color pb.GameEvent_PlayerColor
	var gameState [][]pb.GameEvent_FieldState

	for event.Type != pb.GameEvent_GAME_OVER {
		err := stream.RecvMsg(&event)

		if err != nil {
			log.Fatalf("Failed to receive message: %v", err)
		}

		color = event.CurrentPlayer
		gameState = mapTo2dBoard(event.BoardState)

		fmt.Printf("Received event: %v\n", event.Type)

		switch event.Type {
		case pb.GameEvent_MATCH_FOUND:
			fmt.Printf("Match found! Match ID: %s, Opponent: %s\n", event.MatchId, event.OpponentName)
		case pb.GameEvent_YOUR_TURN:
			fmt.Printf("It's your turn! Current player: %v\n", color)
			moveRequest := makeMove(gameState, color, event.MatchId)
			response, err := client.SubmitMove(authenticatedCtx, moveRequest)
			if err != nil {
				log.Fatalf("Failed to submit move: %v", err)
			}

			fmt.Printf("Submitted move: from (%d, %d) to (%d, %d) with arrow at (%d, %d)\n",
				moveRequest.FromPos.Row, moveRequest.FromPos.Col,
				moveRequest.ToPos.Row, moveRequest.ToPos.Col,
				moveRequest.ArrowPos.Row, moveRequest.ArrowPos.Col)

			if response.Success == false {
				log.Printf("Move was invalid: %s", response.ErrorMessage)
			} else {
				log.Printf("Move accepted")
			}
		case pb.GameEvent_GAME_OVER:
			fmt.Println("Game over!")
		}
	}
}