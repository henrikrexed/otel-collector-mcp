package tools

import (
	"context"
	"log/slog"

	"github.com/hrexed/otel-collector-mcp/pkg/collector"
	"github.com/hrexed/otel-collector-mcp/pkg/types"
)

// GetConfigTool retrieves the running configuration of a collector instance.
type GetConfigTool struct {
	BaseTool
}

func (t *GetConfigTool) Name() string { return "get_config" }

func (t *GetConfigTool) Description() string {
	return "Retrieve the running configuration of a detected OTel Collector instance"
}

func (t *GetConfigTool) InputSchema() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"namespace": map[string]interface{}{
				"type":        "string",
				"description": "Kubernetes namespace of the collector",
			},
			"configmap": map[string]interface{}{
				"type":        "string",
				"description": "Name of the ConfigMap containing the collector configuration",
			},
		},
		"required": []string{"namespace", "configmap"},
	}
}

func (t *GetConfigTool) Run(ctx context.Context, args map[string]interface{}) (*types.StandardResponse, error) {
	namespace, _ := args["namespace"].(string)
	configmap, _ := args["configmap"].(string)

	slog.Info("retrieving collector config", "namespace", namespace, "configmap", configmap)

	rawConfig, err := collector.GetCollectorConfig(ctx, t.Clients.Clientset, namespace, configmap)
	if err != nil {
		return types.NewStandardResponse(t.ClusterMeta(), t.Name(), &types.ToolResult{
			Findings: []types.DiagnosticFinding{{
				Severity: types.SeverityWarning,
				Category: types.CategoryConfig,
				Resource: &types.ResourceRef{Kind: "ConfigMap", Namespace: namespace, Name: configmap},
				Summary:  "Failed to retrieve collector configuration",
				Detail:   err.Error(),
			}},
		}), nil
	}

	parsed, err := collector.ParseConfig(rawConfig)
	if err != nil {
		return types.NewStandardResponse(t.ClusterMeta(), t.Name(), map[string]interface{}{
			"raw":        string(rawConfig),
			"parsed":     nil,
			"parseError": err.Error(),
		}), nil
	}

	return types.NewStandardResponse(t.ClusterMeta(), t.Name(), map[string]interface{}{
		"raw":    string(rawConfig),
		"parsed": parsed,
	}), nil
}
