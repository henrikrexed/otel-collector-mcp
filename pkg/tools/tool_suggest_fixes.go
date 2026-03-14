package tools

import (
	"context"
	"log/slog"

	"github.com/hrexed/otel-collector-mcp/pkg/fixes"
	"github.com/hrexed/otel-collector-mcp/pkg/session"
	"github.com/hrexed/otel-collector-mcp/pkg/types"
)

// SuggestFixesTool generates fix suggestions for detected issues.
type SuggestFixesTool struct {
	BaseTool
	SessionMgr *session.Manager
}

func (t *SuggestFixesTool) Name() string { return "suggest_fixes" }

func (t *SuggestFixesTool) Description() string {
	return "Generate fix suggestions for detected runtime issues."
}

func (t *SuggestFixesTool) InputSchema() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"session_id": map[string]interface{}{"type": "string", "description": "Active session ID"},
		},
		"required": []string{"session_id"},
	}
}

func (t *SuggestFixesTool) Run(ctx context.Context, args map[string]interface{}) (*types.StandardResponse, error) {
	sessionID, _ := args["session_id"].(string)
	if sessionID == "" {
		return nil, types.NewMCPError(types.ErrCodeSessionNotFound, "session_id is required")
	}

	sess, err := t.SessionMgr.Get(sessionID)
	if err != nil {
		return nil, err
	}

	findings, _ := sess.Findings.([]types.DiagnosticFinding)
	if len(findings) == 0 {
		return types.NewStandardResponse(t.ClusterMeta(), t.Name(), map[string]interface{}{
			"session_id":  sessionID,
			"suggestions": []interface{}{},
			"status":      "no_findings",
		}), nil
	}

	generators := fixes.AllFixGenerators()
	var suggestions []fixes.FixSuggestion

	for i, finding := range findings {
		gen, ok := generators[finding.Category]
		if !ok {
			continue
		}
		suggestion := gen(finding, i)
		if suggestion != nil {
			suggestions = append(suggestions, *suggestion)
		}
	}

	sess.SuggestedFixes = suggestions
	slog.Info("fixes suggested", "session_id", sessionID, "count", len(suggestions))

	return types.NewStandardResponse(t.ClusterMeta(), t.Name(), map[string]interface{}{
		"session_id":  sessionID,
		"suggestions": suggestions,
		"total":       len(suggestions),
		"status":      "suggestions_ready",
	}), nil
}
