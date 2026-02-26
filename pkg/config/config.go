package config

import (
	"log/slog"
	"os"
	"strconv"

	"github.com/hrexed/otel-collector-mcp/pkg/types"
)

// Config holds server configuration read from environment variables.
type Config struct {
	Port         int
	LogLevel     string
	ClusterName  string
	OTelEnabled  bool
	OTelEndpoint string
}

// NewFromEnv creates a Config by reading environment variables with defaults.
func NewFromEnv() *Config {
	port := 8080
	if v := os.Getenv("PORT"); v != "" {
		if p, err := strconv.Atoi(v); err == nil {
			port = p
		}
	}

	logLevel := "info"
	if v := os.Getenv("LOG_LEVEL"); v != "" {
		logLevel = v
	}

	otelEnabled := false
	if v := os.Getenv("OTEL_ENABLED"); v != "" {
		parsed, err := strconv.ParseBool(v)
		if err != nil {
			slog.Warn("invalid OTEL_ENABLED value, defaulting to false") // #nosec G706
		} else {
			otelEnabled = parsed
		}
	}

	otelEndpoint := os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT")
	if otelEndpoint == "" {
		otelEndpoint = os.Getenv("OTEL_ENDPOINT")
	}

	return &Config{
		Port:         port,
		LogLevel:     logLevel,
		ClusterName:  os.Getenv("CLUSTER_NAME"),
		OTelEnabled:  otelEnabled,
		OTelEndpoint: otelEndpoint,
	}
}

// SetupLogging configures slog with a JSON handler at the configured log level.
func (c *Config) SetupLogging() {
	var level slog.Level
	switch c.LogLevel {
	case "debug":
		level = slog.LevelDebug
	case "warn":
		level = slog.LevelWarn
	case "error":
		level = slog.LevelError
	default:
		level = slog.LevelInfo
	}

	handler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: level})
	slog.SetDefault(slog.New(handler))
}

// SlogLevel returns the configured slog.Level for use with OTel log bridge setup.
func (c *Config) SlogLevel() slog.Level {
	switch c.LogLevel {
	case "debug":
		return slog.LevelDebug
	case "warn":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

// ClusterMetadata returns the ClusterMetadata for use in StandardResponse.
func (c *Config) ClusterMetadata() types.ClusterMetadata {
	ns := os.Getenv("POD_NAMESPACE")
	if ns == "" {
		ns = "default"
	}
	return types.ClusterMetadata{
		Cluster:   c.ClusterName,
		Namespace: ns,
	}
}
