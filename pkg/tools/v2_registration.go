package tools

import (
	"log/slog"

	"github.com/hrexed/otel-collector-mcp/pkg/session"
)

// RegisterV2Tools registers all 10 v2 tools into the registry.
// Call this only when V2Enabled is true.
func RegisterV2Tools(registry *Registry, base BaseTool, sessionMgr *session.Manager) {
	registry.Register(&CheckHealthTool{BaseTool: base})
	registry.Register(&StartAnalysisTool{BaseTool: base, SessionMgr: sessionMgr})
	registry.Register(&RollbackConfigTool{BaseTool: base, SessionMgr: sessionMgr})
	registry.Register(&CaptureSignalsTool{BaseTool: base, SessionMgr: sessionMgr})
	registry.Register(&CleanupDebugTool{BaseTool: base, SessionMgr: sessionMgr})
	registry.Register(&DetectIssuesTool{BaseTool: base, SessionMgr: sessionMgr})
	registry.Register(&SuggestFixesTool{BaseTool: base, SessionMgr: sessionMgr})
	registry.Register(&ApplyFixTool{BaseTool: base, SessionMgr: sessionMgr})
	registry.Register(&RecommendSamplingTool{BaseTool: base, SessionMgr: sessionMgr})
	registry.Register(&RecommendSizingTool{BaseTool: base, SessionMgr: sessionMgr})

	slog.Info("v2 tools registered", "count", 10)
}
