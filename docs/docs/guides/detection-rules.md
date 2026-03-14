# Detection Rules

The `detect_issues` tool runs 8 runtime analyzers against captured signal data. Each analyzer inspects a specific anti-pattern and produces findings with severity, category, and remediation guidance.

## Rule Summary

| # | Rule | Category | Severity | Signal Type | Threshold |
|---|------|----------|----------|-------------|-----------|
| 1 | High Cardinality | `cardinality` | Warning | Metrics | > 100 unique label combinations |
| 2 | PII Detection | `pii` | Warning | Logs, Traces | Regex pattern match |
| 3 | Bloated Attributes | `bloated_attrs` | Warning | Logs, Traces | > 1024 bytes per value |
| 4 | Missing Resources | `missing_resource` | Warning | Logs, Traces | Required attributes absent |
| 5 | Duplicate Signals | `duplicates` | Info | Metrics | > 10 metrics with multiple data points |
| 6 | Missing Sampling | `sampling` | Info | Config | No sampling processor configured |
| 7 | Orphan Spans | `orphan_spans` | Warning | Traces | Spans with no parent and no children |
| 8 | Resource Sizing | `sizing` | Warning | All | > 10,000 data points/sec |

---

## Rule 1: High Cardinality

**Category:** `cardinality`
**Severity:** Warning
**Signal type:** Metrics

### What It Detects

Metrics with more than **100 unique label combinations**. High cardinality inflates storage costs and slows queries in backends like Prometheus and Mimir.

### How It Works

1. Groups captured metric data points by metric name
2. For each metric, builds unique keys from label key-value pairs
3. Counts distinct combinations
4. Reports any metric exceeding the threshold

### Example Finding

```
Metric http_requests_total has 342 unique label combinations (threshold: 100)
```

### Remediation

Use an attributes processor to drop or aggregate unbounded label keys (e.g., `user_id`, `request_id`, `trace_id`).

### Auto-Fix Available

Yes — `suggest_fixes` generates an attributes processor config that drops the high-cardinality keys. Risk: **medium**.

---

## Rule 2: PII Detection

**Category:** `pii`
**Severity:** Warning
**Signal type:** Logs, Traces

### What It Detects

Personally identifiable information in log attributes and span attributes.

### Patterns

| PII Type | Pattern | Notes |
|----------|---------|-------|
| Email | `[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}` | — |
| IPv4 | `\b(?:\d{1,3}\.){3}\d{1,3}\b` | Excludes 10.* and 192.168.* private ranges |
| IPv6 | `([0-9a-f]{1,4}:){7}[0-9a-f]{1,4}` | Case-insensitive |
| Phone | `\+?\d{1,4}[-.\s]?\(?\d{1,3}\)?[-.\s]?\d{1,4}[-.\s]?\d{1,9}` | Min 10 characters |

### Excluded Attributes

These attributes are skipped to avoid false positives:

- `trace_id`, `span_id`, `parent_span_id`
- `http.url`
- `net.peer.ip`, `net.host.ip`
- `k8s.pod.ip`, `k8s.node.name`

### Example Finding

```
Attribute user.email contains email pattern in 15 log records
```

### Remediation

Add a transform processor with OTTL to redact or hash PII attributes.

### Auto-Fix Available

Yes — `suggest_fixes` generates an OTTL transform that redacts matched patterns. Risk: **low**.

---

## Rule 3: Bloated Attributes

**Category:** `bloated_attrs`
**Severity:** Warning
**Signal type:** Logs, Traces

### What It Detects

Attributes with values exceeding **1024 bytes**. Common culprits include stack traces serialized as attributes, large JSON blobs, and base64 payloads.

### How It Works

Inspects every attribute value on span attributes and log record attributes. Reports any attribute where `len(value) > 1024`.

### Example Finding

```
Attribute exception.stacktrace exceeds 1024 bytes (actual: 4832 bytes) on 23 spans
```

### Remediation

Use an OTTL `truncate_all` transform to cap attribute values at 1KB.

### Auto-Fix Available

Yes — generates `truncate_all(attributes, 1024)` OTTL statement. Risk: **low**.

---

## Rule 4: Missing Resource Attributes

**Category:** `missing_resource`
**Severity:** Warning
**Signal type:** Logs, Traces

### What It Detects

Missing or invalid values for required resource attributes:

| Attribute | Invalid Values |
|-----------|---------------|
| `service.name` | Missing, empty string, `"unknown"` |
| `service.version` | Missing, empty string, `"unknown"` |
| `deployment.environment` | Missing, empty string, `"unknown"` |

### Example Finding

```
Resource attribute service.name is missing or set to "unknown" on 47 spans
```

### Remediation

Add a resource processor to set the required attributes.

### Auto-Fix Available

Yes — generates a resource processor config that sets the missing attributes. Risk: **low**.

---

## Rule 5: Duplicate Signals

**Category:** `duplicates`
**Severity:** Info
**Signal type:** Metrics

### What It Detects

Duplicate metrics arriving from multiple sources. Reports when more than **10 metrics** have multiple data points with the same name, suggesting double-collection.

### How It Works

Groups captured metrics by name and counts data points per metric. If many metrics have multiple data points, it indicates overlapping collection (e.g., both host agent and in-app SDK reporting the same metric).

### Example Finding

```
15 metrics have multiple data points suggesting duplicate collection sources
```

### Remediation

Review collection configuration for overlapping sources. Consider using a filter processor to deduplicate.

### Auto-Fix Available

Yes — generates a filter processor to exclude specific duplicate metrics. Risk: **medium**.

---

## Rule 6: Missing Sampling

**Category:** `sampling`
**Severity:** Info
**Signal type:** Config

### What It Detects

Absence of sampling processors in the collector configuration. Checks for:

- `probabilistic_sampler`
- `tail_sampling`

### How It Works

Inspects the collector's configuration for processor definitions. If neither sampling processor is found, reports an informational finding.

### Example Finding

```
No sampling processor configured — consider adding probabilistic_sampler or tail_sampling
```

### Remediation

Use `recommend_sampling` to get a tailored sampling configuration based on observed traffic.

### Auto-Fix Available

No — use `recommend_sampling` for a targeted recommendation.

---

## Rule 7: Orphan Spans

**Category:** `orphan_spans`
**Severity:** Warning
**Signal type:** Traces

### What It Detects

Spans that have no parent span AND no child spans in the observation window. These typically indicate broken context propagation between services.

### How It Works

1. Builds a parent-child relationship map from all captured spans
2. Identifies spans where `parent_span_id` is empty/zero AND no other span references it as a parent
3. Reports isolated spans

### Example Finding

```
12 orphan spans detected with no parent or children (possible broken context propagation)
```

### Remediation

Check instrumentation for missing context propagation. Common causes:

- Missing W3C trace context propagator in HTTP clients
- Async operations not propagating context
- Cross-service calls without instrumented HTTP/gRPC clients

### Auto-Fix Available

No — this requires instrumentation changes, not collector config changes.

---

## Rule 8: Resource Sizing

**Category:** `sizing`
**Severity:** Warning
**Signal type:** All (Metrics + Logs + Traces)

### What It Detects

High signal throughput exceeding **10,000 data points per second**, suggesting the collector may be under-resourced.

### How It Works

Calculates total throughput:

```
total_points = metrics_count + logs_count + spans_count
points_per_sec = total_points / capture_duration_seconds
```

### Example Finding

```
Throughput of 15,234 data points/sec exceeds 10,000 — review CPU/memory limits
```

### Remediation

Use `recommend_sizing` for specific CPU/memory recommendations based on observed throughput.

### Auto-Fix Available

No — use `recommend_sizing` for tailored resource recommendations.

---

## Finding Severity Levels

Findings are sorted by severity (most severe first):

| Severity | Meaning | Action |
|----------|---------|--------|
| `critical` | Immediate action required | Fix before proceeding |
| `error` | Significant issue | Should be addressed |
| `warning` | Potential problem | Review and decide |
| `info` | Informational | Consider for optimization |

## Panic Recovery

Each analyzer runs in isolation with panic recovery. If an analyzer panics, it produces an internal error finding instead of crashing the entire analysis:

```
internal error in analyzer [name]: [panic message]
```

This ensures that a bug in one analyzer never blocks results from the other seven.
