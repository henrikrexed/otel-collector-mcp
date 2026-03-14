package session

import (
	"context"
	"log/slog"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	"github.com/hrexed/otel-collector-mcp/pkg/mutator"
)

// RecoverOrphanedSessions scans ConfigMaps and CRDs for orphaned session annotations
// and cleans them up. Called on server startup.
func RecoverOrphanedSessions(ctx context.Context, clientset kubernetes.Interface) {
	// Scan all namespaces for ConfigMaps with session annotations
	namespaces, err := clientset.CoreV1().Namespaces().List(ctx, metav1.ListOptions{})
	if err != nil {
		slog.Warn("failed to list namespaces for orphan recovery", "error", err)
		return
	}

	for _, ns := range namespaces.Items {
		configMaps, err := clientset.CoreV1().ConfigMaps(ns.Name).List(ctx, metav1.ListOptions{})
		if err != nil {
			slog.Warn("failed to list ConfigMaps", "namespace", ns.Name, "error", err)
			continue
		}

		for _, cm := range configMaps.Items {
			sessionID, hasSession := cm.Annotations[mutator.AnnotationSessionID]
			_, hasBackup := cm.Annotations[mutator.AnnotationConfigBackup]

			if hasSession || hasBackup {
				slog.Warn("recovering orphaned session",
					"namespace", ns.Name,
					"configmap", cm.Name,
					"session_id", sessionID,
				)

				// Clean up annotations
				cmCopy := cm.DeepCopy()
				delete(cmCopy.Annotations, mutator.AnnotationSessionID)
				delete(cmCopy.Annotations, mutator.AnnotationConfigBackup)

				if _, err := clientset.CoreV1().ConfigMaps(ns.Name).Update(ctx, cmCopy, metav1.UpdateOptions{}); err != nil {
					slog.Error("failed to cleanup orphaned session", "configmap", cm.Name, "error", err)
				} else {
					slog.Info("orphaned session recovered", "configmap", cm.Name, "session_id", sessionID)
				}
			}
		}
	}
}
