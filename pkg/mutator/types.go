package mutator

import "context"

// Mutator defines the interface for safely mutating collector configurations
// with backup and rollback capability.
type Mutator interface {
	// Backup stores the current config for later rollback.
	Backup(ctx context.Context, sessionID string) error

	// ApplyConfig applies new YAML config to the collector.
	ApplyConfig(ctx context.Context, configYAML string) error

	// Rollback restores the backed-up config and triggers a rollout.
	Rollback(ctx context.Context) error

	// TriggerRollout restarts the collector workload to pick up config changes.
	TriggerRollout(ctx context.Context) error

	// Cleanup removes backup annotations and session metadata.
	Cleanup(ctx context.Context) error

	// DetectGitOps checks for ArgoCD or Flux annotations and returns a warning if found.
	DetectGitOps(ctx context.Context) (bool, string)
}

// DeploymentMode represents how a collector is deployed.
type DeploymentMode string

const (
	ModeDeployment  DeploymentMode = "Deployment"
	ModeDaemonSet   DeploymentMode = "DaemonSet"
	ModeStatefulSet DeploymentMode = "StatefulSet"
	ModeOperatorCRD DeploymentMode = "OperatorCRD"
)

// CollectorRef identifies a specific collector instance.
type CollectorRef struct {
	Name           string
	Namespace      string
	DeploymentMode DeploymentMode
	ConfigMapName  string // For ConfigMap-based collectors
	ConfigKey      string // The key in ConfigMap.data holding the collector YAML
	OwnerKind      string // Deployment, DaemonSet, or StatefulSet
	OwnerName      string // Name of the owning workload
}

// Backup annotation keys.
const (
	AnnotationConfigBackup = "mcp.otel.dev/config-backup"
	AnnotationSessionID    = "mcp.otel.dev/session-id"
)
