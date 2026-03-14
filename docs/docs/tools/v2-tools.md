# v2 Tools Reference

v2 tools provide **runtime analysis and safe mutation** capabilities for OpenTelemetry Collectors running in Kubernetes. They are gated behind the `v2.enabled` feature flag.

## Overview

| Tool | Purpose | Session Required |
|------|---------|:----------------:|
| `check_health` | Real-time pod health assessment | No |
| `start_analysis` | Create analysis session for a collector | No |
| `capture_signals` | Inject debug exporter and capture live signals | Yes |
| `detect_issues` | Run 8 analyzers on captured data | Yes |
| `suggest_fixes` | Generate fix suggestions from findings | Yes |
| `apply_fix` | Apply a fix with backup and auto-rollback | Yes |
| `recommend_sampling` | Recommend tail/probabilistic sampling | Yes |
| `recommend_sizing` | Recommend CPU/memory resource limits | Yes |
| `rollback_config` | Restore pre-mutation config from backup | Yes |
| `cleanup_debug` | Remove debug exporter and close session | Yes |

## Typical Workflow

```
start_analysis → capture_signals → detect_issues → suggest_fixes → apply_fix → check_health → cleanup_debug
```

---

## check_health

Check real-time health of a collector: pod phase, readiness, CrashLoopBackOff detection, per-pod status.

### Input

| Parameter | Type | Required | Description |
|-----------|------|:--------:|-------------|
| `name` | string | Yes | Collector name |
| `namespace` | string | Yes | Kubernetes namespace |

### Output

| Field | Type | Description |
|-------|------|-------------|
| `healthy` | boolean | Overall health status |
| `status` | string | `healthy`, `not_ready`, `crash_loop`, `not_found`, `unhealthy` |
| `pods` | integer | Number of pods |
| `details` | string | Markdown table with per-pod status |

### Example

**Request:**
```json
{
  "name": "my-collector",
  "namespace": "observability"
}
```

**Response:**
```json
{
  "healthy": true,
  "status": "healthy",
  "pods": 2,
  "details": "| Pod | Phase | Ready | Restarts | Age |\n|-----|-------|-------|----------|-----|\n| my-collector-abc12 | Running | true | 0 | 2d |\n| my-collector-def34 | Running | true | 0 | 2d |"
}
```

---

## start_analysis

Start a v2 analysis session for a collector, enabling signal capture and mutation operations.

!!! warning "Production Gate"
    Sessions in `production` environment are **always refused** with error code `PRODUCTION_REFUSED`. There is no override or force flag.

### Input

| Parameter | Type | Required | Description |
|-----------|------|:--------:|-------------|
| `collector_name` | string | Yes | Collector name |
| `namespace` | string | Yes | Kubernetes namespace |
| `environment` | string | Yes | `dev`, `staging`, or `production` |

### Output

| Field | Type | Description |
|-------|------|-------------|
| `session_id` | string | UUID for this analysis session |
| `environment` | string | Declared environment |
| `collector` | string | `namespace/name` |
| `status` | string | `ready_for_capture` |

### Example

```json
{
  "collector_name": "gateway-collector",
  "namespace": "observability",
  "environment": "staging"
}
```

### Error Codes

| Code | Cause |
|------|-------|
| `PRODUCTION_REFUSED` | Environment set to `production` |
| `CONCURRENT_SESSION` | Another session already active for this collector |
| `COLLECTOR_NOT_FOUND` | Collector does not exist |
| `GITOPS_CONFLICT` | ArgoCD/Flux detected (warning, session still created) |

---

## capture_signals

Inject a debug exporter to capture live signal samples from a collector pipeline.

### Input

| Parameter | Type | Required | Description |
|-----------|------|:--------:|-------------|
| `session_id` | string | Yes | Active session ID |
| `duration_seconds` | integer | No | Capture duration (30–120, default 60) |

### Output

| Field | Type | Description |
|-------|------|-------------|
| `status` | string | `capture_complete` |
| `duration_seconds` | integer | Actual capture duration |
| `metrics_count` | integer | Number of metric data points captured |
| `logs_count` | integer | Number of log records captured |
| `spans_count` | integer | Number of spans captured |

### Example

```json
{
  "session_id": "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
  "duration_seconds": 60
}
```

---

## detect_issues

Analyze captured signal data for runtime anti-patterns. Runs all 8 detection rules.

### Input

| Parameter | Type | Required | Description |
|-----------|------|:--------:|-------------|
| `session_id` | string | Yes | Active session ID (must have captured signals) |

### Output

| Field | Type | Description |
|-------|------|-------------|
| `findings` | array | List of `DiagnosticFinding` objects |
| `metadata.session_id` | string | Session ID |
| `metadata.analyzers_run` | string | Number of analyzers executed (always `"8"`) |
| `metadata.total_findings` | string | Total number of findings |

Each finding contains:

| Field | Type | Description |
|-------|------|-------------|
| `severity` | string | `critical`, `error`, `warning`, `info` |
| `category` | string | Rule category (e.g., `cardinality`, `pii`) |
| `summary` | string | Human-readable description |

### Example Response

```json
{
  "findings": [
    {
      "severity": "warning",
      "category": "cardinality",
      "summary": "Metric http_requests_total has 342 unique label combinations (threshold: 100)"
    },
    {
      "severity": "warning",
      "category": "pii",
      "summary": "Attribute user.email contains email pattern in 15 log records"
    }
  ],
  "metadata": {
    "session_id": "a1b2c3d4-...",
    "analyzers_run": "8",
    "total_findings": "2"
  }
}
```

---

## suggest_fixes

Generate fix suggestions for detected runtime issues.

### Input

| Parameter | Type | Required | Description |
|-----------|------|:--------:|-------------|
| `session_id` | string | Yes | Active session ID (must have findings from `detect_issues`) |

### Output

| Field | Type | Description |
|-------|------|-------------|
| `session_id` | string | Session ID |
| `suggestions` | array | List of `FixSuggestion` objects |
| `total` | integer | Number of suggestions |
| `status` | string | `suggestions_ready` or `no_findings` |

Each suggestion contains:

| Field | Type | Description |
|-------|------|-------------|
| `fix_type` | string | `ottl`, `filter`, `attribute`, `resource`, `config` |
| `description` | string | What the fix does |
| `processor_config` | string | YAML config block ready to inject |
| `pipeline_changes` | string | Pipeline modifications needed |
| `risk` | string | `low`, `medium`, `high` |

### Fix Types by Category

| Category | Fix Type | Risk | Description |
|----------|----------|------|-------------|
| `cardinality` | `attribute` | medium | Drop high-cardinality label keys |
| `pii` | `ottl` | low | Redact email/IP patterns with OTTL transform |
| `bloated_attrs` | `ottl` | low | Truncate attributes to 1KB max |
| `duplicates` | `filter` | medium | Filter processor to exclude duplicate metrics |
| `missing_resource` | `resource` | low | Set service.name, service.version, deployment.environment |

---

## apply_fix

Apply a suggested fix to the collector configuration with safety checks and backup.

### Input

| Parameter | Type | Required | Description |
|-----------|------|:--------:|-------------|
| `session_id` | string | Yes | Active session ID |
| `suggestion_index` | integer | Yes | Index of the fix suggestion to apply (0-based) |

### Output

| Field | Type | Description |
|-------|------|-------------|
| `session_id` | string | Session ID |
| `fix_type` | string | Type of fix applied |
| `fix_index` | integer | Index of applied suggestion |
| `status` | string | `fix_applied` |
| `risk` | string | Risk level of the applied fix |

### Safety Chain

When a fix is applied, the following sequence executes automatically:

1. **Backup** — Full config stored as Kubernetes annotation
2. **Apply** — Processor config merged into collector config
3. **Rollout** — Workload restart triggered
4. **Health Check** — Polls every 2s for 30s, verifying all pods are Ready
5. **Auto-Rollback** — If health check fails, config is automatically restored

---

## recommend_sampling

Analyze signal volume and recommend tail-sampling or probabilistic-sampling strategies.

### Input

| Parameter | Type | Required | Description |
|-----------|------|:--------:|-------------|
| `session_id` | string | Yes | Active session ID (must have captured signals) |

### Output

| Field | Type | Description |
|-------|------|-------------|
| `session_id` | string | Session ID |
| `trace_analysis.total_spans` | integer | Total spans captured |
| `trace_analysis.spans_per_sec` | string | Calculated throughput |
| `trace_analysis.unique_services` | integer | Distinct service.name values |
| `recommendation.strategy` | string | `none`, `tail_sampling`, `probabilistic` |
| `recommendation.config` | string | Ready-to-use YAML processor config |
| `recommendation.estimated_reduction` | string | Expected volume reduction percentage |
| `recommendation.rationale` | string | Explanation of the recommendation |

### Strategy Thresholds

| Throughput | Strategy | Description |
|------------|----------|-------------|
| < 10 spans/sec | `none` | Volume too low to benefit from sampling |
| 10–100 spans/sec | `probabilistic` | Simple percentage-based sampling |
| > 100 spans/sec | `tail_sampling` | Decision-based sampling with error/latency policies |

---

## recommend_sizing

Analyze resource usage and recommend CPU/memory limits for the collector.

### Input

| Parameter | Type | Required | Description |
|-----------|------|:--------:|-------------|
| `session_id` | string | Yes | Active session ID (must have captured signals) |

### Output

| Field | Type | Description |
|-------|------|-------------|
| `session_id` | string | Session ID |
| `observed_throughput.metrics_per_sec` | string | Metrics throughput |
| `observed_throughput.logs_per_sec` | string | Logs throughput |
| `observed_throughput.spans_per_sec` | string | Traces throughput |
| `observed_throughput.total_per_sec` | string | Combined throughput |
| `recommendation.cpu_request` | string | Recommended CPU request |
| `recommendation.cpu_limit` | string | Recommended CPU limit |
| `recommendation.mem_request` | string | Recommended memory request |
| `recommendation.mem_limit` | string | Recommended memory limit |
| `recommendation.rationale` | string | Sizing justification |

### Sizing Tiers

| Throughput | CPU Req/Limit | Memory Req/Limit | Profile |
|------------|---------------|-------------------|---------|
| < 1K signals/sec | 100m / 500m | 128Mi / 256Mi | Low |
| 1K–10K signals/sec | 250m / 1000m | 256Mi / 512Mi | Moderate |
| > 10K signals/sec | 500m / 2000m | 512Mi / 1Gi | High |

---

## rollback_config

Rollback a collector's configuration to the pre-mutation backup.

### Input

| Parameter | Type | Required | Description |
|-----------|------|:--------:|-------------|
| `session_id` | string | Yes | Active session ID |

### Output

| Field | Type | Description |
|-------|------|-------------|
| `session_id` | string | Session ID |
| `status` | string | `rollback_complete` |
| `restored_from` | string | `backup annotation` |

---

## cleanup_debug

Remove debug exporter from a collector and restore original configuration. Closes the session.

### Input

| Parameter | Type | Required | Description |
|-----------|------|:--------:|-------------|
| `session_id` | string | Yes | Active session ID |

### Output

| Field | Type | Description |
|-------|------|-------------|
| `session_id` | string | Session ID |
| `status` | string | `cleanup_complete` |
| `duration_seconds` | string | Total session duration |

!!! note
    `cleanup_debug` frees all in-memory signal data, findings, and suggestions before closing the session.

---

## Error Codes

All v2 tools may return structured errors with these codes:

| Code | Description |
|------|-------------|
| `SESSION_NOT_FOUND` | Session ID does not exist or was already closed |
| `SESSION_EXPIRED` | Session exceeded TTL (default 10 minutes) |
| `CONCURRENT_SESSION` | Another session is active for this collector |
| `PRODUCTION_REFUSED` | Analysis refused in production environment |
| `BACKUP_FAILED` | Could not create config backup before mutation |
| `ROLLBACK_FAILED` | Rollback operation failed (critical) |
| `HEALTH_CHECK_FAILED` | Post-mutation health check did not pass |
| `MUTATION_FAILED` | Config mutation could not be applied |
| `CAPTURE_FAILED` | Signal capture encountered an error |
| `GITOPS_CONFLICT` | ArgoCD or Flux manages this resource (warning) |
