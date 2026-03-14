package runtime

import (
	"context"
	"fmt"
	"strings"

	"github.com/hrexed/otel-collector-mcp/pkg/types"
)

var requiredResourceAttrs = []string{"service.name", "service.version", "deployment.environment"}

// AnalyzeMissingResources detects missing or invalid resource attributes.
func AnalyzeMissingResources(_ context.Context, input *RuntimeAnalysisInput) []types.DiagnosticFinding {
	if input.Signals == nil {
		return nil
	}

	// Collect all resource attributes seen
	seen := make(map[string]map[string]struct{})
	for _, log := range input.Signals.Logs {
		for k, v := range log.ResourceAttributes {
			if seen[k] == nil {
				seen[k] = make(map[string]struct{})
			}
			seen[k][v] = struct{}{}
		}
	}

	var findings []types.DiagnosticFinding
	var missing []string

	for _, attr := range requiredResourceAttrs {
		values, exists := seen[attr]
		if !exists {
			missing = append(missing, attr)
			continue
		}
		for v := range values {
			if v == "" || v == "unknown" {
				missing = append(missing, attr+" (set to '"+v+"')")
			}
		}
	}

	if len(missing) > 0 {
		findings = append(findings, types.DiagnosticFinding{
			Severity:    "warning",
			Category:    "missing_resource",
			Summary:     fmt.Sprintf("Missing or invalid resource attributes: %s", strings.Join(missing, ", ")),
			Remediation: "Add a resource processor to set these attributes.",
		})
	}

	return findings
}
