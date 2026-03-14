package tools

import (
	"sort"
	"testing"
	"time"

	"github.com/hrexed/otel-collector-mcp/pkg/config"
	"github.com/hrexed/otel-collector-mcp/pkg/session"
)

func TestRegistryV1Only(t *testing.T) {
	registry := NewRegistry()
	names := registry.List()
	if len(names) != 0 {
		t.Errorf("expected 0 tools in empty registry, got %d", len(names))
	}
}

func TestRegisterV2Tools(t *testing.T) {
	registry := NewRegistry()
	base := BaseTool{
		Cfg: &config.Config{V2Enabled: true},
	}
	mgr := session.NewManager(10*time.Minute, 5)

	RegisterV2Tools(registry, base, mgr)

	names := registry.List()
	if len(names) != 10 {
		t.Errorf("expected 10 v2 tools, got %d", len(names))
	}

	expected := []string{
		"check_health",
		"start_analysis",
		"rollback_config",
		"capture_signals",
		"cleanup_debug",
		"detect_issues",
		"suggest_fixes",
		"apply_fix",
		"recommend_sampling",
		"recommend_sizing",
	}

	sort.Strings(names)
	sort.Strings(expected)

	for i, name := range expected {
		if i >= len(names) || names[i] != name {
			t.Errorf("expected tool %q at index %d, got %q", name, i, names[i])
		}
	}
}
