package analysis

import (
	"context"
	"fmt"

	"github.com/hrexed/otel-collector-mcp/pkg/types"
)

// AnalyzeMissingBatch checks each pipeline for the presence of a batch processor.
func AnalyzeMissingBatch(_ context.Context, input *AnalysisInput) []types.DiagnosticFinding {
	if input.Config == nil {
		return nil
	}

	var findings []types.DiagnosticFinding
	for name, pipeline := range input.Config.Service.Pipelines {
		if !pipelineHasProcessor(pipeline, "batch") {
			findings = append(findings, types.DiagnosticFinding{
				Severity: types.SeverityWarning,
				Category: types.CategoryPerformance,
				Summary:  fmt.Sprintf("Pipeline %q is missing the batch processor", name),
				Detail:   "The batch processor groups data before sending to exporters, reducing network overhead and improving throughput. Without it, each piece of telemetry is sent individually.",
				Suggestion: "Add the batch processor to this pipeline",
				Remediation: fmt.Sprintf(`processors:
  batch:
    send_batch_size: 8192
    timeout: 200ms

service:
  pipelines:
    %s:
      processors: [batch, %s]`, name, joinProcessors(pipeline.Processors)),
			})
		}
	}
	return findings
}

func joinProcessors(processors []string) string {
	result := ""
	for i, p := range processors {
		if i > 0 {
			result += ", "
		}
		result += p
	}
	return result
}
