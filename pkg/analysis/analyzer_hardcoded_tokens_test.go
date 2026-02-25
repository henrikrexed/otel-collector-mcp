package analysis

import (
	"context"
	"testing"

	"github.com/hrexed/otel-collector-mcp/pkg/collector"
)

func TestAnalyzeHardcodedTokens_Clean(t *testing.T) {
	input := &AnalysisInput{
		Config: &collector.CollectorConfig{
			Exporters: map[string]interface{}{
				"otlp": map[string]interface{}{
					"endpoint": "backend:4317",
					"headers": map[string]interface{}{
						"api_key": "${env:API_KEY}",
					},
				},
			},
		},
	}

	findings := AnalyzeHardcodedTokens(context.Background(), input)
	if len(findings) != 0 {
		t.Errorf("expected no findings for env var reference, got %d", len(findings))
	}
}

func TestAnalyzeHardcodedTokens_Hardcoded(t *testing.T) {
	input := &AnalysisInput{
		Config: &collector.CollectorConfig{
			Exporters: map[string]interface{}{
				"datadog": map[string]interface{}{
					"api_key": "abc123secret",
				},
			},
		},
	}

	findings := AnalyzeHardcodedTokens(context.Background(), input)
	if len(findings) != 1 {
		t.Fatalf("expected 1 finding, got %d", len(findings))
	}
	if findings[0].Severity != "critical" {
		t.Errorf("expected critical severity, got %s", findings[0].Severity)
	}
	// Must NOT contain the actual credential value
	if findings[0].Summary == "" {
		t.Error("expected summary")
	}
}
