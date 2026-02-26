package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"time"

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
	AttrGenAIToolName       = "gen_ai.tool.name"
	AttrGenAIOperationName  = "gen_ai.operation.name"
	AttrGenAIToolCallArgs   = "gen_ai.tool.call.arguments"
	AttrGenAIToolCallResult = "gen_ai.tool.call.result"
	AttrJSONRPCRequestID    = "jsonrpc.request.id"
	AttrJSONRPCVersion      = "jsonrpc.protocol.version"
	AttrErrorType           = "error.type"

	MCPProtocolVersion = "2025-06-18"
	maxArgBytes        = 1024
	maxResultBytes     = 1024
)

// Server wraps the MCP tool registry and provides HTTP handlers.
type Server struct {
	registry *tools.Registry
	mux      *http.ServeMux
	ready    func() bool
	tracer   trace.Tracer
	metrics  *telemetry.Metrics
	port     int
}

// NewServer creates a new MCP server with the given tool registry, readiness check, and port.
func NewServer(registry *tools.Registry, readyFn func() bool, port int) *Server {
	s := &Server{
		registry: registry,
		mux:      http.NewServeMux(),
		ready:    readyFn,
		tracer:   otel.Tracer(serviceName),
		metrics:  telemetry.NewMetrics(),
		port:     port,
	}
	s.mux.HandleFunc("/mcp", s.handleMCP)
	s.mux.HandleFunc("/healthz", s.handleHealthz)
	s.mux.HandleFunc("/readyz", s.handleReadyz)
	return s
}

const serviceName = "otel-collector-mcp"

// Handler returns the HTTP handler for the server.
func (s *Server) Handler() http.Handler {
	return s.mux
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

// mcpRequest represents a simplified MCP tool call request.
type mcpRequest struct {
	JSONRPC string    `json:"jsonrpc"`
	ID      any       `json:"id"`
	Method  string    `json:"method"`
	Params  mcpParams `json:"params"`
}

type mcpParams struct {
	Name      string                 `json:"name"`
	Arguments map[string]interface{} `json:"arguments"`
	Meta      map[string]interface{} `json:"_meta,omitempty"`
}

// mcpResponse represents a simplified MCP tool call response.
type mcpResponse struct {
	Content []mcpContent `json:"content"`
	IsError bool         `json:"isError,omitempty"`
}

type mcpContent struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

func (s *Server) handleMCP(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		s.handleToolList(w, r)
		return
	}

	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req mcpRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		slog.Error("failed to decode MCP request", "error", err)
		writeJSONError(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if req.Method == "tools/list" {
		s.handleToolList(w, r)
		return
	}

	if req.Method != "tools/call" {
		writeJSONError(w, fmt.Sprintf("unsupported method: %s", req.Method), http.StatusBadRequest)
		return
	}

	tool := s.registry.Get(req.Params.Name)
	if tool == nil {
		slog.Warn("tool not found", "tool", req.Params.Name)
		writeJSONError(w, fmt.Sprintf("tool not found: %s", req.Params.Name), http.StatusNotFound)
		return
	}

	// Context propagation: extract traceparent/tracestate from _meta
	ctx := r.Context()
	if req.Params.Meta != nil {
		carrier := extractMetaCarrier(req.Params.Meta)
		ctx = otel.GetTextMapPropagator().Extract(ctx, propagation.MapCarrier(carrier))
	}

	// Create server span with MCP semantic conventions
	spanName := req.Method + " " + req.Params.Name
	ctx, span := s.tracer.Start(ctx, spanName, trace.WithSpanKind(trace.SpanKindServer))
	defer span.End()

	// Required attributes
	span.SetAttributes(
		attribute.String(AttrMCPMethodName, req.Method),
		attribute.String(AttrGenAIToolName, req.Params.Name),
		attribute.String(AttrGenAIOperationName, "execute_tool"),
		attribute.String(AttrMCPProtocolVersion, MCPProtocolVersion),
	)

	// Recommended attributes
	jsonrpcVersion := req.JSONRPC
	if jsonrpcVersion == "" {
		jsonrpcVersion = "2.0"
	}
	hostname, _ := os.Hostname()
	span.SetAttributes(
		attribute.String(AttrJSONRPCRequestID, fmt.Sprintf("%v", req.ID)),
		attribute.String(AttrJSONRPCVersion, jsonrpcVersion),
		attribute.String("network.transport", "tcp"),
		attribute.String("server.address", hostname),
		attribute.Int("server.port", s.port),
	)

	// Opt-in: sanitized arguments
	span.SetAttributes(attribute.String(AttrGenAIToolCallArgs, sanitizeArgs(req.Params.Arguments)))

	slog.InfoContext(ctx, "tool invoked", "tool", req.Params.Name)

	start := time.Now()
	result, err := tool.Run(ctx, req.Params.Arguments)
	duration := time.Since(start).Seconds()

	// Record duration metric
	toolAttr := attribute.String(AttrGenAIToolName, req.Params.Name)
	if s.metrics.ToolRequestDuration != nil {
		s.metrics.ToolRequestDuration.Record(ctx, duration, telemetry.WithAttributes(toolAttr))
	}

	if err != nil {
		slog.ErrorContext(ctx, "tool execution failed", "tool", req.Params.Name, "error", err)
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		span.SetAttributes(attribute.String(AttrErrorType, "tool_error"))

		if s.metrics.ToolRequestCount != nil {
			s.metrics.ToolRequestCount.Add(ctx, 1, telemetry.WithAttributes(
				toolAttr,
				attribute.String(AttrErrorType, "tool_error"),
			))
		}
		if s.metrics.ErrorsTotal != nil {
			s.metrics.ErrorsTotal.Add(ctx, 1, telemetry.WithAttributes(
				attribute.String(AttrErrorType, "tool_error"),
			))
		}

		resp := mcpResponse{
			Content: []mcpContent{{Type: "text", Text: err.Error()}},
			IsError: true,
		}
		writeJSON(w, resp, http.StatusOK)
		return
	}

	// Success path
	if s.metrics.ToolRequestCount != nil {
		s.metrics.ToolRequestCount.Add(ctx, 1, telemetry.WithAttributes(toolAttr))
	}

	// Record findings as span events and metrics
	s.recordFindings(ctx, span, result, req.Params.Name)

	// Opt-in: truncated result
	resultJSON, err := json.Marshal(result)
	if err != nil {
		slog.ErrorContext(ctx, "failed to marshal tool result", "tool", req.Params.Name, "error", err)
		writeJSONError(w, "internal error", http.StatusInternalServerError)
		return
	}
	span.SetAttributes(attribute.String(AttrGenAIToolCallResult, truncateString(string(resultJSON), maxResultBytes)))

	resp := mcpResponse{
		Content: []mcpContent{{Type: "text", Text: string(resultJSON)}},
	}
	writeJSON(w, resp, http.StatusOK)
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
	// The Data field may be a map with a "collectors" slice
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

func (s *Server) handleToolList(w http.ResponseWriter, _ *http.Request) {
	type toolInfo struct {
		Name        string                 `json:"name"`
		Description string                 `json:"description"`
		InputSchema map[string]interface{} `json:"inputSchema"`
	}

	allTools := s.registry.All()
	toolList := make([]toolInfo, 0, len(allTools))
	for _, t := range allTools {
		toolList = append(toolList, toolInfo{
			Name:        t.Name(),
			Description: t.Description(),
			InputSchema: t.InputSchema(),
		})
	}

	writeJSON(w, map[string]interface{}{"tools": toolList}, http.StatusOK)
}

// ListenAndServe starts the MCP server, blocking until ctx is done.
func (s *Server) ListenAndServe(ctx context.Context, addr string) error {
	srv := &http.Server{
		Addr:              addr,
		Handler:           s.Handler(),
		ReadHeaderTimeout: 10 * time.Second,
	}

	go func() {
		<-ctx.Done()
		slog.Info("shutting down MCP server")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := srv.Shutdown(shutdownCtx); err != nil {
			slog.Error("MCP server shutdown error", "error", err)
		}
	}()

	slog.Info("MCP server starting", "addr", addr)
	return srv.ListenAndServe()
}

func writeJSON(w http.ResponseWriter, v interface{}, status int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		slog.Error("failed to encode JSON response", "error", err)
	}
}

func writeJSONError(w http.ResponseWriter, msg string, status int) {
	resp := mcpResponse{
		Content: []mcpContent{{Type: "text", Text: msg}},
		IsError: true,
	}
	writeJSON(w, resp, status)
}

// extractMetaCarrier converts the _meta map to a string-to-string map for trace propagation.
func extractMetaCarrier(meta map[string]interface{}) map[string]string {
	carrier := make(map[string]string, len(meta))
	for k, v := range meta {
		if s, ok := v.(string); ok {
			carrier[k] = s
		}
	}
	return carrier
}

// sanitizeArgs marshals tool arguments to JSON, truncating to maxArgBytes.
func sanitizeArgs(args map[string]interface{}) string {
	if args == nil {
		return "{}"
	}
	b, err := json.Marshal(args)
	if err != nil {
		return "{}"
	}
	return truncateString(string(b), maxArgBytes)
}

// truncateString truncates s to max bytes, appending "..." if truncated.
func truncateString(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max] + "..."
}
