package analysis

import (
	"context"
	"fmt"
	"strings"

	"github.com/hrexed/otel-collector-mcp/pkg/types"
)

// tokenPatterns are field names that may contain sensitive credentials.
var tokenPatterns = []string{
	"api_key", "apikey", "api_token", "token", "secret",
	"password", "auth", "bearer", "authorization",
	"access_key", "secret_key", "api-key", "api-token",
}

// AnalyzeHardcodedTokens scans exporter configs for hardcoded credentials.
func AnalyzeHardcodedTokens(_ context.Context, input *AnalysisInput) []types.DiagnosticFinding {
	if input.Config == nil {
		return nil
	}

	var findings []types.DiagnosticFinding
	for exporterName, exporterCfg := range input.Config.Exporters {
		cfgMap, ok := exporterCfg.(map[string]interface{})
		if !ok {
			continue
		}
		findings = append(findings, scanMapForTokens(cfgMap, exporterName, "")...)
	}
	return findings
}

func scanMapForTokens(m map[string]interface{}, exporterName, path string) []types.DiagnosticFinding {
	var findings []types.DiagnosticFinding

	for key, val := range m {
		fullPath := key
		if path != "" {
			fullPath = path + "." + key
		}

		switch v := val.(type) {
		case string:
			if isTokenField(key) && isHardcoded(v) {
				findings = append(findings, types.DiagnosticFinding{
					Severity:   types.SeverityCritical,
					Category:   types.CategorySecurity,
					Summary:    fmt.Sprintf("Hardcoded credential detected in exporter %q at %q", exporterName, fullPath),
					Detail:     "Hardcoded credentials in collector configuration are a security risk. They can be exposed in version control, logs, and ConfigMaps.",
					Suggestion: "Use environment variable references or Kubernetes secrets instead",
					Remediation: fmt.Sprintf(`# Replace the hardcoded value with an environment variable reference:
exporters:
  %s:
    %s: ${env:YOUR_SECRET_ENV_VAR}

# Or mount a Kubernetes secret as an environment variable in the collector pod`, exporterName, fullPath),
				})
			}
		case map[string]interface{}:
			findings = append(findings, scanMapForTokens(v, exporterName, fullPath)...)
		}
	}
	return findings
}

func isTokenField(key string) bool {
	lower := strings.ToLower(key)
	for _, pattern := range tokenPatterns {
		if strings.Contains(lower, pattern) {
			return true
		}
	}
	return false
}

func isHardcoded(value string) bool {
	// Not hardcoded if it uses env var substitution
	if strings.HasPrefix(value, "${") && strings.Contains(value, "}") {
		return false
	}
	// Not hardcoded if empty
	if value == "" {
		return false
	}
	return true
}
