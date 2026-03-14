package session

import (
	"context"
	"testing"
	"time"

	"github.com/hrexed/otel-collector-mcp/pkg/mutator"
)

type mockMutator struct{}

func (m *mockMutator) Backup(_ context.Context, _ string) error     { return nil }
func (m *mockMutator) ApplyConfig(_ context.Context, _ string) error { return nil }
func (m *mockMutator) Rollback(_ context.Context) error              { return nil }
func (m *mockMutator) TriggerRollout(_ context.Context) error        { return nil }
func (m *mockMutator) Cleanup(_ context.Context) error               { return nil }
func (m *mockMutator) DetectGitOps(_ context.Context) (bool, string) { return false, "" }

// Compile-time check: mockMutator satisfies mutator.Mutator.
var _ mutator.Mutator = (*mockMutator)(nil)

func TestCreateSession(t *testing.T) {
	mgr := NewManager(10*time.Minute, 5)
	ref := mutator.CollectorRef{Name: "test-collector", Namespace: "default"}

	session, err := mgr.Create(ref, "dev", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if session.ID == "" {
		t.Error("expected non-empty session ID")
	}
	if session.State != StateCreated {
		t.Errorf("expected state Created, got %s", session.State)
	}
	if session.Environment != "dev" {
		t.Errorf("expected environment dev, got %s", session.Environment)
	}
}

func TestGetSession(t *testing.T) {
	mgr := NewManager(10*time.Minute, 5)
	ref := mutator.CollectorRef{Name: "test-collector", Namespace: "default"}

	session, _ := mgr.Create(ref, "staging", nil)

	retrieved, err := mgr.Get(session.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if retrieved.ID != session.ID {
		t.Errorf("expected session ID %s, got %s", session.ID, retrieved.ID)
	}
}

func TestGetSessionNotFound(t *testing.T) {
	mgr := NewManager(10*time.Minute, 5)

	_, err := mgr.Get("nonexistent")
	if err == nil {
		t.Error("expected error for nonexistent session")
	}
}

func TestGetSessionExpired(t *testing.T) {
	mgr := NewManager(1*time.Millisecond, 5)
	ref := mutator.CollectorRef{Name: "test-collector", Namespace: "default"}

	session, _ := mgr.Create(ref, "dev", nil)
	time.Sleep(5 * time.Millisecond)

	_, err := mgr.Get(session.ID)
	if err == nil {
		t.Error("expected error for expired session")
	}
}

func TestMaxConcurrentSessions(t *testing.T) {
	mgr := NewManager(10*time.Minute, 2)

	for i := 0; i < 2; i++ {
		ref := mutator.CollectorRef{
			Name:      "collector-" + string(rune('a'+i)),
			Namespace: "default",
		}
		_, err := mgr.Create(ref, "dev", nil)
		if err != nil {
			t.Fatalf("unexpected error creating session %d: %v", i, err)
		}
	}

	ref := mutator.CollectorRef{Name: "collector-c", Namespace: "default"}
	_, err := mgr.Create(ref, "dev", nil)
	if err == nil {
		t.Error("expected error when max sessions exceeded")
	}
}

func TestConcurrentSessionSameCollector(t *testing.T) {
	mgr := NewManager(10*time.Minute, 5)
	ref := mutator.CollectorRef{Name: "test-collector", Namespace: "default"}

	_, err := mgr.Create(ref, "dev", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	_, err = mgr.Create(ref, "staging", nil)
	if err == nil {
		t.Error("expected error for duplicate collector session")
	}
}

func TestCloseSession(t *testing.T) {
	mgr := NewManager(10*time.Minute, 1)
	ref := mutator.CollectorRef{Name: "test-collector", Namespace: "default"}

	session, _ := mgr.Create(ref, "dev", nil)
	mgr.Close(session.ID)

	// After closing, should be able to create new session for same collector
	ref2 := mutator.CollectorRef{Name: "test-collector-2", Namespace: "default"}
	_, err := mgr.Create(ref2, "dev", nil)
	if err != nil {
		t.Fatalf("expected to create session after closing previous: %v", err)
	}
}
