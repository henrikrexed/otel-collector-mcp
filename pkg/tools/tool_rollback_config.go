package tools

import (
	"context"
	"log/slog"

	"github.com/hrexed/otel-collector-mcp/pkg/session"
	"github.com/hrexed/otel-collector-mcp/pkg/types"
)

// RollbackConfigTool restores the backed-up config for an active session.
type RollbackConfigTool struct {
	BaseTool
	SessionMgr *session.Manager
}

func (t *RollbackConfigTool) Name() string { return "rollback_config" }

func (t *RollbackConfigTool) Description() string {
	return "Rollback a collector's configuration to the pre-mutation backup."
}

func (t *RollbackConfigTool) InputSchema() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"session_id": map[string]interface{}{"type": "string", "description": "Active session ID"},
		},
		"required": []string{"session_id"},
	}
}

func (t *RollbackConfigTool) Run(ctx context.Context, args map[string]interface{}) (*types.StandardResponse, error) {
	sessionID, _ := args["session_id"].(string)
	if sessionID == "" {
		return nil, types.NewMCPError(types.ErrCodeSessionNotFound, "session_id is required")
	}

	sess, err := t.SessionMgr.Get(sessionID)
	if err != nil {
		return nil, err
	}

	slog.Info("rolling back config", "session_id", sessionID, "collector", sess.Collector.Name)

	if sess.Mutator == nil {
		return nil, types.NewMCPError(types.ErrCodeRollbackFailed, "no mutator available for this session")
	}

	if err := sess.Mutator.Rollback(ctx); err != nil {
		return nil, types.NewMCPError(types.ErrCodeRollbackFailed, err.Error())
	}

	return types.NewStandardResponse(t.ClusterMeta(), t.Name(), map[string]interface{}{
		"session_id":    sessionID,
		"status":        "rollback_complete",
		"restored_from": "backup annotation",
	}), nil
}
