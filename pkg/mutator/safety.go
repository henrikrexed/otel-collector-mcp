package mutator

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"k8s.io/client-go/kubernetes"
)

// SafeApplyResult holds the result of a safe mutation operation.
type SafeApplyResult struct {
	Applied    bool
	RolledBack bool
	HealthOK   bool
	Error      error
	Message    string
}

// SafeApply performs a config mutation with automatic health check and rollback.
// Sequence: backup → apply → rollout → wait healthy → success OR rollback.
func SafeApply(ctx context.Context, mut Mutator, clientset kubernetes.Interface, ref CollectorRef, sessionID, configYAML string) *SafeApplyResult {
	result := &SafeApplyResult{}

	// Step 1: Backup
	if err := mut.Backup(ctx, sessionID); err != nil {
		result.Error = fmt.Errorf("backup failed, mutation refused: %w", err)
		result.Message = "Backup failed — no config change attempted"
		return result
	}
	slog.Info("config backed up", "collector", ref.Name, "session", sessionID)

	// Step 2: Apply config
	if err := mut.ApplyConfig(ctx, configYAML); err != nil {
		result.Error = fmt.Errorf("apply failed: %w", err)
		result.Message = "Config apply failed"
		rollbackErr := mut.Rollback(ctx)
		if rollbackErr != nil {
			result.Error = fmt.Errorf("apply failed AND rollback failed: apply=%w, rollback=%v", err, rollbackErr)
			result.Message = "CRITICAL: Apply failed and rollback also failed"
		} else {
			result.RolledBack = true
			result.Message = "Config apply failed — rolled back to backup"
		}
		return result
	}
	result.Applied = true
	slog.Info("config applied", "collector", ref.Name)

	// Step 3: Trigger rollout
	if err := mut.TriggerRollout(ctx); err != nil {
		slog.Warn("rollout trigger failed, will still check health", "error", err)
	}

	// Step 4: Wait for health (30-second timeout)
	healthErr := WaitHealthy(ctx, clientset, ref.Namespace, ref.Name, 30*time.Second)
	if healthErr != nil {
		slog.Warn("health check failed, triggering auto-rollback", "error", healthErr, "collector", ref.Name)

		rollbackErr := mut.Rollback(ctx)
		if rollbackErr != nil {
			result.Error = fmt.Errorf("health check failed AND rollback failed: health=%w, rollback=%v", healthErr, rollbackErr)
			result.Message = "CRITICAL: Health check failed and rollback also failed"
			return result
		}

		// Verify recovery after rollback
		recoveryErr := WaitHealthy(ctx, clientset, ref.Namespace, ref.Name, 30*time.Second)
		result.RolledBack = true
		if recoveryErr != nil {
			result.Error = fmt.Errorf("auto-rollback completed but recovery verification failed: %w", recoveryErr)
			result.Message = "Rolled back but recovery not verified"
		} else {
			result.Error = fmt.Errorf("health check failed after mutation: %w", healthErr)
			result.Message = "Auto-rollback successful — collector recovered"
		}
		return result
	}

	result.HealthOK = true
	result.Message = "Config applied and collector is healthy"
	slog.Info("mutation successful, collector healthy", "collector", ref.Name)
	return result
}
