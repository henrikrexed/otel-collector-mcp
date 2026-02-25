package types

import "time"

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
