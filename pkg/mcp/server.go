package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"

	"github.com/hrexed/otel-collector-mcp/pkg/telemetry"
	"github.com/hrexed/otel-collector-mcp/pkg/tools"
	"github.com/hrexed/otel-collector-mcp/pkg/types"
)

// MCP semantic convention attribute keys.
const (
	AttrMCPMethodName       = "mcp.method.name"
	AttrMCPProtocolVersion  = "mcp.protocol.version"
	AttrMCPSessionID        = "mcp.session.id"
	AttrGenAIToolName       = "gen_ai.tool.name"
	AttrGenAIOperationName  = "gen_ai.operation.name"
	AttrGenAIToolCallArgs   = "gen_ai.tool.call.arguments"
	AttrGenAIToolCallResult = "gen_ai.tool.call.result"
	AttrErrorType           = "error.type"

	MCPProtocolVersion = "2025-06-18"
	maxArgBytes        = 1024
	maxResultBytes     = 1024
)

// sensitiveKeys are argument key substrings that should be redacted from span attributes.
var sensitiveKeys = []string{"secret", "token", "key", "password", "credential"}

const serviceName = "otel-collector-mcp"

// Server wraps the MCP SDK server and provides OTel-instrumented tool execution.
type Server struct {
	mcpServer  *mcpsdk.Server
	httpServer *http.Server
	registry   *tools.Registry
	metrics    *telemetry.Metrics
	ready      func() bool
	port       int

	mu              sync.Mutex
	registeredTools map[string]struct{}
}

// NewServer creates a new MCP server with the given tool registry, readiness check, and port.
func NewServer(registry *tools.Registry, readyFn func() bool, port int) *Server {
	mcpServer := mcpsdk.NewServer(&mcpsdk.Implementation{
		Name:    serviceName,
		Version: "1.0.0",
	}, nil)

	return &Server{
		mcpServer:       mcpServer,
		registry:        registry,
		metrics:         telemetry.NewMetrics(),
		ready:           readyFn,
		port:            port,
		registeredTools: make(map[string]struct{}),
	}
}

// SyncTools diffs the registry against what is currently registered in the MCP server,
// adding new tools and removing stale ones.
func (s *Server) SyncTools() {
	s.mu.Lock()
	defer s.mu.Unlock()

	allTools := s.registry.All()

	// Build a set of tool names currently in the registry
	wanted := make(map[string]struct{}, len(allTools))
	for _, t := range allTools {
		wanted[t.Name()] = struct{}{}
	}

	// Remove tools that are registered but no longer in the registry
	var toRemove []string
	for name := range s.registeredTools {
		if _, ok := wanted[name]; !ok {
			toRemove = append(toRemove, name)
		}
	}
	if len(toRemove) > 0 {
		s.mcpServer.RemoveTools(toRemove...)
		for _, name := range toRemove {
			delete(s.registeredTools, name)
		}
		slog.Info("mcp: removed tools", "tools", toRemove)
	}

	// Add tools that are in the registry but not yet registered
	added := 0
	for _, t := range allTools {
		if _, ok := s.registeredTools[t.Name()]; ok {
			continue
		}
		mcpTool := buildMCPTool(t)
		handler := s.buildInstrumentedHandler(t)
		s.mcpServer.AddTool(mcpTool, handler)
		s.registeredTools[t.Name()] = struct{}{}
		added++
	}

	slog.Info("mcp: synced tools", "total", len(s.registeredTools), "added", added, "removed", len(toRemove))
}

// Start begins serving the MCP Streamable HTTP protocol.
func (s *Server) Start(addr string) error {
	s.SyncTools()

	handler := mcpsdk.NewStreamableHTTPHandler(func(r *http.Request) *mcpsdk.Server {
		return s.mcpServer
	}, nil)

	mux := http.NewServeMux()
	mux.Handle("/mcp", otelhttp.NewHandler(handler, "MCP",
		otelhttp.WithTracerProvider(otel.GetTracerProvider()),
		otelhttp.WithPropagators(otel.GetTextMapPropagator()),
	))
	mux.HandleFunc("/healthz", s.handleHealthz)
	mux.HandleFunc("/readyz", s.handleReadyz)

	s.httpServer = &http.Server{
		Addr:              addr,
		Handler:           mux,
		ReadHeaderTimeout: 10 * time.Second,
	}

	slog.Info("mcp: starting Streamable HTTP server", "addr", addr)
	return s.httpServer.ListenAndServe()
}

// Shutdown gracefully shuts down the HTTP server.
func (s *Server) Shutdown(ctx context.Context) error {
	if s.httpServer != nil {
		return s.httpServer.Shutdown(ctx)
	}
	return nil
}

func (s *Server) handleHealthz(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusOK)
	_, _ = fmt.Fprint(w, "ok")
}

func (s *Server) handleReadyz(w http.ResponseWriter, _ *http.Request) {
	if s.ready != nil && !s.ready() {
		w.WriteHeader(http.StatusServiceUnavailable)
		_, _ = fmt.Fprint(w, "not ready")
		return
	}
	w.WriteHeader(http.StatusOK)
	_, _ = fmt.Fprint(w, "ready")
}

func buildMCPTool(t tools.Tool) *mcpsdk.Tool {
	schema := t.InputSchema()
	schemaJSON, _ := json.Marshal(schema)

	tool := &mcpsdk.Tool{
		Name:        t.Name(),
		Description: t.Description(),
	}

	if err := json.Unmarshal(schemaJSON, &tool.InputSchema); err != nil {
		slog.Warn("mcp: failed to parse input schema", "tool", t.Name(), "error", err)
	}

	return tool
}

// buildInstrumentedHandler creates a ToolHandler that wraps tool execution
// with OTel spans, metrics, and context propagation per GenAI + MCP semantic conventions.
func (s *Server) buildInstrumentedHandler(t tools.Tool) mcpsdk.ToolHandler {
	tracer := otel.Tracer(serviceName)

	return func(ctx context.Context, request *mcpsdk.CallToolRequest) (*mcpsdk.CallToolResult, error) {
		// Context propagation: extract traceparent/tracestate from _meta
		meta := request.Params.GetMeta()
		if meta != nil {
			carrier := propagation.MapCarrier{}
			for k, v := range meta {
				if str, ok := v.(string); ok {
					carrier.Set(k, str)
				}
			}
			ctx = otel.GetTextMapPropagator().Extract(ctx, carrier)
		}

		// Extract session ID
		sessionID := ""
		if request.Session != nil {
			sessionID = request.Session.ID()
		}

		// Create server span with MCP semantic conventions
		spanName := "execute_tool " + t.Name()
		ctx, span := tracer.Start(ctx, spanName, trace.WithSpanKind(trace.SpanKindServer))
		defer span.End()

		// Set GenAI + MCP span attributes
		hostname, _ := os.Hostname()
		span.SetAttributes(
			attribute.String(AttrMCPMethodName, "tools/call"),
			attribute.String(AttrGenAIToolName, t.Name()),
			attribute.String(AttrGenAIOperationName, "execute_tool"),
			attribute.String(AttrMCPProtocolVersion, MCPProtocolVersion),
			attribute.String(AttrMCPSessionID, sessionID),
			attribute.String("network.transport", "tcp"),
			attribute.String("server.address", hostname),
			attribute.Int("server.port", s.port),
		)

		// Unmarshal arguments
		var args map[string]interface{}
		if request.Params.Arguments != nil {
			if err := json.Unmarshal(request.Params.Arguments, &args); err != nil {
				s.recordError(ctx, span, t.Name(), "INVALID_INPUT", err)
				return &mcpsdk.CallToolResult{
					Content: []mcpsdk.Content{&mcpsdk.TextContent{Text: fmt.Sprintf("failed to parse arguments: %v", err)}},
					IsError: true,
				}, nil
			}
		}
		if args == nil {
			args = make(map[string]interface{})
		}

		// Sanitized arguments as span attribute
		span.SetAttributes(attribute.String(AttrGenAIToolCallArgs, sanitizeArgs(args)))

		slog.InfoContext(ctx, "tool invoked", "tool", t.Name())

		// Execute tool with timing
		start := time.Now()
		result, err := t.Run(ctx, args)
		duration := time.Since(start).Seconds()

		if err != nil {
			errType := "tool_error"
			if mcpErr, ok := err.(*types.MCPError); ok {
				errType = mcpErr.Code
			}
			s.recordMetrics(ctx, t.Name(), errType, duration)
			s.recordError(ctx, span, t.Name(), errType, err)

			if mcpErr, ok := err.(*types.MCPError); ok {
				errJSON, _ := json.MarshalIndent(mcpErr, "", "  ")
				return &mcpsdk.CallToolResult{
					Content: []mcpsdk.Content{&mcpsdk.TextContent{Text: string(errJSON)}},
					IsError: true,
				}, nil
			}
			return &mcpsdk.CallToolResult{
				Content: []mcpsdk.Content{&mcpsdk.TextContent{Text: err.Error()}},
				IsError: true,
			}, nil
		}

		// Success metrics
		s.recordMetrics(ctx, t.Name(), "", duration)
		span.SetStatus(codes.Ok, "")

		// Record findings as span events and metrics
		s.recordFindings(ctx, span, result, t.Name())

		// Marshal result
		resultJSON, err := json.MarshalIndent(result, "", "  ")
		if err != nil {
			s.recordError(ctx, span, t.Name(), "INTERNAL_ERROR", err)
			return &mcpsdk.CallToolResult{
				Content: []mcpsdk.Content{&mcpsdk.TextContent{Text: fmt.Sprintf("failed to marshal result: %v", err)}},
				IsError: true,
			}, nil
		}

		// Truncated result as span attribute
		span.SetAttributes(attribute.String(AttrGenAIToolCallResult, truncateString(string(resultJSON), maxResultBytes)))

		return &mcpsdk.CallToolResult{
			Content: []mcpsdk.Content{&mcpsdk.TextContent{Text: string(resultJSON)}},
		}, nil
	}
}

// recordMetrics records GenAI request duration and count metrics.
func (s *Server) recordMetrics(ctx context.Context, toolName, errType string, duration float64) {
	toolAttr := attribute.String(AttrGenAIToolName, toolName)

	if s.metrics.ToolRequestDuration != nil {
		attrs := []attribute.KeyValue{toolAttr}
		if errType != "" {
			attrs = append(attrs, attribute.String(AttrErrorType, errType))
		}
		s.metrics.ToolRequestDuration.Record(ctx, duration, telemetry.WithAttributes(attrs...))
	}

	if s.metrics.ToolRequestCount != nil {
		attrs := []attribute.KeyValue{toolAttr}
		if errType != "" {
			attrs = append(attrs, attribute.String(AttrErrorType, errType))
		}
		s.metrics.ToolRequestCount.Add(ctx, 1, telemetry.WithAttributes(attrs...))
	}
}

// recordError records error metrics and sets span error status.
func (s *Server) recordError(ctx context.Context, span trace.Span, toolName, errType string, err error) {
	slog.ErrorContext(ctx, "tool execution failed", "tool", toolName, "error", err)
	span.RecordError(err)
	span.SetStatus(codes.Error, err.Error())
	span.SetAttributes(attribute.String(AttrErrorType, errType))

	if s.metrics.ErrorsTotal != nil {
		s.metrics.ErrorsTotal.Add(ctx, 1, telemetry.WithAttributes(
			attribute.String(AttrErrorType, errType),
			attribute.String(AttrGenAIToolName, toolName),
		))
	}
}

// recordFindings inspects the tool result for diagnostic findings and records them
// as span events and metrics.
func (s *Server) recordFindings(ctx context.Context, span trace.Span, result *types.StandardResponse, toolName string) {
	if result == nil || result.Data == nil {
		return
	}

	toolResult, ok := result.Data.(*types.ToolResult)
	if !ok {
		// Check if it's a list_collectors result with a collector count
		if toolName == "list_collectors" {
			s.recordCollectorCount(ctx, result)
		}
		return
	}

	for _, f := range toolResult.Findings {
		span.AddEvent("diagnostic_finding",
			trace.WithAttributes(
				attribute.String("severity", f.Severity),
				attribute.String("category", f.Category),
				attribute.String("summary", f.Summary),
			),
		)
		if s.metrics.FindingsTotal != nil {
			s.metrics.FindingsTotal.Add(ctx, 1, telemetry.WithAttributes(
				attribute.String("severity", f.Severity),
				attribute.String("analyzer", f.Category),
			))
		}
	}
}

// recordCollectorCount attempts to record the number of discovered collectors
// from a list_collectors response.
func (s *Server) recordCollectorCount(ctx context.Context, result *types.StandardResponse) {
	if s.metrics.CollectorsDiscovered == nil {
		return
	}
	dataMap, ok := result.Data.(map[string]interface{})
	if !ok {
		return
	}
	collectors, ok := dataMap["collectors"]
	if !ok {
		return
	}
	if arr, ok := collectors.([]interface{}); ok {
		s.metrics.CollectorsDiscovered.Record(ctx, int64(len(arr)))
	}
}

// sanitizeArgs marshals tool arguments to JSON with sensitive values redacted, truncating to maxArgBytes.
func sanitizeArgs(args map[string]interface{}) string {
	if args == nil {
		return "{}"
	}
	sanitized := make(map[string]interface{}, len(args))
	for k, v := range args {
		if isSensitiveKey(k) {
			sanitized[k] = "[REDACTED]"
		} else {
			sanitized[k] = v
		}
	}
	b, err := json.Marshal(sanitized)
	if err != nil {
		return "{}"
	}
	return truncateString(string(b), maxArgBytes)
}

// isSensitiveKey checks if a key name suggests it contains sensitive data.
func isSensitiveKey(key string) bool {
	lower := strings.ToLower(key)
	for _, s := range sensitiveKeys {
		if strings.Contains(lower, s) {
			return true
		}
	}
	return false
}

// truncateString truncates s to max bytes, appending "..." if truncated.
func truncateString(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max] + "..."
}
