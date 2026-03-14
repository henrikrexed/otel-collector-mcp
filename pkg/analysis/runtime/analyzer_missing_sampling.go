package runtime

import (
	"context"
	"strings"

	"github.com/hrexed/otel-collector-mcp/pkg/types"
)

// AnalyzeMissingSampling detects when no sampling processor is configured.
func AnalyzeMissingSampling(_ context.Context, input *RuntimeAnalysisInput) []types.DiagnosticFinding {
	if input.CollectorConfig == "" {
		return nil
	}

	hasSampling := strings.Contains(input.CollectorConfig, "probabilistic_sampler") ||
		strings.Contains(input.CollectorConfig, "tail_sampling")

	if !hasSampling {
		return []types.DiagnosticFinding{
			{
				Severity:    "info",
				Category:    "sampling",
				Summary:     "No sampling processor configured",
				Remediation: "Consider adding probabilistic_sampler or tail_sampling to reduce trace volume if not intentional.",
			},
		}
	}

	return nil
}
