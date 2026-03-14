package runtime

import (
	"context"
	"fmt"

	"github.com/hrexed/otel-collector-mcp/pkg/types"
)

const bloatedAttrThreshold = 1024 // 1KB

// AnalyzeBloatedAttributes detects attributes with values exceeding 1KB.
func AnalyzeBloatedAttributes(_ context.Context, input *RuntimeAnalysisInput) []types.DiagnosticFinding {
	if input.Signals == nil {
		return nil
	}

	var findings []types.DiagnosticFinding

	// Check span attributes
	for _, span := range input.Signals.Traces {
		for key, value := range span.Attributes {
			if len(value) > bloatedAttrThreshold {
				findings = append(findings, types.DiagnosticFinding{
					Severity:    "warning",
					Category:    "bloated_attrs",
					Summary:     fmt.Sprintf("Bloated attribute '%s' on span '%s' (~%d bytes)", key, span.Name, len(value)),
					Remediation: "Use a transform processor to truncate or remove this attribute.",
				})
			}
		}
	}

	// Check log attributes
	for _, log := range input.Signals.Logs {
		for key, value := range log.Attributes {
			if len(value) > bloatedAttrThreshold {
				findings = append(findings, types.DiagnosticFinding{
					Severity:    "warning",
					Category:    "bloated_attrs",
					Summary:     fmt.Sprintf("Bloated attribute '%s' on log record (~%d bytes)", key, len(value)),
					Remediation: "Use a transform processor to truncate or remove this attribute.",
				})
			}
		}
	}

	return findings
}
