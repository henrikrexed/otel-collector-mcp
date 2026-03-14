package tools

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/hrexed/otel-collector-mcp/pkg/mutator"
	"github.com/hrexed/otel-collector-mcp/pkg/types"
)

// CheckHealthTool reports per-pod health status for a collector.
type CheckHealthTool struct {
	BaseTool
}

func (t *CheckHealthTool) Name() string { return "check_health" }

func (t *CheckHealthTool) Description() string {
	return "Check real-time health of a collector: pod phase, readiness, CrashLoopBackOff detection, per-pod status."
}

func (t *CheckHealthTool) InputSchema() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"name":      map[string]interface{}{"type": "string", "description": "Collector name"},
			"namespace": map[string]interface{}{"type": "string", "description": "Kubernetes namespace"},
		},
		"required": []string{"name", "namespace"},
	}
}

func (t *CheckHealthTool) Run(ctx context.Context, args map[string]interface{}) (*types.StandardResponse, error) {
	name, _ := args["name"].(string)
	namespace, _ := args["namespace"].(string)

	if name == "" || namespace == "" {
		return nil, types.NewMCPError(types.ErrCodeCollectorNotFound, "name and namespace are required")
	}

	slog.Info("checking collector health", "name", name, "namespace", namespace)

	health, err := mutator.CheckCollectorHealth(ctx, t.Clients.Clientset, namespace, name)
	if err != nil {
		return nil, types.NewMCPError(types.ErrCodeHealthCheckFailed, fmt.Sprintf("health check failed: %v", err))
	}

	if health.Status == mutator.StatusNotFound {
		return nil, types.NewMCPError(types.ErrCodeCollectorNotFound, fmt.Sprintf("no pods found for collector %s/%s", namespace, name))
	}

	// Build markdown table response
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("healthy=%v status=%s pods=%d\n\n", health.Healthy, health.Status, len(health.Pods)))
	sb.WriteString("| Pod | Phase | Ready | Restarts | Age |\n")
	sb.WriteString("|-----|-------|-------|----------|-----|\n")
	for _, pod := range health.Pods {
		age := pod.Age.Truncate(time.Second).String()
		sb.WriteString(fmt.Sprintf("| %s | %s | %v | %d | %s |\n", pod.Name, pod.Phase, pod.Ready, pod.Restarts, age))
	}

	return types.NewStandardResponse(t.ClusterMeta(), t.Name(), map[string]interface{}{
		"healthy": health.Healthy,
		"status":  string(health.Status),
		"pods":    len(health.Pods),
		"details": sb.String(),
	}), nil
}
