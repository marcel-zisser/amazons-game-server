package game

import (
	"fmt"

	pb "github.com/marcel-zisser/amazons-game-server/api/proto/gen"
)

// GameEngine manages game state and validates moves
type GameEngine struct {
	MatchID     string
	Board       []int32 // 1D array: row*10 + col
	Player1     string
	Player2     string
	CurrentTurn int // 1 for Player1, 2 for Player2
	GameOver    bool
	Winner      string
}

// NewGameEngine creates a new game engine
func NewGameEngine(matchID string, player1, player2 string, initialState *pb.GameState) *GameEngine {
	return &GameEngine{
		MatchID:     matchID,
		Board:       initialState.Grid,
		Player1:     player1,
		Player2:     player2,
		CurrentTurn: 1, // Player1 starts
		GameOver:    false,
		Winner:      "",
	}
}

// MakeMove processes a player's move (piece + arrow)
func (g *GameEngine) MakeMove(playerName string, fromRow, fromCol, toRow, toCol, arrowRow, arrowCol int32) error {
	if g.GameOver {
		return fmt.Errorf("game is already over")
	}

	// Verify it's this player's turn
	if g.CurrentTurn == 1 && g.Player1 != playerName {
		return fmt.Errorf("not player 1's turn")
	}
	if g.CurrentTurn == 2 && g.Player2 != playerName {
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
	if g.CurrentTurn == 1 {
		g.CurrentTurn = 2
	} else {
		g.CurrentTurn = 1
	}

	// Check for win condition (simplified: check if opponent has any moves)
	if !g.hasValidMoves(g.CurrentTurn) {
		g.GameOver = true
		if g.CurrentTurn == 1 {
			g.Winner = g.Player2
		} else {
			g.Winner = g.Player1
		}
	}

	return nil
}

// GetBoardState returns the current board as a proto message
func (g *GameEngine) GetBoardState() *pb.GameState {
	return &pb.GameState{Grid: g.Board}
}

// GetCurrentPlayer returns whose turn it is
func (g *GameEngine) GetCurrentPlayer() string {
	if g.CurrentTurn == 1 {
		return g.Player1
	}
	return g.Player2
}

// Helper methods

func (g *GameEngine) movePiece(fromRow, fromCol, toRow, toCol int32) bool {
	fromIdx := fromRow*10 + fromCol
	toIdx := toRow*10 + toCol

	// Check source has a piece (1 or 2, not 0 or 3)
	piece := g.Board[fromIdx]
	if piece == 0 || piece == 3 {
		return false
	}

	// Check destination is empty
	if g.Board[toIdx] != 0 {
		return false
	}

	// Move the piece
	g.Board[toIdx] = piece
	g.Board[fromIdx] = 0

	return true
}

func (g *GameEngine) placeArrow(row, col int32) bool {
	idx := row*10 + col
	if g.Board[idx] != 0 {
		return false
	}
	g.Board[idx] = 3
	return true
}

func (g *GameEngine) hasValidMoves(playerTurn int) bool {
	playerPiece := int32(playerTurn)

	// For each amazon of the player
	for i := 0; i < len(g.Board); i++ {
		if g.Board[i] == playerPiece {
			// Check if this amazon can move anywhere
			row, col := int32(i/10), int32(i%10)

			// Check all adjacent squares (simplified - just 8 directions)
			for dr := int32(-1); dr <= 1; dr++ {
				for dc := int32(-1); dc <= 1; dc++ {
					if dr == 0 && dc == 0 {
						continue
					}
					nr, nc := row+dr, col+dc
					if isValidPosition(nr, nc) && g.Board[nr*10+nc] == 0 {
						return true
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
