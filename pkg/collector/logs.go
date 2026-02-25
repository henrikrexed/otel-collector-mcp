package collector

import (
	"bufio"
	"context"
	"fmt"
	"log/slog"
	"strings"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// LogCategory classifies a collector log entry.
type LogCategory string

const (
	LogCategoryOTTLSyntax     LogCategory = "ottl_syntax_error"
	LogCategoryExporterFail   LogCategory = "exporter_failure"
	LogCategoryOOM            LogCategory = "oom_event"
	LogCategoryReceiverIssue  LogCategory = "receiver_issue"
	LogCategoryProcessorError LogCategory = "processor_error"
	LogCategoryOperatorCRD    LogCategory = "operator_crd_rejection"
	LogCategoryReconciliation LogCategory = "reconciliation_failure"
	LogCategoryOther          LogCategory = "other"
)

// ClassifiedLog represents a log entry with its classification.
type ClassifiedLog struct {
	Category LogCategory `json:"category"`
	Line     string      `json:"line"`
	Message  string      `json:"message"`
}

// DefaultTailLines is the default number of log lines to fetch.
const DefaultTailLines int64 = 1000

// FetchPodLogs retrieves recent log lines from a pod.
func FetchPodLogs(ctx context.Context, clientset kubernetes.Interface, namespace, podName string, tailLines int64) ([]string, error) {
	opts := &corev1.PodLogOptions{
		TailLines: &tailLines,
	}

	req := clientset.CoreV1().Pods(namespace).GetLogs(podName, opts)
	stream, err := req.Stream(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to stream logs for %s/%s: %w", namespace, podName, err)
	}
	defer stream.Close()

	var lines []string
	scanner := bufio.NewScanner(stream)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	if err := scanner.Err(); err != nil {
		return lines, fmt.Errorf("error reading log stream: %w", err)
	}

	return lines, nil
}

// FindPodsByLabel finds pods matching a label selector in a namespace.
func FindPodsByLabel(ctx context.Context, clientset kubernetes.Interface, namespace, labelSelector string) ([]string, error) {
	pods, err := clientset.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{
		LabelSelector: labelSelector,
	})
	if err != nil {
		return nil, err
	}
	var names []string
	for _, p := range pods.Items {
		names = append(names, p.Name)
	}
	return names, nil
}

// ClassifyCollectorLogs categorizes collector log lines by error type.
func ClassifyCollectorLogs(lines []string) []ClassifiedLog {
	var classified []ClassifiedLog

	for _, line := range lines {
		lower := strings.ToLower(line)

		var cat LogCategory
		var msg string

		switch {
		case strings.Contains(lower, "ottl") && (strings.Contains(lower, "error") || strings.Contains(lower, "parse")):
			cat = LogCategoryOTTLSyntax
			msg = "OTTL syntax or parse error detected"
		case strings.Contains(lower, "exporter") && (strings.Contains(lower, "error") || strings.Contains(lower, "failed") || strings.Contains(lower, "dropping")):
			cat = LogCategoryExporterFail
			msg = "Exporter failure or data loss"
		case strings.Contains(lower, "oom") || strings.Contains(lower, "out of memory") || strings.Contains(lower, "memory limit"):
			cat = LogCategoryOOM
			msg = "Out of memory event"
		case strings.Contains(lower, "receiver") && (strings.Contains(lower, "error") || strings.Contains(lower, "failed")):
			cat = LogCategoryReceiverIssue
			msg = "Receiver error"
		case strings.Contains(lower, "processor") && (strings.Contains(lower, "error") || strings.Contains(lower, "failed")):
			cat = LogCategoryProcessorError
			msg = "Processor error"
		default:
			// Only classify error/warning lines
			if strings.Contains(lower, "error") || strings.Contains(lower, "warn") || strings.Contains(lower, "fatal") {
				cat = LogCategoryOther
				msg = "Unclassified error/warning"
			} else {
				continue // Skip non-error lines
			}
		}

		classified = append(classified, ClassifiedLog{
			Category: cat,
			Line:     line,
			Message:  msg,
		})
	}

	return classified
}

// ClassifyOperatorLogs categorizes OTel Operator log lines.
func ClassifyOperatorLogs(lines []string) []ClassifiedLog {
	var classified []ClassifiedLog

	for _, line := range lines {
		lower := strings.ToLower(line)

		var cat LogCategory
		var msg string

		switch {
		case strings.Contains(lower, "rejected") || (strings.Contains(lower, "validation") && strings.Contains(lower, "failed")):
			cat = LogCategoryOperatorCRD
			msg = "CRD validation or rejection error"
		case strings.Contains(lower, "reconcil") && (strings.Contains(lower, "error") || strings.Contains(lower, "failed")):
			cat = LogCategoryReconciliation
			msg = "Reconciliation failure"
		default:
			if strings.Contains(lower, "error") || strings.Contains(lower, "warn") {
				cat = LogCategoryOther
				msg = "Unclassified operator error/warning"
			} else {
				continue
			}
		}

		classified = append(classified, ClassifiedLog{
			Category: cat,
			Line:     line,
			Message:  msg,
		})
	}

	slog.Debug("classified operator logs", "total", len(lines), "classified", len(classified))
	return classified
}

// GetCollectorConfig retrieves the collector configuration from a ConfigMap or CRD.
func GetCollectorConfig(ctx context.Context, clientset kubernetes.Interface, namespace, configMapName string) ([]byte, error) {
	cm, err := clientset.CoreV1().ConfigMaps(namespace).Get(ctx, configMapName, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to get configmap %s/%s: %w", namespace, configMapName, err)
	}

	// Try common config keys
	for _, key := range []string{"relay", "config.yaml", "collector.yaml", "otel-collector-config"} {
		if data, ok := cm.Data[key]; ok {
			return []byte(data), nil
		}
	}

	// Return first available key
	for _, data := range cm.Data {
		return []byte(data), nil
	}

	return nil, fmt.Errorf("no configuration data found in configmap %s/%s", namespace, configMapName)
}
