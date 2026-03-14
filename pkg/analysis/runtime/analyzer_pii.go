package runtime

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/hrexed/otel-collector-mcp/pkg/types"
)

var (
	emailRegex = regexp.MustCompile(`[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}`)
	ipv4Regex  = regexp.MustCompile(`\b(?:\d{1,3}\.){3}\d{1,3}\b`)
	ipv6Regex  = regexp.MustCompile(`(?i)([0-9a-f]{1,4}:){7}[0-9a-f]{1,4}`)
	phoneRegex = regexp.MustCompile(`\+?\d{1,4}[-.\s]?\(?\d{1,3}\)?[-.\s]?\d{1,4}[-.\s]?\d{1,9}`)

	// False positive exclusion keys
	falsePositiveKeys = map[string]bool{
		"trace_id": true, "span_id": true, "parent_span_id": true,
		"traceID": true, "spanID": true, "parentSpanID": true,
		"http.url": true, "net.peer.ip": true, "net.host.ip": true,
		"k8s.pod.ip": true, "k8s.node.name": true,
	}
)

// AnalyzePII detects PII patterns in captured log and span data.
func AnalyzePII(_ context.Context, input *RuntimeAnalysisInput) []types.DiagnosticFinding {
	if input.Signals == nil {
		return nil
	}

	var findings []types.DiagnosticFinding

	// Check log attributes
	for _, log := range input.Signals.Logs {
		for key, value := range log.Attributes {
			if falsePositiveKeys[key] {
				continue
			}
			if piiType := detectPII(value); piiType != "" {
				findings = append(findings, types.DiagnosticFinding{
					Severity:    "warning",
					Category:    "pii",
					Summary:     fmt.Sprintf("Potential %s detected in log attribute '%s'", piiType, key),
					Remediation: "Add a filter or transform processor to redact this attribute before export.",
				})
			}
		}
	}

	// Check span attributes
	for _, span := range input.Signals.Traces {
		for key, value := range span.Attributes {
			if falsePositiveKeys[key] {
				continue
			}
			if piiType := detectPII(value); piiType != "" {
				findings = append(findings, types.DiagnosticFinding{
					Severity:    "warning",
					Category:    "pii",
					Summary:     fmt.Sprintf("Potential %s detected in span attribute '%s'", piiType, key),
					Remediation: "Add a transform processor to redact this attribute.",
				})
			}
		}
	}

	return findings
}

func detectPII(value string) string {
	if emailRegex.MatchString(value) {
		return "email address"
	}
	if ipv4Regex.MatchString(value) && !strings.HasPrefix(value, "10.") && !strings.HasPrefix(value, "192.168.") {
		return "IP address"
	}
	if ipv6Regex.MatchString(value) {
		return "IPv6 address"
	}
	if phoneRegex.MatchString(value) && len(value) >= 10 {
		return "phone number"
	}
	return ""
}
