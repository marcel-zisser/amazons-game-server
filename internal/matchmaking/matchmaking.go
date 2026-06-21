package matchmaking

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"os"
	"path/filepath"
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
	Name     string
	MatchCh  chan *Match              // Channel to notify player when match is found
	Color    pb.GameEvent_PlayerColor // Player color (WHITE or BLACK)
	Stream   pb.GameService_PlayGameServer
	StreamMu sync.Mutex // Protect stream access
}

// Match represents an active game match
type Match struct {
	MatchID    string
	Player1    *Player
	Player2    *Player
	CreatedAt  time.Time
	Logger     *log.Logger
	fileHandle *os.File
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
func (ms *MatchmakingService) JoinQueue(ctx context.Context, playerName string) (chan *Match, error) {
	ms.mu.Lock()
	defer ms.mu.Unlock()

	matchCh := make(chan *Match, 1)
	player := &Player{
		Name:    playerName,
		MatchCh: matchCh,
	}

	// Check if there's already a player waiting
	if len(ms.queue) > 0 {
		// Found a match! Pair with the first player in queue
		opponent := ms.queue[0]
		ms.queue = ms.queue[1:] // Remove from queue

		opponent.Color = pb.GameEvent_PLAYER_WHITE
		player.Color = pb.GameEvent_PLAYER_BLACK

		logDir := "logs"
		if err := os.MkdirAll(logDir, 0755); err != nil {
			return nil, fmt.Errorf("failed to create log directory: %w", err)
		}

		matchId := generateMatchID()

		fileName := filepath.Join(logDir, fmt.Sprintf("match_%s.log", matchId))
		file, err := os.OpenFile(fileName, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
		if err != nil {
			return nil, fmt.Errorf("failed to create match log file: %w", err)
		}

		matchLogger := log.New(file, fmt.Sprintf("[MATCH-%s] ", matchId), log.LstdFlags)

		match := &Match{
			MatchID:    matchId,
			Player1:    opponent,
			Player2:    player,
			CreatedAt:  time.Now(),
			Logger:     matchLogger,
			fileHandle: file,
		}

		// Store active match
		ms.activeMatches[match.MatchID] = match

		// Notify both players
		opponent.MatchCh <- match
		matchCh <- match

		match.Logger.Printf("Match found: %s vs %s (Match ID: %s)", match.Player1.Name, match.Player2.Name, match.MatchID)

		return matchCh, nil
	}

	// No one waiting, add this player to queue
	ms.queue = append(ms.queue, player)
	return matchCh, nil
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
		if p.Name == playerName {
			ms.queue = append(ms.queue[:i], ms.queue[i+1:]...)
			break
		}
	}
}

// Helper functions
func generateMatchID() string {
	return fmt.Sprintf("match_%d_%d", time.Now().Unix(), rand.Intn(10000))
}

func (m *Match) CloseLogFile() {
	if m.fileHandle != nil {
		m.Logger.Println("🏁 Match terminated. Closing game records.")
		m.fileHandle.Close()
	}
}
