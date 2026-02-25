package analysis

import (
	"context"
	"fmt"
	"regexp"

	"github.com/hrexed/otel-collector-mcp/pkg/types"
)

// AnalyzeInvalidRegex validates regex patterns in filter processor configurations.
func AnalyzeInvalidRegex(_ context.Context, input *AnalysisInput) []types.DiagnosticFinding {
	if input.Config == nil {
		return nil
	}

	var findings []types.DiagnosticFinding

	for procName, procCfg := range input.Config.Processors {
		cfgMap, ok := procCfg.(map[string]interface{})
		if !ok {
			continue
		}

		// Look for filter processor configurations with regex patterns
		findings = append(findings, scanForRegexPatterns(cfgMap, procName, "")...)
	}

	return findings
}

func scanForRegexPatterns(m map[string]interface{}, processorName, path string) []types.DiagnosticFinding {
	var findings []types.DiagnosticFinding

	for key, val := range m {
		fullPath := key
		if path != "" {
			fullPath = path + "." + key
		}

		switch v := val.(type) {
		case string:
			// Check if this looks like a regex field
			if isRegexField(key) {
				if _, err := regexp.Compile(v); err != nil {
					findings = append(findings, types.DiagnosticFinding{
						Severity:   types.SeverityWarning,
						Category:   types.CategoryConfig,
						Summary:    fmt.Sprintf("Invalid regex pattern in processor %q at %q", processorName, fullPath),
						Detail:     fmt.Sprintf("The regex pattern %q is invalid: %v", v, err),
						Suggestion: "Fix the regex pattern syntax",
					})
				}
			}
		case map[string]interface{}:
			findings = append(findings, scanForRegexPatterns(v, processorName, fullPath)...)
		}
	}

	return findings
}

func isRegexField(key string) bool {
	return key == "regexp" || key == "regex" || key == "match_type" || key == "pattern"
}
