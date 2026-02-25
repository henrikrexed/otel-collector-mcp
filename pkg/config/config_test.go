package config

import (
	"os"
	"testing"
)

func TestNewFromEnvDefaults(t *testing.T) {
	// Clear env vars for clean test
	os.Unsetenv("PORT")
	os.Unsetenv("LOG_LEVEL")
	os.Unsetenv("CLUSTER_NAME")
	os.Unsetenv("OTEL_ENABLED")
	os.Unsetenv("OTEL_ENDPOINT")

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
	os.Setenv("PORT", "9090")
	os.Setenv("LOG_LEVEL", "debug")
	os.Setenv("CLUSTER_NAME", "prod-us-east")
	os.Setenv("OTEL_ENABLED", "true")
	os.Setenv("OTEL_ENDPOINT", "localhost:4317")
	defer func() {
		os.Unsetenv("PORT")
		os.Unsetenv("LOG_LEVEL")
		os.Unsetenv("CLUSTER_NAME")
		os.Unsetenv("OTEL_ENABLED")
		os.Unsetenv("OTEL_ENDPOINT")
	}()

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
	os.Setenv("CLUSTER_NAME", "test-cluster")
	os.Setenv("POD_NAMESPACE", "monitoring")
	defer func() {
		os.Unsetenv("CLUSTER_NAME")
		os.Unsetenv("POD_NAMESPACE")
	}()

	cfg := NewFromEnv()
	meta := cfg.ClusterMetadata()

	if meta.Cluster != "test-cluster" {
		t.Errorf("expected cluster test-cluster, got %s", meta.Cluster)
	}
	if meta.Namespace != "monitoring" {
		t.Errorf("expected namespace monitoring, got %s", meta.Namespace)
	}
}
