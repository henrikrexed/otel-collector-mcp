package telemetry

import (
	"log/slog"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

// Metrics holds all OTel metric instruments for the MCP server.
type Metrics struct {
	// GenAI semconv metrics
	ToolRequestDuration metric.Float64Histogram // gen_ai.server.request.duration
	ToolRequestCount    metric.Int64Counter     // gen_ai.server.request.count

	// Custom domain metrics
	FindingsTotal        metric.Int64Counter // mcp.findings.total
	CollectorsDiscovered metric.Int64Gauge   // mcp.collectors.discovered
	ErrorsTotal          metric.Int64Counter // mcp.errors.total
}

// NewMetrics creates all metric instruments from the global MeterProvider.
func NewMetrics() *Metrics {
	meter := otel.Meter(serviceName)
	m := &Metrics{}

	var err error

	m.ToolRequestDuration, err = meter.Float64Histogram(
		"gen_ai.server.request.duration",
		metric.WithDescription("Duration of MCP tool execution in seconds"),
		metric.WithUnit("s"),
	)
	if err != nil {
		slog.Error("failed to create gen_ai.server.request.duration metric", "error", err)
	}

	m.ToolRequestCount, err = meter.Int64Counter(
		"gen_ai.server.request.count",
		metric.WithDescription("Number of MCP tool requests"),
	)
	if err != nil {
		slog.Error("failed to create gen_ai.server.request.count metric", "error", err)
	}

	m.FindingsTotal, err = meter.Int64Counter(
		"mcp.findings.total",
		metric.WithDescription("Total diagnostic findings by severity and analyzer"),
	)
	if err != nil {
		slog.Error("failed to create mcp.findings.total metric", "error", err)
	}

	m.CollectorsDiscovered, err = meter.Int64Gauge(
		"mcp.collectors.discovered",
		metric.WithDescription("Number of OTel Collector instances discovered"),
	)
	if err != nil {
		slog.Error("failed to create mcp.collectors.discovered metric", "error", err)
	}

	m.ErrorsTotal, err = meter.Int64Counter(
		"mcp.errors.total",
		metric.WithDescription("Total MCP tool execution errors"),
	)
	if err != nil {
		slog.Error("failed to create mcp.errors.total metric", "error", err)
	}

	return m
}

// WithAttributes is a convenience wrapper for metric.WithAttributes.
func WithAttributes(attrs ...attribute.KeyValue) metric.MeasurementOption {
	return metric.WithAttributes(attrs...)
}
