# Use Cases

Real-world workflows demonstrating v2 capabilities end-to-end.

## Use Case 1: Find and Fix High Cardinality Metrics

**Scenario:** Your Prometheus storage costs are growing 15% month-over-month. You suspect high-cardinality metrics but don't know which ones.

### Step 1: Check collector health

```
> check_health(name="gateway-collector", namespace="observability")
```

```
healthy: true | pods: 3 | status: healthy
```

### Step 2: Start analysis session

```
> start_analysis(collector_name="gateway-collector", namespace="observability", environment="staging")
```

```
session_id: a1b2c3d4-e5f6-7890-abcd-ef1234567890
status: ready_for_capture
```

### Step 3: Capture live signals

```
> capture_signals(session_id="a1b2c3d4-...", duration_seconds=120)
```

```
status: capture_complete
metrics_count: 48,230 | logs_count: 1,205 | spans_count: 12,891
```

### Step 4: Detect issues

```
> detect_issues(session_id="a1b2c3d4-...")
```

```
Findings (3):
| # | Severity | Category    | Summary                                                          |
|---|----------|-------------|------------------------------------------------------------------|
| 1 | warning  | cardinality | http_requests_total has 342 unique label combos (threshold: 100) |
| 2 | warning  | cardinality | grpc_server_duration has 218 unique label combos                 |
| 3 | info     | duplicates  | 12 metrics have multiple data points                             |
```

### Step 5: Get fix suggestions

```
> suggest_fixes(session_id="a1b2c3d4-...")
```

```
Suggestions (2):
| # | Type      | Risk   | Description                                           |
|---|-----------|--------|-------------------------------------------------------|
| 0 | attribute | medium | Drop high-cardinality keys from http_requests_total   |
| 1 | attribute | medium | Drop high-cardinality keys from grpc_server_duration  |
```

### Step 6: Apply the fix

```
> apply_fix(session_id="a1b2c3d4-...", suggestion_index=0)
```

The system automatically:

1. Backs up current config to annotation
2. Adds attributes processor to drop unbounded label keys
3. Triggers rolling restart
4. Verifies all 3 pods are healthy within 30 seconds

```
status: fix_applied | risk: medium
```

### Step 7: Verify and clean up

```
> check_health(name="gateway-collector", namespace="observability")
> cleanup_debug(session_id="a1b2c3d4-...")
```

**Result:** Cardinality dropped from 342 to 15 unique combinations for `http_requests_total`. If the fix had caused issues, the health check would have auto-rolled back.

---

## Use Case 2: Detect PII Leaking Through Logs

**Scenario:** Your compliance team needs to verify that no PII (email addresses, IP addresses) is flowing through your log pipeline before a SOC 2 audit.

### Step 1: Start analysis on the log collector

```
> start_analysis(collector_name="log-collector", namespace="logging", environment="staging")
```

### Step 2: Capture a representative sample

```
> capture_signals(session_id="<id>", duration_seconds=120)
```

Use 120 seconds to capture a broad sample of log patterns.

### Step 3: Run detection

```
> detect_issues(session_id="<id>")
```

```
Findings (2):
| # | Severity | Category | Summary                                               |
|---|----------|----------|-------------------------------------------------------|
| 1 | warning  | pii      | Attribute user.email contains email pattern in 87 logs|
| 2 | warning  | pii      | Attribute client.ip contains IPv4 pattern in 203 logs |
```

### Step 4: Get redaction fixes

```
> suggest_fixes(session_id="<id>")
```

```
Suggestions (2):
| # | Type | Risk | Description                                    |
|---|------|------|------------------------------------------------|
| 0 | ottl | low  | OTTL transform to redact email patterns        |
| 1 | ottl | low  | OTTL transform to delete client.ip attribute   |
```

### Step 5: Apply the OTTL redaction

```
> apply_fix(session_id="<id>", suggestion_index=0)
> apply_fix(session_id="<id>", suggestion_index=1)
```

Both fixes are low-risk OTTL transforms. After each apply, health is verified automatically.

### Step 6: Re-capture and verify

Start a new session and re-run `capture_signals` → `detect_issues` to confirm PII is no longer present.

```
> cleanup_debug(session_id="<id>")
```

**Result:** PII redaction processors are in place. Capture a second sample to confirm zero PII findings for your audit evidence.

---

## Use Case 3: Right-Size Your Collector Resources

**Scenario:** Your collectors are OOMKilled periodically but you don't know the actual throughput to set appropriate resource limits.

### Step 1: Start analysis

```
> start_analysis(collector_name="metrics-collector", namespace="monitoring", environment="dev")
```

### Step 2: Capture during peak traffic

```
> capture_signals(session_id="<id>", duration_seconds=120)
```

Time this during your peak traffic window for accurate measurements.

### Step 3: Get sizing recommendation

```
> recommend_sizing(session_id="<id>")
```

```
Observed Throughput:
| Signal  | Rate         |
|---------|-------------|
| Metrics | 3,240/sec   |
| Logs    | 1,890/sec   |
| Spans   | 2,100/sec   |
| Total   | 7,230/sec   |

Recommendation (Moderate tier):
| Resource    | Value  |
|-------------|--------|
| CPU request | 250m   |
| CPU limit   | 1000m  |
| Mem request | 256Mi  |
| Mem limit   | 512Mi  |

Rationale: Total throughput 7,230 signals/sec falls in moderate tier (1K-10K).
Recommend 250m/1000m CPU and 256Mi/512Mi memory.
```

### Step 4: Also check for sampling opportunities

```
> recommend_sampling(session_id="<id>")
```

```
Trace Analysis:
| Metric          | Value     |
|-----------------|-----------|
| Total spans     | 252,000   |
| Spans/sec       | 2,100     |
| Unique services | 12        |

Recommendation:
| Setting             | Value                    |
|---------------------|--------------------------|
| Strategy            | tail_sampling            |
| Estimated reduction | ~60%                     |
| Rationale           | High span volume benefits from tail sampling |
```

### Step 5: Clean up

```
> cleanup_debug(session_id="<id>")
```

**Result:** Update your Helm values or resource manifest with the recommended CPU/memory limits. Optionally apply the tail sampling config to reduce volume before it hits your backend.

---

## Use Case 4: Set Up Tail Sampling from Observed Trace Patterns

**Scenario:** You want to implement tail sampling but don't know what policies to set. You want data-driven sampling rules based on actual traffic patterns.

### Step 1: Start analysis on the trace pipeline

```
> start_analysis(collector_name="trace-gateway", namespace="observability", environment="staging")
```

### Step 2: Capture trace data

```
> capture_signals(session_id="<id>", duration_seconds=120)
```

### Step 3: Get sampling recommendation

```
> recommend_sampling(session_id="<id>")
```

The tool analyzes span volume, service distribution, and error rates, then generates a ready-to-use config:

```yaml
# Recommended tail_sampling processor config
tail_sampling:
  decision_wait: 10s
  num_traces: 100
  expected_new_traces_per_sec: 100
  policies:
    - name: errors
      type: status_code
      status_code:
        status_codes:
          - ERROR
    - name: slow-traces
      type: latency
      latency:
        threshold_ms: 1000
    - name: probabilistic-fallback
      type: probabilistic
      probabilistic:
        sampling_percentage: 10
```

### Step 4: Detect additional issues while you're here

```
> detect_issues(session_id="<id>")
```

```
Findings (2):
| # | Severity | Category     | Summary                                              |
|---|----------|-------------|------------------------------------------------------|
| 1 | warning  | orphan_spans | 34 orphan spans with no parent or children           |
| 2 | warning  | sizing       | Throughput 15,234 points/sec exceeds 10,000 threshold|
```

The orphan spans finding reveals broken context propagation — a good signal to fix instrumentation before investing in sampling.

### Step 5: Clean up

```
> cleanup_debug(session_id="<id>")
```

**Result:** You have a data-driven tail sampling config that keeps all errors and slow traces while sampling 10% of normal traffic. Plus, you've identified orphan spans that need instrumentation fixes.

---

## Workflow Summary

All use cases follow the same core loop:

```
start_analysis → capture_signals → [detect_issues / recommend_*] → [suggest_fixes → apply_fix] → cleanup_debug
```

| Step | Tool | Purpose |
|------|------|---------|
| 1 | `start_analysis` | Create session, declare environment |
| 2 | `capture_signals` | Inject debug exporter, collect data |
| 3a | `detect_issues` | Find anti-patterns |
| 3b | `recommend_sampling` | Get sampling config |
| 3c | `recommend_sizing` | Get resource limits |
| 4 | `suggest_fixes` | Generate fix configs |
| 5 | `apply_fix` | Apply with safety chain |
| 6 | `check_health` | Verify health post-fix |
| 7 | `cleanup_debug` | Remove debug exporter, close session |
| — | `rollback_config` | Emergency: restore pre-mutation config |
