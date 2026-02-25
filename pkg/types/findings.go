package types

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
