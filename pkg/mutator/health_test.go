package mutator

import (
	"testing"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestCheckPodHealth_Healthy(t *testing.T) {
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:              "collector-abc",
			CreationTimestamp: metav1.NewTime(time.Now().Add(-10 * time.Minute)),
		},
		Status: corev1.PodStatus{
			Phase: corev1.PodRunning,
			Conditions: []corev1.PodCondition{
				{Type: corev1.PodReady, Status: corev1.ConditionTrue},
			},
			ContainerStatuses: []corev1.ContainerStatus{
				{RestartCount: 0},
			},
		},
	}

	ph := CheckPodHealth(pod)

	if ph.Status != StatusHealthy {
		t.Errorf("expected Healthy, got %s", ph.Status)
	}
	if !ph.Ready {
		t.Error("expected Ready=true")
	}
	if ph.Restarts != 0 {
		t.Errorf("expected 0 restarts, got %d", ph.Restarts)
	}
}

func TestCheckPodHealth_CrashLoopBackOff(t *testing.T) {
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:              "collector-crash",
			CreationTimestamp: metav1.NewTime(time.Now().Add(-5 * time.Minute)),
		},
		Status: corev1.PodStatus{
			Phase: corev1.PodRunning,
			ContainerStatuses: []corev1.ContainerStatus{
				{
					RestartCount: 5,
					State: corev1.ContainerState{
						Waiting: &corev1.ContainerStateWaiting{
							Reason: "CrashLoopBackOff",
						},
					},
				},
			},
		},
	}

	ph := CheckPodHealth(pod)

	if ph.Status != StatusCrashLoop {
		t.Errorf("expected CrashLoop, got %s", ph.Status)
	}
	if ph.Ready {
		t.Error("expected Ready=false for CrashLoop")
	}
}

func TestCheckPodHealth_Pending(t *testing.T) {
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:              "collector-pending",
			CreationTimestamp: metav1.NewTime(time.Now()),
		},
		Status: corev1.PodStatus{
			Phase: corev1.PodPending,
		},
	}

	ph := CheckPodHealth(pod)

	if ph.Status != StatusNotReady {
		t.Errorf("expected NotReady, got %s", ph.Status)
	}
	if ph.Ready {
		t.Error("expected Ready=false for Pending")
	}
}

func TestCheckPodHealth_RunningNotReady(t *testing.T) {
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:              "collector-notready",
			CreationTimestamp: metav1.NewTime(time.Now().Add(-2 * time.Minute)),
		},
		Status: corev1.PodStatus{
			Phase: corev1.PodRunning,
			Conditions: []corev1.PodCondition{
				{Type: corev1.PodReady, Status: corev1.ConditionFalse},
			},
			ContainerStatuses: []corev1.ContainerStatus{
				{RestartCount: 0},
			},
		},
	}

	ph := CheckPodHealth(pod)

	if ph.Status != StatusUnhealthy {
		t.Errorf("expected Unhealthy, got %s", ph.Status)
	}
	if ph.Ready {
		t.Error("expected Ready=false for Unhealthy")
	}
}
