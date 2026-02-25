package analysis

import (
	"context"

	"github.com/hrexed/otel-collector-mcp/pkg/collector"
	"github.com/hrexed/otel-collector-mcp/pkg/types"
	corev1 "k8s.io/api/core/v1"
)

// Analyzer is the function signature for all detection rules.
type Analyzer func(ctx context.Context, input *AnalysisInput) []types.DiagnosticFinding

// AnalysisInput provides data for analyzers to inspect.
type AnalysisInput struct {
	Config       *collector.CollectorConfig
	DeployMode   collector.DeploymentMode
	Logs         []string
	OperatorLogs []string
	PodInfo      *corev1.Pod
}

// AllAnalyzers returns all registered config-based analyzers.
func AllAnalyzers() []Analyzer {
	return []Analyzer{
		AnalyzeMissingBatch,
		AnalyzeMissingMemoryLimiter,
		AnalyzeHardcodedTokens,
		AnalyzeMissingRetryQueue,
		AnalyzeReceiverBindings,
		AnalyzeTailSamplingDaemonSet,
		AnalyzeInvalidRegex,
		AnalyzeConnectorMisconfig,
		AnalyzeResourceDetectorConflicts,
		AnalyzeCumulativeDelta,
		AnalyzeHighCardinality,
	}
}

// AllAnalyzersIncludingLogs returns all analyzers including log-based ones.
func AllAnalyzersIncludingLogs() []Analyzer {
	all := AllAnalyzers()
	all = append(all, AnalyzeExporterBackpressure)
	return all
}
