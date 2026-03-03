package collector

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
)

// DeploymentMode represents the deployment type of a collector.
type DeploymentMode string

const (
	ModeDaemonSet   DeploymentMode = "DaemonSet"
	ModeDeployment  DeploymentMode = "Deployment"
	ModeStatefulSet DeploymentMode = "StatefulSet"
	ModeOperatorCRD DeploymentMode = "OperatorCRD"
	ModeUnknown     DeploymentMode = "Unknown"
)

// CollectorInstance represents a discovered collector in the cluster.
type CollectorInstance struct {
	Name            string            `json:"name"`
	Namespace       string            `json:"namespace"`
	DeploymentMode  DeploymentMode    `json:"deploymentMode"`
	Version         string            `json:"version,omitempty"`
	PodCount        int               `json:"podCount"`
	Labels          map[string]string `json:"labels,omitempty"`
	ManagedBy       string            `json:"managedBy,omitempty"`
	OperatorCRDName string            `json:"operatorCRDName,omitempty"`
	ManagedWorkload string            `json:"managedWorkload,omitempty"`
}

// DetectDeploymentMode determines the deployment type of a collector workload.
func DetectDeploymentMode(ctx context.Context, clientset kubernetes.Interface, namespace, name string) (DeploymentMode, error) {
	// Check DaemonSet
	_, err := clientset.AppsV1().DaemonSets(namespace).Get(ctx, name, metav1.GetOptions{})
	if err == nil {
		return ModeDaemonSet, nil
	}

	// Check Deployment
	_, err = clientset.AppsV1().Deployments(namespace).Get(ctx, name, metav1.GetOptions{})
	if err == nil {
		return ModeDeployment, nil
	}

	// Check StatefulSet
	_, err = clientset.AppsV1().StatefulSets(namespace).Get(ctx, name, metav1.GetOptions{})
	if err == nil {
		return ModeStatefulSet, nil
	}

	return ModeUnknown, fmt.Errorf("no workload found for %s/%s", namespace, name)
}

// DetectDeploymentModeWithCRD checks both standard workloads and OTel Operator CRDs.
func DetectDeploymentModeWithCRD(ctx context.Context, clientset kubernetes.Interface, dynClient dynamic.Interface, namespace, name string, hasOperator bool) (DeploymentMode, error) {
	// Check standard workloads first
	mode, err := DetectDeploymentMode(ctx, clientset, namespace, name)
	if err == nil {
		return mode, nil
	}

	// Check Operator CRD if available
	if hasOperator {
		gvr := schema.GroupVersionResource{
			Group:    "opentelemetry.io",
			Version:  "v1beta1",
			Resource: "opentelemetrycollectors",
		}
		_, crdErr := dynClient.Resource(gvr).Namespace(namespace).Get(ctx, name, metav1.GetOptions{})
		if crdErr == nil {
			return ModeOperatorCRD, nil
		}

		// Try v1alpha1
		gvr.Version = "v1alpha1"
		_, crdErr = dynClient.Resource(gvr).Namespace(namespace).Get(ctx, name, metav1.GetOptions{})
		if crdErr == nil {
			return ModeOperatorCRD, nil
		}
	}

	return ModeUnknown, fmt.Errorf("no workload or CRD found for %s/%s", namespace, name)
}

// ListCollectors discovers all OTel Collector instances in the cluster.
// When the OTel Operator is present, CRD entries are merged with the workloads
// they manage (typically named <crd-name>-collector) so each collector appears
// as a single entry with the correct pod count and operator metadata.
func ListCollectors(ctx context.Context, clientset kubernetes.Interface, dynClient dynamic.Interface, namespace string, hasOperator bool) ([]CollectorInstance, error) {
	var collectors []CollectorInstance
	seen := make(map[string]int) // key -> index in collectors slice

	listNS := namespace
	// Empty namespace means all namespaces
	if listNS == "" {
		listNS = metav1.NamespaceAll
	}

	// Search for DaemonSets
	dsList, err := clientset.AppsV1().DaemonSets(listNS).List(ctx, metav1.ListOptions{})
	if err != nil {
		slog.Warn("failed to list DaemonSets", "error", err)
	} else {
		for _, ds := range dsList.Items {
			if isCollector(&ds.ObjectMeta) {
				key := ds.Namespace + "/" + ds.Name
				if _, exists := seen[key]; !exists {
					seen[key] = len(collectors)
					collectors = append(collectors, CollectorInstance{
						Name:           ds.Name,
						Namespace:      ds.Namespace,
						DeploymentMode: ModeDaemonSet,
						Version:        extractVersion(ds.Spec.Template.Spec.Containers),
						PodCount:       int(ds.Status.NumberReady),
						Labels:         ds.Labels,
					})
				}
			}
		}
	}

	// Search for Deployments
	depList, err := clientset.AppsV1().Deployments(listNS).List(ctx, metav1.ListOptions{})
	if err != nil {
		slog.Warn("failed to list Deployments", "error", err)
	} else {
		for _, dep := range depList.Items {
			if isCollector(&dep.ObjectMeta) {
				key := dep.Namespace + "/" + dep.Name
				if _, exists := seen[key]; !exists {
					seen[key] = len(collectors)
					collectors = append(collectors, CollectorInstance{
						Name:           dep.Name,
						Namespace:      dep.Namespace,
						DeploymentMode: ModeDeployment,
						Version:        extractVersion(dep.Spec.Template.Spec.Containers),
						PodCount:       int(dep.Status.ReadyReplicas),
						Labels:         dep.Labels,
					})
				}
			}
		}
	}

	// Search for StatefulSets
	ssList, err := clientset.AppsV1().StatefulSets(listNS).List(ctx, metav1.ListOptions{})
	if err != nil {
		slog.Warn("failed to list StatefulSets", "error", err)
	} else {
		for _, ss := range ssList.Items {
			if isCollector(&ss.ObjectMeta) {
				key := ss.Namespace + "/" + ss.Name
				if _, exists := seen[key]; !exists {
					seen[key] = len(collectors)
					collectors = append(collectors, CollectorInstance{
						Name:           ss.Name,
						Namespace:      ss.Namespace,
						DeploymentMode: ModeStatefulSet,
						Version:        extractVersion(ss.Spec.Template.Spec.Containers),
						PodCount:       int(ss.Status.ReadyReplicas),
						Labels:         ss.Labels,
					})
				}
			}
		}
	}

	// Search for Operator CRDs and merge with discovered workloads
	if hasOperator {
		gvr := schema.GroupVersionResource{
			Group:    "opentelemetry.io",
			Version:  "v1beta1",
			Resource: "opentelemetrycollectors",
		}
		crdList, err := dynClient.Resource(gvr).Namespace(listNS).List(ctx, metav1.ListOptions{})
		if err != nil {
			slog.Warn("failed to list OpenTelemetryCollector CRDs", "error", err)
		} else {
			for _, item := range crdList.Items {
				crdName := item.GetName()
				crdNS := item.GetNamespace()

				// Extract spec.mode from the CRD (e.g. "daemonset", "deployment", "statefulset")
				crdMode := extractCRDSpecMode(item.Object)

				// The operator creates workloads named <crd-name>-collector
				workloadName := crdName + "-collector"
				workloadKey := crdNS + "/" + workloadName

				if idx, found := seen[workloadKey]; found {
					// Workload already discovered — annotate it with operator metadata
					collectors[idx].DeploymentMode = ModeOperatorCRD
					collectors[idx].ManagedBy = "opentelemetry-operator"
					collectors[idx].OperatorCRDName = crdName
					collectors[idx].ManagedWorkload = workloadName
					if crdMode != "" {
						collectors[idx].Labels = mergeLabels(collectors[idx].Labels, "opentelemetry.io/spec-mode", crdMode)
					}
					// Also mark the CRD key as seen so we don't add a duplicate
					crdKey := crdNS + "/" + crdName
					seen[crdKey] = idx
					continue
				}

				// Workload not yet seen — try to find it directly
				inst, found := findOperatorWorkload(ctx, clientset, crdNS, workloadName)
				if found {
					inst.DeploymentMode = ModeOperatorCRD
					inst.ManagedBy = "opentelemetry-operator"
					inst.OperatorCRDName = crdName
					inst.ManagedWorkload = workloadName
					if crdMode != "" {
						inst.Labels = mergeLabels(inst.Labels, "opentelemetry.io/spec-mode", crdMode)
					}
					seen[workloadKey] = len(collectors)
					crdKey := crdNS + "/" + crdName
					seen[crdKey] = len(collectors)
					collectors = append(collectors, inst)
					continue
				}

				// No matching workload — standalone CRD entry
				crdKey := crdNS + "/" + crdName
				if _, exists := seen[crdKey]; !exists {
					seen[crdKey] = len(collectors)
					collectors = append(collectors, CollectorInstance{
						Name:            crdName,
						Namespace:       crdNS,
						DeploymentMode:  ModeOperatorCRD,
						Labels:          item.GetLabels(),
						ManagedBy:       "opentelemetry-operator",
						OperatorCRDName: crdName,
					})
				}
			}
		}
	}

	return collectors, nil
}

// findOperatorWorkload checks DaemonSets, Deployments, and StatefulSets for
// a workload with the given name and returns a CollectorInstance if found.
func findOperatorWorkload(ctx context.Context, clientset kubernetes.Interface, namespace, name string) (CollectorInstance, bool) {
	if ds, err := clientset.AppsV1().DaemonSets(namespace).Get(ctx, name, metav1.GetOptions{}); err == nil {
		return CollectorInstance{
			Name:      ds.Name,
			Namespace: ds.Namespace,
			Version:   extractVersion(ds.Spec.Template.Spec.Containers),
			PodCount:  int(ds.Status.NumberReady),
			Labels:    ds.Labels,
		}, true
	}
	if dep, err := clientset.AppsV1().Deployments(namespace).Get(ctx, name, metav1.GetOptions{}); err == nil {
		return CollectorInstance{
			Name:      dep.Name,
			Namespace: dep.Namespace,
			Version:   extractVersion(dep.Spec.Template.Spec.Containers),
			PodCount:  int(dep.Status.ReadyReplicas),
			Labels:    dep.Labels,
		}, true
	}
	if ss, err := clientset.AppsV1().StatefulSets(namespace).Get(ctx, name, metav1.GetOptions{}); err == nil {
		return CollectorInstance{
			Name:      ss.Name,
			Namespace: ss.Namespace,
			Version:   extractVersion(ss.Spec.Template.Spec.Containers),
			PodCount:  int(ss.Status.ReadyReplicas),
			Labels:    ss.Labels,
		}, true
	}
	return CollectorInstance{}, false
}

// extractCRDSpecMode reads .spec.mode from an unstructured CRD object.
func extractCRDSpecMode(obj map[string]interface{}) string {
	spec, ok := obj["spec"].(map[string]interface{})
	if !ok {
		return ""
	}
	mode, ok := spec["mode"].(string)
	if !ok {
		return ""
	}
	return strings.ToLower(mode)
}

// mergeLabels returns a copy of labels with the given key-value pair added.
func mergeLabels(labels map[string]string, key, value string) map[string]string {
	out := make(map[string]string, len(labels)+1)
	for k, v := range labels {
		out[k] = v
	}
	out[key] = value
	return out
}

func isCollector(meta *metav1.ObjectMeta) bool {
	labels := meta.Labels
	if labels == nil {
		return false
	}
	// Check common OTel collector labels
	if labels["app.kubernetes.io/component"] == "opentelemetry-collector" {
		return true
	}
	if labels["app.kubernetes.io/name"] == "opentelemetry-collector" {
		return true
	}
	if labels["app"] == "opentelemetry-collector" {
		return true
	}
	if labels["component"] == "otel-collector" {
		return true
	}
	// Check for OTel Operator managed labels
	if _, ok := labels["app.kubernetes.io/managed-by"]; ok {
		if labels["app.kubernetes.io/part-of"] == "opentelemetry" {
			return true
		}
	}
	return false
}

func extractVersion(containers []corev1.Container) string {
	for _, c := range containers {
		img := c.Image
		// Extract tag from image like "otel/opentelemetry-collector:0.96.0"
		if idx := strings.LastIndex(img, ":"); idx != -1 {
			return img[idx+1:]
		}
	}
	return ""
}
