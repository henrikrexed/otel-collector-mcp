package analysis

import (
	"context"
	"testing"

	"github.com/hrexed/otel-collector-mcp/pkg/collector"
)

func TestAnalyzeMissingBatch_WithBatch(t *testing.T) {
	input := &AnalysisInput{
		Config: &collector.CollectorConfig{
			Service: collector.ServiceConfig{
				Pipelines: map[string]collector.PipelineConfig{
					"traces": {
						Receivers:  []string{"otlp"},
						Processors: []string{"batch"},
						Exporters:  []string{"otlp"},
					},
				},
			},
		},
	}

	findings := AnalyzeMissingBatch(context.Background(), input)
	if len(findings) != 0 {
		t.Errorf("expected no findings when batch is present, got %d", len(findings))
	}
}

func TestAnalyzeMissingBatch_WithoutBatch(t *testing.T) {
	input := &AnalysisInput{
		Config: &collector.CollectorConfig{
			Service: collector.ServiceConfig{
				Pipelines: map[string]collector.PipelineConfig{
					"traces": {
						Receivers:  []string{"otlp"},
						Processors: []string{"memory_limiter"},
						Exporters:  []string{"otlp"},
					},
				},
			},
		},
	}

	findings := AnalyzeMissingBatch(context.Background(), input)
	if len(findings) != 1 {
		t.Fatalf("expected 1 finding, got %d", len(findings))
	}
	if findings[0].Remediation == "" {
		t.Error("expected remediation to be present")
	}
}

func TestAnalyzeMissingBatch_MultiplePipelines(t *testing.T) {
	input := &AnalysisInput{
		Config: &collector.CollectorConfig{
			Service: collector.ServiceConfig{
				Pipelines: map[string]collector.PipelineConfig{
					"traces": {
						Receivers:  []string{"otlp"},
						Processors: []string{"batch"},
						Exporters:  []string{"otlp"},
					},
					"metrics": {
						Receivers:  []string{"otlp"},
						Processors: []string{},
						Exporters:  []string{"otlp"},
					},
				},
			},
		},
	}

	findings := AnalyzeMissingBatch(context.Background(), input)
	if len(findings) != 1 {
		t.Errorf("expected 1 finding for metrics pipeline, got %d", len(findings))
	}
}

func TestAnalyzeMissingBatch_NilConfig(t *testing.T) {
	findings := AnalyzeMissingBatch(context.Background(), &AnalysisInput{})
	if len(findings) != 0 {
		t.Errorf("expected no findings for nil config, got %d", len(findings))
	}
}
