package analysis

import (
	"context"
	"fmt"
	"strings"

	"github.com/hrexed/otel-collector-mcp/pkg/types"
)

// backpressurePatterns are log patterns indicating exporter queue saturation.
var backpressurePatterns = []struct {
	pattern string
	message string
}{
	{"queue is full", "Exporter queue is full — data is being dropped"},
	{"dropping data", "Exporter is dropping data due to backpressure"},
	{"sending queue is full", "Sending queue is full — increase queue_size or add consumers"},
	{"failed to send", "Exporter failed to send data to backend"},
	{"context deadline exceeded", "Export timed out — backend may be slow or unreachable"},
	{"connection refused", "Backend connection refused — check endpoint and network"},
	{"retry limit reached", "Exporter retry limit reached — data permanently lost"},
}

// AnalyzeExporterBackpressure detects exporter queue saturation from collector logs.
func AnalyzeExporterBackpressure(_ context.Context, input *AnalysisInput) []types.DiagnosticFinding {
	if len(input.Logs) == 0 {
		return nil
	}

	var findings []types.DiagnosticFinding
	patternCounts := make(map[string]int)

	for _, line := range input.Logs {
		lower := strings.ToLower(line)
		for _, bp := range backpressurePatterns {
			if strings.Contains(lower, bp.pattern) {
				patternCounts[bp.message]++
			}
		}
	}

	for message, count := range patternCounts {
		severity := types.SeverityWarning
		if count > 10 {
			severity = types.SeverityCritical
		}

		findings = append(findings, types.DiagnosticFinding{
			Severity: severity,
			Category: types.CategoryRuntime,
			Summary:  message,
			Detail:   fmt.Sprintf("Detected %d occurrences in recent logs. This indicates the collector is unable to keep up with the data volume or the backend is too slow.", count),
			Suggestion: "Increase exporter queue size, add more consumers, or investigate backend performance",
			Remediation: `exporters:
  <exporter_name>:
    sending_queue:
      enabled: true
      num_consumers: 10
      queue_size: 10000
    retry_on_failure:
      enabled: true
      initial_interval: 5s
      max_interval: 30s`,
		})
	}

	return findings
}
