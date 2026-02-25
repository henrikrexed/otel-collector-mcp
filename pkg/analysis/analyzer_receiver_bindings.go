package analysis

import (
	"context"
	"fmt"

	"github.com/hrexed/otel-collector-mcp/pkg/types"
)

// AnalyzeReceiverBindings checks for receiver endpoint port conflicts and missing protocols.
func AnalyzeReceiverBindings(_ context.Context, input *AnalysisInput) []types.DiagnosticFinding {
	if input.Config == nil {
		return nil
	}

	var findings []types.DiagnosticFinding
	portUsage := make(map[string][]string) // port -> list of receivers using it

	for name, receiverCfg := range input.Config.Receivers {
		cfgMap, ok := receiverCfg.(map[string]interface{})
		if !ok {
			continue
		}

		// Check top-level endpoint
		if endpoint, ok := getNestedString(cfgMap, "endpoint"); ok {
			portUsage[endpoint] = append(portUsage[endpoint], name)
		}

		// Check protocol-level endpoints (e.g., otlp receiver with grpc/http)
		if protocols, ok := getNestedMap(cfgMap, "protocols"); ok {
			for proto, protoCfg := range protocols {
				protoCfgMap, ok := protoCfg.(map[string]interface{})
				if !ok {
					continue
				}
				if endpoint, ok := getNestedString(protoCfgMap, "endpoint"); ok {
					portUsage[endpoint] = append(portUsage[endpoint], fmt.Sprintf("%s/%s", name, proto))
				}
			}
		}
	}

	// Check for port conflicts
	for endpoint, receivers := range portUsage {
		if len(receivers) > 1 {
			findings = append(findings, types.DiagnosticFinding{
				Severity:   types.SeverityCritical,
				Category:   types.CategoryConfig,
				Summary:    fmt.Sprintf("Port conflict: endpoint %q is used by multiple receivers", endpoint),
				Detail:     fmt.Sprintf("Receivers %v are all configured to listen on %s. Only one receiver can bind to a given endpoint.", receivers, endpoint),
				Suggestion: "Assign unique endpoints to each receiver",
			})
		}
	}

	return findings
}
