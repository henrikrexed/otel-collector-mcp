package collector

import "testing"

func TestParseConfig(t *testing.T) {
	yamlData := `
receivers:
  otlp:
    protocols:
      grpc:
        endpoint: "0.0.0.0:4317"
      http:
        endpoint: "0.0.0.0:4318"
processors:
  batch: {}
  memory_limiter:
    limit_mib: 512
exporters:
  otlp:
    endpoint: "backend:4317"
service:
  pipelines:
    traces:
      receivers: [otlp]
      processors: [memory_limiter, batch]
      exporters: [otlp]
`
	cfg, err := ParseConfig([]byte(yamlData))
	if err != nil {
		t.Fatalf("failed to parse config: %v", err)
	}

	if len(cfg.Receivers) != 1 {
		t.Errorf("expected 1 receiver, got %d", len(cfg.Receivers))
	}
	if len(cfg.Processors) != 2 {
		t.Errorf("expected 2 processors, got %d", len(cfg.Processors))
	}
	if len(cfg.Exporters) != 1 {
		t.Errorf("expected 1 exporter, got %d", len(cfg.Exporters))
	}

	pipeline, ok := cfg.Service.Pipelines["traces"]
	if !ok {
		t.Fatal("expected traces pipeline")
	}
	if len(pipeline.Receivers) != 1 {
		t.Errorf("expected 1 receiver in pipeline, got %d", len(pipeline.Receivers))
	}
	if len(pipeline.Processors) != 2 {
		t.Errorf("expected 2 processors in pipeline, got %d", len(pipeline.Processors))
	}
	if len(pipeline.Exporters) != 1 {
		t.Errorf("expected 1 exporter in pipeline, got %d", len(pipeline.Exporters))
	}
}

func TestParseConfigInvalid(t *testing.T) {
	_, err := ParseConfig([]byte("not: [valid: yaml"))
	if err == nil {
		t.Error("expected error for invalid YAML")
	}
}
