package game

import (
	"fmt"

	pb "github.com/marcel-zisser/amazons-game-server/api/proto/gen"
	"github.com/marcel-zisser/amazons-game-server/internal/matchmaking"
)

// GameEngine manages game state and validates moves
type GameEngine struct {
	MatchID       string
	Board         [][]pb.GameEvent_FieldState
	Player1       *matchmaking.Player
	Player2       *matchmaking.Player
	CurrentPlayer pb.GameEvent_PlayerColor // 1 for Player1, 2 for Player2
	GameOver      bool
	Winner        string
}

// NewGameEngine creates a new game engine
func NewGameEngine(matchID string, player1, player2 *matchmaking.Player) *GameEngine {
	return &GameEngine{
		MatchID:       matchID,
		Board:         initializeGameState(),
		Player1:       player1,
		Player2:       player2,
		CurrentPlayer: 1, // Player1 starts
		GameOver:      false,
		Winner:        "",
	}
}

// MakeMove processes a player's move (piece + arrow)
func (g *GameEngine) MakeMove(playerName string, fromRow, fromCol, toRow, toCol, arrowRow, arrowCol int32) error {
	if g.GameOver {
		return fmt.Errorf("game is already over")
	}

	// Verify it's this player's turn
	if g.CurrentPlayer == 1 && g.Player1.PlayerName != playerName {
		return fmt.Errorf("not player 1's turn")
	}
	if g.CurrentPlayer == 2 && g.Player2.PlayerName != playerName {
		return fmt.Errorf("not player 2's turn")
	}

	// Validate board positions
	if !isValidPosition(fromRow, fromCol) || !isValidPosition(toRow, toCol) || !isValidPosition(arrowRow, arrowCol) {
		return fmt.Errorf("invalid board position")
	}

	// Move piece
	if !g.movePiece(fromRow, fromCol, toRow, toCol) {
		return fmt.Errorf("invalid piece move")
	}

	// Place arrow
	if !g.placeArrow(arrowRow, arrowCol) {
		return fmt.Errorf("invalid arrow placement")
	}

	// Switch turns
	if g.CurrentPlayer == pb.GameEvent_PLAYER_WHITE {
		g.CurrentPlayer = pb.GameEvent_PLAYER_BLACK
	} else {
		g.CurrentPlayer = pb.GameEvent_PLAYER_WHITE
	}

	// Check for win condition (simplified: check if opponent has any moves)
	if !g.hasValidMoves(g.CurrentPlayer) {
		g.GameOver = true
		if g.CurrentPlayer == pb.GameEvent_PLAYER_WHITE {
			g.Winner = g.Player2.PlayerName
		} else {
			g.Winner = g.Player1.PlayerName
		}
	}

	return nil
}

// GetBoardState returns the current board as a proto message
func (g *GameEngine) GetBoardState() [][]pb.GameEvent_FieldState {
	return g.Board
}

// GetCurrentPlayer returns whose turn it is
func (g *GameEngine) GetCurrentPlayer() string {
	if g.CurrentPlayer == pb.GameEvent_PLAYER_WHITE {
		return g.Player1.PlayerName
	}
	return g.Player2.PlayerName
}

// Helper methods

func (g *GameEngine) movePiece(fromRow, fromCol, toRow, toCol int32) bool {

	// Check source has a piece (1 or 2, not 0 or 3)
	piece := g.Board[fromRow][fromCol]
	if piece == pb.GameEvent_FIELD_EMPTY || piece == pb.GameEvent_FIELD_ARROW {
		return false
	}

	// Check destination is empty
	if g.Board[toRow][toCol] != pb.GameEvent_FIELD_EMPTY {
		return false
	}

	// Move the piece
	g.Board[toRow][toCol] = piece
	g.Board[fromRow][fromCol] = pb.GameEvent_FIELD_EMPTY

	return true
}

func (g *GameEngine) placeArrow(row, col int32) bool {
	if g.Board[row][col] != pb.GameEvent_FIELD_EMPTY {
		return false
	}
	g.Board[row][col] = pb.GameEvent_FIELD_ARROW
	return true
}

func (g *GameEngine) hasValidMoves(playerTurn pb.GameEvent_PlayerColor) bool {
	playerPiece := pb.GameEvent_FieldState(playerTurn)

	// For each amazon of the player
	for i := 0; i < len(g.Board); i++ {
		for j := 0; j < len(g.Board); j++ {
			if g.Board[i][j] == playerPiece {
				// Check if this amazon can move anywhere
				row, col := int32(i/10), int32(i%10)

				// Check all adjacent squares (simplified - just 8 directions)
				for dr := int32(-1); dr <= 1; dr++ {
					for dc := int32(-1); dc <= 1; dc++ {
						if dr == 0 && dc == 0 {
							continue
						}
						nr, nc := row+dr, col+dc
						if isValidPosition(nr, nc) && g.Board[nr][nc] == pb.GameEvent_FIELD_EMPTY {
							return true
						}
					}
				}
			}
		}
	}
	return false
}

func isValidPosition(row, col int32) bool {
	return row >= 0 && row < 10 && col >= 0 && col < 10
}

func initializeGameState() [][]pb.GameEvent_FieldState {
	// Initialize 10x10 board for Amazons
	// 0 = Empty, 1 = White Amazon, 2 = Black Amazon, 3 = Burned (arrow)
	board := make([][]pb.GameEvent_FieldState, 10)
	for i := range board {
		board[i] = make([]pb.GameEvent_FieldState, 10)
	}

	// Place white amazons
	board[0][3] = pb.GameEvent_FIELD_WHITE_AMAZON
	board[0][6] = pb.GameEvent_FIELD_WHITE_AMAZON
	board[3][0] = pb.GameEvent_FIELD_WHITE_AMAZON
	board[3][9] = pb.GameEvent_FIELD_WHITE_AMAZON

	// Place black amazons at opposite corners
	board[9][3] = pb.GameEvent_FIELD_BLACK_AMAZON
	board[9][6] = pb.GameEvent_FIELD_BLACK_AMAZON
	board[6][0] = pb.GameEvent_FIELD_BLACK_AMAZON
	board[6][9] = pb.GameEvent_FIELD_BLACK_AMAZON

	return board
}
