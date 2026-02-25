package telemetry

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"time"

	"go.opentelemetry.io/contrib/bridges/otelslog"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploggrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/propagation"
	sdklog "go.opentelemetry.io/otel/sdk/log"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
	"go.opentelemetry.io/otel/trace/noop"

	"go.opentelemetry.io/otel/log/global"
)

const serviceName = "otel-collector-mcp"

// InitTelemetry initializes all 3 OTel signal providers (traces, metrics, logs).
// When enabled is false, a noop TracerProvider is set and no other providers are configured.
// Returns a shutdown function that cleanly shuts down all providers.
func InitTelemetry(ctx context.Context, enabled bool, endpoint string) (func(), error) {
	if !enabled {
		otel.SetTracerProvider(noop.NewTracerProvider())
		return func() {}, nil
	}

	res, err := resource.New(ctx,
		resource.WithAttributes(
			semconv.ServiceNameKey.String(serviceName),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create resource: %w", err)
	}

	// TracerProvider
	traceExporter, err := otlptracegrpc.New(ctx,
		otlptracegrpc.WithEndpoint(endpoint),
		otlptracegrpc.WithInsecure(),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create trace exporter: %w", err)
	}

	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(traceExporter),
		sdktrace.WithResource(res),
	)
	otel.SetTracerProvider(tp)

	// MeterProvider
	metricExporter, err := otlpmetricgrpc.New(ctx,
		otlpmetricgrpc.WithEndpoint(endpoint),
		otlpmetricgrpc.WithInsecure(),
	)
	if err != nil {
		_ = traceExporter.Shutdown(ctx)
		return nil, fmt.Errorf("failed to create metric exporter: %w", err)
	}

	mp := sdkmetric.NewMeterProvider(
		sdkmetric.WithReader(sdkmetric.NewPeriodicReader(metricExporter, sdkmetric.WithInterval(10*time.Second))),
		sdkmetric.WithResource(res),
	)
	otel.SetMeterProvider(mp)

	// LoggerProvider
	logExporter, err := otlploggrpc.New(ctx,
		otlploggrpc.WithEndpoint(endpoint),
		otlploggrpc.WithInsecure(),
	)
	if err != nil {
		_ = traceExporter.Shutdown(ctx)
		_ = mp.Shutdown(ctx)
		return nil, fmt.Errorf("failed to create log exporter: %w", err)
	}

	lp := sdklog.NewLoggerProvider(
		sdklog.WithProcessor(sdklog.NewBatchProcessor(logExporter)),
		sdklog.WithResource(res),
	)
	global.SetLoggerProvider(lp)

	// W3C TraceContext propagator
	otel.SetTextMapPropagator(propagation.TraceContext{})

	slog.Info("OpenTelemetry initialized (traces, metrics, logs)", "endpoint", endpoint)

	shutdown := func() {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := tp.Shutdown(shutdownCtx); err != nil {
			slog.Error("failed to shutdown tracer provider", "error", err)
		}
		if err := mp.Shutdown(shutdownCtx); err != nil {
			slog.Error("failed to shutdown meter provider", "error", err)
		}
		if err := lp.Shutdown(shutdownCtx); err != nil {
			slog.Error("failed to shutdown logger provider", "error", err)
		}
	}

	return shutdown, nil
}

// SetupOTelLogging reconfigures slog with a tee handler that writes to both
// stdout (JSON) and OTel log bridge (for OTLP export with trace correlation).
func SetupOTelLogging(level slog.Level) {
	stdoutHandler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: level})
	otelHandler := otelslog.NewHandler(serviceName)

	slog.SetDefault(slog.New(&teeHandler{
		stdout: stdoutHandler,
		otel:   otelHandler,
	}))
	slog.Info("slog reconfigured with OTel log bridge")
}

// teeHandler fans out log records to both stdout and OTel handlers.
type teeHandler struct {
	stdout slog.Handler
	otel   slog.Handler
}

func (h *teeHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return h.stdout.Enabled(ctx, level) || h.otel.Enabled(ctx, level)
}

func (h *teeHandler) Handle(ctx context.Context, r slog.Record) error {
	if h.stdout.Enabled(ctx, r.Level) {
		_ = h.stdout.Handle(ctx, r.Clone())
	}
	if h.otel.Enabled(ctx, r.Level) {
		_ = h.otel.Handle(ctx, r.Clone())
	}
	return nil
}

func (h *teeHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &teeHandler{
		stdout: h.stdout.WithAttrs(attrs),
		otel:   h.otel.WithAttrs(attrs),
	}
}

func (h *teeHandler) WithGroup(name string) slog.Handler {
	return &teeHandler{
		stdout: h.stdout.WithGroup(name),
		otel:   h.otel.WithGroup(name),
	}
}
