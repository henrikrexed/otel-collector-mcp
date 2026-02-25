package tools

import (
	"context"

	"github.com/hrexed/otel-collector-mcp/pkg/config"
	"github.com/hrexed/otel-collector-mcp/pkg/k8s"
	"github.com/hrexed/otel-collector-mcp/pkg/types"
)

// Tool is the interface all MCP tools must implement.
type Tool interface {
	Name() string
	Description() string
	InputSchema() map[string]interface{}
	Run(ctx context.Context, args map[string]interface{}) (*types.StandardResponse, error)
}

// BaseTool provides common fields for all tools.
type BaseTool struct {
	Cfg     *config.Config
	Clients *k8s.Clients
}

// ClusterMeta returns the cluster metadata for responses.
func (b *BaseTool) ClusterMeta() types.ClusterMetadata {
	return b.Cfg.ClusterMetadata()
}
