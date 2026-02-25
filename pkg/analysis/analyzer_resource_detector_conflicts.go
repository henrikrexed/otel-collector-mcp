package analysis

import (
	"context"
	"fmt"
	"strings"

	"github.com/hrexed/otel-collector-mcp/pkg/types"
)

// AnalyzeResourceDetectorConflicts checks for multiple resource detection processors
// in the same pipeline that may overwrite each other.
func AnalyzeResourceDetectorConflicts(_ context.Context, input *AnalysisInput) []types.DiagnosticFinding {
	if input.Config == nil {
		return nil
	}

	var findings []types.DiagnosticFinding

	for pipelineName, pipeline := range input.Config.Service.Pipelines {
		var resourceDetectors []string
		for _, proc := range pipeline.Processors {
			if strings.HasPrefix(proc, "resourcedetection") || proc == "resource" {
				resourceDetectors = append(resourceDetectors, proc)
			}
		}

		if len(resourceDetectors) > 1 {
			findings = append(findings, types.DiagnosticFinding{
				Severity: types.SeverityWarning,
				Category: types.CategoryConfig,
				Summary:  fmt.Sprintf("Pipeline %q has multiple resource detection processors: %v", pipelineName, resourceDetectors),
				Detail:   "Multiple resource detection processors in the same pipeline may overwrite each other's attributes. The later processor's attributes take precedence, potentially losing information from earlier detectors.",
				Suggestion: "Merge resource detection processors into a single instance with multiple detectors, or ensure the override order is intentional",
				Remediation: `# Merge multiple resource detection processors into one:
processors:
  resourcedetection:
    detectors: [env, system, gcp, eks, azure]
    timeout: 5s
    override: false  # Set to false to preserve existing attributes`,
			})
		}
	}

	return findings
}
