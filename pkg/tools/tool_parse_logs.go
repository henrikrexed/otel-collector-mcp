package tools

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/hrexed/otel-collector-mcp/pkg/collector"
	"github.com/hrexed/otel-collector-mcp/pkg/types"
)

// ParseCollectorLogsTool parses collector pod logs and classifies errors.
type ParseCollectorLogsTool struct {
	BaseTool
}

func (t *ParseCollectorLogsTool) Name() string { return "parse_collector_logs" }

func (t *ParseCollectorLogsTool) Description() string {
	return "Parse OTel Collector pod logs and classify errors into categories (OTTL syntax, exporter failure, OOM, receiver issue, processor error)"
}

func (t *ParseCollectorLogsTool) InputSchema() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"namespace": map[string]interface{}{
				"type":        "string",
				"description": "Kubernetes namespace of the collector",
			},
			"pod": map[string]interface{}{
				"type":        "string",
				"description": "Pod name of the collector",
			},
			"tail_lines": map[string]interface{}{
				"type":        "integer",
				"description": "Number of log lines to fetch (default: 1000)",
			},
		},
		"required": []string{"namespace", "pod"},
	}
}

func (t *ParseCollectorLogsTool) Run(ctx context.Context, args map[string]interface{}) (*types.StandardResponse, error) {
	namespace, _ := args["namespace"].(string)
	pod, _ := args["pod"].(string)
	tailLines := collector.DefaultTailLines
	if v, ok := args["tail_lines"].(float64); ok {
		tailLines = int64(v)
	}

	slog.Info("parsing collector logs", "namespace", namespace, "pod", pod, "tailLines", tailLines)

	lines, err := collector.FetchPodLogs(ctx, t.Clients.Clientset, namespace, pod, tailLines)
	if err != nil {
		return types.NewStandardResponse(t.ClusterMeta(), t.Name(), &types.ToolResult{
			Findings: []types.DiagnosticFinding{{
				Severity:   types.SeverityWarning,
				Category:   types.CategoryRuntime,
				Resource:   &types.ResourceRef{Kind: "Pod", Namespace: namespace, Name: pod},
				Summary:    "Failed to fetch collector logs",
				Detail:     err.Error(),
				Suggestion: "Check RBAC permissions for pods/log access",
			}},
		}), nil
	}

	classified := collector.ClassifyCollectorLogs(lines)

	var findings []types.DiagnosticFinding
	for _, cl := range classified {
		severity := types.SeverityInfo
		switch cl.Category {
		case collector.LogCategoryOOM:
			severity = types.SeverityCritical
		case collector.LogCategoryExporterFail:
			severity = types.SeverityCritical
		case collector.LogCategoryOTTLSyntax:
			severity = types.SeverityWarning
		case collector.LogCategoryReceiverIssue, collector.LogCategoryProcessorError:
			severity = types.SeverityWarning
		}

		findings = append(findings, types.DiagnosticFinding{
			Severity: severity,
			Category: types.CategoryRuntime,
			Resource: &types.ResourceRef{Kind: "Pod", Namespace: namespace, Name: pod},
			Summary:  cl.Message,
			Detail:   cl.Line,
		})
	}

	return types.NewStandardResponse(t.ClusterMeta(), t.Name(), &types.ToolResult{
		Findings: findings,
		Metadata: map[string]string{
			"totalLines":      fmt.Sprintf("%d", len(lines)),
			"classifiedCount": fmt.Sprintf("%d", len(classified)),
		},
	}), nil
}
