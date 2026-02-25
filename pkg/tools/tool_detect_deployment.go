package tools

import (
	"context"
	"log/slog"

	"github.com/hrexed/otel-collector-mcp/pkg/collector"
	"github.com/hrexed/otel-collector-mcp/pkg/types"
)

// DetectDeploymentTool detects the deployment type of an OTel Collector instance.
type DetectDeploymentTool struct {
	BaseTool
	HasOperator func() bool
}

func (t *DetectDeploymentTool) Name() string { return "detect_deployment_type" }

func (t *DetectDeploymentTool) Description() string {
	return "Auto-detect the deployment type (DaemonSet, Deployment, StatefulSet, or OTel Operator CRD) of an OTel Collector instance"
}

func (t *DetectDeploymentTool) InputSchema() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"namespace": map[string]interface{}{
				"type":        "string",
				"description": "Kubernetes namespace of the collector",
			},
			"name": map[string]interface{}{
				"type":        "string",
				"description": "Name of the collector workload",
			},
		},
		"required": []string{"namespace", "name"},
	}
}

func (t *DetectDeploymentTool) Run(ctx context.Context, args map[string]interface{}) (*types.StandardResponse, error) {
	namespace, _ := args["namespace"].(string)
	name, _ := args["name"].(string)

	slog.Info("detecting deployment type", "namespace", namespace, "name", name)

	hasOperator := false
	if t.HasOperator != nil {
		hasOperator = t.HasOperator()
	}

	mode, err := collector.DetectDeploymentModeWithCRD(ctx, t.Clients.Clientset, t.Clients.DynamicClient, namespace, name, hasOperator)
	if err != nil {
		return types.NewStandardResponse(t.ClusterMeta(), t.Name(), &types.ToolResult{
			Findings: []types.DiagnosticFinding{{
				Severity: types.SeverityWarning,
				Category: types.CategoryConfig,
				Resource: &types.ResourceRef{Namespace: namespace, Name: name},
				Summary:  "Collector workload not found",
				Detail:   err.Error(),
			}},
		}), nil
	}

	return types.NewStandardResponse(t.ClusterMeta(), t.Name(), map[string]interface{}{
		"namespace":      namespace,
		"name":           name,
		"deploymentMode": string(mode),
	}), nil
}
