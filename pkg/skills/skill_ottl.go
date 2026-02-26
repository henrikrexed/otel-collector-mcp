package skills

import (
	"context"
	"fmt"
	"strings"

	"github.com/hrexed/otel-collector-mcp/pkg/config"
	"github.com/hrexed/otel-collector-mcp/pkg/types"
)

// OTTLSkill generates OTTL transform processor configurations.
type OTTLSkill struct {
	Cfg *config.Config
}

func (s *OTTLSkill) Definition() SkillDefinition {
	return SkillDefinition{
		Name:        "generate_ottl",
		Description: "Generate OTTL transform processor statements for log parsing, span manipulation, or metric operations",
		Parameters: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"signal_type": map[string]interface{}{
					"type":        "string",
					"enum":        []string{"logs", "traces", "metrics"},
					"description": "Signal type: logs, traces, or metrics",
				},
				"operation": map[string]interface{}{
					"type":        "string",
					"description": "Natural language description of the desired transformation",
				},
			},
			"required": []string{"signal_type", "operation"},
		},
	}
}

func (s *OTTLSkill) Execute(_ context.Context, args map[string]interface{}) (*types.StandardResponse, error) {
	signalType, _ := args["signal_type"].(string)
	operation, _ := args["operation"].(string)

	result := generateOTTL(signalType, operation)

	meta := s.Cfg.ClusterMetadata()
	return types.NewStandardResponse(meta, "generate_ottl", result), nil
}

func generateOTTL(signalType, operation string) *SkillResult {
	var statements []string
	var contextStr string

	lower := strings.ToLower(operation)

	switch signalType {
	case "logs":
		contextStr = "log"
		statements = generateLogOTTL(lower)
	case "traces":
		contextStr = "span"
		statements = generateTraceOTTL(lower)
	case "metrics":
		contextStr = "datapoint"
		statements = generateMetricOTTL(lower)
	}

	configBlock := buildTransformConfig(signalType, contextStr, statements)

	return &SkillResult{
		Skill: "generate_ottl",
		Recommendation: map[string]interface{}{
			"signalType": signalType,
			"operation":  operation,
			"statements": statements,
			"context":    contextStr,
		},
		ConfigSnippet: configBlock,
	}
}

func generateLogOTTL(operation string) []string {
	var statements []string

	switch {
	case strings.Contains(operation, "parse json") || strings.Contains(operation, "json parse"):
		statements = append(statements, `merge_maps(cache, ParseJSON(body), "insert")`)
		statements = append(statements, `set(attributes["parsed"], cache)`)
	case strings.Contains(operation, "severity") || strings.Contains(operation, "log level"):
		statements = append(statements, `set(severity_text, attributes["level"]) where attributes["level"] != nil`)
	case strings.Contains(operation, "extract") || strings.Contains(operation, "regex"):
		statements = append(statements, `# Replace <pattern> with your regex:`)
		statements = append(statements, `merge_maps(attributes, ExtractPatterns(body, "(?P<field>pattern)"), "insert")`)
	default:
		statements = append(statements, fmt.Sprintf(`# Custom log transform for: %s`, operation))
		statements = append(statements, `set(attributes["custom_field"], "value") where body != nil`)
	}

	return statements
}

func generateTraceOTTL(operation string) []string {
	var statements []string

	switch {
	case strings.Contains(operation, "add attribute") || strings.Contains(operation, "set attribute"):
		statements = append(statements, `set(attributes["custom.attribute"], "value")`)
	case strings.Contains(operation, "delete") || strings.Contains(operation, "remove"):
		statements = append(statements, `delete_key(attributes, "attribute.to.remove")`)
	case strings.Contains(operation, "rename"):
		statements = append(statements, `set(attributes["new.name"], attributes["old.name"]) where attributes["old.name"] != nil`)
		statements = append(statements, `delete_key(attributes, "old.name")`)
	default:
		statements = append(statements, fmt.Sprintf(`# Custom span transform for: %s`, operation))
		statements = append(statements, `set(attributes["custom"], "value")`)
	}

	return statements
}

func generateMetricOTTL(operation string) []string {
	var statements []string

	switch {
	case strings.Contains(operation, "rename") || strings.Contains(operation, "label"):
		statements = append(statements, `set(attributes["new_label"], attributes["old_label"]) where attributes["old_label"] != nil`)
		statements = append(statements, `delete_key(attributes, "old_label")`)
	case strings.Contains(operation, "drop") || strings.Contains(operation, "delete"):
		statements = append(statements, `delete_key(attributes, "high_cardinality_label")`)
	case strings.Contains(operation, "aggregate") || strings.Contains(operation, "sum"):
		statements = append(statements, `# Use metricstransform processor for aggregation instead of OTTL`)
	default:
		statements = append(statements, fmt.Sprintf(`# Custom metric transform for: %s`, operation))
		statements = append(statements, `set(attributes["custom"], "value")`)
	}

	return statements
}

func buildTransformConfig(signalType, context string, statements []string) string {
	var b strings.Builder
	b.WriteString("processors:\n")
	fmt.Fprintf(&b, "  transform/%s_transform:\n", signalType)
	fmt.Fprintf(&b, "    %s_statements:\n", context)
	b.WriteString("      - context: " + context + "\n")
	b.WriteString("        statements:\n")
	for _, stmt := range statements {
		fmt.Fprintf(&b, "          - '%s'\n", stmt)
	}

	fmt.Fprintf(&b, "\nservice:\n  pipelines:\n    %s:\n      processors: [transform/%s_transform]\n", signalType, signalType)

	return b.String()
}
