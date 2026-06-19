package matchmaking

import (
	"context"
	"fmt"
	"math/rand"
	"sync"
	"time"

	pb "github.com/marcel-zisser/amazons-game-server/api/proto/gen"
)

// MatchmakingService handles player queuing and match creation
type MatchmakingService struct {
	mu            sync.Mutex
	queue         []*Player
	activeMatches map[string]*Match
}

// Player represents a player waiting in the queue
type Player struct {
	PlayerName string
	MatchCh    chan *Match // Channel to notify player when match is found
}

// Match represents an active game match
type Match struct {
	MatchID   string
	Player1   *Player
	Player2   *Player
	GameState *pb.GameState
	CreatedAt time.Time
}

// NewMatchmakingService creates a new matchmaking service
func NewMatchmakingService() *MatchmakingService {
	return &MatchmakingService{
		queue:         []*Player{},
		activeMatches: make(map[string]*Match),
	}
}

// JoinQueue adds a player to the matchmaking queue and returns a channel to receive match
// When a match is found, it will be sent on the returned channel
func (ms *MatchmakingService) JoinQueue(ctx context.Context, playerName string) chan *Match {
	ms.mu.Lock()
	defer ms.mu.Unlock()

	matchCh := make(chan *Match, 1)
	player := &Player{
		PlayerName: playerName,
		MatchCh:    matchCh,
	}

	// Check if there's already a player waiting
	if len(ms.queue) > 0 {
		// Found a match! Pair with the first player in queue
		opponent := ms.queue[0]
		ms.queue = ms.queue[1:] // Remove from queue

		// Create match
		match := &Match{
			MatchID:   generateMatchID(),
			Player1:   opponent,
			Player2:   player,
			GameState: initializeGameState(),
			CreatedAt: time.Now(),
		}

		// Store active match
		ms.activeMatches[match.MatchID] = match

		// Notify both players
		opponent.MatchCh <- match
		matchCh <- match

		return matchCh
	}

	// No one waiting, add this player to queue
	ms.queue = append(ms.queue, player)
	return matchCh
}

// GetMatch retrieves an active match by ID
func (ms *MatchmakingService) GetMatch(matchID string) *Match {
	ms.mu.Lock()
	defer ms.mu.Unlock()
	return ms.activeMatches[matchID]
}

// RemoveMatch removes a match when it ends
func (ms *MatchmakingService) RemoveMatch(matchID string) {
	ms.mu.Lock()
	defer ms.mu.Unlock()
	delete(ms.activeMatches, matchID)
}

// GetQueueSize returns the current queue size
func (ms *MatchmakingService) GetQueueSize() int {
	ms.mu.Lock()
	defer ms.mu.Unlock()
	return len(ms.queue)
}

// RemoveFromQueue removes a player from the queue (e.g., if they disconnect)
func (ms *MatchmakingService) RemoveFromQueue(playerName string) {
	ms.mu.Lock()
	defer ms.mu.Unlock()
	for i, p := range ms.queue {
		if p.PlayerName == playerName {
			ms.queue = append(ms.queue[:i], ms.queue[i+1:]...)
			break
		}
	}
}

// Helper functions

func generateMatchID() string {
	rand.Seed(time.Now().UnixNano())
	return fmt.Sprintf("match_%d_%d", time.Now().Unix(), rand.Intn(10000))
}

func initializeGameState() *pb.GameState {
	// Initialize 10x10 board for Amazons
	// 0 = Empty, 1 = White Amazon, 2 = Black Amazon, 3 = Burned (arrow)
	board := make([]int32, 100)

	// Place white amazons at corners: (3,0), (6,0), (0,3), (0,6), (9,3), (9,6), (3,9), (6,9)
	// Row-major order: row*10 + col
	board[0*10+3] = 1 // (0,3)
	board[0*10+6] = 1 // (0,6)
	board[3*10+0] = 1 // (3,0)
	board[3*10+9] = 1 // (3,9)
	board[6*10+0] = 1 // (6,0)
	board[6*10+9] = 1 // (6,9)
	board[9*10+3] = 1 // (9,3)
	board[9*10+6] = 1 // (9,6)

	// Place black amazons at opposite corners
	board[0*10+0] = 2 // (0,0)
	board[0*10+9] = 2 // (0,9)
	board[9*10+0] = 2 // (9,0)
	board[9*10+9] = 2 // (9,9)

	return &pb.GameState{Grid: board}
}
