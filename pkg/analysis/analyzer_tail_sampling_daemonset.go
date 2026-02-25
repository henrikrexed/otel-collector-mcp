package analysis

import (
	"context"

	"github.com/hrexed/otel-collector-mcp/pkg/collector"
	"github.com/hrexed/otel-collector-mcp/pkg/types"
)

// AnalyzeTailSamplingDaemonSet detects tail_sampling processor on DaemonSet deployments.
func AnalyzeTailSamplingDaemonSet(_ context.Context, input *AnalysisInput) []types.DiagnosticFinding {
	if input.Config == nil || input.DeployMode != collector.ModeDaemonSet {
		return nil
	}

	var findings []types.DiagnosticFinding
	for name, pipeline := range input.Config.Service.Pipelines {
		if pipelineHasProcessor(pipeline, "tail_sampling") {
			findings = append(findings, types.DiagnosticFinding{
				Severity: types.SeverityCritical,
				Category: types.CategoryConfig,
				Summary:  "Tail sampling configured on a DaemonSet collector in pipeline " + name,
				Detail:   "Tail sampling requires all spans for a trace to reach the same collector instance. In a DaemonSet deployment, spans from different pods land on different collector nodes, making tail sampling decisions incorrect. This leads to incomplete traces and sampling bias.",
				Suggestion: "Move tail sampling to a gateway collector running as a Deployment or StatefulSet",
				Remediation: `# Tail sampling must run on a centralized gateway, not per-node agents.
# Architecture pattern:
#   DaemonSet (agent) -> Deployment/StatefulSet (gateway with tail_sampling)
#
# Remove tail_sampling from the DaemonSet pipeline and configure it
# on a centralized gateway collector instead.`,
			})
		}
	}
	return findings
}
