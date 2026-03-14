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

	// v2 metrics
	AnalysisDuration  metric.Float64Histogram // mcp.analysis.duration_seconds
	AnalysisSessions  metric.Int64Counter     // mcp.analysis.sessions_total
	CaptureSignals    metric.Int64Counter     // mcp.capture.signals_total
	DetectionHits     metric.Int64Counter     // mcp.detection.hits_total
	FixesApplied      metric.Int64Counter     // mcp.fixes.applied_total
	RollbacksTotal    metric.Int64Counter     // mcp.rollbacks.total
	HealthChecksTotal metric.Int64Counter     // mcp.health_checks.total
	BackupActive      metric.Int64Gauge       // mcp.backup.active
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

	// v2 metrics
	m.AnalysisDuration, err = meter.Float64Histogram(
		"mcp.analysis.duration_seconds",
		metric.WithDescription("Duration of analysis sessions"),
		metric.WithUnit("s"),
	)
	if err != nil {
		slog.Error("failed to create mcp.analysis.duration_seconds metric", "error", err)
	}

	m.AnalysisSessions, err = meter.Int64Counter(
		"mcp.analysis.sessions_total",
		metric.WithDescription("Total analysis sessions by environment and outcome"),
	)
	if err != nil {
		slog.Error("failed to create mcp.analysis.sessions_total metric", "error", err)
	}

	m.CaptureSignals, err = meter.Int64Counter(
		"mcp.capture.signals_total",
		metric.WithDescription("Total captured signals by type"),
	)
	if err != nil {
		slog.Error("failed to create mcp.capture.signals_total metric", "error", err)
	}

	m.DetectionHits, err = meter.Int64Counter(
		"mcp.detection.hits_total",
		metric.WithDescription("Total detection rule hits by rule and severity"),
	)
	if err != nil {
		slog.Error("failed to create mcp.detection.hits_total metric", "error", err)
	}

	m.FixesApplied, err = meter.Int64Counter(
		"mcp.fixes.applied_total",
		metric.WithDescription("Total fixes applied by type and outcome"),
	)
	if err != nil {
		slog.Error("failed to create mcp.fixes.applied_total metric", "error", err)
	}

	m.RollbacksTotal, err = meter.Int64Counter(
		"mcp.rollbacks.total",
		metric.WithDescription("Total rollbacks by trigger and reason"),
	)
	if err != nil {
		slog.Error("failed to create mcp.rollbacks.total metric", "error", err)
	}

	m.HealthChecksTotal, err = meter.Int64Counter(
		"mcp.health_checks.total",
		metric.WithDescription("Total health checks by result"),
	)
	if err != nil {
		slog.Error("failed to create mcp.health_checks.total metric", "error", err)
	}

	m.BackupActive, err = meter.Int64Gauge(
		"mcp.backup.active",
		metric.WithDescription("Number of active config backups"),
	)
	if err != nil {
		slog.Error("failed to create mcp.backup.active metric", "error", err)
	}

	return m
}

// WithAttributes is a convenience wrapper for metric.WithAttributes.
func WithAttributes(attrs ...attribute.KeyValue) metric.MeasurementOption {
	return metric.WithAttributes(attrs...)
}
