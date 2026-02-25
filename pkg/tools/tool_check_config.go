package tools

import (
	"context"
	"log/slog"

	"github.com/hrexed/otel-collector-mcp/pkg/analysis"
	"github.com/hrexed/otel-collector-mcp/pkg/collector"
	"github.com/hrexed/otel-collector-mcp/pkg/types"
)

// CheckConfigTool runs the misconfig detection suite without log analysis.
type CheckConfigTool struct {
	BaseTool
	HasOperator func() bool
}

func (t *CheckConfigTool) Name() string { return "check_config" }

func (t *CheckConfigTool) Description() string {
	return "Run the misconfig detection suite against a collector's configuration without log analysis"
}

func (t *CheckConfigTool) InputSchema() map[string]interface{} {
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
			"configmap": map[string]interface{}{
				"type":        "string",
				"description": "Name of the ConfigMap containing collector configuration",
			},
		},
		"required": []string{"namespace", "name", "configmap"},
	}
}

func (t *CheckConfigTool) Run(ctx context.Context, args map[string]interface{}) (*types.StandardResponse, error) {
	namespace, _ := args["namespace"].(string)
	name, _ := args["name"].(string)
	configmap, _ := args["configmap"].(string)

	slog.Info("running config check", "namespace", namespace, "name", name)

	hasOperator := false
	if t.HasOperator != nil {
		hasOperator = t.HasOperator()
	}

	// Detect deployment mode
	mode, err := collector.DetectDeploymentModeWithCRD(ctx, t.Clients.Clientset, t.Clients.DynamicClient, namespace, name, hasOperator)
	if err != nil {
		mode = collector.ModeUnknown
	}

	// Get and parse config
	rawConfig, err := collector.GetCollectorConfig(ctx, t.Clients.Clientset, namespace, configmap)
	if err != nil {
		return types.NewStandardResponse(t.ClusterMeta(), t.Name(), &types.ToolResult{
			Findings: []types.DiagnosticFinding{{
				Severity: types.SeverityWarning,
				Category: types.CategoryConfig,
				Summary:  "Failed to retrieve collector configuration",
				Detail:   err.Error(),
			}},
		}), nil
	}

	cfg, err := collector.ParseConfig(rawConfig)
	if err != nil {
		return types.NewStandardResponse(t.ClusterMeta(), t.Name(), &types.ToolResult{
			Findings: []types.DiagnosticFinding{{
				Severity: types.SeverityWarning,
				Category: types.CategoryConfig,
				Summary:  "Failed to parse collector configuration",
				Detail:   err.Error(),
			}},
		}), nil
	}

	// Run config-only analyzers (not log-based)
	input := &analysis.AnalysisInput{
		Config:     cfg,
		DeployMode: mode,
	}

	analyzers := analysis.AllAnalyzers()
	var allFindings []types.DiagnosticFinding

	for _, analyzer := range analyzers {
		func() {
			defer func() {
				if r := recover(); r != nil {
					slog.Error("analyzer panicked", "error", r)
					allFindings = append(allFindings, types.DiagnosticFinding{
						Severity: types.SeverityInfo,
						Category: types.CategoryConfig,
						Summary:  "An analyzer failed to execute",
						Detail:   "One of the detection rules encountered an unexpected error. Other rules were not affected.",
					})
				}
			}()
			findings := analyzer(ctx, input)
			allFindings = append(allFindings, findings...)
		}()
	}

	sortFindings(allFindings)

	return types.NewStandardResponse(t.ClusterMeta(), t.Name(), &types.ToolResult{
		Findings: allFindings,
		Metadata: map[string]string{
			"deploymentMode": string(mode),
		},
	}), nil
}
