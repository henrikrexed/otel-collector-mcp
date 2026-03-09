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
	return "Retrieve the running configuration of an OTel Collector instance from an Operator CRD (spec.config) or a ConfigMap. When collector_name is provided, the CRD is tried first; falls back to ConfigMap."
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
			"collector_name": map[string]interface{}{
				"type":        "string",
				"description": "Name of the OpenTelemetryCollector CR (optional). When provided, reads config from the CRD spec.config field first, falling back to ConfigMap.",
			},
		},
		"required": []string{"namespace"},
	}
}

func (t *GetConfigTool) Run(ctx context.Context, args map[string]interface{}) (*types.StandardResponse, error) {
	namespace, _ := args["namespace"].(string)
	configmap, _ := args["configmap"].(string)
	collectorName, _ := args["collector_name"].(string)

	// Try CRD first when collector_name is provided
	if collectorName != "" {
		slog.Info("trying CRD config", "namespace", namespace, "collector", collectorName)

		rawConfig, err := collector.GetConfigFromCRD(ctx, t.Clients.DynamicClient, namespace, collectorName)
		if err == nil {
			return t.buildResponse(namespace, rawConfig, &types.ResourceRef{
				Kind:      "OpenTelemetryCollector",
				Namespace: namespace,
				Name:      collectorName,
			})
		}
		slog.Info("CRD config not found, falling back to ConfigMap", "error", err)
	}

	// Determine ConfigMap name: explicit > operator-derived > ""
	if configmap == "" && collectorName != "" {
		configmap = collectorName + "-collector"
		slog.Info("using operator-derived ConfigMap name", "configmap", configmap)
	}

	if configmap == "" {
		return types.NewStandardResponse(t.ClusterMeta(), t.Name(), &types.ToolResult{
			Findings: []types.DiagnosticFinding{{
				Severity:   types.SeverityWarning,
				Category:   types.CategoryConfig,
				Summary:    "No config source specified",
				Detail:     "Provide either collector_name (for CRD) or configmap (for ConfigMap)",
				Suggestion: "Use list_collectors to find the collector name, then pass it as collector_name.",
			}},
		}), nil
	}

	slog.Info("retrieving collector config from ConfigMap", "namespace", namespace, "configmap", configmap)

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

	return t.buildResponse(namespace, rawConfig, &types.ResourceRef{
		Kind:      "ConfigMap",
		Namespace: namespace,
		Name:      configmap,
	})
}

// buildResponse parses raw config YAML and returns a compact StandardResponse.
// Returns a structured summary instead of the full raw config to reduce token usage.
func (t *GetConfigTool) buildResponse(namespace string, rawConfig []byte, source *types.ResourceRef) (*types.StandardResponse, error) {
	parsed, err := collector.ParseConfig(rawConfig)
	if err != nil {
		return types.NewStandardResponse(t.ClusterMeta(), t.Name(), map[string]interface{}{
			"parseError": err.Error(),
			"source":     source,
			"configSize": len(rawConfig),
		}), nil
	}

	// Build compact summary: list receivers, processors, exporters, pipelines
	summary := map[string]interface{}{
		"source": source,
	}
	if parsed.Receivers != nil {
		names := make([]string, 0, len(parsed.Receivers))
		for k := range parsed.Receivers {
			names = append(names, k)
		}
		summary["receivers"] = names
	}
	if parsed.Processors != nil {
		names := make([]string, 0, len(parsed.Processors))
		for k := range parsed.Processors {
			names = append(names, k)
		}
		summary["processors"] = names
	}
	if parsed.Exporters != nil {
		names := make([]string, 0, len(parsed.Exporters))
		for k := range parsed.Exporters {
			names = append(names, k)
		}
		summary["exporters"] = names
	}
	if parsed.Connectors != nil {
		names := make([]string, 0, len(parsed.Connectors))
		for k := range parsed.Connectors {
			names = append(names, k)
		}
		summary["connectors"] = names
	}
	if len(parsed.Service.Pipelines) > 0 && parsed.Service.Pipelines != nil {
		pipelines := make(map[string]interface{})
		for name, p := range parsed.Service.Pipelines {
			pipelines[name] = map[string]interface{}{
				"receivers":  p.Receivers,
				"processors": p.Processors,
				"exporters":  p.Exporters,
			}
		}
		summary["pipelines"] = pipelines
	}
	if len(parsed.Service.Extensions) > 0 {
		summary["extensions"] = parsed.Service.Extensions
	}

	return types.NewStandardResponse(t.ClusterMeta(), t.Name(), summary), nil
}
