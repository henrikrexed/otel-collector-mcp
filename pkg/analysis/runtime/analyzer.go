package runtime

import (
	"context"

	"github.com/hrexed/otel-collector-mcp/pkg/signals"
	"github.com/hrexed/otel-collector-mcp/pkg/types"
)

// RuntimeAnalysisInput provides all context needed for runtime analysis.
type RuntimeAnalysisInput struct {
	Signals        *signals.CapturedSignals
	CollectorConfig string
	DeploymentMode string
}

// RuntimeAnalyzer is a function that analyzes captured signal data for issues.
type RuntimeAnalyzer func(ctx context.Context, input *RuntimeAnalysisInput) []types.DiagnosticFinding

// AllRuntimeAnalyzers returns all registered runtime analyzers.
func AllRuntimeAnalyzers() map[string]RuntimeAnalyzer {
	return map[string]RuntimeAnalyzer{
		"high_cardinality":    AnalyzeHighCardinality,
		"pii_detection":       AnalyzePII,
		"orphan_spans":        AnalyzeOrphanSpans,
		"bloated_attributes":  AnalyzeBloatedAttributes,
		"missing_resources":   AnalyzeMissingResources,
		"duplicate_signals":   AnalyzeDuplicateSignals,
		"missing_sampling":    AnalyzeMissingSampling,
		"resource_sizing":     AnalyzeResourceSizing,
	}
}
