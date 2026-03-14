package mutator

import (
	"context"
	"encoding/json"
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	k8stypes "k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
)

var otelCollectorGVR = schema.GroupVersionResource{
	Group:    "opentelemetry.io",
	Version:  "v1beta1",
	Resource: "opentelemetrycollectors",
}

// CRDMutator implements Mutator for OTel Operator CRD-based collectors.
type CRDMutator struct {
	clientset     kubernetes.Interface
	dynamicClient dynamic.Interface
	ref           CollectorRef
}

// NewCRDMutator creates a CRDMutator for the given collector reference.
func NewCRDMutator(clientset kubernetes.Interface, ref CollectorRef) *CRDMutator {
	return &CRDMutator{
		clientset: clientset,
		ref:       ref,
	}
}

// SetDynamicClient sets the dynamic client for CRD operations.
func (m *CRDMutator) SetDynamicClient(dc dynamic.Interface) {
	m.dynamicClient = dc
}

func (m *CRDMutator) Backup(ctx context.Context, sessionID string) error {
	if m.dynamicClient == nil {
		return fmt.Errorf("dynamic client not configured for CRD operations")
	}

	cr, err := m.dynamicClient.Resource(otelCollectorGVR).Namespace(m.ref.Namespace).Get(ctx, m.ref.Name, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("failed to get OpenTelemetryCollector CR %s/%s: %w", m.ref.Namespace, m.ref.Name, err)
	}

	// Store full .spec as JSON annotation
	spec, found, err := unstructured.NestedMap(cr.Object, "spec")
	if err != nil {
		return fmt.Errorf("failed to extract spec from CR: %w", err)
	}
	if !found {
		return fmt.Errorf("no spec found in CR %s/%s", m.ref.Namespace, m.ref.Name)
	}

	specJSON, err := json.Marshal(spec)
	if err != nil {
		return fmt.Errorf("failed to marshal CR spec: %w", err)
	}

	// Set annotations
	annotations := cr.GetAnnotations()
	if annotations == nil {
		annotations = make(map[string]string)
	}
	annotations[AnnotationConfigBackup] = string(specJSON)
	annotations[AnnotationSessionID] = sessionID
	cr.SetAnnotations(annotations)

	_, err = m.dynamicClient.Resource(otelCollectorGVR).Namespace(m.ref.Namespace).Update(ctx, cr, metav1.UpdateOptions{})
	if err != nil {
		return fmt.Errorf("failed to update CR with backup annotation: %w", err)
	}

	return nil
}

func (m *CRDMutator) ApplyConfig(ctx context.Context, configYAML string) error {
	if m.dynamicClient == nil {
		return fmt.Errorf("dynamic client not configured for CRD operations")
	}

	patch := map[string]interface{}{
		"spec": map[string]interface{}{
			"config": configYAML,
		},
	}

	patchJSON, err := json.Marshal(patch)
	if err != nil {
		return fmt.Errorf("failed to marshal config patch: %w", err)
	}

	_, err = m.dynamicClient.Resource(otelCollectorGVR).Namespace(m.ref.Namespace).Patch(
		ctx, m.ref.Name, k8stypes.MergePatchType, patchJSON, metav1.PatchOptions{})
	if err != nil {
		return fmt.Errorf("failed to patch CR spec.config: %w", err)
	}

	return nil
}

func (m *CRDMutator) Rollback(ctx context.Context) error {
	if m.dynamicClient == nil {
		return fmt.Errorf("dynamic client not configured for CRD operations")
	}

	cr, err := m.dynamicClient.Resource(otelCollectorGVR).Namespace(m.ref.Namespace).Get(ctx, m.ref.Name, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("failed to get CR for rollback: %w", err)
	}

	annotations := cr.GetAnnotations()
	backupJSON, ok := annotations[AnnotationConfigBackup]
	if !ok {
		return fmt.Errorf("no backup annotation found on CR %s/%s", m.ref.Namespace, m.ref.Name)
	}

	var backupSpec map[string]interface{}
	if err := json.Unmarshal([]byte(backupJSON), &backupSpec); err != nil {
		return fmt.Errorf("failed to unmarshal backup spec: %w", err)
	}

	// Restore spec
	if err := unstructured.SetNestedMap(cr.Object, backupSpec, "spec"); err != nil {
		return fmt.Errorf("failed to set spec from backup: %w", err)
	}

	// Clean up annotations
	delete(annotations, AnnotationConfigBackup)
	delete(annotations, AnnotationSessionID)
	cr.SetAnnotations(annotations)

	_, err = m.dynamicClient.Resource(otelCollectorGVR).Namespace(m.ref.Namespace).Update(ctx, cr, metav1.UpdateOptions{})
	if err != nil {
		return fmt.Errorf("failed to restore CR from backup: %w", err)
	}

	// Operator handles rollout automatically
	return nil
}

func (m *CRDMutator) TriggerRollout(_ context.Context) error {
	// Operator handles rollout automatically when CR spec changes
	return nil
}

func (m *CRDMutator) Cleanup(ctx context.Context) error {
	if m.dynamicClient == nil {
		return fmt.Errorf("dynamic client not configured for CRD operations")
	}

	cr, err := m.dynamicClient.Resource(otelCollectorGVR).Namespace(m.ref.Namespace).Get(ctx, m.ref.Name, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("failed to get CR for cleanup: %w", err)
	}

	annotations := cr.GetAnnotations()
	delete(annotations, AnnotationConfigBackup)
	delete(annotations, AnnotationSessionID)
	cr.SetAnnotations(annotations)

	_, err = m.dynamicClient.Resource(otelCollectorGVR).Namespace(m.ref.Namespace).Update(ctx, cr, metav1.UpdateOptions{})
	if err != nil {
		return fmt.Errorf("failed to cleanup CR annotations: %w", err)
	}

	return nil
}

func (m *CRDMutator) DetectGitOps(ctx context.Context) (bool, string) {
	if m.dynamicClient == nil {
		return false, ""
	}

	cr, err := m.dynamicClient.Resource(otelCollectorGVR).Namespace(m.ref.Namespace).Get(ctx, m.ref.Name, metav1.GetOptions{})
	if err != nil {
		return false, ""
	}

	annotations := cr.GetAnnotations()
	gitopsAnnotations := []string{
		"argocd.argoproj.io/managed-by",
		"fluxcd.io/automated",
	}

	for _, ann := range gitopsAnnotations {
		if _, ok := annotations[ann]; ok {
			return true, fmt.Sprintf("GitOps managed resource detected (%s). Mutations may be reverted.", ann)
		}
	}

	return false, ""
}

var _ Mutator = (*CRDMutator)(nil)
