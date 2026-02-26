package skills

import (
	"context"
	"fmt"
	"strings"

	"github.com/hrexed/otel-collector-mcp/pkg/config"
	"github.com/hrexed/otel-collector-mcp/pkg/types"
)

// ArchitectureSkill recommends collector deployment topologies.
type ArchitectureSkill struct {
	Cfg *config.Config
}

func (s *ArchitectureSkill) Definition() SkillDefinition {
	return SkillDefinition{
		Name:        "design_architecture",
		Description: "Recommend OTel Collector deployment topology based on workload description",
		Parameters: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"signal_types": map[string]interface{}{
					"type":        "array",
					"items":       map[string]interface{}{"type": "string"},
					"description": "Signal types to collect: traces, metrics, logs",
				},
				"scale": map[string]interface{}{
					"type":        "string",
					"description": "Expected scale: small (<50 pods), medium (50-500), large (500+)",
				},
				"backends": map[string]interface{}{
					"type":        "array",
					"items":       map[string]interface{}{"type": "string"},
					"description": "Backend targets: e.g., jaeger, prometheus, datadog, dynatrace, otlp",
				},
				"needs_sampling": map[string]interface{}{
					"type":        "boolean",
					"description": "Whether tail sampling is needed",
				},
				"needs_prometheus_scraping": map[string]interface{}{
					"type":        "boolean",
					"description": "Whether Prometheus target scraping is needed",
				},
			},
		},
	}
}

func (s *ArchitectureSkill) Execute(_ context.Context, args map[string]interface{}) (*types.StandardResponse, error) {
	signalTypes := extractStringSlice(args, "signal_types")
	scale, _ := args["scale"].(string)
	backends := extractStringSlice(args, "backends")
	needsSampling, _ := args["needs_sampling"].(bool)
	needsPromScraping, _ := args["needs_prometheus_scraping"].(bool)

	rec := buildRecommendation(signalTypes, scale, backends, needsSampling, needsPromScraping)

	meta := s.Cfg.ClusterMetadata()
	return types.NewStandardResponse(meta, "design_architecture", rec), nil
}

type architectureRecommendation struct {
	Topology   string            `json:"topology"`
	Components []componentRec    `json:"components"`
	Rationale  []string          `json:"rationale"`
	Config     string            `json:"configSkeleton"`
}

type componentRec struct {
	Name           string `json:"name"`
	DeploymentMode string `json:"deploymentMode"`
	Role           string `json:"role"`
	Reason         string `json:"reason"`
}

func buildRecommendation(signals []string, scale string, backends []string, needsSampling, needsPromScraping bool) *architectureRecommendation {
	rec := &architectureRecommendation{}

	hasLogs := contains(signals, "logs")
	hasTraces := contains(signals, "traces")
	hasMetrics := contains(signals, "metrics")

	needsGateway := needsSampling || scale == "large" || len(backends) > 1

	if hasLogs && (hasTraces || hasMetrics) && needsGateway {
		rec.Topology = "Hybrid Agent→Gateway"
		rec.Rationale = append(rec.Rationale, "Mixed signal types with gateway requirements call for the hybrid Agent→Gateway pattern")
	} else if hasLogs && !hasTraces && !hasMetrics {
		rec.Topology = "DaemonSet Only"
		rec.Rationale = append(rec.Rationale, "Log collection requires node-level access to /var/log, which requires a DaemonSet")
	} else if needsGateway {
		rec.Topology = "Gateway (Deployment/StatefulSet)"
		rec.Rationale = append(rec.Rationale, "Centralized processing is needed for tail sampling or multi-backend fan-out")
	} else {
		rec.Topology = "DaemonSet Agent"
		rec.Rationale = append(rec.Rationale, "Simple signal collection at scale is best served by per-node agents")
	}

	// Add log agent
	if hasLogs {
		rec.Components = append(rec.Components, componentRec{
			Name:           "otel-agent-logs",
			DeploymentMode: "DaemonSet",
			Role:           "Log collection agent",
			Reason:         "Logs require node-level /var/log access, which only DaemonSets provide",
		})
	}

	// Add gateway for traces/metrics if needed
	if needsGateway && (hasTraces || hasMetrics) {
		mode := "Deployment"
		reason := "Centralized gateway for trace/metric processing and backend fan-out"

		if needsPromScraping {
			mode = "StatefulSet"
			reason = "StatefulSet required for Target Allocator to assign scrape targets to specific pods"
			rec.Rationale = append(rec.Rationale, "Prometheus scraping with Target Allocator requires StatefulSet")
		}

		if needsSampling {
			rec.Rationale = append(rec.Rationale, "Tail sampling requires all spans for a trace to reach the same collector, necessitating a centralized gateway")
		}

		rec.Components = append(rec.Components, componentRec{
			Name:           "otel-gateway",
			DeploymentMode: mode,
			Role:           "Centralized gateway",
			Reason:         reason,
		})
	} else if (hasTraces || hasMetrics) && !hasLogs {
		rec.Components = append(rec.Components, componentRec{
			Name:           "otel-agent",
			DeploymentMode: "DaemonSet",
			Role:           "Telemetry agent",
			Reason:         "Per-node agents for trace/metric collection",
		})
	}

	rec.Config = generateSkeletonConfig(rec.Components, signals, backends)

	return rec
}

func generateSkeletonConfig(components []componentRec, signals, backends []string) string {
	var b strings.Builder
	b.WriteString("# Skeleton collector configuration\n")

	b.WriteString("receivers:\n  otlp:\n    protocols:\n      grpc:\n        endpoint: \"0.0.0.0:4317\"\n      http:\n        endpoint: \"0.0.0.0:4318\"\n\n")

	b.WriteString("processors:\n  memory_limiter:\n    check_interval: 1s\n    limit_mib: 512\n    spike_limit_mib: 128\n  batch:\n    send_batch_size: 8192\n    timeout: 200ms\n\n")

	b.WriteString("exporters:\n")
	for _, backend := range backends {
		fmt.Fprintf(&b, "  %s:\n    endpoint: \"<configure-%s-endpoint>\"\n", backend, backend)
	}
	if len(backends) == 0 {
		b.WriteString("  otlp:\n    endpoint: \"<configure-endpoint>\"\n")
	}

	b.WriteString("\nservice:\n  pipelines:\n")
	exporterList := strings.Join(backends, ", ")
	if exporterList == "" {
		exporterList = "otlp"
	}
	for _, sig := range signals {
		fmt.Fprintf(&b, "    %s:\n      receivers: [otlp]\n      processors: [memory_limiter, batch]\n      exporters: [%s]\n",
			sig, exporterList)
	}

	return b.String()
}

func extractStringSlice(args map[string]interface{}, key string) []string {
	v, ok := args[key]
	if !ok {
		return nil
	}
	arr, ok := v.([]interface{})
	if !ok {
		return nil
	}
	result := make([]string, 0, len(arr))
	for _, item := range arr {
		if s, ok := item.(string); ok {
			result = append(result, s)
		}
	}
	return result
}

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}
