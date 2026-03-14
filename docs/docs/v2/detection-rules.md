# Detection Rules Reference

v2 includes 8 runtime detection rules that analyze captured signal data for anti-patterns.

## Rules

| # | Rule | Category | Severity | Threshold | Fix Available |
|---|------|----------|----------|-----------|---------------|
| 1 | High Cardinality | `cardinality` | warning | >100 unique label combinations | Yes (attributes processor) |
| 2 | PII Detection | `pii` | warning | Regex match (email, IP, phone, credit card) | Yes (OTTL redaction) |
| 3 | Orphan Spans | `orphan_spans` | warning | No parent AND no children | No |
| 4 | Bloated Attributes | `bloated_attrs` | warning | Attribute value >1KB | Yes (OTTL truncate) |
| 5 | Missing Resources | `missing_resource` | warning | service.name missing/unknown | Yes (resource processor) |
| 6 | Duplicate Signals | `duplicates` | info | Same metric name, multiple sources | Yes (filter processor) |
| 7 | Missing Sampling | `sampling` | info | No sampling processor configured | No (recommendation only) |
| 8 | Resource Sizing | `sizing` | warning | >10k data points/sec | No (recommendation only) |

## Rule Details

### 1. High Cardinality

Detects metrics with more than 100 unique label value combinations in the capture window. Reports the metric name and high-cardinality label keys.

**Fix**: Attributes processor to drop high-cardinality labels.

### 2. PII Detection

Detects PII patterns using regex matching:
- Email: RFC 5322 simplified regex
- IPv4/IPv6: Standard IP patterns (excludes private ranges)
- Phone: International format
- Credit card: 13-19 digit sequences with Luhn validation

**False positive exclusions**: trace_id, span_id, http.url, net.peer.ip, k8s.pod.ip, and other OTel semantic convention keys.

### 3. Orphan Spans

Detects spans with no parent span AND no child spans in the observation window. Indicates broken context propagation.

### 4. Bloated Attributes

Detects attribute values exceeding 1KB. Reports the attribute key and approximate size.

### 5. Missing Resource Attributes

Checks for `service.name`, `service.version`, and `deployment.environment`. Flags missing, empty, or "unknown" values.

### 6. Duplicate Signals

Detects metrics with identical names from multiple sources, suggesting redundant collection.

### 7. Missing Sampling

Flags when no `probabilistic_sampler` or `tail_sampling` processor is configured. Informational — may be intentional.

### 8. Resource Sizing

Compares observed throughput against resource limits. Flags when throughput exceeds 10,000 data points/sec.
