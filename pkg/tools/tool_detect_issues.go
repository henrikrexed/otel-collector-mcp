package tools

import (
	"context"
	"log/slog"
	"sort"
	"strconv"

	"github.com/hrexed/otel-collector-mcp/pkg/analysis/runtime"
	"github.com/hrexed/otel-collector-mcp/pkg/session"
	"github.com/hrexed/otel-collector-mcp/pkg/signals"
	"github.com/hrexed/otel-collector-mcp/pkg/types"
)

// DetectIssuesTool runs all runtime detection rules against captured signal data.
type DetectIssuesTool struct {
	BaseTool
	SessionMgr *session.Manager
}

func (t *DetectIssuesTool) Name() string { return "detect_issues" }

func (t *DetectIssuesTool) Description() string {
	return "Analyze captured signal data for runtime anti-patterns: high cardinality, PII, orphan spans, bloated attributes."
}

func (t *DetectIssuesTool) InputSchema() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"session_id": map[string]interface{}{"type": "string", "description": "Active session ID"},
		},
		"required": []string{"session_id"},
	}
}

func (t *DetectIssuesTool) Run(ctx context.Context, args map[string]interface{}) (*types.StandardResponse, error) {
	sessionID, _ := args["session_id"].(string)
	if sessionID == "" {
		return nil, types.NewMCPError(types.ErrCodeSessionNotFound, "session_id is required")
	}

	sess, err := t.SessionMgr.Get(sessionID)
	if err != nil {
		return nil, err
	}

	captured, _ := sess.CapturedSignals.(*signals.CapturedSignals)
	if captured == nil {
		captured = &signals.CapturedSignals{}
	}

	sess.SetState(session.StateAnalyzing)

	input := &runtime.RuntimeAnalysisInput{
		Signals:        captured,
		CollectorConfig: sess.BackupConfig,
		DeploymentMode: string(sess.Collector.DeploymentMode),
	}

	// Run all analyzers with panic recovery
	var allFindings []types.DiagnosticFinding
	analyzers := runtime.AllRuntimeAnalyzers()

	for name, analyzer := range analyzers {
		func() {
			defer func() {
				if r := recover(); r != nil {
					slog.Error("runtime analyzer panicked", "analyzer", name, "panic", r)
					allFindings = append(allFindings, types.DiagnosticFinding{
						Severity: "error",
						Category: name,
						Summary:  "Analyzer failed with internal error",
					})
				}
			}()
			findings := analyzer(ctx, input)
			allFindings = append(allFindings, findings...)
		}()
	}

	// Sort by severity: critical > warning > info
	severityOrder := map[string]int{"critical": 0, "error": 1, "warning": 2, "info": 3}
	sort.Slice(allFindings, func(i, j int) bool {
		return severityOrder[allFindings[i].Severity] < severityOrder[allFindings[j].Severity]
	})

	// Store findings in session
	sess.Findings = allFindings

	return types.NewStandardResponse(t.ClusterMeta(), t.Name(), &types.ToolResult{
		Findings: allFindings,
		Metadata: map[string]string{
			"session_id":    sessionID,
			"analyzers_run": "8",
			"total_findings": strconv.Itoa(len(allFindings)),
		},
	}), nil
}
