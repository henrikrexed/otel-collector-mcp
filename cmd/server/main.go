package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

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

	// Initialize OpenTelemetry (traces, metrics, logs)
	shutdown, err := telemetry.InitTelemetry(ctx, cfg.OTelEnabled, cfg.OTelEndpoint)
	if err != nil {
		slog.Error("failed to initialize telemetry", "error", err)
		os.Exit(1)
	}
	defer shutdown()

	// Reconfigure slog with OTel log bridge (tee: stdout + OTel export)
	if cfg.OTelEnabled {
		telemetry.SetupOTelLogging(cfg.SlogLevel())
	}

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

	// Create MCP server (Streamable HTTP via official Go MCP SDK)
	srv := mcp.NewServer(registry, watcher.Features().IsReady, cfg.Port)

	// Start MCP server in background
	go func() {
		addr := fmt.Sprintf(":%d", cfg.Port)
		if err := srv.Start(addr); err != nil && err != http.ErrServerClosed {
			slog.Error("MCP server error", "error", err)
			os.Exit(1)
		}
	}()

	slog.Info("server ready", "port", cfg.Port)

	// Block until shutdown signal
	<-ctx.Done()
	slog.Info("shutting down")

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		slog.Error("MCP server shutdown error", "error", err)
	}

	slog.Info("otel-collector-mcp stopped")
}
