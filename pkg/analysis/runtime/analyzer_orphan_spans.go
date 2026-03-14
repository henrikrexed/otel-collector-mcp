package runtime

import (
	"context"
	"fmt"

	"github.com/hrexed/otel-collector-mcp/pkg/types"
)

// AnalyzeOrphanSpans detects spans with no parent AND no children in the observation window.
func AnalyzeOrphanSpans(_ context.Context, input *RuntimeAnalysisInput) []types.DiagnosticFinding {
	if input.Signals == nil || len(input.Signals.Traces) == 0 {
		return nil
	}

	// Build parent/child relationships
	hasChildren := make(map[string]bool)
	hasParent := make(map[string]bool)

	for _, span := range input.Signals.Traces {
		if span.ParentSpanID != "" {
			hasParent[span.SpanID] = true
			hasChildren[span.ParentSpanID] = true
		}
	}

	var findings []types.DiagnosticFinding
	for _, span := range input.Signals.Traces {
		if !hasParent[span.SpanID] && !hasChildren[span.SpanID] {
			findings = append(findings, types.DiagnosticFinding{
				Severity:    "warning",
				Category:    "orphan_spans",
				Summary:     fmt.Sprintf("Orphan span detected: %s (no parent or children)", span.Name),
				Remediation: "Check instrumentation for missing context propagation.",
			})
		}
	}

	return findings
}
