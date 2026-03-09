package types

import "strings"

// Severity constants for diagnostic findings.
const (
	SeverityCritical = "critical"
	SeverityWarning  = "warning"
	SeverityInfo     = "info"
	SeverityOk       = "ok"
)

// Category constants for diagnostic findings.
const (
	CategoryPipeline    = "pipeline"
	CategoryConfig      = "config"
	CategorySecurity    = "security"
	CategoryPerformance = "performance"
	CategoryOperator    = "operator"
	CategoryRuntime     = "runtime"
)

// ResourceRef identifies a Kubernetes resource associated with a finding.
type ResourceRef struct {
	Kind       string `json:"kind"`
	Namespace  string `json:"namespace"`
	Name       string `json:"name"`
	APIVersion string `json:"apiVersion,omitempty"`
}

// DiagnosticFinding represents a single diagnostic finding from analysis.
type DiagnosticFinding struct {
	Severity    string       `json:"severity"`
	Category    string       `json:"category"`
	Resource    *ResourceRef `json:"resource,omitempty"`
	Summary     string       `json:"summary"`
	Detail      string       `json:"detail"`
	Suggestion  string       `json:"suggestion"`
	Remediation string       `json:"remediation,omitempty"`
}

// SeverityIcon returns a compact emoji for the severity level.
func SeverityIcon(severity string) string {
	switch severity {
	case SeverityCritical:
		return "❗"
	case SeverityWarning:
		return "⚠️"
	case SeverityOk:
		return "✅"
	default:
		return "ℹ️"
	}
}

// FindingsToText renders findings as a compact markdown table.
func FindingsToText(findings []DiagnosticFinding) string {
	if len(findings) == 0 {
		return "(no findings)"
	}

	var sb strings.Builder
	sb.WriteString("| St | Resource | Summary | Detail |\n")
	sb.WriteString("|----|----------|---------|--------|\n")
	for _, f := range findings {
		res := "-"
		if f.Resource != nil {
			res = f.Resource.Kind
			if f.Resource.Namespace != "" {
				res += " " + f.Resource.Namespace + "/" + f.Resource.Name
			} else {
				res += " " + f.Resource.Name
			}
		}
		detail := f.Detail
		if f.Suggestion != "" {
			if detail != "" {
				detail += " → "
			}
			detail += f.Suggestion
		}
		if f.Remediation != "" {
			if detail != "" {
				detail += " "
			}
			detail += "[fix: " + f.Remediation + "]"
		}
		// Escape pipes in content
		res = strings.ReplaceAll(res, "|", "\\|")
		summary := strings.ReplaceAll(f.Summary, "|", "\\|")
		detail = strings.ReplaceAll(detail, "|", "\\|")
		// Replace newlines
		summary = strings.ReplaceAll(summary, "\n", " ")
		detail = strings.ReplaceAll(detail, "\n", " ")

		sb.WriteString("| " + SeverityIcon(f.Severity) + " | " + res + " | " + summary + " | " + detail + " |\n")
	}
	return sb.String()
}
