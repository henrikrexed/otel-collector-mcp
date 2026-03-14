package tools

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/hrexed/otel-collector-mcp/pkg/mutator"
	"github.com/hrexed/otel-collector-mcp/pkg/session"
	"github.com/hrexed/otel-collector-mcp/pkg/types"
)

// StartAnalysisTool initiates a safe analysis session on a collector.
type StartAnalysisTool struct {
	BaseTool
	SessionMgr *session.Manager
}

func (t *StartAnalysisTool) Name() string { return "start_analysis" }

func (t *StartAnalysisTool) Description() string {
	return "Start a v2 analysis session for a collector, enabling dynamic signal capture and mutation operations."
}

func (t *StartAnalysisTool) InputSchema() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"collector_name": map[string]interface{}{"type": "string", "description": "Collector name"},
			"namespace":      map[string]interface{}{"type": "string", "description": "Kubernetes namespace"},
			"environment":    map[string]interface{}{"type": "string", "description": "Environment type: dev, staging, production", "enum": []string{"dev", "staging", "production"}},
		},
		"required": []string{"collector_name", "namespace", "environment"},
	}
}

func (t *StartAnalysisTool) Run(ctx context.Context, args map[string]interface{}) (*types.StandardResponse, error) {
	collectorName, _ := args["collector_name"].(string)
	namespace, _ := args["namespace"].(string)
	environment, _ := args["environment"].(string)

	if collectorName == "" || namespace == "" || environment == "" {
		return nil, types.NewMCPError(types.ErrCodeMutationFailed, "collector_name, namespace, and environment are required")
	}

	// Production gate — absolute refusal, no override
	if environment == "production" {
		return nil, types.NewMCPError(types.ErrCodeProductionRefused,
			"analysis sessions are not allowed in production environments. No override or force flag exists.")
	}

	slog.Info("starting analysis session", "collector", collectorName, "namespace", namespace, "environment", environment)

	ref := mutator.CollectorRef{
		Name:      collectorName,
		Namespace: namespace,
	}

	// Create mutator
	mut := mutator.NewMutator(t.Clients.Clientset, ref)

	// Check for GitOps conflicts
	if isGitOps, warning := mut.DetectGitOps(ctx); isGitOps {
		slog.Warn("gitops conflict detected", "warning", warning)
	}

	// Create session
	sess, err := t.SessionMgr.Create(ref, environment, mut)
	if err != nil {
		return nil, err
	}

	return types.NewStandardResponse(t.ClusterMeta(), t.Name(), map[string]interface{}{
		"session_id":  sess.ID,
		"environment": environment,
		"collector":   fmt.Sprintf("%s/%s", namespace, collectorName),
		"status":      "ready_for_capture",
	}), nil
}
