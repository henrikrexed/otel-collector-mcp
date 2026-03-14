package tools

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/hrexed/otel-collector-mcp/pkg/session"
	"github.com/hrexed/otel-collector-mcp/pkg/signals"
	"github.com/hrexed/otel-collector-mcp/pkg/types"
)

// RecommendSizingTool estimates recommended resource limits based on observed throughput.
type RecommendSizingTool struct {
	BaseTool
	SessionMgr *session.Manager
}

func (t *RecommendSizingTool) Name() string { return "recommend_sizing" }

func (t *RecommendSizingTool) Description() string {
	return "Analyze resource usage and recommend CPU/memory limits for the collector."
}

func (t *RecommendSizingTool) InputSchema() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"session_id": map[string]interface{}{"type": "string", "description": "Active session ID"},
		},
		"required": []string{"session_id"},
	}
}

func (t *RecommendSizingTool) Run(ctx context.Context, args map[string]interface{}) (*types.StandardResponse, error) {
	sessionID, _ := args["session_id"].(string)
	if sessionID == "" {
		return nil, types.NewMCPError(types.ErrCodeSessionNotFound, "session_id is required")
	}

	sess, err := t.SessionMgr.Get(sessionID)
	if err != nil {
		return nil, err
	}

	captured, _ := sess.CapturedSignals.(*signals.CapturedSignals)
	if captured == nil {
		captured = &signals.CapturedSignals{}
	}

	duration := captured.Duration.Seconds()
	if duration <= 0 {
		duration = 1
	}

	metricsPerSec := float64(len(captured.Metrics)) / duration
	logsPerSec := float64(len(captured.Logs)) / duration
	spansPerSec := float64(len(captured.Traces)) / duration
	totalPerSec := metricsPerSec + logsPerSec + spansPerSec

	// Simple sizing heuristic based on throughput
	cpuRequest := "100m"
	cpuLimit := "500m"
	memRequest := "128Mi"
	memLimit := "256Mi"
	rationale := "Low throughput — default sizing is adequate."

	if totalPerSec > 1000 {
		cpuRequest = "250m"
		cpuLimit = "1000m"
		memRequest = "256Mi"
		memLimit = "512Mi"
		rationale = fmt.Sprintf("Moderate throughput (%.0f signals/sec) — increased limits recommended.", totalPerSec)
	}
	if totalPerSec > 10000 {
		cpuRequest = "500m"
		cpuLimit = "2000m"
		memRequest = "512Mi"
		memLimit = "1Gi"
		rationale = fmt.Sprintf("High throughput (%.0f signals/sec) — significantly increased limits recommended.", totalPerSec)
	}

	slog.Info("sizing recommendation", "session_id", sessionID, "total_per_sec", totalPerSec)

	return types.NewStandardResponse(t.ClusterMeta(), t.Name(), map[string]interface{}{
		"session_id": sessionID,
		"observed_throughput": map[string]interface{}{
			"metrics_per_sec": fmt.Sprintf("%.1f", metricsPerSec),
			"logs_per_sec":    fmt.Sprintf("%.1f", logsPerSec),
			"spans_per_sec":   fmt.Sprintf("%.1f", spansPerSec),
			"total_per_sec":   fmt.Sprintf("%.1f", totalPerSec),
		},
		"recommendation": map[string]interface{}{
			"cpu_request": cpuRequest,
			"cpu_limit":   cpuLimit,
			"mem_request": memRequest,
			"mem_limit":   memLimit,
			"rationale":   rationale,
		},
	}), nil
}
