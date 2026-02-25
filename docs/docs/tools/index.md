# Tools Reference

otel-collector-mcp exposes 7 MCP tools that AI assistants can invoke to discover, inspect, and diagnose OpenTelemetry Collector instances running in your Kubernetes cluster.

All tools return responses wrapped in a standard envelope:

```json
{
  "cluster": "<cluster-name>",
  "namespace": "<pod-namespace>",
  "timestamp": "<RFC3339>",
  "tool": "<tool-name>",
  "data": { ... }
}
```

---

## detect_deployment_type

Auto-detect the deployment type (DaemonSet, Deployment, StatefulSet, or OTel Operator CRD) of an OTel Collector instance.

The tool checks standard Kubernetes workload types first, then falls back to querying the OTel Operator `OpenTelemetryCollector` CRD if the operator is detected in the cluster.

### Parameters

| Parameter | Type | Required | Description |
|---|---|---|---|
| `namespace` | string | Yes | Kubernetes namespace of the collector |
| `name` | string | Yes | Name of the collector workload |

### Example Invocation

```json
{
  "method": "tools/call",
  "params": {
    "name": "detect_deployment_type",
    "arguments": {
      "namespace": "observability",
      "name": "otel-collector-agent"
    }
  }
}
```

### Sample Output

```json
{
  "cluster": "production-us-east",
  "namespace": "observability",
  "timestamp": "2025-01-15T10:30:00Z",
  "tool": "detect_deployment_type",
  "data": {
    "namespace": "observability",
    "name": "otel-collector-agent",
    "deploymentMode": "DaemonSet"
  }
}
```

If the workload is not found, the response includes a diagnostic finding:

```json
{
  "cluster": "production-us-east",
  "namespace": "observability",
  "timestamp": "2025-01-15T10:30:00Z",
  "tool": "detect_deployment_type",
  "data": {
    "findings": [
      {
        "severity": "warning",
        "category": "config",
        "resource": {
          "kind": "",
          "namespace": "observability",
          "name": "nonexistent-collector"
        },
        "summary": "Collector workload not found",
        "detail": "no workload or CRD found for observability/nonexistent-collector"
      }
    ]
  }
}
```

---

## list_collectors

List all OTel Collector instances across all namespaces or a specified namespace.

This tool scans DaemonSets, Deployments, StatefulSets, and OTel Operator CRDs. It identifies collectors by checking for common OpenTelemetry labels such as `app.kubernetes.io/component=opentelemetry-collector`, `app.kubernetes.io/name=opentelemetry-collector`, `app=opentelemetry-collector`, and `component=otel-collector`.

### Parameters

| Parameter | Type | Required | Description |
|---|---|---|---|
| `namespace` | string | No | Kubernetes namespace to search. Leave empty to search all namespaces. |

### Example Invocation

```json
{
  "method": "tools/call",
  "params": {
    "name": "list_collectors",
    "arguments": {
      "namespace": ""
    }
  }
}
```

### Sample Output

```json
{
  "cluster": "production-us-east",
  "namespace": "observability",
  "timestamp": "2025-01-15T10:30:00Z",
  "tool": "list_collectors",
  "data": {
    "collectors": [
      {
        "name": "otel-collector-agent",
        "namespace": "observability",
        "deploymentMode": "DaemonSet",
        "version": "0.96.0",
        "podCount": 5,
        "labels": {
          "app.kubernetes.io/name": "opentelemetry-collector",
          "app.kubernetes.io/component": "opentelemetry-collector"
        }
      },
      {
        "name": "otel-collector-gateway",
        "namespace": "observability",
        "deploymentMode": "Deployment",
        "version": "0.96.0",
        "podCount": 2,
        "labels": {
          "app.kubernetes.io/name": "opentelemetry-collector"
        }
      },
      {
        "name": "sidecar-collector",
        "namespace": "app-team",
        "deploymentMode": "OperatorCRD",
        "podCount": 0,
        "labels": {
          "app.kubernetes.io/managed-by": "opentelemetry-operator",
          "app.kubernetes.io/part-of": "opentelemetry"
        }
      }
    ],
    "count": 3
  }
}
```

---

## get_config

Retrieve the running configuration of a detected OTel Collector instance from its ConfigMap.

The tool fetches the raw YAML configuration and attempts to parse it into a structured representation. If parsing fails, the raw content is still returned along with the parse error.

### Parameters

| Parameter | Type | Required | Description |
|---|---|---|---|
| `namespace` | string | Yes | Kubernetes namespace of the collector |
| `configmap` | string | Yes | Name of the ConfigMap containing the collector configuration |

### Example Invocation

```json
{
  "method": "tools/call",
  "params": {
    "name": "get_config",
    "arguments": {
      "namespace": "observability",
      "configmap": "otel-collector-gateway-config"
    }
  }
}
```

### Sample Output

```json
{
  "cluster": "production-us-east",
  "namespace": "observability",
  "timestamp": "2025-01-15T10:30:00Z",
  "tool": "get_config",
  "data": {
    "raw": "receivers:\n  otlp:\n    protocols:\n      grpc:\n        endpoint: 0.0.0.0:4317\n...",
    "parsed": {
      "receivers": {
        "otlp": {
          "protocols": {
            "grpc": { "endpoint": "0.0.0.0:4317" },
            "http": { "endpoint": "0.0.0.0:4318" }
          }
        }
      },
      "processors": {
        "batch": { "send_batch_size": 8192, "timeout": "200ms" },
        "memory_limiter": { "check_interval": "1s", "limit_mib": 512 }
      },
      "exporters": {
        "otlp/backend": { "endpoint": "tempo.observability:4317" }
      },
      "service": {
        "pipelines": {
          "traces": {
            "receivers": ["otlp"],
            "processors": ["memory_limiter", "batch"],
            "exporters": ["otlp/backend"]
          }
        }
      }
    }
  }
}
```

---

## parse_collector_logs

Parse OTel Collector pod logs and classify errors into categories: OTTL syntax errors, exporter failures, OOM events, receiver issues, and processor errors.

The tool fetches the most recent log lines from a specified collector pod and runs classification rules against each line. Each classified entry becomes a diagnostic finding with an appropriate severity level.

### Parameters

| Parameter | Type | Required | Description |
|---|---|---|---|
| `namespace` | string | Yes | Kubernetes namespace of the collector |
| `pod` | string | Yes | Pod name of the collector |
| `tail_lines` | integer | No | Number of log lines to fetch. Default: 1000. |

### Example Invocation

```json
{
  "method": "tools/call",
  "params": {
    "name": "parse_collector_logs",
    "arguments": {
      "namespace": "observability",
      "pod": "otel-collector-agent-xk7j2",
      "tail_lines": 500
    }
  }
}
```

### Sample Output

```json
{
  "cluster": "production-us-east",
  "namespace": "observability",
  "timestamp": "2025-01-15T10:30:00Z",
  "tool": "parse_collector_logs",
  "data": {
    "findings": [
      {
        "severity": "critical",
        "category": "runtime",
        "resource": {
          "kind": "Pod",
          "namespace": "observability",
          "name": "otel-collector-agent-xk7j2"
        },
        "summary": "Exporter failure detected: connection refused to backend",
        "detail": "2025-01-15T10:28:00Z error exporterhelper/queue_sender.go:229 Exporting failed. Dropping data. {\"error\": \"connection refused\"}"
      },
      {
        "severity": "warning",
        "category": "runtime",
        "resource": {
          "kind": "Pod",
          "namespace": "observability",
          "name": "otel-collector-agent-xk7j2"
        },
        "summary": "OTTL syntax error in transform processor",
        "detail": "2025-01-15T10:25:00Z error transform/processor.go:45 failed to parse OTTL statement"
      }
    ],
    "metadata": {
      "totalLines": "500",
      "classifiedCount": "2"
    }
  }
}
```

---

## parse_operator_logs

Parse OTel Operator pod logs to detect rejected CRDs and reconciliation failures.

The tool locates operator pods using the label `app.kubernetes.io/name=opentelemetry-operator` in the specified namespace (defaults to `opentelemetry-operator-system`), fetches their logs, and classifies entries related to CRD rejections and reconciliation errors.

### Parameters

| Parameter | Type | Required | Description |
|---|---|---|---|
| `namespace` | string | No | Namespace where the OTel Operator is running. Default: `opentelemetry-operator-system`. |

### Example Invocation

```json
{
  "method": "tools/call",
  "params": {
    "name": "parse_operator_logs",
    "arguments": {
      "namespace": "opentelemetry-operator-system"
    }
  }
}
```

### Sample Output

```json
{
  "cluster": "production-us-east",
  "namespace": "observability",
  "timestamp": "2025-01-15T10:30:00Z",
  "tool": "parse_operator_logs",
  "data": {
    "findings": [
      {
        "severity": "critical",
        "category": "operator",
        "resource": {
          "kind": "Pod",
          "namespace": "opentelemetry-operator-system",
          "name": "opentelemetry-operator-controller-manager-7b4c8f9d6-abc12"
        },
        "summary": "CRD validation failure: invalid collector spec",
        "detail": "2025-01-15T10:28:00Z error controller/reconciler.go:120 Reconciler error {\"error\": \"spec.config is invalid\"}"
      },
      {
        "severity": "warning",
        "category": "operator",
        "resource": {
          "kind": "Pod",
          "namespace": "opentelemetry-operator-system",
          "name": "opentelemetry-operator-controller-manager-7b4c8f9d6-abc12"
        },
        "summary": "Reconciliation retry detected",
        "detail": "2025-01-15T10:29:00Z warn controller/reconciler.go:85 Retrying reconciliation for observability/my-collector"
      }
    ]
  }
}
```

If the operator is not found:

```json
{
  "cluster": "production-us-east",
  "namespace": "observability",
  "timestamp": "2025-01-15T10:30:00Z",
  "tool": "parse_operator_logs",
  "data": {
    "findings": [
      {
        "severity": "warning",
        "category": "operator",
        "summary": "OTel Operator pods not found",
        "detail": "No pods found with label app.kubernetes.io/name=opentelemetry-operator in namespace opentelemetry-operator-system",
        "suggestion": "Verify the Operator is installed and the namespace is correct"
      }
    ]
  }
}
```

---

## triage_scan

Run all detection rules against a specified OTel Collector and return a prioritized issue list with severity rankings and specific remediation.

This is the most comprehensive diagnostic tool. It performs the following steps:

1. Detects the deployment mode (DaemonSet, Deployment, StatefulSet, OperatorCRD).
2. Retrieves and parses the collector configuration from the specified ConfigMap.
3. Optionally fetches pod logs if a pod name is provided.
4. Runs all 12 configuration analyzers plus log-based analyzers (if logs are available).
5. Sorts findings by severity: critical, warning, info, ok.

### Detection Rules

The triage scan runs the following analyzers:

| Analyzer | Category | Description |
|---|---|---|
| Missing batch processor | performance | Checks each pipeline for a batch processor |
| Missing memory_limiter | performance | Checks each pipeline for a memory_limiter processor |
| Hardcoded tokens | security | Detects hardcoded authentication tokens in exporter configs |
| Missing retry/queue | config | Checks exporters for retry and sending queue configuration |
| Receiver bindings | config | Validates receiver endpoint bindings |
| Tail sampling on DaemonSet | config | Flags tail_sampling processor on DaemonSet deployments |
| Invalid regex | config | Validates regex patterns in processor configurations |
| Connector misconfiguration | pipeline | Detects misconfigured connectors between pipelines |
| Resource detector conflicts | config | Finds conflicting resource detection processors |
| Cumulative-to-delta issues | config | Detects problematic cumulative-to-delta metric conversions |
| High cardinality | performance | Flags attributes likely to cause high cardinality |
| Exporter backpressure | runtime | Detects exporter backpressure from log patterns (log-based) |

### Parameters

| Parameter | Type | Required | Description |
|---|---|---|---|
| `namespace` | string | Yes | Kubernetes namespace of the collector |
| `name` | string | Yes | Name of the collector workload |
| `configmap` | string | Yes | Name of the ConfigMap containing collector configuration |
| `pod` | string | No | Pod name for log analysis. If omitted, only config-based analysis runs. |

### Example Invocation

```json
{
  "method": "tools/call",
  "params": {
    "name": "triage_scan",
    "arguments": {
      "namespace": "observability",
      "name": "otel-collector-gateway",
      "configmap": "otel-collector-gateway-config",
      "pod": "otel-collector-gateway-5d8f7c6b9-m2k4x"
    }
  }
}
```

### Sample Output

```json
{
  "cluster": "production-us-east",
  "namespace": "observability",
  "timestamp": "2025-01-15T10:31:00Z",
  "tool": "triage_scan",
  "data": {
    "findings": [
      {
        "severity": "critical",
        "category": "runtime",
        "resource": {
          "kind": "Pod",
          "namespace": "observability",
          "name": "otel-collector-gateway-5d8f7c6b9-m2k4x"
        },
        "summary": "Exporter backpressure detected: queue full, dropping data",
        "detail": "Log line indicates the exporter sending queue is full and data is being dropped."
      },
      {
        "severity": "warning",
        "category": "performance",
        "summary": "Pipeline \"traces\" is missing the batch processor",
        "detail": "The batch processor groups data before sending to exporters, reducing network overhead and improving throughput.",
        "suggestion": "Add the batch processor to this pipeline",
        "remediation": "processors:\n  batch:\n    send_batch_size: 8192\n    timeout: 200ms\n\nservice:\n  pipelines:\n    traces:\n      processors: [batch, memory_limiter]"
      },
      {
        "severity": "warning",
        "category": "security",
        "summary": "Hardcoded authentication token found in exporter configuration",
        "detail": "Exporter 'otlp/backend' contains a hardcoded bearer token in its headers.",
        "suggestion": "Use environment variable substitution or a Kubernetes Secret"
      },
      {
        "severity": "info",
        "category": "config",
        "summary": "Tail sampling processor detected on DaemonSet deployment",
        "detail": "Tail sampling requires all spans for a given trace to be processed by the same collector instance.",
        "suggestion": "Move tail sampling to a centralized Gateway (Deployment) and use load-balanced routing"
      }
    ],
    "metadata": {
      "deploymentMode": "Deployment",
      "configSource": "otel-collector-gateway-config"
    }
  }
}
```

---

## check_config

Run the misconfiguration detection suite against a collector's configuration without log analysis.

This tool is a lighter alternative to `triage_scan` -- it runs only the configuration-based analyzers (11 rules) and skips any log-based analysis. Use this when you want a fast configuration review without needing pod access.

### Parameters

| Parameter | Type | Required | Description |
|---|---|---|---|
| `namespace` | string | Yes | Kubernetes namespace of the collector |
| `name` | string | Yes | Name of the collector workload |
| `configmap` | string | Yes | Name of the ConfigMap containing collector configuration |

### Example Invocation

```json
{
  "method": "tools/call",
  "params": {
    "name": "check_config",
    "arguments": {
      "namespace": "observability",
      "name": "otel-collector-agent",
      "configmap": "otel-collector-agent-config"
    }
  }
}
```

### Sample Output

```json
{
  "cluster": "production-us-east",
  "namespace": "observability",
  "timestamp": "2025-01-15T10:32:00Z",
  "tool": "check_config",
  "data": {
    "findings": [
      {
        "severity": "warning",
        "category": "performance",
        "summary": "Pipeline \"logs\" is missing the batch processor",
        "detail": "The batch processor groups data before sending to exporters, reducing network overhead and improving throughput.",
        "suggestion": "Add the batch processor to this pipeline",
        "remediation": "processors:\n  batch:\n    send_batch_size: 8192\n    timeout: 200ms\n\nservice:\n  pipelines:\n    logs:\n      processors: [batch]"
      },
      {
        "severity": "warning",
        "category": "performance",
        "summary": "Pipeline \"logs\" is missing the memory_limiter processor",
        "detail": "Without memory_limiter, the collector may consume unbounded memory under load.",
        "suggestion": "Add memory_limiter as the first processor in the pipeline",
        "remediation": "processors:\n  memory_limiter:\n    check_interval: 1s\n    limit_mib: 512\n    spike_limit_mib: 128"
      }
    ],
    "metadata": {
      "deploymentMode": "DaemonSet"
    }
  }
}
```
