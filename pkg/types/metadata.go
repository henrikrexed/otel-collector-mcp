package types

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// ClusterMetadata provides cluster identification for multi-cluster disambiguation.
type ClusterMetadata struct {
	Cluster   string `json:"cluster"`
	Namespace string `json:"namespace"`
	Context   string `json:"context,omitempty"`
}

// StandardResponse is the envelope for all MCP tool responses.
type StandardResponse struct {
	Cluster   string      `json:"cluster"`
	Namespace string      `json:"namespace"`
	Context   string      `json:"context,omitempty"`
	Timestamp string      `json:"timestamp"`
	Tool      string      `json:"tool"`
	Data      interface{} `json:"data"`
}

// NewStandardResponse creates a new StandardResponse with the given metadata.
func NewStandardResponse(meta ClusterMetadata, tool string, data interface{}) *StandardResponse {
	return &StandardResponse{
		Cluster:   meta.Cluster,
		Namespace: meta.Namespace,
		Context:   meta.Context,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Tool:      tool,
		Data:      data,
	}
}

// ToolResult wraps diagnostic findings with optional metadata.
type ToolResult struct {
	Findings []DiagnosticFinding `json:"findings"`
	Metadata map[string]string   `json:"metadata,omitempty"`
}

// ToText renders a ToolResult as compact text.
func (tr *ToolResult) ToText() string {
	meta := ""
	for k, v := range tr.Metadata {
		if meta != "" {
			meta += " "
		}
		meta += k + "=" + v
	}
	if meta != "" {
		return meta + "\n" + FindingsToText(tr.Findings)
	}
	return FindingsToText(tr.Findings)
}

// ToText renders the StandardResponse as compact text for LLM consumption.
func (r *StandardResponse) ToText() string {
	header := fmt.Sprintf("[%s] cluster=%s", r.Tool, r.Cluster)
	if r.Namespace != "" {
		header += " ns=" + r.Namespace
	}

	if tr, ok := r.Data.(*ToolResult); ok {
		return header + "\n" + tr.ToText()
	}

	// For map responses (e.g. get_config summary), render key=value
	if m, ok := r.Data.(map[string]interface{}); ok {
		var parts []string
		for k, v := range m {
			parts = append(parts, fmt.Sprintf("%s=%v", k, v))
		}
		return header + "\n" + strings.Join(parts, "\n")
	}

	// Fallback: compact JSON
	b, err := json.Marshal(r.Data)
	if err != nil {
		return header + " | (error formatting data)"
	}
	return header + "\n" + string(b)
}
