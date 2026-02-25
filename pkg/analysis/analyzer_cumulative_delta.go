package analysis

import (
	"context"
	"fmt"

	"github.com/hrexed/otel-collector-mcp/pkg/collector"
	"github.com/hrexed/otel-collector-mcp/pkg/types"
)

// AnalyzeCumulativeDelta detects cumulative-to-delta processor on non-stateful deployments.
func AnalyzeCumulativeDelta(_ context.Context, input *AnalysisInput) []types.DiagnosticFinding {
	if input.Config == nil {
		return nil
	}

	var findings []types.DiagnosticFinding

	for pipelineName, pipeline := range input.Config.Service.Pipelines {
		if pipelineHasProcessor(pipeline, "cumulativetodelta") {
			if input.DeployMode == collector.ModeDeployment || input.DeployMode == collector.ModeDaemonSet {
				findings = append(findings, types.DiagnosticFinding{
					Severity: types.SeverityWarning,
					Category: types.CategoryConfig,
					Summary:  fmt.Sprintf("cumulativetodelta processor in pipeline %q on non-stateful deployment (%s)", pipelineName, input.DeployMode),
					Detail:   "The cumulativetodelta processor maintains state to track metric starting points. On Deployments or DaemonSets, pod restarts lose this state, causing counter resets and incorrect delta values. A StatefulSet with persistent storage preserves state across restarts.",
					Suggestion: "Use a StatefulSet with persistent storage, or accept potential counter resets on pod restarts",
					Remediation: `# Option 1: Use a StatefulSet for stateful processing
# Option 2: Accept counter resets and configure your backend to handle them
# Option 3: Use the metrics_transform processor for simpler conversions`,
				})
			}
		}
	}

	return findings
}
