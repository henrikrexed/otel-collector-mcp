package tools

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/hrexed/otel-collector-mcp/pkg/session"
	"github.com/hrexed/otel-collector-mcp/pkg/types"
)

// CleanupDebugTool removes debug exporter and closes the analysis session.
type CleanupDebugTool struct {
	BaseTool
	SessionMgr *session.Manager
}

func (t *CleanupDebugTool) Name() string { return "cleanup_debug" }

func (t *CleanupDebugTool) Description() string {
	return "Remove debug exporter from a collector and restore original configuration."
}

func (t *CleanupDebugTool) InputSchema() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"session_id": map[string]interface{}{"type": "string", "description": "Active session ID"},
		},
		"required": []string{"session_id"},
	}
}

func (t *CleanupDebugTool) Run(ctx context.Context, args map[string]interface{}) (*types.StandardResponse, error) {
	sessionID, _ := args["session_id"].(string)
	if sessionID == "" {
		return nil, types.NewMCPError(types.ErrCodeSessionNotFound, "session_id is required")
	}

	sess, err := t.SessionMgr.Get(sessionID)
	if err != nil {
		return nil, err
	}

	if sess.State == session.StateClosed {
		return nil, types.NewMCPError(types.ErrCodeSessionExpired, "session is already closed")
	}

	slog.Info("cleaning up debug exporter", "session_id", sessionID, "collector", sess.Collector.Name)

	// Cleanup mutator resources
	if sess.Mutator != nil {
		if err := sess.Mutator.Cleanup(ctx); err != nil {
			slog.Warn("cleanup error", "error", err)
		}
	}

	duration := time.Since(sess.CreatedAt).Seconds()

	// Free signal data (thread-safe)
	sess.ClearData()

	// Close session
	t.SessionMgr.Close(sessionID)

	return types.NewStandardResponse(t.ClusterMeta(), t.Name(), map[string]interface{}{
		"session_id":       sessionID,
		"status":           "cleanup_complete",
		"duration_seconds": fmt.Sprintf("%.0f", duration),
	}), nil
}
