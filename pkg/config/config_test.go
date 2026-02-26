package config

import (
	"testing"
)

func TestNewFromEnvDefaults(t *testing.T) {
	// Clear env vars for clean test
	t.Setenv("PORT", "")
	t.Setenv("LOG_LEVEL", "")
	t.Setenv("CLUSTER_NAME", "")
	t.Setenv("OTEL_ENABLED", "")
	t.Setenv("OTEL_ENDPOINT", "")
	t.Setenv("OTEL_EXPORTER_OTLP_ENDPOINT", "")

	cfg := NewFromEnv()

	if cfg.Port != 8080 {
		t.Errorf("expected default port 8080, got %d", cfg.Port)
	}
	if cfg.LogLevel != "info" {
		t.Errorf("expected default log level info, got %s", cfg.LogLevel)
	}
	if cfg.ClusterName != "" {
		t.Errorf("expected empty cluster name, got %s", cfg.ClusterName)
	}
	if cfg.OTelEnabled {
		t.Errorf("expected OTel disabled by default")
	}
}

func TestNewFromEnvOverrides(t *testing.T) {
	t.Setenv("PORT", "9090")
	t.Setenv("LOG_LEVEL", "debug")
	t.Setenv("CLUSTER_NAME", "prod-us-east")
	t.Setenv("OTEL_ENABLED", "true")
	t.Setenv("OTEL_ENDPOINT", "localhost:4317")

	cfg := NewFromEnv()

	if cfg.Port != 9090 {
		t.Errorf("expected port 9090, got %d", cfg.Port)
	}
	if cfg.LogLevel != "debug" {
		t.Errorf("expected log level debug, got %s", cfg.LogLevel)
	}
	if cfg.ClusterName != "prod-us-east" {
		t.Errorf("expected cluster name prod-us-east, got %s", cfg.ClusterName)
	}
	if !cfg.OTelEnabled {
		t.Errorf("expected OTel enabled")
	}
	if cfg.OTelEndpoint != "localhost:4317" {
		t.Errorf("expected OTEL_ENDPOINT localhost:4317, got %s", cfg.OTelEndpoint)
	}
}

func TestClusterMetadata(t *testing.T) {
	t.Setenv("CLUSTER_NAME", "test-cluster")
	t.Setenv("POD_NAMESPACE", "monitoring")

	cfg := NewFromEnv()
	meta := cfg.ClusterMetadata()

	if meta.Cluster != "test-cluster" {
		t.Errorf("expected cluster test-cluster, got %s", meta.Cluster)
	}
	if meta.Namespace != "monitoring" {
		t.Errorf("expected namespace monitoring, got %s", meta.Namespace)
	}
}
