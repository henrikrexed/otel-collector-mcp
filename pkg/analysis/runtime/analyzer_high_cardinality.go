package runtime

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/hrexed/otel-collector-mcp/pkg/types"
)

const cardinalityThreshold = 100

// AnalyzeHighCardinality detects metrics with high-cardinality label dimensions.
func AnalyzeHighCardinality(_ context.Context, input *RuntimeAnalysisInput) []types.DiagnosticFinding {
	if input.Signals == nil || len(input.Signals.Metrics) == 0 {
		return nil
	}

	// Group by metric name, collect unique label combinations
	type metricStats struct {
		uniqueCombos map[string]struct{}
		labelKeys    map[string]struct{}
	}

	stats := make(map[string]*metricStats)
	for _, dp := range input.Signals.Metrics {
		ms, ok := stats[dp.Name]
		if !ok {
			ms = &metricStats{
				uniqueCombos: make(map[string]struct{}),
				labelKeys:    make(map[string]struct{}),
			}
			stats[dp.Name] = ms
		}

		// Build combo key from sorted labels
		var parts []string
		for k, v := range dp.Labels {
			ms.labelKeys[k] = struct{}{}
			parts = append(parts, k+"="+v)
		}
		sort.Strings(parts)
		combo := strings.Join(parts, ",")
		ms.uniqueCombos[combo] = struct{}{}
	}

	var findings []types.DiagnosticFinding
	for name, ms := range stats {
		if len(ms.uniqueCombos) > cardinalityThreshold {
			var keys []string
			for k := range ms.labelKeys {
				keys = append(keys, k)
			}
			sort.Strings(keys)

			findings = append(findings, types.DiagnosticFinding{
				Severity:    "warning",
				Category:    "cardinality",
				Summary:     fmt.Sprintf("High-cardinality metric: %s (%d unique combinations)", name, len(ms.uniqueCombos)),
				Remediation: fmt.Sprintf("Review label keys [%s] for unbounded values. Consider using OTTL to drop or aggregate high-cardinality dimensions.", strings.Join(keys, ", ")),
			})
		}
	}

	return findings
}
