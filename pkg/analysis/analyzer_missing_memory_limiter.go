package analysis

import (
	"context"
	"fmt"

	"github.com/hrexed/otel-collector-mcp/pkg/types"
)

// AnalyzeMissingMemoryLimiter checks each pipeline for the memory_limiter processor.
func AnalyzeMissingMemoryLimiter(_ context.Context, input *AnalysisInput) []types.DiagnosticFinding {
	if input.Config == nil {
		return nil
	}

	var findings []types.DiagnosticFinding
	for name, pipeline := range input.Config.Service.Pipelines {
		if !pipelineHasProcessor(pipeline, "memory_limiter") {
			findings = append(findings, types.DiagnosticFinding{
				Severity: types.SeverityCritical,
				Category: types.CategoryPerformance,
				Summary:  fmt.Sprintf("Pipeline %q is missing the memory_limiter processor", name),
				Detail:   "Without memory_limiter, the collector can consume unbounded memory and be OOM-killed. The memory_limiter processor should be the first processor in every pipeline to prevent this.",
				Suggestion: "Add memory_limiter as the first processor in this pipeline",
				Remediation: fmt.Sprintf(`processors:
  memory_limiter:
    check_interval: 1s
    limit_mib: 512
    spike_limit_mib: 128

service:
  pipelines:
    %s:
      processors: [memory_limiter, %s]`, name, joinProcessors(pipeline.Processors)),
			})
		}
	}
	return findings
}
