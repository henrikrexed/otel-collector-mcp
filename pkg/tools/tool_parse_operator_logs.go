package tools

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/hrexed/otel-collector-mcp/pkg/collector"
	"github.com/hrexed/otel-collector-mcp/pkg/types"
)

// ParseOperatorLogsTool parses OTel Operator pod logs for CRD and reconciliation failures.
type ParseOperatorLogsTool struct {
	BaseTool
	HasOperator func() bool
}

func (t *ParseOperatorLogsTool) Name() string { return "parse_operator_logs" }

func (t *ParseOperatorLogsTool) Description() string {
	return "Parse OTel Operator pod logs to detect rejected CRDs and reconciliation failures"
}

func (t *ParseOperatorLogsTool) InputSchema() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"namespace": map[string]interface{}{
				"type":        "string",
				"description": "Namespace where the OTel Operator is running (default: opentelemetry-operator-system)",
			},
		},
	}
}

func (t *ParseOperatorLogsTool) Run(ctx context.Context, args map[string]interface{}) (*types.StandardResponse, error) {
	namespace := "opentelemetry-operator-system"
	if v, ok := args["namespace"].(string); ok && v != "" {
		namespace = v
	}

	slog.Info("parsing operator logs", "namespace", namespace)

	// Find operator pods
	podNames, err := collector.FindPodsByLabel(ctx, t.Clients.Clientset, namespace, "app.kubernetes.io/name=opentelemetry-operator")
	if err != nil || len(podNames) == 0 {
		return types.NewStandardResponse(t.ClusterMeta(), t.Name(), &types.ToolResult{
			Findings: []types.DiagnosticFinding{{
				Severity:   types.SeverityWarning,
				Category:   types.CategoryOperator,
				Summary:    "OTel Operator pods not found",
				Detail:     fmt.Sprintf("No pods found with label app.kubernetes.io/name=opentelemetry-operator in namespace %s", namespace),
				Suggestion: "Verify the Operator is installed and the namespace is correct",
			}},
		}), nil
	}

	var allFindings []types.DiagnosticFinding
	for _, podName := range podNames {
		lines, err := collector.FetchPodLogs(ctx, t.Clients.Clientset, namespace, podName, collector.DefaultTailLines)
		if err != nil {
			allFindings = append(allFindings, types.DiagnosticFinding{
				Severity: types.SeverityWarning,
				Category: types.CategoryOperator,
				Resource: &types.ResourceRef{Kind: "Pod", Namespace: namespace, Name: podName},
				Summary:  "Failed to fetch operator logs",
				Detail:   err.Error(),
			})
			continue
		}

		classified := collector.ClassifyOperatorLogs(lines)
		for _, cl := range classified {
			severity := types.SeverityWarning
			if cl.Category == collector.LogCategoryOperatorCRD {
				severity = types.SeverityCritical
			}

			allFindings = append(allFindings, types.DiagnosticFinding{
				Severity: severity,
				Category: types.CategoryOperator,
				Resource: &types.ResourceRef{Kind: "Pod", Namespace: namespace, Name: podName},
				Summary:  cl.Message,
				Detail:   cl.Line,
			})
		}
	}

	return types.NewStandardResponse(t.ClusterMeta(), t.Name(), &types.ToolResult{
		Findings: allFindings,
	}), nil
}
