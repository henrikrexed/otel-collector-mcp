package analysis

import (
	"context"
	"fmt"
	"strings"

	"github.com/hrexed/otel-collector-mcp/pkg/types"
)

// highCardinalityPatterns are attribute names that commonly produce high cardinality.
var highCardinalityPatterns = []string{
	"http.url", "url.full", "http.target",
	"db.statement", "db.query.text",
	"user.id", "session.id", "request.id",
	"ip", "client.address",
}

// AnalyzeHighCardinality checks for patterns that may produce high-cardinality metric labels.
func AnalyzeHighCardinality(_ context.Context, input *AnalysisInput) []types.DiagnosticFinding {
	if input.Config == nil {
		return nil
	}

	var findings []types.DiagnosticFinding

	// Check metric pipeline processors for attributes that commonly cause high cardinality
	for pipelineName, pipeline := range input.Config.Service.Pipelines {
		if !strings.HasPrefix(pipelineName, "metrics") {
			continue
		}

		// Check if there's a filter or attributes processor to control cardinality
		hasFilter := false
		for _, proc := range pipeline.Processors {
			if strings.HasPrefix(proc, "filter") || strings.HasPrefix(proc, "attributes") || strings.HasPrefix(proc, "metricstransform") {
				hasFilter = true
				break
			}
		}

		if !hasFilter && len(pipeline.Receivers) > 0 {
			findings = append(findings, types.DiagnosticFinding{
				Severity: types.SeverityInfo,
				Category: types.CategoryPerformance,
				Summary:  fmt.Sprintf("Metrics pipeline %q has no cardinality control processor", pipelineName),
				Detail:   "Without filter, attributes, or metricstransform processors, all metric labels are passed through unmodified. High-cardinality labels (like URLs, user IDs, or IP addresses) can cause metric explosion in your backend.",
				Suggestion: "Consider adding a filter or attributes processor to control metric cardinality",
				Remediation: fmt.Sprintf(`# Add an attributes processor to drop high-cardinality labels:
processors:
  attributes/drop-high-card:
    actions:
      - key: http.url
        action: delete
      - key: url.full
        action: delete

service:
  pipelines:
    %s:
      processors: [attributes/drop-high-card, %s]`, pipelineName, joinProcessors(pipeline.Processors)),
			})
		}
	}

	return findings
}
