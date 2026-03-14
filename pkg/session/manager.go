package session

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/hrexed/otel-collector-mcp/pkg/mutator"
	"github.com/hrexed/otel-collector-mcp/pkg/types"
)

// Manager manages v2 analysis sessions with lifecycle tracking.
type Manager struct {
	sessions    sync.Map
	ttl         time.Duration
	maxSessions int
	mu          sync.Mutex // protects count operations
}

// NewManager creates a new session manager.
func NewManager(ttl time.Duration, maxSessions int) *Manager {
	return &Manager{
		ttl:         ttl,
		maxSessions: maxSessions,
	}
}

// Create creates a new analysis session for the given collector.
func (m *Manager) Create(ref mutator.CollectorRef, environment string, mut mutator.Mutator) (*Session, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Check max concurrent sessions
	count := 0
	m.sessions.Range(func(_, value interface{}) bool {
		s := value.(*Session)
		if s.State != StateClosed {
			count++
		}
		return true
	})

	if count >= m.maxSessions {
		return nil, types.NewMCPError(types.ErrCodeConcurrentSession,
			fmt.Sprintf("maximum concurrent sessions reached (%d)", m.maxSessions))
	}

	// Check if collector is already targeted by an active session
	var conflictSessionID string
	m.sessions.Range(func(_, value interface{}) bool {
		s := value.(*Session)
		if s.State != StateClosed &&
			s.Collector.Name == ref.Name &&
			s.Collector.Namespace == ref.Namespace {
			conflictSessionID = s.ID
			return false
		}
		return true
	})

	if conflictSessionID != "" {
		return nil, types.NewMCPError(types.ErrCodeConcurrentSession,
			fmt.Sprintf("collector %s/%s already has an active session: %s", ref.Namespace, ref.Name, conflictSessionID))
	}

	now := time.Now()
	session := &Session{
		ID:           uuid.New().String(),
		Collector:    ref,
		Environment:  environment,
		State:        StateCreated,
		CreatedAt:    now,
		LastActivity: now,
		Mutator:      mut,
	}

	m.sessions.Store(session.ID, session)
	slog.Info("session created", "session_id", session.ID, "collector", ref.Name, "environment", environment)
	return session, nil
}

// Get retrieves a session by ID and updates its last activity.
func (m *Manager) Get(sessionID string) (*Session, error) {
	value, ok := m.sessions.Load(sessionID)
	if !ok {
		return nil, types.NewMCPError(types.ErrCodeSessionNotFound,
			fmt.Sprintf("session %s not found", sessionID))
	}

	session := value.(*Session)
	if session.IsExpired(m.ttl) {
		return nil, types.NewMCPError(types.ErrCodeSessionExpired,
			fmt.Sprintf("session %s has expired", sessionID))
	}

	session.Touch()
	return session, nil
}

// Close marks a session as closed.
func (m *Manager) Close(sessionID string) {
	if value, ok := m.sessions.Load(sessionID); ok {
		session := value.(*Session)
		session.SetState(StateClosed)
		slog.Info("session closed", "session_id", sessionID)
	}
}

// StartCleanupLoop runs a background goroutine that cleans up expired sessions every 30 seconds.
func (m *Manager) StartCleanupLoop(ctx context.Context) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			m.cleanupExpired(ctx)
		}
	}
}

func (m *Manager) cleanupExpired(ctx context.Context) {
	m.sessions.Range(func(key, value interface{}) bool {
		sess := value.(*Session)
		// TryExpire atomically checks expiry and transitions to Closed,
		// preventing a race where a tool call re-activates the session
		// between the expiry check and the state transition.
		if sess.TryExpire(m.ttl) {
			slog.Info("cleaning up expired session", "session_id", sess.ID)

			// Cleanup: remove debug exporter if present
			if sess.Mutator != nil {
				if err := sess.Mutator.Cleanup(ctx); err != nil {
					slog.Warn("failed to cleanup expired session", "session_id", sess.ID, "error", err)
				}
			}
		}
		return true
	})
}

// ActiveCount returns the number of active (non-closed) sessions.
func (m *Manager) ActiveCount() int {
	count := 0
	m.sessions.Range(func(_, value interface{}) bool {
		s := value.(*Session)
		if s.State != StateClosed {
			count++
		}
		return true
	})
	return count
}
