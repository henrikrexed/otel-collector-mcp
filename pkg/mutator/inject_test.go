package mutator

import (
	"strings"
	"testing"
)

const testConfig = `
receivers:
  otlp:
    protocols:
      grpc:
      http:
exporters:
  otlp:
    endpoint: "otel-backend:4317"
service:
  pipelines:
    traces:
      receivers: [otlp]
      exporters: [otlp]
    metrics:
      receivers: [otlp]
      exporters: [otlp]
`

func TestInjectDebugExporter_AllPipelines(t *testing.T) {
	result, injected, err := InjectDebugExporter(testConfig, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(injected) != 2 {
		t.Errorf("expected 2 injected pipelines, got %d: %v", len(injected), injected)
	}
	if !strings.Contains(result, "debug") {
		t.Error("expected debug exporter in result")
	}
}

func TestInjectDebugExporter_SpecificPipeline(t *testing.T) {
	_, injected, err := InjectDebugExporter(testConfig, []string{"traces"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(injected) != 1 || injected[0] != "traces" {
		t.Errorf("expected [traces], got %v", injected)
	}
}

func TestInjectDebugExporter_Idempotent(t *testing.T) {
	result1, _, err := InjectDebugExporter(testConfig, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	_, injected2, err := InjectDebugExporter(result1, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(injected2) != 0 {
		t.Errorf("expected idempotent (0 injected), got %d", len(injected2))
	}
}

func TestRemoveDebugExporter(t *testing.T) {
	injected, _, err := InjectDebugExporter(testConfig, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	result, removed, err := RemoveDebugExporter(injected)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(removed) != 2 {
		t.Errorf("expected 2 removed pipelines, got %d", len(removed))
	}
	if strings.Contains(result, "debug") {
		t.Error("expected debug exporter to be removed")
	}
}
