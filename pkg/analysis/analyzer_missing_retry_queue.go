package analysis

import (
	"context"
	"fmt"

	"github.com/hrexed/otel-collector-mcp/pkg/types"
)

// AnalyzeMissingRetryQueue checks exporters for retry_on_failure and sending_queue settings.
func AnalyzeMissingRetryQueue(_ context.Context, input *AnalysisInput) []types.DiagnosticFinding {
	if input.Config == nil {
		return nil
	}

	var findings []types.DiagnosticFinding
	for name, exporterCfg := range input.Config.Exporters {
		cfgMap, ok := exporterCfg.(map[string]interface{})
		if !ok {
			continue
		}

		_, hasRetry := cfgMap["retry_on_failure"]
		_, hasQueue := cfgMap["sending_queue"]

		if !hasRetry {
			findings = append(findings, types.DiagnosticFinding{
				Severity:   types.SeverityWarning,
				Category:   types.CategoryPerformance,
				Summary:    fmt.Sprintf("Exporter %q is missing retry_on_failure configuration", name),
				Detail:     "Without retry configuration, transient export failures will cause permanent data loss. The retry_on_failure setting enables automatic retries with exponential backoff.",
				Suggestion: "Add retry_on_failure to this exporter",
				Remediation: fmt.Sprintf(`exporters:
  %s:
    retry_on_failure:
      enabled: true
      initial_interval: 5s
      max_interval: 30s
      max_elapsed_time: 300s`, name),
			})
		}

		if !hasQueue {
			findings = append(findings, types.DiagnosticFinding{
				Severity:   types.SeverityWarning,
				Category:   types.CategoryPerformance,
				Summary:    fmt.Sprintf("Exporter %q is missing sending_queue configuration", name),
				Detail:     "Without a sending queue, the exporter processes data synchronously. If the backend is slow, this causes backpressure that propagates to receivers. A sending queue buffers data and decouples the exporter from the pipeline.",
				Suggestion: "Add sending_queue to this exporter",
				Remediation: fmt.Sprintf(`exporters:
  %s:
    sending_queue:
      enabled: true
      num_consumers: 10
      queue_size: 5000`, name),
			})
		}
	}
	return findings
}
