package session

import (
	"sync"
	"time"

	"github.com/hrexed/otel-collector-mcp/pkg/mutator"
)

// State represents the lifecycle state of a session.
type State string

const (
	StateCreated   State = "Created"
	StateCapturing State = "Capturing"
	StateAnalyzing State = "Analyzing"
	StateClosed    State = "Closed"
)

// Session holds all state for a v2 analysis session.
type Session struct {
	mu sync.Mutex

	ID           string
	Collector    mutator.CollectorRef
	Environment  string
	State        State
	CreatedAt    time.Time
	LastActivity time.Time

	// Mutation state
	BackupConfig    string
	InjectedPipelines []string

	// Analysis state
	CapturedSignals interface{}
	Findings        interface{}
	SuggestedFixes  interface{}

	// Mutator for this session
	Mutator mutator.Mutator
}

// Touch updates the last activity timestamp.
func (s *Session) Touch() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.LastActivity = time.Now()
}

// SetState updates the session state.
func (s *Session) SetState(state State) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.State = state
	s.LastActivity = time.Now()
}

// IsExpired returns true if the session has been inactive longer than ttl.
func (s *Session) IsExpired(ttl time.Duration) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return time.Since(s.LastActivity) > ttl
}

// TryExpire atomically checks if the session is expired and marks it closed.
// Returns true if the session was expired and transitioned to StateClosed.
// This prevents a TOCTOU race between checking expiry and closing.
func (s *Session) TryExpire(ttl time.Duration) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.State == StateClosed {
		return false
	}
	if time.Since(s.LastActivity) > ttl {
		s.State = StateClosed
		return true
	}
	return false
}

// ClearData nils out captured signals, findings, and suggested fixes under the lock.
func (s *Session) ClearData() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.CapturedSignals = nil
	s.Findings = nil
	s.SuggestedFixes = nil
}
