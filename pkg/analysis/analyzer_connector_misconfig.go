package analysis

import (
	"context"
	"fmt"

	"github.com/hrexed/otel-collector-mcp/pkg/types"
)

// AnalyzeConnectorMisconfig verifies connectors reference existing pipelines.
func AnalyzeConnectorMisconfig(_ context.Context, input *AnalysisInput) []types.DiagnosticFinding {
	if input.Config == nil || len(input.Config.Connectors) == 0 {
		return nil
	}

	var findings []types.DiagnosticFinding
	pipelineNames := make(map[string]bool)
	for name := range input.Config.Service.Pipelines {
		pipelineNames[name] = true
	}

	// Connectors appear as both exporters in one pipeline and receivers in another.
	// Check that each connector name appears in at least one pipeline as an exporter
	// and at least one pipeline as a receiver.
	for connName := range input.Config.Connectors {
		usedAsExporter := false
		usedAsReceiver := false

		for _, pipeline := range input.Config.Service.Pipelines {
			for _, exp := range pipeline.Exporters {
				if exp == connName {
					usedAsExporter = true
				}
			}
			for _, recv := range pipeline.Receivers {
				if recv == connName {
					usedAsReceiver = true
				}
			}
		}

		if !usedAsExporter && !usedAsReceiver {
			findings = append(findings, types.DiagnosticFinding{
				Severity:   types.SeverityWarning,
				Category:   types.CategoryConfig,
				Summary:    fmt.Sprintf("Connector %q is defined but not used in any pipeline", connName),
				Detail:     "This connector is configured but does not appear as an exporter or receiver in any pipeline. It will have no effect.",
				Suggestion: "Add the connector to the appropriate pipelines or remove it",
			})
		} else if !usedAsExporter {
			findings = append(findings, types.DiagnosticFinding{
				Severity:   types.SeverityWarning,
				Category:   types.CategoryConfig,
				Summary:    fmt.Sprintf("Connector %q is not used as an exporter in any pipeline", connName),
				Detail:     "A connector must appear as an exporter in one pipeline (source) and a receiver in another (destination). This connector is missing its source pipeline.",
				Suggestion: "Add the connector as an exporter in the source pipeline",
			})
		} else if !usedAsReceiver {
			findings = append(findings, types.DiagnosticFinding{
				Severity:   types.SeverityWarning,
				Category:   types.CategoryConfig,
				Summary:    fmt.Sprintf("Connector %q is not used as a receiver in any pipeline", connName),
				Detail:     "A connector must appear as an exporter in one pipeline (source) and a receiver in another (destination). This connector is missing its destination pipeline.",
				Suggestion: "Add the connector as a receiver in the destination pipeline",
			})
		}
	}

	return findings
}
