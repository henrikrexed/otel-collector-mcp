package mutator

import (
	"context"
	"fmt"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// PodHealthStatus represents the health state of a single pod.
type PodHealthStatus string

const (
	StatusHealthy  PodHealthStatus = "Healthy"
	StatusUnhealthy PodHealthStatus = "Unhealthy"
	StatusNotReady PodHealthStatus = "NotReady"
	StatusCrashLoop PodHealthStatus = "CrashLoop"
	StatusNotFound PodHealthStatus = "NotFound"
)

// PodHealth holds health information for a single pod.
type PodHealth struct {
	Name     string          `json:"name"`
	Status   PodHealthStatus `json:"status"`
	Phase    string          `json:"phase"`
	Ready    bool            `json:"ready"`
	Restarts int32           `json:"restarts"`
	Age      time.Duration   `json:"age"`
}

// CollectorHealth holds the aggregate health of a collector's pods.
type CollectorHealth struct {
	Healthy bool            `json:"healthy"`
	Status  PodHealthStatus `json:"status"`
	Pods    []PodHealth     `json:"pods"`
}

// CheckPodHealth assesses the health status of a single pod.
func CheckPodHealth(pod *corev1.Pod) PodHealth {
	ph := PodHealth{
		Name:  pod.Name,
		Phase: string(pod.Status.Phase),
	}

	if pod.CreationTimestamp.Time.IsZero() {
		ph.Age = 0
	} else {
		ph.Age = time.Since(pod.CreationTimestamp.Time)
	}

	// Check container statuses for CrashLoopBackOff and restart counts
	for _, cs := range pod.Status.ContainerStatuses {
		ph.Restarts += cs.RestartCount
		if cs.State.Waiting != nil && cs.State.Waiting.Reason == "CrashLoopBackOff" {
			ph.Status = StatusCrashLoop
			ph.Ready = false
			return ph
		}
	}

	// Pod not running
	if pod.Status.Phase != corev1.PodRunning {
		ph.Status = StatusNotReady
		ph.Ready = false
		return ph
	}

	// Check readiness conditions
	ready := true
	for _, cond := range pod.Status.Conditions {
		if cond.Type == corev1.PodReady {
			ready = cond.Status == corev1.ConditionTrue
			break
		}
	}
	ph.Ready = ready

	if ready {
		ph.Status = StatusHealthy
	} else {
		ph.Status = StatusUnhealthy
	}

	return ph
}

// CheckCollectorHealth checks the health of all pods matching a collector's label selector.
func CheckCollectorHealth(ctx context.Context, clientset kubernetes.Interface, namespace, name string) (*CollectorHealth, error) {
	labelSelector := fmt.Sprintf("app.kubernetes.io/instance=%s", name)

	pods, err := clientset.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{
		LabelSelector: labelSelector,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list pods: %w", err)
	}

	if len(pods.Items) == 0 {
		return &CollectorHealth{
			Healthy: false,
			Status:  StatusNotFound,
			Pods:    nil,
		}, nil
	}

	health := &CollectorHealth{
		Healthy: true,
		Status:  StatusHealthy,
		Pods:    make([]PodHealth, 0, len(pods.Items)),
	}

	for i := range pods.Items {
		ph := CheckPodHealth(&pods.Items[i])
		health.Pods = append(health.Pods, ph)

		if ph.Status != StatusHealthy {
			health.Healthy = false
			// Worst status wins
			if ph.Status == StatusCrashLoop {
				health.Status = StatusCrashLoop
			} else if health.Status == StatusHealthy {
				health.Status = ph.Status
			}
		}
	}

	return health, nil
}

// WaitHealthy polls pod health at 2-second intervals until all pods are healthy
// or the context deadline is exceeded.
func WaitHealthy(ctx context.Context, clientset kubernetes.Interface, namespace, name string, timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for {
		health, err := CheckCollectorHealth(ctx, clientset, namespace, name)
		if err != nil {
			return err
		}
		if health.Healthy {
			return nil
		}

		select {
		case <-ctx.Done():
			return fmt.Errorf("timed out waiting for collector %s/%s to become healthy (last status: %s)", namespace, name, health.Status)
		case <-ticker.C:
		}
	}
}
