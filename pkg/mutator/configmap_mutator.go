package mutator

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8stypes "k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
)

// ConfigMapMutator implements Mutator for ConfigMap-based collectors.
type ConfigMapMutator struct {
	clientset       kubernetes.Interface
	ref             CollectorRef
	resourceVersion string
}

// NewConfigMapMutator creates a ConfigMapMutator for the given collector reference.
func NewConfigMapMutator(clientset kubernetes.Interface, ref CollectorRef) *ConfigMapMutator {
	return &ConfigMapMutator{
		clientset: clientset,
		ref:       ref,
	}
}

func (m *ConfigMapMutator) Backup(ctx context.Context, sessionID string) error {
	cm, err := m.clientset.CoreV1().ConfigMaps(m.ref.Namespace).Get(ctx, m.ref.ConfigMapName, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("failed to get ConfigMap %s/%s: %w", m.ref.Namespace, m.ref.ConfigMapName, err)
	}

	// Store full .data as JSON annotation
	dataJSON, err := json.Marshal(cm.Data)
	if err != nil {
		return fmt.Errorf("failed to marshal ConfigMap data: %w", err)
	}

	if cm.Annotations == nil {
		cm.Annotations = make(map[string]string)
	}
	cm.Annotations[AnnotationConfigBackup] = string(dataJSON)
	cm.Annotations[AnnotationSessionID] = sessionID

	updated, err := m.clientset.CoreV1().ConfigMaps(m.ref.Namespace).Update(ctx, cm, metav1.UpdateOptions{})
	if err != nil {
		return fmt.Errorf("failed to update ConfigMap with backup annotation: %w", err)
	}

	m.resourceVersion = updated.ResourceVersion
	return nil
}

func (m *ConfigMapMutator) ApplyConfig(ctx context.Context, configYAML string) error {
	cm, err := m.clientset.CoreV1().ConfigMaps(m.ref.Namespace).Get(ctx, m.ref.ConfigMapName, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("failed to get ConfigMap: %w", err)
	}

	// Only update the collector config key, preserving other data keys
	if cm.Data == nil {
		cm.Data = make(map[string]string)
	}
	cm.Data[m.ref.ConfigKey] = configYAML

	_, err = m.clientset.CoreV1().ConfigMaps(m.ref.Namespace).Update(ctx, cm, metav1.UpdateOptions{})
	if err != nil {
		return fmt.Errorf("failed to update ConfigMap config: %w", err)
	}

	return nil
}

func (m *ConfigMapMutator) Rollback(ctx context.Context) error {
	cm, err := m.clientset.CoreV1().ConfigMaps(m.ref.Namespace).Get(ctx, m.ref.ConfigMapName, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("failed to get ConfigMap for rollback: %w", err)
	}

	backupJSON, ok := cm.Annotations[AnnotationConfigBackup]
	if !ok {
		return fmt.Errorf("no backup annotation found on ConfigMap %s/%s", m.ref.Namespace, m.ref.ConfigMapName)
	}

	// Restore data from backup
	var backupData map[string]string
	if err := json.Unmarshal([]byte(backupJSON), &backupData); err != nil {
		return fmt.Errorf("failed to unmarshal backup data: %w", err)
	}

	cm.Data = backupData
	delete(cm.Annotations, AnnotationConfigBackup)
	delete(cm.Annotations, AnnotationSessionID)

	if _, err := m.clientset.CoreV1().ConfigMaps(m.ref.Namespace).Update(ctx, cm, metav1.UpdateOptions{}); err != nil {
		return fmt.Errorf("failed to restore ConfigMap from backup: %w", err)
	}

	return m.TriggerRollout(ctx)
}

func (m *ConfigMapMutator) TriggerRollout(ctx context.Context) error {
	patch := fmt.Sprintf(`{"spec":{"template":{"metadata":{"annotations":{"kubectl.kubernetes.io/restartedAt":"%s"}}}}}`,
		time.Now().UTC().Format(time.RFC3339))

	var err error
	switch m.ref.OwnerKind {
	case "Deployment":
		_, err = m.clientset.AppsV1().Deployments(m.ref.Namespace).Patch(
			ctx, m.ref.OwnerName, k8stypes.StrategicMergePatchType, []byte(patch), metav1.PatchOptions{})
	case "DaemonSet":
		_, err = m.clientset.AppsV1().DaemonSets(m.ref.Namespace).Patch(
			ctx, m.ref.OwnerName, k8stypes.StrategicMergePatchType, []byte(patch), metav1.PatchOptions{})
	case "StatefulSet":
		_, err = m.clientset.AppsV1().StatefulSets(m.ref.Namespace).Patch(
			ctx, m.ref.OwnerName, k8stypes.StrategicMergePatchType, []byte(patch), metav1.PatchOptions{})
	default:
		return fmt.Errorf("unsupported owner kind: %s", m.ref.OwnerKind)
	}

	if err != nil {
		return fmt.Errorf("failed to trigger rollout for %s/%s: %w", m.ref.OwnerKind, m.ref.OwnerName, err)
	}

	return nil
}

func (m *ConfigMapMutator) Cleanup(ctx context.Context) error {
	cm, err := m.clientset.CoreV1().ConfigMaps(m.ref.Namespace).Get(ctx, m.ref.ConfigMapName, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("failed to get ConfigMap for cleanup: %w", err)
	}

	delete(cm.Annotations, AnnotationConfigBackup)
	delete(cm.Annotations, AnnotationSessionID)

	_, err = m.clientset.CoreV1().ConfigMaps(m.ref.Namespace).Update(ctx, cm, metav1.UpdateOptions{})
	if err != nil {
		return fmt.Errorf("failed to cleanup ConfigMap annotations: %w", err)
	}

	return nil
}

func (m *ConfigMapMutator) DetectGitOps(ctx context.Context) (bool, string) {
	cm, err := m.clientset.CoreV1().ConfigMaps(m.ref.Namespace).Get(ctx, m.ref.ConfigMapName, metav1.GetOptions{})
	if err != nil {
		return false, ""
	}

	gitopsAnnotations := []string{
		"argocd.argoproj.io/managed-by",
		"fluxcd.io/automated",
	}

	var detected []string
	for _, ann := range gitopsAnnotations {
		if _, ok := cm.Annotations[ann]; ok {
			detected = append(detected, ann)
		}
	}

	if len(detected) > 0 {
		return true, fmt.Sprintf("GitOps managed resource detected (%s). Mutations may be reverted by the GitOps controller.", strings.Join(detected, ", "))
	}

	return false, ""
}

// Ensure ConfigMapMutator implements Mutator at compile time.
var _ Mutator = (*ConfigMapMutator)(nil)

// NewMutator factory creates the appropriate mutator based on deployment mode.
func NewMutator(clientset kubernetes.Interface, ref CollectorRef) Mutator {
	if ref.DeploymentMode == ModeOperatorCRD {
		return NewCRDMutator(clientset, ref)
	}
	return NewConfigMapMutator(clientset, ref)
}
