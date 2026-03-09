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

// ToText renders a DiagnosticFinding as a compact single line.
func (f DiagnosticFinding) ToText() string {
	line := SeverityIcon(f.Severity) + " "
	if f.Resource != nil {
		line += f.Resource.Kind + " "
		if f.Resource.Namespace != "" {
			line += f.Resource.Namespace + "/"
		}
		line += f.Resource.Name + " | "
	}
	line += f.Summary
	if f.Detail != "" {
		line += " | " + f.Detail
	}
	if f.Suggestion != "" {
		line += " → " + f.Suggestion
	}
	if f.Remediation != "" {
		line += " [fix: " + f.Remediation + "]"
	}
	return line
}

// FindingsToText renders a slice of findings as compact newline-separated text.
func FindingsToText(findings []DiagnosticFinding) string {
	if len(findings) == 0 {
		return "(no findings)"
	}
	lines := make([]string, len(findings))
	for i, f := range findings {
		lines[i] = f.ToText()
	}
	return strings.Join(lines, "\n")
}
