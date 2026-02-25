package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/hrexed/otel-collector-mcp/pkg/config"
	"github.com/hrexed/otel-collector-mcp/pkg/discovery"
	"github.com/hrexed/otel-collector-mcp/pkg/k8s"
	"github.com/hrexed/otel-collector-mcp/pkg/mcp"
	"github.com/hrexed/otel-collector-mcp/pkg/telemetry"
	"github.com/hrexed/otel-collector-mcp/pkg/tools"
)

func main() {
	// Initialize configuration
	cfg := config.NewFromEnv()
	cfg.SetupLogging()

	slog.Info("starting otel-collector-mcp",
		"port", cfg.Port,
		"clusterName", cfg.ClusterName,
		"otelEnabled", cfg.OTelEnabled,
	)

	// Create context with signal handling
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	// Initialize OpenTelemetry tracing
	_, shutdown, err := telemetry.InitTracer(ctx, cfg.OTelEnabled, cfg.OTelEndpoint)
	if err != nil {
		slog.Error("failed to initialize tracer", "error", err)
		os.Exit(1)
	}
	defer shutdown()

	// Initialize Kubernetes clients
	clients, err := k8s.NewClients()
	if err != nil {
		slog.Error("failed to initialize kubernetes clients", "error", err)
		os.Exit(1)
	}

	// Initialize tool registry
	registry := tools.NewRegistry()

	// Base tool config for all tools
	baseTool := tools.BaseTool{
		Cfg:     cfg,
		Clients: clients,
	}

	// Initialize CRD discovery
	watcher := discovery.NewCRDWatcher(clients.Discovery, func(hasOTelOperator, hasTargetAllocator bool) {
		slog.Info("features changed, re-syncing tools",
			"hasOTelOperator", hasOTelOperator,
			"hasTargetAllocator", hasTargetAllocator,
		)
	})

	hasOperator := func() bool {
		op, _ := watcher.Features().Get()
		return op
	}

	// Register discovery tools
	registry.Register(&tools.DetectDeploymentTool{BaseTool: baseTool, HasOperator: hasOperator})
	registry.Register(&tools.ListCollectorsTool{BaseTool: baseTool, HasOperator: hasOperator})
	registry.Register(&tools.GetConfigTool{BaseTool: baseTool})

	// Register log parsing tools
	registry.Register(&tools.ParseCollectorLogsTool{BaseTool: baseTool})
	registry.Register(&tools.ParseOperatorLogsTool{BaseTool: baseTool, HasOperator: hasOperator})

	// Register analysis tools
	registry.Register(&tools.TriageScanTool{BaseTool: baseTool, HasOperator: hasOperator})
	registry.Register(&tools.CheckConfigTool{BaseTool: baseTool, HasOperator: hasOperator})

	// Start CRD discovery in background
	go watcher.Start(ctx)

	// Create and start MCP server
	mcpServer := mcp.NewServer(registry, watcher.Features().IsReady)

	addr := fmt.Sprintf(":%d", cfg.Port)
	if err := mcpServer.ListenAndServe(ctx, addr); err != nil {
		slog.Error("MCP server stopped", "error", err)
	}

	slog.Info("otel-collector-mcp stopped")
}
