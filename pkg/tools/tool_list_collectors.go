package tools

import (
	"context"
	"log/slog"

	"github.com/hrexed/otel-collector-mcp/pkg/collector"
	"github.com/hrexed/otel-collector-mcp/pkg/types"
)

// ListCollectorsTool lists all OTel Collector instances in the cluster.
type ListCollectorsTool struct {
	BaseTool
	HasOperator func() bool
}

func (t *ListCollectorsTool) Name() string { return "list_collectors" }

func (t *ListCollectorsTool) Description() string {
	return "List all OTel Collector instances across all namespaces or a specified namespace"
}

func (t *ListCollectorsTool) InputSchema() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"namespace": map[string]interface{}{
				"type":        "string",
				"description": "Kubernetes namespace to search (empty for all namespaces)",
			},
		},
	}
}

func (t *ListCollectorsTool) Run(ctx context.Context, args map[string]interface{}) (*types.StandardResponse, error) {
	namespace, _ := args["namespace"].(string)

	slog.Info("listing collectors", "namespace", namespace)

	hasOperator := false
	if t.HasOperator != nil {
		hasOperator = t.HasOperator()
	}

	collectors, err := collector.ListCollectors(ctx, t.Clients.Clientset, t.Clients.DynamicClient, namespace, hasOperator)
	if err != nil {
		return nil, err
	}

	return types.NewStandardResponse(t.ClusterMeta(), t.Name(), map[string]interface{}{
		"collectors": collectors,
		"count":      len(collectors),
	}), nil
}
