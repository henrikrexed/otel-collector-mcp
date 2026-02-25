package tools

import (
	"context"
	"log/slog"
	"sort"

	"github.com/hrexed/otel-collector-mcp/pkg/analysis"
	"github.com/hrexed/otel-collector-mcp/pkg/collector"
	"github.com/hrexed/otel-collector-mcp/pkg/types"
)

// TriageScanTool runs all detection rules against a collector and returns prioritized findings.
type TriageScanTool struct {
	BaseTool
	HasOperator func() bool
}

func (t *TriageScanTool) Name() string { return "triage_scan" }

func (t *TriageScanTool) Description() string {
	return "Run all detection rules against a specified OTel Collector and return a prioritized issue list with severity rankings and specific remediation"
}

func (t *TriageScanTool) InputSchema() map[string]interface{} {
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
			"pod": map[string]interface{}{
				"type":        "string",
				"description": "Pod name for log analysis (optional — auto-discovered if not provided)",
			},
		},
		"required": []string{"namespace", "name", "configmap"},
	}
}

func (t *TriageScanTool) Run(ctx context.Context, args map[string]interface{}) (*types.StandardResponse, error) {
	namespace, _ := args["namespace"].(string)
	name, _ := args["name"].(string)
	configmap, _ := args["configmap"].(string)
	pod, _ := args["pod"].(string)

	slog.Info("running triage scan", "namespace", namespace, "name", name)

	hasOperator := false
	if t.HasOperator != nil {
		hasOperator = t.HasOperator()
	}

	// 1. Detect deployment mode
	mode, err := collector.DetectDeploymentModeWithCRD(ctx, t.Clients.Clientset, t.Clients.DynamicClient, namespace, name, hasOperator)
	if err != nil {
		mode = collector.ModeUnknown
		slog.Warn("could not detect deployment mode", "error", err)
	}

	// 2. Get collector config
	var cfg *collector.CollectorConfig
	rawConfig, err := collector.GetCollectorConfig(ctx, t.Clients.Clientset, namespace, configmap)
	if err != nil {
		slog.Warn("could not retrieve collector config", "error", err)
	} else {
		cfg, err = collector.ParseConfig(rawConfig)
		if err != nil {
			slog.Warn("could not parse collector config", "error", err)
		}
	}

	// 3. Get logs if pod name available
	var logs []string
	if pod != "" {
		logs, err = collector.FetchPodLogs(ctx, t.Clients.Clientset, namespace, pod, collector.DefaultTailLines)
		if err != nil {
			slog.Warn("could not fetch collector logs", "error", err)
		}
	}

	// 4. Build analysis input
	input := &analysis.AnalysisInput{
		Config:     cfg,
		DeployMode: mode,
		Logs:       logs,
	}

	// 5. Run all analyzers
	analyzers := analysis.AllAnalyzersIncludingLogs()
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

	// 6. Sort by severity
	sortFindings(allFindings)

	return types.NewStandardResponse(t.ClusterMeta(), t.Name(), &types.ToolResult{
		Findings: allFindings,
		Metadata: map[string]string{
			"deploymentMode": string(mode),
			"configSource":   configmap,
		},
	}), nil
}

// sortFindings sorts findings by severity: critical > warning > info > ok.
func sortFindings(findings []types.DiagnosticFinding) {
	severityOrder := map[string]int{
		types.SeverityCritical: 0,
		types.SeverityWarning:  1,
		types.SeverityInfo:     2,
		types.SeverityOk:       3,
	}

	sort.Slice(findings, func(i, j int) bool {
		return severityOrder[findings[i].Severity] < severityOrder[findings[j].Severity]
	})
}
