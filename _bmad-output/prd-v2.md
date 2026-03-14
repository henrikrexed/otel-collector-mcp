---
stepsCompleted: [step-01-init, step-02-discovery, step-02b-vision, step-02c-executive-summary, step-03-success, step-04-journeys, step-05-domain, step-06-innovation, step-07-project-type, step-08-scoping, step-09-functional, step-10-nonfunctional, step-11-polish, step-12-complete]
inputDocuments: ['product-brief-v2.md', 'product-brief-otel-collector-mcp-2026-02-25.md', 'prd.md (v1)', 'docs/index.md', 'docs/tools/index.md', 'docs/architecture/index.md']
workflowType: 'prd'
classification:
  projectType: developer_tool
  domain: observability_infrastructure
  complexity: high
  projectContext: brownfield
documentCounts:
  briefs: 2
  research: 0
  brainstorming: 0
  projectDocs: 8
---

# Product Requirements Document - otel-collector-mcp v2

**Author:** Henrik.rexed
**Date:** 2026-03-13

## Executive Summary

otel-collector-mcp v2 extends the existing MCP server from a static configuration linter into a **dynamic pipeline analyzer** capable of temporarily instrumenting live OTel Collectors in dev/staging environments to observe real signal data, detect runtime anti-patterns, and suggest targeted fixes. It targets the class of problems invisible to static config analysis: high-cardinality metric dimensions, PII leaking through log bodies, orphan spans from broken instrumentation, and bloated attributes inflating storage costs. Engineers discover these problems weeks later in their observability backend — after budget and compliance damage is done.

v2 adds 10 new MCP tools on top of the existing 7 v1 tools (which remain unchanged). The new tools implement a controlled mutation workflow: safety gate (environment check, production refusal) → config backup → debug exporter injection → live signal capture → detection rule execution → fix suggestion → user-approved fix application → cleanup. Every mutation is guarded by automatic health checks and instant rollback on failure. The tool explicitly refuses to operate on production collectors.

The primary users remain Platform/DevOps engineers, but v2 shifts the usage context from "diagnose a broken config" to "analyze a running pipeline's actual behavior." The tool is used via AI agents (HolmesGPT, kagent, Claude) or directly as an MCP server in dev/staging Kubernetes environments.

### What Makes This Special

- **Runtime visibility from config tooling:** No other MCP or collector tool bridges the gap between static config analysis and live traffic observation. v2 instruments collectors temporarily, captures real data, and runs detection rules against actual payloads — not YAML guesses.
- **Safety-first mutation model:** The write operations required for dynamic analysis (injecting debug exporters, applying fixes) are wrapped in a comprehensive safety model: explicit environment gate (user declares dev/staging), config backup before any mutation, automatic health checks after every restart, and instant rollback on crash detection. This makes controlled mutation safe enough for non-production use.
- **Detection rules that need live data:** 8 detection rules target problems only visible at runtime — high cardinality (>100 unique label combos in 60s), PII patterns (email, IP, credit card, phone), orphan spans, bloated attributes (>1KB values), missing resource attributes, duplicate signals, missing sampling config, and resource sizing recommendations based on actual throughput.
- **RBAC escalation with clear boundaries:** v1 is read-only. v2 requires write access to ConfigMaps and OTel Operator CRs — but only in the MCP server's target namespace, only with user approval per change, and only in non-production environments.

## Project Classification

- **Project Type:** Developer Tool (MCP Server)
- **Domain:** Observability Infrastructure (Kubernetes + OpenTelemetry)
- **Complexity:** High — introduces controlled write operations, safety gates, automatic rollback, PII detection, live signal parsing, and RBAC escalation on top of the existing v1 read-only architecture
- **Project Context:** Brownfield — extending a fully implemented v1 system with 7 MCP tools, 12 analyzers, Helm chart, CI/CD, and documentation site

## Success Criteria

### User Success

- Engineers detect high-cardinality metrics, PII leaks, and orphan spans in dev/staging **before** they reach production backends
- Dynamic analysis cycle (start → capture → detect → suggest) completes in under 3 minutes end-to-end
- Suggested OTTL transforms and filter configs are syntactically valid and apply cleanly without collector crashes
- Rollback restores the exact pre-analysis config within 10 seconds of detecting a health failure
- Engineers who previously tuned collector pipelines over hours can complete the same work in minutes using the analysis workflow

### Business Success

- v2 tools become the primary differentiation that drives new adoption beyond the v1 user base
- Community engagement increases — v2's detection rules attract contributors who add domain-specific patterns (e.g., PII patterns for GDPR, HIPAA-specific attribute checks)
- Positions otel-collector-mcp as the first MCP server that can safely mutate infrastructure state (not just read it), establishing a new category

### Technical Success

- Zero production incidents caused by the tool — the safety model (environment gate, backup, health check, rollback) must be airtight
- All 8 runtime detection rules produce zero false positives on known-good collector pipelines
- Health check detects CrashLoopBackOff within 30 seconds and triggers automatic rollback
- Debug exporter injection and cleanup leave no trace in the final collector config
- v1 tools continue to work identically — zero regressions from v2 additions
- RBAC escalation (write access to ConfigMaps/CRs) is scoped to the minimum required permissions

### Measurable Outcomes

| Metric | Target |
|--------|--------|
| High-cardinality detection accuracy | >95% true positive rate on metrics with >100 unique label combos |
| PII detection coverage | Catches email, IPv4/v6, credit card (Luhn), phone patterns with <5% false positive rate |
| Rollback success rate | 100% — every failed health check results in successful config restoration |
| Analysis cycle time | <3 minutes from `start_analysis` to `detect_issues` results |
| Fix application success rate | >90% of suggested fixes apply without collector crash |
| v1 tool regression rate | 0 — all existing tests pass unchanged |

## Product Scope

### MVP Strategy & Philosophy

**MVP Approach:** Problem-solving MVP — demonstrate the complete inject → observe → detect → fix → cleanup loop with at least 3 detection rules working end-to-end. The safety model (environment gate, backup, health check, rollback) must be 100% complete in MVP — there is no "partial safety."

**Resource Requirements:** Single developer (same as v1). Go backend, client-go for write operations, existing MCP SDK. No new external dependencies. The v1 codebase provides ~60% of the infrastructure (K8s client, MCP server, tool registry, telemetry, Helm chart).

### MVP Feature Set (Phase 1)

**Must-Have (10 new tools):**
1. `start_analysis` — Safety gate + environment check + config backup + debug exporter injection
2. `capture_signals` — Observe debug output for 30-60s, parse metrics/logs/traces from stdout
3. `detect_issues` — Run all 8 runtime detection rules on captured data
4. `suggest_fixes` — Generate OTTL/filter/attribute processor fixes for detected issues
5. `apply_fix` — Apply a single user-approved fix to the collector config
6. `rollback_config` — Restore backup config and restart collector
7. `cleanup_debug` — Remove debug exporter, restore clean pipeline, verify healthy
8. `check_health` — Verify collector is running, not in CrashLoopBackOff, processing data
9. `recommend_sampling` — Analyze captured traces and suggest tail sampling config
10. `recommend_sizing` — Estimate CPU/memory resource needs from observed throughput

**Must-Have (8 runtime detection rules):**
1. High cardinality metric dimensions (>100 unique combos in 60s window)
2. PII in log bodies/span attributes (email, IP, credit card, phone)
3. Single/orphan spans (no parent, no children in observation window)
4. Bloated attributes (values >1KB, extremely high unique counts)
5. Missing resource attributes (service.name, service.version, deployment.environment)
6. Duplicate signals (identical metric names from different sources)
7. Sampling check (no sampling processor configured — prompt user)
8. Resource sizing (throughput vs. resource limits comparison)

**Must-Have (safety model):**
- Environment validation (user declares dev/staging/prod, refuse prod)
- Config backup before any mutation
- Health check after every config change + restart
- Automatic rollback on crash detection
- User approval gate for every suggested fix

**Must-Have (RBAC):**
- v2 ClusterRole adds: `update`, `patch` on ConfigMaps and OpenTelemetryCollector CRs
- v1 read-only permissions remain unchanged

**All 7 v1 tools unchanged:** `triage_scan`, `detect_deployment_type`, `list_collectors`, `get_config`, `parse_collector_logs`, `parse_operator_logs`, `check_config`

**Core User Journeys Supported:**
- Alex (Cardinality): Full analysis loop — primary validation journey
- Sam (Rollback): Safety model validation — must work before shipping anything

**Must-Have Capabilities (ordered by implementation dependency):**

1. **Safety infrastructure** (prerequisite for everything):
   - Session manager (`pkg/session/`)
   - Config backup/restore (`pkg/mutator/`)
   - `check_health` tool
   - `rollback_config` tool
   - Environment gate in `start_analysis`

2. **Debug exporter lifecycle**:
   - `start_analysis` tool (inject debug exporter)
   - `cleanup_debug` tool (remove debug exporter)
   - Rollout trigger mechanism (annotation-based restart)

3. **Signal capture & parsing**:
   - `capture_signals` tool
   - Debug exporter stdout parser (`pkg/signals/`)
   - Metric, log, and trace data models

4. **Detection rules** (MVP subset — 3 minimum):
   - High cardinality (metrics) — highest user value
   - PII detection (logs/spans) — compliance differentiator
   - Missing resource attributes — simplest to implement
   - `detect_issues` tool

5. **Fix generation & application**:
   - `suggest_fixes` tool
   - `apply_fix` tool
   - OTTL transform generation for cardinality and PII fixes

6. **Recommendations** (can ship slightly after core loop):
   - `recommend_sampling` tool
   - `recommend_sizing` tool

**Deferred from MVP (Phase 1.5):**
- Remaining 5 detection rules (orphan spans, bloated attributes, duplicate signals, sampling check, resource sizing rule) — add incrementally after core loop is validated
- v2 self-instrumentation metrics — existing v1 telemetry covers tool calls; v2-specific metrics can be added after core functionality works

### Growth Features (Phase 2)

- All 8 detection rules complete
- Custom PII patterns (user-configurable regex)
- Continuous monitoring mode (re-run detection rules periodically during a dev session)
- Multi-collector topology analysis (analyze an Agent→Gateway pipeline as a connected topology)
- Fix templates library (pre-built OTTL transforms for common issues)
- Dry-run mode (show what would change without applying)
- Full v2 telemetry metrics suite

### Vision (Phase 3+)

- Production-safe read-only analysis (capture signals without injecting debug exporter — using existing zpages or metrics endpoints)
- CI/CD pipeline integration (run dynamic analysis as a pipeline gate before promoting configs to production)
- Cross-cluster analysis (compare runtime behavior across dev/staging/prod)
- Community detection rule plugin system (pluggable detection patterns)

### Risk Mitigation Strategy

**Technical Risks:**

| Risk | Likelihood | Impact | Mitigation |
|------|-----------|--------|------------|
| Debug exporter output format changes between collector versions | Medium | High — parser breaks | Pin to collector v0.90+ format, add version detection, build parser test suite against multiple collector versions |
| ConfigMap mutation races with GitOps controllers (ArgoCD, Flux) | High | Medium — GitOps reverts changes | Document that v2 analysis sessions should pause GitOps sync. Detect GitOps annotations and warn user. |
| Session state lost on MCP server restart | Medium | High — orphaned debug exporters | Store session state as ConfigMap annotations, not just in-memory. Startup recovery scans for orphaned debug exporters. |
| Go MCP SDK limitations for long-running operations (capture_signals blocks for 60s) | Low | Medium — timeout issues | Use streaming response or return immediately with a "capture in progress" status, then poll. |

**Market Risks:**
- OTel community builds similar functionality into the Operator → Differentiate on MCP integration and AI-agent-driven workflow. The Operator would need years to match the detection rule depth.
- Low adoption if engineers don't trust write operations → Ship with `v2.enabled=false` by default. Require explicit opt-in. Safety model documentation must be thorough.

**Resource Risks:**
- Single developer bottleneck → Phase the v2 MVP to ship the core loop (tools 1-7 + 3 detection rules) first. Recommendations and remaining detection rules follow.
- Scope creep into production support → Hard boundary: v2 MVP is dev/staging only. Production-safe analysis is Phase 3.

## User Journeys

### Journey 1: Alex the Observability Engineer — "Why Is My Metrics Bill So High?" (Primary — Success Path)

Alex is a mid-level observability engineer whose team just got a $12,000 monthly bill from their metrics backend. Management wants to know why costs tripled. Alex suspects high-cardinality metrics but doesn't know which ones — the collector config looks fine.

**Opening Scene:** Alex stares at a perfectly valid collector config. Nothing looks wrong. But the metrics backend is ingesting 2 million unique time series, and nobody knows which service or metric is responsible. The config has no cardinality controls — no filter processor, no attributes processor, no metricstransform. But which metrics actually need them?

**Rising Action:** Alex connects to otel-collector-mcp via their AI agent in the staging environment (which mirrors production traffic patterns). They invoke `start_analysis` targeting the metrics gateway collector. The tool asks "What environment is this collector in?" — Alex confirms "staging." The tool backs up the current config, injects a debug exporter with `verbosity: basic` into the metrics pipeline, applies the change, and waits for the collector to pass health checks.

Next, Alex runs `capture_signals` with a 60-second window. The tool captures the debug exporter's stdout, parsing metric data points and counting unique label combinations per metric name.

Then `detect_issues` runs — and it finds the culprit: `http_server_request_duration_seconds` has 47,000 unique label combinations because it includes `http.target` (full URL path with query parameters) and `user.id` as dimensions. Two other metrics have >500 unique combos each.

**Climax:** Alex runs `suggest_fixes`. The tool generates three OTTL transforms: (1) drop `http.target` and `user.id` dimensions from the high-cardinality metric using the attributes processor, (2) add a filter processor rule to aggregate the two medium-cardinality metrics, (3) add a metricstransform processor to rename and consolidate duplicate metrics it also detected. Each fix includes the complete processor config block ready to paste.

Alex approves fix #1 via `apply_fix`. The tool applies the config change, restarts the collector, runs `check_health` — collector is healthy and processing. Alex approves #2 and #3 the same way.

**Resolution:** Alex runs `cleanup_debug` to remove the debug exporter. The tool restores a clean pipeline (with the approved fixes included), verifies health, and reports: "3 issues detected, 3 fixes applied, debug exporter removed, collector healthy." The staging metrics bill drops 60% in the next billing cycle. Alex applies the same config changes to production with confidence.

### Journey 2: Jordan the Platform Engineer — "Is That PII in Our Logs?" (Primary — Compliance Edge Case)

Jordan is a senior platform engineer who just received an urgent message from the security team: an internal audit found that some application logs may contain customer email addresses and IP addresses being shipped to their log analytics backend, which violates their data handling policy.

**Opening Scene:** Jordan needs to determine the scope of the PII leak — which services, which log attributes, what types of PII — without manually reading thousands of log lines. The collector config has no redaction rules because nobody thought PII would appear in structured logs.

**Rising Action:** Jordan targets the log collector (DaemonSet) in the staging environment with `start_analysis`. After safety gate and backup, the debug exporter is injected into the logs pipeline. `capture_signals` runs for 60 seconds, capturing structured log records from the debug output.

`detect_issues` runs the PII detection rules and flags: (1) 23 log records contain email addresses in the `user.email` attribute (regex match), (2) 156 log records contain IPv4 addresses in the `client.address` attribute, (3) 4 log records contain what appears to be credit card numbers in the `payment.reference` field (Luhn-validated).

**Climax:** `suggest_fixes` generates three OTTL transform statements: (1) `replace_pattern(attributes["user.email"], "([a-zA-Z0-9._%+-]+)@([a-zA-Z0-9.-]+)", "REDACTED@\\2")` to hash the local part of emails, (2) `replace_pattern(attributes["client.address"], "\\d+\\.\\d+\\.\\d+\\.\\d+", "REDACTED")` to redact IPs, (3) `delete_key(attributes, "payment.reference")` to drop the credit card field entirely. Each fix is wrapped in a complete transform processor config block.

Jordan approves all three via `apply_fix`, each followed by automatic health checks. Then `cleanup_debug` removes the debug exporter.

**Resolution:** Jordan reports to the security team with specifics: 3 PII types found across 3 attributes, all redacted at the collector level before reaching the backend. The fix is promoted to production the same day. The audit finding is closed.

### Journey 3: Sam the SRE — "Something Broke After I Applied Fixes" (Edge Case — Rollback)

Sam is an SRE assisting a developer who applied a suggested fix from `suggest_fixes` but the collector entered CrashLoopBackOff immediately after.

**Opening Scene:** The developer ran `apply_fix` with a suggested OTTL transform, but the transform had a context mismatch (the fix targeted span attributes but was applied to a logs pipeline). The collector crashes on startup.

**Rising Action:** The `apply_fix` tool automatically ran `check_health` after the restart. Within 15 seconds, it detected the pod was in CrashLoopBackOff — the readiness probe failed and Kubernetes restarted the container twice.

**Climax:** The tool automatically triggers `rollback_config` — it restores the backup config that was snapshotted before the analysis session began, applies it to the ConfigMap/CR, and waits for the collector to restart. Within 30 seconds, the collector is healthy again on the original config.

**Resolution:** Sam sees the automatic rollback in the tool's output: "Health check failed — CrashLoopBackOff detected. Automatic rollback triggered. Backup config restored. Collector healthy." No data was lost beyond the ~30 seconds of crash time. The developer reports the failed fix to improve the suggestion engine.

### Journey 4: AI Agent (kagent/HolmesGPT) — Autonomous Analysis Workflow (Integration)

An AI agent (kagent) is configured to periodically analyze collector health in the staging environment as part of a CI/CD pipeline check before promoting config changes to production.

**Opening Scene:** A new collector config is deployed to staging via GitOps. The CI pipeline includes a step that invokes otel-collector-mcp through kagent to validate runtime behavior before promotion.

**Rising Action:** kagent invokes `start_analysis` (environment: "staging"), then `capture_signals` (60s window), then `detect_issues`. The tool returns findings in its compact text format optimized for LLM consumption.

**Climax:** kagent receives the detection results: 1 warning (missing `service.version` resource attribute on 40% of spans) and 1 info (no sampling processor configured). kagent evaluates: the missing resource attribute is a known gap already tracked in the team's backlog. The sampling note is informational — the team intentionally sends 100% of traces in staging.

**Resolution:** kagent runs `cleanup_debug`, removes the debug exporter, and reports to the CI pipeline: "2 findings (1 warning, 1 info), no critical issues, config promotion approved." The GitOps pipeline proceeds to production deployment. No human intervention needed for this routine check.

### Journey Requirements Summary

| Journey | v2 Tools Exercised | Capabilities Revealed |
|---------|-------------------|----------------------|
| Alex (Cardinality) | `start_analysis`, `capture_signals`, `detect_issues`, `suggest_fixes`, `apply_fix`, `check_health`, `cleanup_debug` | Full analysis loop, high-cardinality detection, OTTL fix generation, iterative fix application |
| Jordan (PII) | `start_analysis`, `capture_signals`, `detect_issues`, `suggest_fixes`, `apply_fix`, `cleanup_debug` | PII detection (email, IP, credit card), OTTL redaction transforms, compliance workflow |
| Sam (Rollback) | `apply_fix`, `check_health`, `rollback_config` | Automatic crash detection, automatic rollback, safety model validation |
| AI Agent (CI/CD) | `start_analysis`, `capture_signals`, `detect_issues`, `cleanup_debug` | Autonomous operation, LLM-optimized output, no-fix-needed path, CI integration |

## Domain-Specific Requirements

### OTel Collector Runtime Domain Knowledge

v2 detection rules require deep understanding of OTel Collector **runtime behavior**, beyond v1's static config knowledge:

- **Debug exporter output format:** The debug exporter (with `verbosity: basic`) writes structured telemetry data to stdout. v2 must parse this output format — metric data points with labels, log records with bodies and attributes, trace spans with attributes and parent/child relationships. The format varies between collector versions; v2 targets collector v0.90+.
- **Signal data models:** Metrics (gauge, sum, histogram, summary with data points and labels), logs (body, attributes, resource, severity), traces (spans with traceID, spanID, parentSpanID, attributes, events, links). Each has distinct parsing and analysis requirements.
- **Collector restart behavior:** When a ConfigMap or CRD spec.config changes, the collector pod must be restarted. For CRDs, the Operator handles rolling restarts. For ConfigMaps, the pod needs a delete/recreate (or annotation-triggered rollout). v2 must handle both restart mechanisms.
- **Health signal semantics:** A "healthy" collector means: pod is in Running phase, readiness probe passes, no CrashLoopBackOff in the last 30s, and the collector is actively processing data (not just idle). v2's `check_health` must distinguish all four states.

### Kubernetes Write Operations Domain

v2 introduces write operations — a fundamental shift from v1's read-only model:

- **ConfigMap mutations:** Updating the `data` field of a ConfigMap containing collector YAML. Must preserve other keys in the ConfigMap. Must handle ConfigMaps with multiple data keys (e.g., `relay` key used by Helm-deployed collectors).
- **CRD spec.config mutations:** Updating the `spec.config` field of an OpenTelemetryCollector CR. The Operator validates this field and may reject invalid configs — v2 must handle rejection gracefully.
- **Rollout triggers:** ConfigMap changes don't automatically restart pods. v2 must trigger a rollout (e.g., annotate the Deployment/DaemonSet/StatefulSet with a timestamp) or delete pods to pick up the new config. For CRDs, the Operator handles this.
- **Concurrent access:** Multiple users or agents could target the same collector simultaneously. v2's backup/rollback mechanism must handle this (at minimum: detect if config changed since backup and warn).

### Safety & Blast Radius Constraints

- **Environment gate is user-declared, not heuristic:** v2 asks the user what environment the collector is in. It does NOT attempt to guess (e.g., by reading namespace names or labels). The user is the authority.
- **Production refusal is absolute:** If the user declares "production," v2 refuses all mutation operations. There is no override, no force flag, no workaround.
- **Backup scope:** The backup includes the full ConfigMap data or CRD spec, not just the collector YAML. This ensures rollback restores the exact original state.
- **Debug exporter injection is minimal:** v2 adds only `debug` exporter with `verbosity: basic` to the target pipeline. It does not modify receivers, processors, or existing exporters. The injection is append-only to the exporters list and pipeline exporter references.
- **Cleanup is mandatory:** The `cleanup_debug` tool must remove the debug exporter even if the user abandons the analysis workflow midway. Consider a TTL-based auto-cleanup mechanism.

### RBAC Escalation Requirements

v1 RBAC (read-only):
```yaml
verbs: [get, list, watch]
resources: [daemonsets, deployments, statefulsets, configmaps, pods, pods/log]
```

v2 RBAC (adds write access):
```yaml
# New permissions for v2
- apiGroups: [""]
  resources: [configmaps]
  verbs: [update, patch]
- apiGroups: [opentelemetry.io]
  resources: [opentelemetrycollectors]
  verbs: [update, patch]
- apiGroups: [apps]
  resources: [daemonsets, deployments, statefulsets]
  verbs: [patch]  # For triggering rollouts via annotation
```

The Helm chart must support a `v2.enabled` flag that controls whether write RBAC is included. Clusters that only want v1 functionality should not need to grant write permissions.

### Self-Instrumentation (v2 Extensions)

v1 already provides OTel self-instrumentation via `pkg/telemetry/` — spans for every tool call (GenAI semantic conventions), `gen_ai.server.request.duration` histogram, `gen_ai.server.request.count` counter, `mcp.findings.total` counter, and OTLP gRPC export for all three signals. v2 extends this with metrics specific to the dynamic analysis workflow:

**v2 Span Requirements:**
- Every v2 tool call produces a span following existing GenAI/MCP semantic conventions (inherited from v1 instrumentation)
- `start_analysis` span includes child spans for: environment validation, config backup, debug exporter injection, config apply, health check
- `capture_signals` span includes attributes: `capture.duration_seconds`, `capture.metrics_count`, `capture.logs_count`, `capture.spans_count`
- `detect_issues` span includes child spans per detection rule executed
- `apply_fix` span includes child spans for: config mutation, rollout trigger, health check
- `rollback_config` span includes attributes: `rollback.trigger` (manual|automatic), `rollback.reason`

**v2 Metrics (new counters/histograms):**

| Metric | Type | Labels | Description |
|--------|------|--------|-------------|
| `mcp.analysis.duration_seconds` | histogram | `collector_name`, `namespace` | End-to-end analysis session duration |
| `mcp.analysis.sessions_total` | counter | `environment`, `outcome` (completed/rolled_back/abandoned) | Analysis sessions started |
| `mcp.capture.signals_total` | counter | `signal_type` (metrics/logs/traces) | Signal data points captured |
| `mcp.detection.hits_total` | counter | `rule`, `severity` | Detection rule matches |
| `mcp.fixes.applied_total` | counter | `fix_type` (ottl/filter/attribute/resource), `outcome` (success/rollback) | Fixes applied |
| `mcp.rollbacks.total` | counter | `trigger` (automatic/manual), `reason` | Rollback events |
| `mcp.health_checks.total` | counter | `result` (healthy/crash_loop/timeout/unhealthy) | Health check outcomes |
| `mcp.backup.active` | gauge | `collector_name`, `namespace` | Currently active config backups |

**v2 Structured Logging:**
- All v2 operations log with `trace_id`/`span_id` correlation (inherited from v1 slog→OTel bridge)
- Mutation operations (config apply, rollback) log at INFO level with before/after config diff summary
- Safety gate decisions (environment refusal) log at WARN level

### PII Detection Domain Knowledge

- **Email regex:** RFC 5322 simplified — `[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}` — targets common formats, not edge cases
- **IPv4:** `\b\d{1,3}\.\d{1,3}\.\d{1,3}\.\d{1,3}\b` with optional CIDR suffix
- **IPv6:** Abbreviated and full forms — `[0-9a-fA-F:]{2,39}` with at least 2 colons
- **Credit card:** 13-19 digit sequences validated with Luhn algorithm. Must not flag random numeric IDs.
- **Phone numbers:** International format `+\d{1,3}[\s-]?\d{3,14}` — high false positive risk, so this rule should have configurable sensitivity
- **False positive mitigation:** PII patterns must not flag: trace IDs (hex strings), span IDs, metric names, Kubernetes resource names, OTel semantic convention values (e.g., `http.url` values that contain IPs in URLs)

### Risk Mitigations

| Risk | Impact | Mitigation |
|------|--------|------------|
| Debug exporter causes collector OOM (high-throughput pipeline) | Collector crash, data loss | Set `verbosity: basic` (not `detailed`), add memory_limiter if not present, monitor memory during capture |
| Config backup lost (MCP server pod restart during analysis) | Cannot rollback | Store backup as ConfigMap annotation on the target ConfigMap, not in MCP server memory |
| Concurrent analysis sessions on same collector | Config corruption | Track active sessions per collector, reject concurrent analysis with clear error |
| OTTL fix has syntax error | Collector crash after apply | Validate OTTL syntax before suggesting (parse with OTel collector config validator if available), automatic rollback catches remaining failures |
| User abandons session without cleanup | Debug exporter left in pipeline permanently | Implement session TTL — auto-cleanup after 10 minutes of inactivity |

## Innovation & Novel Patterns

### Detected Innovation Areas

**1. MCP Server with Controlled Infrastructure Mutation**
Every MCP server in the current ecosystem is read-only — they query systems and return information. otel-collector-mcp v2 is the first to safely mutate infrastructure state (inject debug exporters, apply config fixes, trigger rollouts) through the MCP protocol with a comprehensive safety model. This establishes a new pattern: MCP servers as controlled actuators, not just sensors.

**2. Temporary Instrumentation as a Diagnostic Technique**
The "inject → observe → analyze → cleanup" pattern is novel in the observability tooling space. Existing tools either require permanent instrumentation changes or analyze static configs. v2 treats instrumentation as ephemeral — add it just long enough to capture data, then remove it. This is analogous to how debuggers attach and detach from processes, applied to observability pipelines.

**3. Runtime Anti-Pattern Detection via Live Signal Parsing**
Static linters exist (v1 is one). APM backends can detect cardinality issues after ingestion. But no tool sits at the collector level, captures live signal data in-flight, and detects anti-patterns before data reaches the backend. v2 occupies this unique position in the data pipeline.

### Validation Approach

- **Safety model validation:** Test the full mutation lifecycle (inject → capture → detect → fix → rollback → cleanup) against collectors with known-bad configs in a test cluster. Verify zero-state-leakage — the collector must be identical before and after analysis.
- **Detection rule accuracy:** Build a corpus of collector pipelines with known runtime issues (high-cardinality metrics, PII in logs, orphan spans). Run v2 detection rules and measure true positive / false positive rates against ground truth.
- **Rollback reliability:** Chaos-test the rollback mechanism — inject invalid configs, crash the collector, kill the MCP server mid-analysis — and verify backup restoration succeeds in every case.

### Risk Mitigation

- **Risk:** Safety model has gaps → **Mitigation:** Start with the most conservative approach (user declares environment, no heuristics), add defense-in-depth (automatic health checks, TTL-based cleanup), and chaos-test extensively before release.
- **Risk:** The "temporary instrumentation" concept confuses users → **Mitigation:** Clear tool descriptions explain what each tool does to the collector. The AI agent mediates the workflow and explains each step.
- **Risk:** Other MCP servers copy the mutation pattern without safety guarantees → **Mitigation:** Establish the safety model as the reference implementation. Document it thoroughly so copies at least follow the same pattern.

## MCP Server Specific Requirements

### Project-Type Overview

otel-collector-mcp is an MCP server — it exposes capabilities as MCP tools invoked by AI assistants or automation agents. It has no UI, no CLI, and no SDK. All interaction happens through the MCP protocol (Streamable HTTP transport at `/mcp`). v2 adds 10 new tools to the existing 7, all following the same `StandardResponse` envelope and compact text output format established in v1.

### Technical Architecture Considerations

**Transport & Protocol:**
- MCP Streamable HTTP (v1 used SSE, migrated to Streamable HTTP via go-sdk v1.3.1)
- Endpoint: `POST /mcp` — handles `tools/list` and `tools/call`
- Health: `/healthz` (liveness) and `/readyz` (readiness, including K8s client check)

**v2 State Management:**
v1 is stateless — every tool call is independent. v2 introduces **session state** for the analysis workflow:
- Active analysis sessions (which collector is being analyzed, backup config, debug exporter injected)
- Session state must survive across multiple tool calls (`start_analysis` → `capture_signals` → `detect_issues` → etc.)
- State stored in-memory with ConfigMap annotation backup for rollback durability
- Session cleanup on TTL expiry (10 minutes of inactivity)

**v2 Tool Input/Output Schemas:**

#### 1. `start_analysis`
```json
Input: {
  "namespace": "string (required) — collector namespace",
  "name": "string (required) — collector workload name",
  "environment": "string (required) — dev|staging|production",
  "pipelines": "string[] (optional) — target pipeline names, default: all"
}
Output: {
  "session_id": "string — unique analysis session identifier",
  "environment": "string — confirmed environment",
  "backup_id": "string — config backup reference",
  "collector": "object — {name, namespace, deploymentMode}",
  "injected_pipelines": "string[] — pipelines with debug exporter added",
  "status": "string — ready_for_capture"
}
Error: refuses if environment=production
```

#### 2. `capture_signals`
```json
Input: {
  "session_id": "string (required) — from start_analysis",
  "duration_seconds": "integer (optional, default: 60, range: 30-120)"
}
Output: {
  "session_id": "string",
  "duration_seconds": "integer — actual capture duration",
  "signals": {
    "metrics": { "data_points": "integer", "unique_metric_names": "integer" },
    "logs": { "records": "integer" },
    "traces": { "spans": "integer", "unique_trace_ids": "integer" }
  },
  "status": "string — capture_complete"
}
```

#### 3. `detect_issues`
```json
Input: {
  "session_id": "string (required) — from start_analysis"
}
Output: {
  "session_id": "string",
  "findings": [
    {
      "rule": "string — detection rule name",
      "severity": "string — critical|warning|info",
      "category": "string — cardinality|pii|orphan_spans|bloated_attrs|missing_resource|duplicates|sampling|sizing",
      "summary": "string",
      "detail": "string",
      "affected_signals": "string[] — metric names, attribute keys, span names",
      "fix_available": "boolean"
    }
  ],
  "rules_executed": "integer",
  "status": "string — detection_complete"
}
```

#### 4. `suggest_fixes`
```json
Input: {
  "session_id": "string (required)",
  "finding_index": "integer (optional) — specific finding, default: all"
}
Output: {
  "session_id": "string",
  "suggestions": [
    {
      "finding_index": "integer",
      "fix_type": "string — ottl|filter|attribute|resource|config",
      "description": "string — what this fix does",
      "processor_config": "string — complete YAML config block",
      "pipeline_changes": "string — where to add the processor",
      "risk": "string — low|medium|high"
    }
  ]
}
```

#### 5. `apply_fix`
```json
Input: {
  "session_id": "string (required)",
  "suggestion_index": "integer (required) — which suggestion to apply"
}
Output: {
  "session_id": "string",
  "applied": "object — {fix_type, description}",
  "health_check": "object — {status, pod_phase, ready, restarts}",
  "status": "string — fix_applied|rolled_back"
}
Error: auto-rollback if health check fails
```

#### 6. `rollback_config`
```json
Input: {
  "session_id": "string (required)"
}
Output: {
  "session_id": "string",
  "restored_from": "string — backup reference",
  "health_check": "object — {status, pod_phase, ready}",
  "status": "string — rollback_complete"
}
```

#### 7. `cleanup_debug`
```json
Input: {
  "session_id": "string (required)"
}
Output: {
  "session_id": "string",
  "removed_from_pipelines": "string[] — pipelines cleaned",
  "health_check": "object — {status, pod_phase, ready}",
  "session_summary": {
    "findings_count": "integer",
    "fixes_applied": "integer",
    "rollbacks": "integer",
    "duration_seconds": "integer"
  },
  "status": "string — cleanup_complete"
}
```

#### 8. `check_health`
```json
Input: {
  "namespace": "string (required)",
  "name": "string (required)"
}
Output: {
  "collector": "object — {name, namespace, deploymentMode}",
  "pods": [
    {
      "name": "string",
      "phase": "string — Running|Pending|Failed|CrashLoopBackOff",
      "ready": "boolean",
      "restarts": "integer",
      "age_seconds": "integer"
    }
  ],
  "healthy": "boolean",
  "status": "string — healthy|unhealthy|crash_loop|not_found"
}
```

#### 9. `recommend_sampling`
```json
Input: {
  "session_id": "string (required)"
}
Output: {
  "session_id": "string",
  "trace_analysis": {
    "total_spans": "integer",
    "error_spans": "integer",
    "error_rate": "float",
    "p99_duration_ms": "float",
    "unique_services": "integer"
  },
  "recommendation": {
    "strategy": "string — tail_sampling|probabilistic|hybrid",
    "config": "string — complete YAML config block",
    "estimated_reduction": "string — e.g. 70% volume reduction",
    "rationale": "string"
  }
}
```

#### 10. `recommend_sizing`
```json
Input: {
  "session_id": "string (required)"
}
Output: {
  "session_id": "string",
  "observed_throughput": {
    "metrics_per_second": "float",
    "logs_per_second": "float",
    "spans_per_second": "float"
  },
  "current_resources": {
    "cpu_request": "string", "cpu_limit": "string",
    "memory_request": "string", "memory_limit": "string"
  },
  "recommendation": {
    "cpu_request": "string", "cpu_limit": "string",
    "memory_request": "string", "memory_limit": "string",
    "rationale": "string"
  }
}
```

### Migration Guide (v1 → v2)

- **v1 tools unchanged:** All 7 v1 tools retain identical input schemas, output formats, and behavior. No client changes needed.
- **RBAC upgrade:** Clusters upgrading to v2 must update the ClusterRole to add `update`/`patch` on ConfigMaps and OpenTelemetryCollector CRs. The Helm chart `v2.enabled=true` flag controls this.
- **New Helm values:** `v2.enabled` (default: false), `v2.sessionTTL` (default: 10m), `v2.maxConcurrentSessions` (default: 5).
- **Backward compatibility:** MCP clients that only use v1 tools see no changes. v2 tools simply appear in `tools/list` when `v2.enabled=true`.

### Implementation Considerations

- **Session manager:** New `pkg/session/` package to manage analysis session state (create, get, expire, cleanup). Thread-safe map keyed by session ID.
- **Config mutator:** New `pkg/mutator/` package to handle ConfigMap and CRD mutations, backup/restore, and rollout triggers. Separated from v1's read-only `pkg/collector/` package.
- **Signal parser:** New `pkg/signals/` package to parse debug exporter stdout into structured metric/log/trace data models.
- **Runtime analyzers:** New `pkg/analysis/runtime/` subdirectory for v2 detection rules, separate from v1's `pkg/analysis/` static analyzers.
- **Fix generator:** New `pkg/fixes/` package to generate OTTL transforms, filter rules, and processor configs from detection findings.

## Functional Requirements

### v1 Tool Preservation

- FR1: All 7 v1 MCP tools (`triage_scan`, `detect_deployment_type`, `list_collectors`, `get_config`, `parse_collector_logs`, `parse_operator_logs`, `check_config`) retain identical input schemas, output formats, and behavior with zero regressions
- FR2: v1 tools remain available regardless of whether v2 features are enabled

### Safety & Environment Control

- FR3: MCP server can ask the user to declare the environment type (dev, staging, production) before any mutation operation
- FR4: MCP server refuses all mutation operations when the user declares "production" — no override, no force flag
- FR5: MCP server creates a complete config backup (full ConfigMap data or CRD spec) before any mutation
- FR6: MCP server stores config backups durably (ConfigMap annotation) so they survive MCP server pod restarts
- FR7: MCP server can restore a backed-up config and trigger a collector restart (rollback)
- FR8: MCP server automatically triggers rollback when a health check detects collector failure after a mutation
- FR9: MCP server can detect concurrent analysis sessions targeting the same collector and reject with a clear error

### Collector Health Monitoring

- FR10: MCP server can check whether a collector is healthy (pod Running, readiness probe passing, no CrashLoopBackOff, processing data)
- FR11: MCP server can detect CrashLoopBackOff within 30 seconds of a collector restart
- FR12: MCP server runs a health check automatically after every config mutation and restart
- FR13: MCP server can report per-pod health status (phase, ready state, restart count, age) for multi-pod collectors

### Dynamic Analysis Workflow

- FR14: MCP server can inject a debug exporter (`verbosity: basic`) into specified collector pipelines without modifying existing receivers, processors, or exporters
- FR15: MCP server can apply a modified collector config to a ConfigMap and trigger a pod rollout
- FR16: MCP server can apply a modified collector config to an OpenTelemetryCollector CR (Operator handles rollout)
- FR17: MCP server can capture debug exporter output from collector pod stdout for a configurable duration (30-120 seconds)
- FR18: MCP server can parse captured debug output into structured metric data points (with labels), log records (with body and attributes), and trace spans (with attributes and parent/child relationships)
- FR19: MCP server can remove the debug exporter from collector pipelines and restore a clean config (with any approved fixes preserved)
- FR20: MCP server can auto-cleanup debug exporters after a configurable session TTL (default 10 minutes) if the user abandons the workflow
- FR21: MCP server can recover orphaned debug exporters on startup (detect sessions that were active when the MCP server restarted)

### Runtime Detection Rules

- FR22: MCP server can detect high-cardinality metric dimensions by counting unique label value combinations per metric name and flagging metrics exceeding a threshold (default: >100 unique combos in the capture window)
- FR23: MCP server can detect PII patterns in log bodies and span/log attributes — email addresses (regex), IPv4/v6 addresses, credit card numbers (Luhn-validated), and phone numbers (international format)
- FR24: MCP server can identify false positive PII matches (trace IDs, span IDs, metric names, Kubernetes resource names, OTel semantic convention values) and exclude them
- FR25: MCP server can detect single/orphan spans (spans with no parent AND no children in the observation window)
- FR26: MCP server can detect bloated attributes (values exceeding a size threshold, default >1KB, or attributes with extremely high unique value counts)
- FR27: MCP server can detect missing resource attributes (`service.name`, `service.version`, `deployment.environment`) including `service.name=unknown` or empty values
- FR28: MCP server can detect duplicate signals (identical metric names from different sources, semantically equivalent metrics with different names)
- FR29: MCP server can detect missing sampling configuration (no probabilistic or tail sampling processor) and prompt the user about intentionality
- FR30: MCP server can measure observed throughput (data points/sec, spans/sec, log records/sec) and compare against collector resource limits (CPU/memory requests/limits)

### Fix Suggestion & Application

- FR31: MCP server can generate OTTL transform processor statements to fix detected issues (drop dimensions, redact PII, truncate attributes)
- FR32: MCP server can generate filter processor rules to address detected issues (drop metrics, deduplicate signals)
- FR33: MCP server can generate attributes processor configurations to address detected issues (remove or rename attributes)
- FR34: MCP server can generate resource processor configurations to add missing resource attributes
- FR35: Each suggested fix includes a complete YAML config block ready to apply, the target pipeline, and a risk assessment (low/medium/high)
- FR36: MCP server can apply a single user-approved fix to the collector config, followed by automatic health check
- FR37: MCP server presents each fix individually for user approval — no batch auto-apply

### Sampling & Sizing Recommendations

- FR38: MCP server can analyze captured trace data (error rate, latency distribution, service count) and recommend a sampling strategy (tail sampling, probabilistic, or hybrid)
- FR39: MCP server can generate a complete tail sampling processor config based on observed trace patterns (error-biased, latency-biased, or probabilistic)
- FR40: MCP server can estimate recommended CPU and memory resource requests/limits based on observed throughput plus headroom

### Session Management

- FR41: MCP server can create, track, and expire analysis sessions with unique identifiers
- FR42: MCP server can maintain session state across multiple sequential tool calls (start → capture → detect → suggest → apply → cleanup)
- FR43: MCP server can enforce a maximum number of concurrent analysis sessions (configurable, default 5)
- FR44: MCP server can provide a session summary on cleanup (findings count, fixes applied, rollbacks, session duration)

### RBAC & Deployment

- FR45: Helm chart supports a `v2.enabled` flag that controls whether v2 write RBAC permissions (update/patch on ConfigMaps, OpenTelemetryCollector CRs, and apps workloads) are included
- FR46: Clusters with `v2.enabled=false` only get v1 read-only RBAC — no write permissions granted
- FR47: Helm chart exposes configurable v2 settings: `v2.sessionTTL`, `v2.maxConcurrentSessions`
- FR48: v2 tools only appear in MCP `tools/list` response when `v2.enabled=true`

### Self-Instrumentation (v2 Extensions)

- FR49: Every v2 tool call produces an OTel span following GenAI/MCP semantic conventions (inherited from v1 instrumentation framework)
- FR50: MCP server emits v2-specific metrics: analysis session duration, signal capture counts, detection rule hits (by rule and severity), fixes applied (by type and outcome), rollback events (by trigger and reason), health check outcomes, active backup count
- FR51: All v2 mutation operations (config apply, rollback) produce structured log entries with trace_id/span_id correlation and before/after config diff summary
- FR52: Safety gate decisions (production refusal) produce WARN-level log entries with full context

### Documentation

- FR53: MkDocs site includes a Tool Reference for all 10 v2 tools with parameters, examples, and sample output
- FR54: MkDocs site includes a Safety Model guide explaining the environment gate, backup, health check, and rollback mechanisms
- FR55: MkDocs site includes a Migration Guide for upgrading from v1 to v2 (RBAC changes, Helm values, backward compatibility)
- FR56: MkDocs site includes a Detection Rules reference documenting all 8 runtime detection rules with thresholds, examples, and false positive mitigation

## Non-Functional Requirements

### Performance

- v2 tool responses complete within 15 seconds for single-collector operations, excluding `capture_signals` which blocks for the configured capture duration (30-120s)
- `check_health` completes within 5 seconds — this is on the critical path for rollback decisions
- `rollback_config` completes (config restore + rollout trigger) within 10 seconds
- Debug exporter injection (`start_analysis`) completes config mutation + rollout + health verification within 30 seconds
- Signal parsing (`capture_signals`) processes up to 100,000 data points (metrics + logs + spans) captured in a 60-second window within 5 seconds of capture completion
- Detection rule execution (`detect_issues`) completes all 8 rules within 10 seconds on the captured data set
- MCP server memory footprint remains under 512MB during active analysis sessions with captured signal data in memory (up from v1's 256MB limit to accommodate signal buffers)

### Security

- v2 write operations are gated by explicit user environment declaration — no heuristic environment detection
- Production environment declaration results in absolute refusal of all mutation operations with no bypass mechanism
- v2 RBAC permissions (write access) are opt-in via Helm chart flag — not included by default
- Config backups stored as ConfigMap annotations do not contain sensitive data beyond what is already in the ConfigMap
- PII detection results report the PII type and attribute key but do not include the actual PII values in tool output
- Detection rule output (captured signal samples) is truncated to prevent leaking full payloads through MCP responses
- All v1 security requirements remain: no credentials stored in MCP server config, hardcoded credential detection does not echo credentials, MCP transport supports TLS via Gateway

### Scalability

- MCP server supports up to 5 concurrent analysis sessions (configurable) without performance degradation
- Captured signal data for a single session (60-second window) stays under 100MB of memory
- Session cleanup releases all captured data immediately — no memory leak across sessions
- MCP server continues serving v1 read-only tool calls during active v2 analysis sessions without contention

### Reliability

- Rollback success rate: 100% — every failed health check must result in successful config restoration. This is the single most critical reliability requirement.
- Session recovery on MCP server restart: detect orphaned debug exporters and offer cleanup via `check_health` or automatic cleanup
- Detection rules fail independently — one rule panic/error does not prevent other rules from executing (same pattern as v1 analyzers)
- ConfigMap/CRD mutation failures (API errors, conflict, RBAC denied) are reported clearly with the specific Kubernetes API error, not generic failure messages
- Health check correctly distinguishes between: pod not yet ready (still starting), pod in CrashLoopBackOff (bad config), pod running but not processing data (stuck), and pod healthy (normal operation)

### Integration

- Compatible with OTel Collector v0.90+ debug exporter output format
- Compatible with both ConfigMap-based and OTel Operator CRD-based collector deployments
- ConfigMap mutations preserve non-collector data keys in the ConfigMap (e.g., Helm-managed metadata)
- CRD mutations work with OTel Operator v0.90+ reconciliation behavior
- GitOps compatibility: detect ArgoCD/Flux annotations on target resources and warn the user that GitOps may revert changes
- MCP protocol compatibility: all v2 tools follow the same Streamable HTTP transport and `StandardResponse` envelope as v1
