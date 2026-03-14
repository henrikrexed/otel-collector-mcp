package fixes

import "github.com/hrexed/otel-collector-mcp/pkg/types"

// FixSuggestion represents a generated fix for a detected issue.
type FixSuggestion struct {
	FindingIndex    int    `json:"finding_index"`
	FixType         string `json:"fix_type"` // ottl, filter, attribute, resource, config
	Description     string `json:"description"`
	ProcessorConfig string `json:"processor_config"` // Complete YAML config block
	PipelineChanges string `json:"pipeline_changes"`
	Risk            string `json:"risk"` // low, medium, high
}

// FixGenerator generates fix suggestions for a specific category of findings.
type FixGenerator func(finding types.DiagnosticFinding, index int) *FixSuggestion

// AllFixGenerators returns all registered fix generators keyed by finding category.
func AllFixGenerators() map[string]FixGenerator {
	return map[string]FixGenerator{
		"cardinality":     GenerateCardinalityFix,
		"pii":             GeneratePIIFix,
		"bloated_attrs":   GenerateBloatedAttrsFix,
		"duplicates":      GenerateDuplicatesFix,
		"missing_resource": GenerateMissingResourceFix,
	}
}
