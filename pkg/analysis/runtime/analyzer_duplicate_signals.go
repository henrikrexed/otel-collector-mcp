package runtime

import (
	"context"
	"fmt"
	"strings"

	"github.com/hrexed/otel-collector-mcp/pkg/types"
)

// AnalyzeDuplicateSignals detects duplicate metrics from different sources.
func AnalyzeDuplicateSignals(_ context.Context, input *RuntimeAnalysisInput) []types.DiagnosticFinding {
	if input.Signals == nil || len(input.Signals.Metrics) == 0 {
		return nil
	}

	// Group metrics by name, count unique data points
	metricCounts := make(map[string]int)
	for _, dp := range input.Signals.Metrics {
		metricCounts[dp.Name]++
	}

	var findings []types.DiagnosticFinding
	var duplicates []string
	for name, count := range metricCounts {
		if count > 1 {
			duplicates = append(duplicates, fmt.Sprintf("%s (%d points)", name, count))
		}
	}

	if len(duplicates) > 10 {
		findings = append(findings, types.DiagnosticFinding{
			Severity:    "info",
			Category:    "duplicates",
			Summary:     fmt.Sprintf("Found %d metrics with multiple data points: %s", len(duplicates), strings.Join(duplicates[:5], ", ")),
			Remediation: "Review for duplicate collection. Consider using a filter processor to deduplicate.",
		})
	}

	return findings
}
