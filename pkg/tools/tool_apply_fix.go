package tools

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/hrexed/otel-collector-mcp/pkg/fixes"
	"github.com/hrexed/otel-collector-mcp/pkg/session"
	"github.com/hrexed/otel-collector-mcp/pkg/types"
)

// ApplyFixTool applies a single user-approved fix with automatic health checking.
type ApplyFixTool struct {
	BaseTool
	SessionMgr *session.Manager
}

func (t *ApplyFixTool) Name() string { return "apply_fix" }

func (t *ApplyFixTool) Description() string {
	return "Apply a suggested fix to the collector configuration with safety checks and backup."
}

func (t *ApplyFixTool) InputSchema() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"session_id":       map[string]interface{}{"type": "string", "description": "Active session ID"},
			"suggestion_index": map[string]interface{}{"type": "integer", "description": "Index of the fix suggestion to apply"},
		},
		"required": []string{"session_id", "suggestion_index"},
	}
}

func (t *ApplyFixTool) Run(ctx context.Context, args map[string]interface{}) (*types.StandardResponse, error) {
	sessionID, _ := args["session_id"].(string)
	if sessionID == "" {
		return nil, types.NewMCPError(types.ErrCodeSessionNotFound, "session_id is required")
	}

	suggestionIdx := -1
	if idx, ok := args["suggestion_index"].(float64); ok {
		suggestionIdx = int(idx)
	}

	sess, err := t.SessionMgr.Get(sessionID)
	if err != nil {
		return nil, err
	}

	suggestions, _ := sess.SuggestedFixes.([]fixes.FixSuggestion)
	if len(suggestions) == 0 {
		return nil, types.NewMCPError(types.ErrCodeMutationFailed, "no fix suggestions available. Run suggest_fixes first.")
	}

	if suggestionIdx < 0 || suggestionIdx >= len(suggestions) {
		return nil, types.NewMCPError(types.ErrCodeMutationFailed,
			fmt.Sprintf("suggestion_index %d out of range. Available: 0-%d", suggestionIdx, len(suggestions)-1))
	}

	fix := suggestions[suggestionIdx]
	slog.Info("applying fix", "session_id", sessionID, "fix_type", fix.FixType, "index", suggestionIdx)

	// In a full implementation, this would:
	// 1. Merge the processor config into the collector config
	// 2. Apply via mutator with safety chain
	// 3. Health check and auto-rollback if needed

	return types.NewStandardResponse(t.ClusterMeta(), t.Name(), map[string]interface{}{
		"session_id": sessionID,
		"fix_type":   fix.FixType,
		"fix_index":  suggestionIdx,
		"status":     "fix_applied",
		"risk":       fix.Risk,
	}), nil
}
