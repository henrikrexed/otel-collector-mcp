# Product Brief: otel-collector-mcp v2 — Dynamic Pipeline Analyzer

## Vision

Evolve otel-collector-mcp from a static config linter into a **dynamic pipeline analyzer** that temporarily instruments OTel Collectors to observe live signal data, detect runtime anti-patterns, and suggest targeted fixes — all with user approval and automatic safety rollbacks.

## Problem Statement

Static config analysis (v1) catches structural mistakes but misses **runtime issues** that only surface when real data flows:
- A metric with 50,000 unique label combinations (high cardinality) looks fine in config
- PII leaking through log bodies or span attributes is invisible in YAML
- Single orphan spans from broken instrumentation only appear at runtime
- Expensive attributes inflating storage costs can't be detected without seeing actual payloads

Engineers discover these problems weeks later in their observability backend — after damage is done to budgets and compliance.

## Target User

- **Platform/DevOps engineers** building and tuning OTel Collector pipelines in **dev/staging environments**
- Used via AI agents (HolmesGPT, kagent, Claude) or directly as an MCP server
- NOT for production — v2 mutations are explicitly blocked on production collectors

## Core Loop

```
┌─────────────────────────────────────────────────────┐
│                 SAFETY GATE                         │
│  1. Ask user: what environment? (dev/staging/prod)  │
│  2. Refuse if production                            │
└──────────────────────┬──────────────────────────────┘
                       ▼
┌─────────────────────────────────────────────────────┐
│                 BACKUP                              │
│  3. Snapshot current collector config               │
│     (ConfigMap/CR/file — stored for rollback)       │
└──────────────────────┬──────────────────────────────┘
                       ▼
┌─────────────────────────────────────────────────────┐
│                 INJECT                              │
│  4. Detect target pipeline(s) (metrics/logs/traces) │
│  5. Add debug exporter (verbosity: basic)           │
│  6. Apply config → restart collector                │
│  7. Health check — if crash → auto-rollback         │
└──────────────────────┬──────────────────────────────┘
                       ▼
┌─────────────────────────────────────────────────────┐
│                 OBSERVE                             │
│  8. Wait 30-60 seconds                             │
│  9. Capture collector stdout (kubectl logs)         │
│  10. Parse metrics, logs, traces from debug output  │
└──────────────────────┬──────────────────────────────┘
                       ▼
┌─────────────────────────────────────────────────────┐
│                 DETECT                              │
│  11. Run detection rules on captured data:          │
│      - High cardinality metric dimensions           │
│      - PII in log bodies/attributes                 │
│      - Single/orphan spans                          │
│      - Bloated/expensive attributes                 │
│      - Missing resource attributes                  │
│      - Duplicate signals                            │
│  12. Check sampling config — if missing, ask user   │
│      if intentional                                 │
│  13. Resource sizing recommendation based on        │
│      observed throughput                            │
└──────────────────────┬──────────────────────────────┘
                       ▼
┌─────────────────────────────────────────────────────┐
│                 SUGGEST                             │
│  14. Generate fixes (user must approve each):       │
│      - OTTL transforms (transform processor)        │
│      - Filter processor rules                       │
│      - Attribute processor (drop dimensions)        │
│      - Scrape config changes (Prometheus receiver)  │
│      - Tail sampling config                         │
│      - Resource limit recommendations               │
│  15. Present findings + suggested changes to user   │
└──────────────────────┬──────────────────────────────┘
                       ▼
┌─────────────────────────────────────────────────────┐
│                 APPLY (with approval)               │
│  16. User approves/rejects each suggestion          │
│  17. Apply approved changes to config               │
│  18. Restart collector                              │
│  19. Health check — if crash → auto-rollback        │
└──────────────────────┬──────────────────────────────┘
                       ▼
┌─────────────────────────────────────────────────────┐
│                 CLEANUP                             │
│  20. Remove debug exporter from pipeline            │
│  21. Final restart → verify healthy                 │
│  22. Report: what was found, what was fixed         │
└─────────────────────────────────────────────────────┘
```

## Detection Rules

### High Cardinality (metrics)
- Parse debug output for metric data points
- Count unique label value combinations per metric name
- Flag metrics exceeding threshold (e.g., >100 unique combos in 60s window)
- **Suggest**: OTTL to drop/aggregate expensive dimensions, or filter processor to drop the metric entirely

### PII Detection (logs, spans)
- Scan log bodies and span/log attributes for patterns:
  - Email addresses (regex)
  - IP addresses (v4/v6)
  - Credit card numbers (Luhn-validated)
  - Phone numbers
  - Common name patterns (configurable)
- **Suggest**: OTTL transform to redact/hash matching fields

### Single/Orphan Spans (traces)
- Identify spans with no parent AND no children in the observation window
- Flag spans that always appear alone (likely broken instrumentation)
- **Suggest**: Documentation of the issue (instrumentation fix needed upstream — not a collector fix)

### Bloated Attributes
- Detect attributes with values exceeding size thresholds (e.g., >1KB)
- Detect attributes with extremely high unique value counts
- **Suggest**: OTTL to truncate, drop, or hash large attributes

### Missing Resource Attributes
- Check for missing `service.name`, `service.version`, `deployment.environment`
- Flag `service.name=unknown` or empty values
- **Suggest**: resource processor to add defaults, or flag for upstream fix

### Duplicate Signals
- Detect identical metric names from different sources
- Detect semantically equivalent metrics (same data, different names)
- **Suggest**: filter processor to deduplicate

### Sampling Check
- If no sampling processor configured (probabilistic or tail), prompt user: "No sampling configured — sending 100% of traces. Is this intentional?"
- If not intentional: suggest tail sampling config based on observed trace patterns (error-biased, latency-biased, or probabilistic)

### Resource Sizing
- Measure throughput during observation window (data points/sec, spans/sec, log records/sec)
- Compare against collector resource limits (CPU/memory requests/limits)
- **Suggest**: adjusted resource values based on observed load + headroom

## Safety Model

### Environment Validation
- Ask the user explicitly what environment the collector is in (dev/staging/production)
- If production: refuse with explanation
- No heuristic guessing — the user decides

### Config Backup
- Before ANY mutation, snapshot the full collector config
- Store as a ConfigMap annotation, local file, or in-memory (depending on deployment type)
- Backup is used for auto-rollback and manual rollback

### Health Checks
- After every config change + restart:
  - Wait for pod readiness (readiness probe or container status)
  - Check for CrashLoopBackOff within 30s
  - Verify collector is processing data (not just running but healthy)
- If unhealthy: **automatic rollback** to backup config

### Rollback
- Available at any point during the workflow
- Explicit tool: `rollback_config` — restores backup and restarts
- Auto-triggered on crash detection

### Approval Gate
- Every suggested fix is presented to the user
- User can approve, reject, or modify each suggestion
- No changes applied without explicit approval

## v1 Tools (kept as-is)

| Tool | Description |
|------|-------------|
| `triage_scan` | Run all 12 static analyzers |
| `detect_deployment_type` | Identify collector deployment type |
| `list_collectors` | Discover all collector instances |
| `get_config` | Retrieve collector configuration |
| `parse_collector_logs` | Analyze collector logs |
| `parse_operator_logs` | Check OTel Operator logs |
| `check_config` | Static misconfiguration detection |

## v2 Tools (new)

| Tool | Description |
|------|-------------|
| `start_analysis` | Safety gate + backup + inject debug exporter |
| `capture_signals` | Observe debug output for 30-60s, parse signals |
| `detect_issues` | Run all detection rules on captured data |
| `suggest_fixes` | Generate OTTL/filter/attribute fixes for detected issues |
| `apply_fix` | Apply a user-approved fix to the config |
| `rollback_config` | Restore backup config and restart |
| `cleanup_debug` | Remove debug exporter, restore clean pipeline |
| `check_health` | Verify collector is running and processing |
| `recommend_sampling` | Analyze traces and suggest sampling config |
| `recommend_sizing` | Estimate resource needs from observed throughput |

## Success Metrics

- Detects high cardinality issues before they hit production backend
- Catches PII leaks in dev before compliance violations
- Reduces collector tuning time from hours to minutes
- Zero production incidents caused by the tool (safety model)
- Generated OTTL/configs are valid and apply cleanly

## Out of Scope (v2)

- Production environment mutations
- Auto-applying fixes without user approval
- Modifying application instrumentation (only collector config)
- Backend-specific optimizations (Dynatrace/Grafana/etc. — stays backend-agnostic)
- Collector version migration (keep in v1 scope)
