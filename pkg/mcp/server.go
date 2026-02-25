package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/hrexed/otel-collector-mcp/pkg/tools"
)

// Server wraps the MCP tool registry and provides HTTP handlers.
type Server struct {
	registry *tools.Registry
	mux      *http.ServeMux
	ready    func() bool
}

// NewServer creates a new MCP server with the given tool registry and readiness check.
func NewServer(registry *tools.Registry, readyFn func() bool) *Server {
	s := &Server{
		registry: registry,
		mux:      http.NewServeMux(),
		ready:    readyFn,
	}
	s.mux.HandleFunc("/mcp", s.handleMCP)
	s.mux.HandleFunc("/healthz", s.handleHealthz)
	s.mux.HandleFunc("/readyz", s.handleReadyz)
	return s
}

// Handler returns the HTTP handler for the server.
func (s *Server) Handler() http.Handler {
	return s.mux
}

func (s *Server) handleHealthz(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	fmt.Fprint(w, "ok")
}

func (s *Server) handleReadyz(w http.ResponseWriter, r *http.Request) {
	if s.ready != nil && !s.ready() {
		w.WriteHeader(http.StatusServiceUnavailable)
		fmt.Fprint(w, "not ready")
		return
	}
	w.WriteHeader(http.StatusOK)
	fmt.Fprint(w, "ready")
}

// mcpRequest represents a simplified MCP tool call request.
type mcpRequest struct {
	Method string                 `json:"method"`
	Params mcpParams              `json:"params"`
}

type mcpParams struct {
	Name      string                 `json:"name"`
	Arguments map[string]interface{} `json:"arguments"`
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
		// Return tool list for discovery
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

	slog.Info("tool invoked", "tool", req.Params.Name)

	result, err := tool.Run(r.Context(), req.Params.Arguments)
	if err != nil {
		slog.Error("tool execution failed", "tool", req.Params.Name, "error", err)
		resp := mcpResponse{
			Content: []mcpContent{{Type: "text", Text: err.Error()}},
			IsError: true,
		}
		writeJSON(w, resp, http.StatusOK)
		return
	}

	resultJSON, err := json.Marshal(result)
	if err != nil {
		slog.Error("failed to marshal tool result", "tool", req.Params.Name, "error", err)
		writeJSONError(w, "internal error", http.StatusInternalServerError)
		return
	}

	resp := mcpResponse{
		Content: []mcpContent{{Type: "text", Text: string(resultJSON)}},
	}
	writeJSON(w, resp, http.StatusOK)
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
		Addr:    addr,
		Handler: s.Handler(),
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
