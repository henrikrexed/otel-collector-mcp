package tools

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/hrexed/otel-collector-mcp/pkg/session"
	"github.com/hrexed/otel-collector-mcp/pkg/signals"
	"github.com/hrexed/otel-collector-mcp/pkg/types"
)

// RecommendSamplingTool analyzes captured trace data and recommends a sampling strategy.
type RecommendSamplingTool struct {
	BaseTool
	SessionMgr *session.Manager
}

func (t *RecommendSamplingTool) Name() string { return "recommend_sampling" }

func (t *RecommendSamplingTool) Description() string {
	return "Analyze signal volume and recommend tail-sampling or probabilistic-sampling strategies."
}

func (t *RecommendSamplingTool) InputSchema() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"session_id": map[string]interface{}{"type": "string", "description": "Active session ID"},
		},
		"required": []string{"session_id"},
	}
}

func (t *RecommendSamplingTool) Run(ctx context.Context, args map[string]interface{}) (*types.StandardResponse, error) {
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

	totalSpans := len(captured.Traces)
	duration := captured.Duration.Seconds()
	spansPerSec := 0.0
	if duration > 0 {
		spansPerSec = float64(totalSpans) / duration
	}

	// Unique services
	services := make(map[string]struct{})
	for _, span := range captured.Traces {
		if svc, ok := span.Attributes["service.name"]; ok {
			services[svc] = struct{}{}
		}
	}

	// Determine recommendation
	strategy := "none"
	rationale := "Trace volume is low; sampling may not be needed."
	config := ""
	estimatedReduction := "0%"

	if spansPerSec > 100 {
		strategy = "tail_sampling"
		rationale = fmt.Sprintf("High trace volume (%.0f spans/sec). Tail sampling recommended to preserve errors and high-latency traces.", spansPerSec)
		estimatedReduction = "60-80%"
		config = `processors:
  tail_sampling:
    decision_wait: 10s
    num_traces: 100000
    policies:
      - name: errors
        type: status_code
        status_code: {status_codes: [ERROR]}
      - name: high-latency
        type: latency
        latency: {threshold_ms: 1000}
      - name: probabilistic
        type: probabilistic
        probabilistic: {sampling_percentage: 10}`
	} else if spansPerSec > 10 {
		strategy = "probabilistic"
		rationale = fmt.Sprintf("Moderate trace volume (%.0f spans/sec). Probabilistic sampling can reduce volume.", spansPerSec)
		estimatedReduction = "50%"
		config = `processors:
  probabilistic_sampler:
    sampling_percentage: 50`
	}

	slog.Info("sampling recommendation", "session_id", sessionID, "strategy", strategy, "spans_per_sec", spansPerSec)

	return types.NewStandardResponse(t.ClusterMeta(), t.Name(), map[string]interface{}{
		"session_id": sessionID,
		"trace_analysis": map[string]interface{}{
			"total_spans":     totalSpans,
			"spans_per_sec":   fmt.Sprintf("%.1f", spansPerSec),
			"unique_services": len(services),
		},
		"recommendation": map[string]interface{}{
			"strategy":           strategy,
			"config":             config,
			"estimated_reduction": estimatedReduction,
			"rationale":          rationale,
		},
	}), nil
}
