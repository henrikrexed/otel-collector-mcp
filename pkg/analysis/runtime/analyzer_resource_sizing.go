package runtime

import (
	"context"
	"fmt"

	"github.com/hrexed/otel-collector-mcp/pkg/types"
)

// AnalyzeResourceSizing compares observed throughput against resource limits.
func AnalyzeResourceSizing(_ context.Context, input *RuntimeAnalysisInput) []types.DiagnosticFinding {
	if input.Signals == nil {
		return nil
	}

	duration := input.Signals.Duration.Seconds()
	if duration <= 0 {
		return nil
	}

	totalPoints := len(input.Signals.Metrics) + len(input.Signals.Logs) + len(input.Signals.Traces)
	throughput := float64(totalPoints) / duration

	var findings []types.DiagnosticFinding

	// High throughput warning (>10k points/sec suggests potential resource pressure)
	if throughput > 10000 {
		findings = append(findings, types.DiagnosticFinding{
			Severity:    "warning",
			Category:    "sizing",
			Summary:     fmt.Sprintf("High signal throughput: %.0f data points/sec observed", throughput),
			Remediation: "Review collector CPU/memory limits. Consider scaling or adding sampling.",
		})
	}

	return findings
}
