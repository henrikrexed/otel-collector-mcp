package fixes

import (
	"fmt"
	"strings"

	"github.com/hrexed/otel-collector-mcp/pkg/types"
)

// GenerateCardinalityFix generates an attributes processor fix for high-cardinality metrics.
func GenerateCardinalityFix(finding types.DiagnosticFinding, index int) *FixSuggestion {
	// Extract label keys from remediation text
	config := `processors:
  attributes/drop-high-cardinality:
    actions:
      - key: <high-cardinality-label>
        action: delete`

	return &FixSuggestion{
		FindingIndex:    index,
		FixType:         "attribute",
		Description:     "Drop high-cardinality label keys using attributes processor",
		ProcessorConfig: config,
		PipelineChanges: "Add attributes/drop-high-cardinality to pipeline processors",
		Risk:            "medium",
	}
}

// GeneratePIIFix generates an OTTL transform fix for PII detection findings.
func GeneratePIIFix(finding types.DiagnosticFinding, index int) *FixSuggestion {
	fixType := "ottl"
	var config string

	if strings.Contains(finding.Summary, "email") {
		config = `processors:
  transform/redact-pii:
    log_statements:
      - context: log
        statements:
          - replace_pattern(attributes["<attribute>"], "\\b[\\w.+-]+@[\\w-]+\\.[\\w.]+\\b", "***REDACTED***")`
	} else if strings.Contains(finding.Summary, "IP") {
		config = `processors:
  transform/redact-ip:
    log_statements:
      - context: log
        statements:
          - replace_pattern(attributes["<attribute>"], "\\b\\d{1,3}\\.\\d{1,3}\\.\\d{1,3}\\.\\d{1,3}\\b", "***REDACTED***")`
	} else {
		config = `processors:
  attributes/remove-pii:
    actions:
      - key: <pii-attribute>
        action: delete`
		fixType = "attribute"
	}

	return &FixSuggestion{
		FindingIndex:    index,
		FixType:         fixType,
		Description:     fmt.Sprintf("Redact PII from %s", finding.Summary),
		ProcessorConfig: config,
		PipelineChanges: "Add transform/redact-pii to pipeline processors",
		Risk:            "low",
	}
}

// GenerateBloatedAttrsFix generates a fix to truncate or delete bloated attributes.
func GenerateBloatedAttrsFix(finding types.DiagnosticFinding, index int) *FixSuggestion {
	config := `processors:
  transform/truncate-attrs:
    log_statements:
      - context: log
        statements:
          - truncate_all(attributes, 1024)`

	return &FixSuggestion{
		FindingIndex:    index,
		FixType:         "ottl",
		Description:     "Truncate bloated attributes to 1KB maximum",
		ProcessorConfig: config,
		PipelineChanges: "Add transform/truncate-attrs to pipeline processors",
		Risk:            "low",
	}
}

// GenerateDuplicatesFix generates a filter processor fix for duplicate signals.
func GenerateDuplicatesFix(finding types.DiagnosticFinding, index int) *FixSuggestion {
	config := `processors:
  filter/drop-duplicates:
    metrics:
      exclude:
        match_type: strict
        metric_names:
          - <duplicate-metric-name>`

	return &FixSuggestion{
		FindingIndex:    index,
		FixType:         "filter",
		Description:     "Drop duplicate metrics using filter processor",
		ProcessorConfig: config,
		PipelineChanges: "Add filter/drop-duplicates to pipeline processors",
		Risk:            "medium",
	}
}

// GenerateMissingResourceFix generates a resource processor fix for missing attributes.
func GenerateMissingResourceFix(finding types.DiagnosticFinding, index int) *FixSuggestion {
	config := `processors:
  resource/add-missing:
    attributes:
      - key: service.name
        value: <your-service-name>
        action: upsert
      - key: service.version
        value: <your-version>
        action: upsert
      - key: deployment.environment
        value: <your-environment>
        action: upsert`

	return &FixSuggestion{
		FindingIndex:    index,
		FixType:         "resource",
		Description:     "Add missing resource attributes using resource processor",
		ProcessorConfig: config,
		PipelineChanges: "Add resource/add-missing to pipeline processors",
		Risk:            "low",
	}
}
