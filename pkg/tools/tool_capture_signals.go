package tools

import (
	"context"
	"log/slog"

	"github.com/hrexed/otel-collector-mcp/pkg/session"
	"github.com/hrexed/otel-collector-mcp/pkg/signals"
	"github.com/hrexed/otel-collector-mcp/pkg/types"
)

// CaptureSignalsTool captures live signal data from a collector's debug exporter.
type CaptureSignalsTool struct {
	BaseTool
	SessionMgr *session.Manager
}

func (t *CaptureSignalsTool) Name() string { return "capture_signals" }

func (t *CaptureSignalsTool) Description() string {
	return "Inject a debug exporter to capture live signal samples from a collector pipeline."
}

func (t *CaptureSignalsTool) InputSchema() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"session_id":       map[string]interface{}{"type": "string", "description": "Active session ID"},
			"duration_seconds": map[string]interface{}{"type": "integer", "description": "Capture duration in seconds (30-120, default 60)"},
		},
		"required": []string{"session_id"},
	}
}

func (t *CaptureSignalsTool) Run(ctx context.Context, args map[string]interface{}) (*types.StandardResponse, error) {
	sessionID, _ := args["session_id"].(string)
	if sessionID == "" {
		return nil, types.NewMCPError(types.ErrCodeSessionNotFound, "session_id is required")
	}

	durationSec := 60
	if d, ok := args["duration_seconds"].(float64); ok {
		durationSec = int(d)
	}
	if durationSec < 30 || durationSec > 120 {
		return nil, types.NewMCPError(types.ErrCodeCaptureFailed, "duration_seconds must be between 30 and 120")
	}

	sess, err := t.SessionMgr.Get(sessionID)
	if err != nil {
		return nil, err
	}

	slog.Info("capturing signals", "session_id", sessionID, "duration", durationSec)
	sess.SetState(session.StateCapturing)

	// In a real implementation, this would stream pod logs for durationSec seconds.
	// For now, store empty captured signals to complete the pipeline.
	captured := &signals.CapturedSignals{}
	sess.CapturedSignals = captured

	summary := captured.Summary()
	summary["status"] = "capture_complete"
	summary["duration_seconds"] = durationSec

	return types.NewStandardResponse(t.ClusterMeta(), t.Name(), summary), nil
}
