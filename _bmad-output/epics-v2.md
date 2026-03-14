---
stepsCompleted: [step-01-validate-prerequisites, step-02-design-epics, step-03-create-stories, step-04-final-validation]
inputDocuments: ['_bmad-output/prd-v2.md', '_bmad-output/architecture-v2.md', '_bmad-output/planning-artifacts/prd.md (v1)', '_bmad-output/planning-artifacts/architecture.md (v1)']
---

# otel-collector-mcp v2 - Epic Breakdown

## Overview

This document provides the complete epic and story breakdown for otel-collector-mcp v2, decomposing the requirements from the PRD, Architecture, and v1 context into implementable stories.

## Requirements Inventory

### Functional Requirements

FR1: All 7 v1 MCP tools (`triage_scan`, `detect_deployment_type`, `list_collectors`, `get_config`, `parse_collector_logs`, `parse_operator_logs`, `check_config`) retain identical input schemas, output formats, and behavior with zero regressions
FR2: v1 tools remain available regardless of whether v2 features are enabled
FR3: MCP server can ask the user to declare the environment type (dev, staging, production) before any mutation operation
FR4: MCP server refuses all mutation operations when the user declares "production" — no override, no force flag
FR5: MCP server creates a complete config backup (full ConfigMap data or CRD spec) before any mutation
FR6: MCP server stores config backups durably (ConfigMap annotation) so they survive MCP server pod restarts
FR7: MCP server can restore a backed-up config and trigger a collector restart (rollback)
FR8: MCP server automatically triggers rollback when a health check detects collector failure after a mutation
FR9: MCP server can detect concurrent analysis sessions targeting the same collector and reject with a clear error
FR10: MCP server can check whether a collector is healthy (pod Running, readiness probe passing, no CrashLoopBackOff, processing data)
FR11: MCP server can detect CrashLoopBackOff within 30 seconds of a collector restart
FR12: MCP server runs a health check automatically after every config mutation and restart
FR13: MCP server can report per-pod health status (phase, ready state, restart count, age) for multi-pod collectors
FR14: MCP server can inject a debug exporter (`verbosity: basic`) into specified collector pipelines without modifying existing receivers, processors, or exporters
FR15: MCP server can apply a modified collector config to a ConfigMap and trigger a pod rollout
FR16: MCP server can apply a modified collector config to an OpenTelemetryCollector CR (Operator handles rollout)
FR17: MCP server can capture debug exporter output from collector pod stdout for a configurable duration (30-120 seconds)
FR18: MCP server can parse captured debug output into structured metric data points (with labels), log records (with body and attributes), and trace spans (with attributes and parent/child relationships)
FR19: MCP server can remove the debug exporter from collector pipelines and restore a clean config (with any approved fixes preserved)
FR20: MCP server can auto-cleanup debug exporters after a configurable session TTL (default 10 minutes) if the user abandons the workflow
FR21: MCP server can recover orphaned debug exporters on startup (detect sessions that were active when the MCP server restarted)
FR22: MCP server can detect high-cardinality metric dimensions by counting unique label value combinations per metric name and flagging metrics exceeding a threshold (default: >100 unique combos in the capture window)
FR23: MCP server can detect PII patterns in log bodies and span/log attributes — email addresses (regex), IPv4/v6 addresses, credit card numbers (Luhn-validated), and phone numbers (international format)
FR24: MCP server can identify false positive PII matches (trace IDs, span IDs, metric names, Kubernetes resource names, OTel semantic convention values) and exclude them
FR25: MCP server can detect single/orphan spans (spans with no parent AND no children in the observation window)
FR26: MCP server can detect bloated attributes (values exceeding a size threshold, default >1KB, or attributes with extremely high unique value counts)
FR27: MCP server can detect missing resource attributes (`service.name`, `service.version`, `deployment.environment`) including `service.name=unknown` or empty values
FR28: MCP server can detect duplicate signals (identical metric names from different sources, semantically equivalent metrics with different names)
FR29: MCP server can detect missing sampling configuration (no probabilistic or tail sampling processor) and prompt the user about intentionality
FR30: MCP server can measure observed throughput (data points/sec, spans/sec, log records/sec) and compare against collector resource limits (CPU/memory requests/limits)
FR31: MCP server can generate OTTL transform processor statements to fix detected issues (drop dimensions, redact PII, truncate attributes)
FR32: MCP server can generate filter processor rules to address detected issues (drop metrics, deduplicate signals)
FR33: MCP server can generate attributes processor configurations to address detected issues (remove or rename attributes)
FR34: MCP server can generate resource processor configurations to add missing resource attributes
FR35: Each suggested fix includes a complete YAML config block ready to apply, the target pipeline, and a risk assessment (low/medium/high)
FR36: MCP server can apply a single user-approved fix to the collector config, followed by automatic health check
FR37: MCP server presents each fix individually for user approval — no batch auto-apply
FR38: MCP server can analyze captured trace data (error rate, latency distribution, service count) and recommend a sampling strategy (tail sampling, probabilistic, or hybrid)
FR39: MCP server can generate a complete tail sampling processor config based on observed trace patterns (error-biased, latency-biased, or probabilistic)
FR40: MCP server can estimate recommended CPU and memory resource requests/limits based on observed throughput plus headroom
FR41: MCP server can create, track, and expire analysis sessions with unique identifiers
FR42: MCP server can maintain session state across multiple sequential tool calls (start → capture → detect → suggest → apply → cleanup)
FR43: MCP server can enforce a maximum number of concurrent analysis sessions (configurable, default 5)
FR44: MCP server can provide a session summary on cleanup (findings count, fixes applied, rollbacks, session duration)
FR45: Helm chart supports a `v2.enabled` flag that controls whether v2 write RBAC permissions (update/patch on ConfigMaps, OpenTelemetryCollector CRs, and apps workloads) are included
FR46: Clusters with `v2.enabled=false` only get v1 read-only RBAC — no write permissions granted
FR47: Helm chart exposes configurable v2 settings: `v2.sessionTTL`, `v2.maxConcurrentSessions`
FR48: v2 tools only appear in MCP `tools/list` response when `v2.enabled=true`
FR49: Every v2 tool call produces an OTel span following GenAI/MCP semantic conventions (inherited from v1 instrumentation framework)
FR50: MCP server emits v2-specific metrics: analysis session duration, signal capture counts, detection rule hits (by rule and severity), fixes applied (by type and outcome), rollback events (by trigger and reason), health check outcomes, active backup count
FR51: All v2 mutation operations (config apply, rollback) produce structured log entries with trace_id/span_id correlation and before/after config diff summary
FR52: Safety gate decisions (production refusal) produce WARN-level log entries with full context
FR53: MkDocs site includes a Tool Reference for all 10 v2 tools with parameters, examples, and sample output
FR54: MkDocs site includes a Safety Model guide explaining the environment gate, backup, health check, and rollback mechanisms
FR55: MkDocs site includes a Migration Guide for upgrading from v1 to v2 (RBAC changes, Helm values, backward compatibility)
FR56: MkDocs site includes a Detection Rules reference documenting all 8 runtime detection rules with thresholds, examples, and false positive mitigation

### NonFunctional Requirements

NFR1: v2 tool responses complete within 15 seconds for single-collector operations, excluding `capture_signals` which blocks for the configured capture duration (30-120s)
NFR2: `check_health` completes within 5 seconds — this is on the critical path for rollback decisions
NFR3: `rollback_config` completes (config restore + rollout trigger) within 10 seconds
NFR4: Debug exporter injection (`start_analysis`) completes config mutation + rollout + health verification within 30 seconds
NFR5: Signal parsing (`capture_signals`) processes up to 100,000 data points captured in a 60-second window within 5 seconds of capture completion
NFR6: Detection rule execution (`detect_issues`) completes all 8 rules within 10 seconds on the captured data set
NFR7: MCP server memory footprint remains under 512MB during active analysis sessions with captured signal data in memory
NFR8: v2 write operations are gated by explicit user environment declaration — no heuristic environment detection
NFR9: Production environment declaration results in absolute refusal of all mutation operations with no bypass mechanism
NFR10: v2 RBAC permissions (write access) are opt-in via Helm chart flag — not included by default
NFR11: Config backups stored as ConfigMap annotations do not contain sensitive data beyond what is already in the ConfigMap
NFR12: PII detection results report the PII type and attribute key but do not include the actual PII values in tool output
NFR13: Detection rule output (captured signal samples) is truncated to prevent leaking full payloads through MCP responses
NFR14: All v1 security requirements remain: no credentials stored in MCP server config, hardcoded credential detection does not echo credentials, MCP transport supports TLS via Gateway
NFR15: MCP server supports up to 5 concurrent analysis sessions (configurable) without performance degradation
NFR16: Captured signal data for a single session (60-second window) stays under 100MB of memory
NFR17: Session cleanup releases all captured data immediately — no memory leak across sessions
NFR18: MCP server continues serving v1 read-only tool calls during active v2 analysis sessions without contention
NFR19: Rollback success rate: 100% — every failed health check must result in successful config restoration
NFR20: Session recovery on MCP server restart: detect orphaned debug exporters and offer cleanup
NFR21: Detection rules fail independently — one rule panic/error does not prevent other rules from executing
NFR22: ConfigMap/CRD mutation failures are reported clearly with the specific Kubernetes API error, not generic failure messages
NFR23: Health check correctly distinguishes between: pod not yet ready, pod in CrashLoopBackOff, pod running but not processing, and pod healthy
NFR24: Compatible with OTel Collector v0.90+ debug exporter output format
NFR25: Compatible with both ConfigMap-based and OTel Operator CRD-based collector deployments
NFR26: ConfigMap mutations preserve non-collector data keys in the ConfigMap
NFR27: CRD mutations work with OTel Operator v0.90+ reconciliation behavior
NFR28: GitOps compatibility: detect ArgoCD/Flux annotations on target resources and warn the user that GitOps may revert changes
NFR29: MCP protocol compatibility: all v2 tools follow the same Streamable HTTP transport and `StandardResponse` envelope as v1

### Additional Requirements

- No new Go dependencies — all v2 capabilities from existing `go.mod`
- 5 new packages: `pkg/session/`, `pkg/mutator/`, `pkg/signals/`, `pkg/analysis/runtime/`, `pkg/fixes/`
- Brownfield project extending v1 codebase — no starter template
- Acyclic dependency graph: `tools/` → `session/` → `mutator/` → `k8s/`; no v1 package depends on any v2 package
- ConfigMap annotation-based backup for rollback durability using `mcp.otel.dev/` annotation prefix
- Feature gating via `v2.enabled` Helm value controls RBAC, tool registration, and config
- 9 new ADRs (ADR-014 through ADR-022) governing session state, safety model, config mutation, debug injection, signal parsing, runtime rules, fix generation, feature gating, and telemetry
- Implementation priority order defined in architecture: config → types → mutator → session → tools → signals → analyzers → fixes → helm → telemetry → docs
- v2 tool file naming: `tool_v2_<name>.go` to visually distinguish from v1 tools
- v2 runtime analyzer file naming: `pkg/analysis/runtime/analyzer_<name>.go`
- v2 fix generator file naming: `pkg/fixes/fix_<type>.go`
- All v2 tools embed `BaseTool` AND hold a reference to the session manager
- Every v2 tool (except `check_health`) validates session state before executing
- Mutator factory returns correct implementation (ConfigMapMutator or CRDMutator) based on deployment mode
- Safety chain pattern (backup → apply → rollout → health → rollback) mandatory for all mutation operations
- All v2 tool responses must use markdown table format for structured output (same as v1 tools) for token-efficient LLM consumption

### FR Coverage Map

FR1: Epic 1 — v1 tool preservation (zero regressions)
FR2: Epic 1 — v1 tools available regardless of v2 enablement
FR3: Epic 3 — Environment declaration gate before mutations
FR4: Epic 3 — Production refusal (absolute, no override)
FR5: Epic 3 — Config backup before any mutation
FR6: Epic 3 — Durable backup via ConfigMap annotation
FR7: Epic 3 — Rollback (restore backup + restart)
FR8: Epic 3 — Automatic rollback on health failure
FR9: Epic 3 — Concurrent session rejection
FR10: Epic 2 — Health check (Running, readiness, CrashLoopBackOff, processing)
FR11: Epic 2 — CrashLoopBackOff detection within 30s
FR12: Epic 3 — Automatic post-mutation health check
FR13: Epic 2 — Per-pod health status reporting
FR14: Epic 4 — Debug exporter injection (append-only)
FR15: Epic 4 — ConfigMap config apply + pod rollout trigger
FR16: Epic 4 — CRD config apply (Operator rollout)
FR17: Epic 4 — Capture debug exporter stdout (30-120s)
FR18: Epic 4 — Parse debug output into structured signals
FR19: Epic 4 — Remove debug exporter (preserve approved fixes)
FR20: Epic 4 — Auto-cleanup on session TTL expiry
FR21: Epic 4 — Orphaned debug exporter recovery on startup
FR22: Epic 5 — High-cardinality metric dimension detection
FR23: Epic 5 — PII pattern detection (email, IP, CC, phone)
FR24: Epic 5 — PII false positive exclusion
FR25: Epic 5 — Orphan span detection
FR26: Epic 5 — Bloated attribute detection
FR27: Epic 5 — Missing resource attribute detection
FR28: Epic 5 — Duplicate signal detection
FR29: Epic 5 — Missing sampling configuration detection
FR30: Epic 5 — Throughput vs resource limits comparison
FR31: Epic 6 — OTTL transform fix generation
FR32: Epic 6 — Filter processor fix generation
FR33: Epic 6 — Attributes processor fix generation
FR34: Epic 6 — Resource processor fix generation
FR35: Epic 6 — Complete YAML config block + risk assessment
FR36: Epic 6 — Apply single user-approved fix + health check
FR37: Epic 6 — Individual fix approval (no batch)
FR38: Epic 7 — Sampling strategy recommendation
FR39: Epic 7 — Tail sampling config generation
FR40: Epic 7 — Resource sizing recommendation
FR41: Epic 3 — Session create/track/expire
FR42: Epic 3 — Session state across sequential tool calls
FR43: Epic 3 — Max concurrent sessions enforcement
FR44: Epic 3 — Session summary on cleanup
FR45: Epic 1 — v2.enabled Helm flag for write RBAC
FR46: Epic 1 — v2.enabled=false → read-only RBAC only
FR47: Epic 1 — Configurable v2 Helm settings
FR48: Epic 1 — Conditional v2 tool registration
FR49: Epic 8 — v2 tool spans (GenAI/MCP semconv)
FR50: Epic 8 — v2-specific metrics (8 new instruments)
FR51: Epic 8 — Structured mutation logging with trace correlation
FR52: Epic 8 — Safety gate WARN-level logging
FR53: Epic 9 — v2 Tool Reference documentation
FR54: Epic 9 — Safety Model guide
FR55: Epic 9 — v1→v2 Migration Guide
FR56: Epic 9 — Detection Rules reference

## Epic List

### Epic 1: v2 Foundation & Backward Compatibility
Engineers can deploy the v2-enabled MCP server, confirm all 7 v1 tools still work identically, and control v2 features via Helm chart flag.
**FRs covered:** FR1, FR2, FR45, FR46, FR47, FR48

### Epic 2: Collector Health Monitoring
Engineers can check real-time health of any collector — pod phase, readiness, CrashLoopBackOff detection, per-pod status for multi-replica deployments. Standalone tool, no session required.
**FRs covered:** FR10, FR11, FR13

### Epic 3: Session Management & Safety Model
Engineers can safely start analysis sessions on dev/staging collectors — environment gate prevents production mutations, configs are automatically backed up (surviving MCP server restarts), concurrent sessions are rejected, and any failure triggers automatic rollback.
**FRs covered:** FR3, FR4, FR5, FR6, FR7, FR8, FR9, FR12, FR41, FR42, FR43, FR44

### Epic 4: Dynamic Signal Capture
Engineers can inject a debug exporter into live collector pipelines, capture real signal data (metrics, logs, traces) for 30-120 seconds, and clean up — with session TTL auto-cleanup and orphan recovery on server restart.
**FRs covered:** FR14, FR15, FR16, FR17, FR18, FR19, FR20, FR21

### Epic 5: Runtime Issue Detection
Engineers get automated detection of runtime anti-patterns in live signal data — high-cardinality metrics, PII leaking in logs, orphan spans, bloated attributes, missing resource attributes, duplicate signals, missing sampling, and resource sizing mismatches.
**FRs covered:** FR22, FR23, FR24, FR25, FR26, FR27, FR28, FR29, FR30

### Epic 6: Fix Suggestion & Application
Engineers receive actionable fix suggestions (complete YAML config blocks) for detected issues and can apply them one-by-one with automatic health checks — OTTL transforms, filter rules, attributes processors, resource processors, each with risk assessment.
**FRs covered:** FR31, FR32, FR33, FR34, FR35, FR36, FR37

### Epic 7: Sampling & Sizing Recommendations
Engineers get data-driven sampling strategy recommendations (tail/probabilistic/hybrid with complete config) and resource sizing recommendations based on observed throughput.
**FRs covered:** FR38, FR39, FR40

### Epic 8: v2 Self-Instrumentation
Operations teams can observe v2 analysis workflows in their observability backend — session durations, signal capture counts, detection hits, fix outcomes, rollback events, health check results, and active backup counts, all correlated via trace IDs.
**FRs covered:** FR49, FR50, FR51, FR52

### Epic 9: v2 Documentation
Engineers have comprehensive guides to adopt v2 — tool reference for all 10 new tools, safety model explainer, v1→v2 migration guide, and detection rules reference with thresholds and examples.
**FRs covered:** FR53, FR54, FR55, FR56

---

## Epic 1: v2 Foundation & Backward Compatibility

Engineers can deploy the v2-enabled MCP server, confirm all 7 v1 tools still work identically, and control v2 features via Helm chart flag.

### Story 1.1: v2 Configuration & Error Codes

As a platform engineer,
I want the MCP server to support v2 configuration fields (`V2Enabled`, `SessionTTL`, `MaxConcurrentSessions`) via environment variables,
So that I can control v2 behavior without rebuilding the server.

**Acceptance Criteria:**

**Given** the MCP server binary with v2 code
**When** `V2_ENABLED=true`, `V2_SESSION_TTL=10m`, `V2_MAX_SESSIONS=5` environment variables are set
**Then** `pkg/config/config.go` parses these into the Config struct with correct types and defaults
**And** `V2_ENABLED` defaults to `false`, `V2_SESSION_TTL` defaults to `10m`, `V2_MAX_SESSIONS` defaults to `5`

**Given** v2 operations encounter error conditions
**When** errors are returned
**Then** they use v2-specific error codes (`session_not_found`, `session_expired`, `concurrent_session`, `production_refused`, `backup_failed`, `rollback_failed`, `health_check_failed`, `mutation_failed`, `capture_failed`, `gitops_conflict`) defined in `pkg/types/errors.go`

### Story 1.2: Feature-Gated v2 Tool Registration

As a platform engineer,
I want v2 tools to only appear in MCP `tools/list` when `v2.enabled=true`,
So that clusters not ready for v2 see no v2 functionality.

**Acceptance Criteria:**

**Given** the MCP server starts with `V2_ENABLED=false`
**When** a client calls `tools/list`
**Then** only the 7 v1 tools are returned — no v2 tools visible

**Given** the MCP server starts with `V2_ENABLED=true`
**When** a client calls `tools/list`
**Then** all 7 v1 tools AND all 10 v2 tools are returned

**Given** `V2_ENABLED=false`
**When** a client attempts to invoke any v2 tool directly
**Then** the server returns a structured error indicating the tool is not available

### Story 1.3: Feature-Gated Helm Chart RBAC

As a platform engineer,
I want the Helm chart to conditionally include v2 write RBAC permissions based on `v2.enabled`,
So that clusters only grant write access when v2 is explicitly opted into.

**Acceptance Criteria:**

**Given** the Helm chart is installed with `v2.enabled=false` (default)
**When** the ClusterRole is rendered
**Then** only v1 read-only RBAC verbs (`get`, `list`, `watch`) are included — no `update` or `patch`

**Given** the Helm chart is installed with `v2.enabled=true`
**When** the ClusterRole is rendered
**Then** write RBAC is added: `update`/`patch` on ConfigMaps, `update`/`patch` on OpenTelemetryCollector CRs, `patch` on Deployments/DaemonSets/StatefulSets

**Given** the Helm chart with `v2.enabled=true`
**When** `values.yaml` is reviewed
**Then** `v2.sessionTTL` and `v2.maxConcurrentSessions` are configurable and passed as environment variables to the Deployment

**Given** all v1 tools
**When** the server is deployed with v2 enabled
**Then** all v1 tools produce identical outputs with zero regressions

---

## Epic 2: Collector Health Monitoring

Engineers can check real-time health of any collector — pod phase, readiness, CrashLoopBackOff detection, per-pod status for multi-replica deployments. Standalone tool, no session required.

### Story 2.1: Health Check Core Logic

As an observability engineer,
I want a reusable health check module that can assess collector pod health status,
So that both the standalone `check_health` tool and the mutation safety chain can reliably determine collector state.

**Acceptance Criteria:**

**Given** a collector with pods in Running phase with readiness probes passing
**When** `pkg/mutator/health.go` `checkPodHealth()` is called
**Then** it returns `Healthy` status

**Given** a collector pod that entered CrashLoopBackOff
**When** `checkPodHealth()` is called within 30 seconds of the crash
**Then** it returns `CrashLoop` status (FR11)

**Given** a collector pod in Pending phase
**When** `checkPodHealth()` is called
**Then** it returns `NotReady` status

**Given** a collector pod Running but with failing readiness probes
**When** `checkPodHealth()` is called
**Then** it returns `Unhealthy` status (NFR23: distinguishes all 4 states)

**Given** `WaitHealthy()` is called with a 30-second timeout
**When** the pod transitions to healthy within the timeout
**Then** it returns nil (success) with 2-second poll interval

### Story 2.2: check_health MCP Tool

As an observability engineer,
I want a `check_health` MCP tool that reports per-pod health status for any collector,
So that I can quickly assess collector health without needing an analysis session.

**Acceptance Criteria:**

**Given** a collector deployment with 3 healthy pods
**When** `check_health` is called with `namespace` and `name`
**Then** it returns per-pod status (name, phase, ready, restarts, age) for all 3 pods in markdown table format (FR13)
**And** overall `healthy: true` and `status: healthy`
**And** response completes within 5 seconds (NFR2)

**Given** a collector where one pod is in CrashLoopBackOff
**When** `check_health` is called
**Then** it returns `healthy: false`, `status: crash_loop` with the affected pod highlighted

**Given** a non-existent collector name
**When** `check_health` is called
**Then** it returns `status: not_found` with a clear error message

**Given** `check_health` is called
**When** the response is returned
**Then** it follows `StandardResponse` envelope with cluster identification fields and markdown table output format

---

## Epic 3: Session Management & Safety Model

Engineers can safely start analysis sessions on dev/staging collectors — environment gate prevents production mutations, configs are automatically backed up (surviving MCP server restarts), concurrent sessions are rejected, and any failure triggers automatic rollback.

### Story 3.1: Mutator Interface & ConfigMap Mutator

As an observability engineer,
I want the MCP server to safely mutate ConfigMap-based collector configs with backup and rollback capability,
So that any config change can be undone if something goes wrong.

**Acceptance Criteria:**

**Given** `pkg/mutator/types.go` defines the `Mutator` interface
**When** the interface is reviewed
**Then** it includes `Backup()`, `ApplyConfig()`, `Rollback()`, `TriggerRollout()`, `Cleanup()` methods

**Given** a ConfigMap-based collector
**When** `ConfigMapMutator.Backup()` is called
**Then** the full ConfigMap `.data` is stored as a `mcp.otel.dev/config-backup` annotation on the ConfigMap (FR5, FR6)
**And** the session ID is stored as a `mcp.otel.dev/session-id` annotation
**And** `resourceVersion` is captured for optimistic concurrency

**Given** a backup exists on a ConfigMap
**When** `ConfigMapMutator.ApplyConfig()` is called with new YAML
**Then** only the collector config key is updated — other data keys are preserved (NFR26)

**Given** a ConfigMap mutation applied
**When** `ConfigMapMutator.TriggerRollout()` is called
**Then** it patches the owning Deployment/DaemonSet/StatefulSet with `kubectl.kubernetes.io/restartedAt` annotation

**Given** a collector in an unhealthy state after mutation
**When** `ConfigMapMutator.Rollback()` is called
**Then** the backup config is restored from the annotation, rollout is triggered, and the annotation is cleaned up (FR7)
**And** rollback completes within 10 seconds (NFR3)

**Given** a ConfigMap with ArgoCD or Flux annotations
**When** mutation is attempted
**Then** the mutator detects `argocd.argoproj.io/managed-by` or `fluxcd.io/automated` annotations and returns a `gitops_conflict` warning (NFR28)

### Story 3.2: CRD Mutator

As an observability engineer,
I want the MCP server to safely mutate OTel Operator CRD-based collector configs,
So that Operator-managed collectors get the same backup/rollback safety as ConfigMap-based ones.

**Acceptance Criteria:**

**Given** an Operator-managed collector (OpenTelemetryCollector CR)
**When** `CRDMutator.Backup()` is called
**Then** the full CRD `.spec` is stored as a `mcp.otel.dev/config-backup` annotation on the CR (FR5, FR6)

**Given** a CRD backup exists
**When** `CRDMutator.ApplyConfig()` is called with new YAML
**Then** it patches the CR `.spec.config` field
**And** the Operator handles rollout automatically — no explicit `TriggerRollout()` needed (FR16)

**Given** the Operator rejects the CRD spec update
**When** the API returns a validation error
**Then** the error is reported with the specific Kubernetes API error message (NFR22)

**Given** a CRD mutation
**When** `CRDMutator.Rollback()` is called
**Then** the backup spec is restored and the Operator reconciles to the original state

**Given** a collector deployment
**When** a `Mutator` is needed
**Then** `NewMutator()` factory returns `CRDMutator` for `ModeOperatorCRD` and `ConfigMapMutator` for all other modes

### Story 3.3: Safety Chain with Health-Gated Auto-Rollback

As an observability engineer,
I want every config mutation to automatically verify collector health and roll back on failure,
So that bad config changes never leave a collector in a broken state.

**Acceptance Criteria:**

**Given** any config mutation (debug inject, fix apply, rollback)
**When** the mutation is applied and rollout is triggered
**Then** `WaitHealthy()` is called automatically with a 30-second timeout (FR12)

**Given** a health check after mutation detects CrashLoopBackOff
**When** the failure is detected
**Then** automatic rollback is triggered — backup config restored, rollout triggered, recovery verified (FR8)
**And** the tool response indicates `status: rolled_back` with the failure reason

**Given** a health check after mutation times out (pod not ready after 30s)
**When** the timeout occurs
**Then** automatic rollback is triggered with the same recovery flow

**Given** the safety chain `SafeApply()` method
**When** backup fails
**Then** the mutation is refused — no config change is attempted

**Given** rollback is triggered
**When** rollback itself fails
**Then** a `rollback_failed` error is returned with full context (the most critical error path)
**And** rollback success rate target is 100% (NFR19)

### Story 3.4: Session Manager

As an observability engineer,
I want the MCP server to manage analysis sessions with lifecycle tracking,
So that multiple sequential tool calls share state and sessions are properly cleaned up.

**Acceptance Criteria:**

**Given** `pkg/session/manager.go` with a `Manager` struct
**When** `Create()` is called with collector reference and environment
**Then** a new session is created with UUID v4 ID, stored in `sync.Map`, and returned (FR41)

**Given** an active session
**When** `Get()` is called with the session ID
**Then** the session is returned with all accumulated state (backup config, captured signals, findings, suggested fixes) (FR42)
**And** the session's last-activity timestamp is updated

**Given** a session with no activity for longer than `SessionTTL`
**When** the cleanup loop runs (every 30 seconds)
**Then** the expired session is cleaned up (debug exporter removed if present, session closed) (FR20)

**Given** 5 active sessions (max concurrent = 5)
**When** a 6th `Create()` is attempted
**Then** it is rejected with `concurrent_session` error (FR43)

**Given** a collector already targeted by an active session
**When** another session targets the same collector
**Then** it is rejected with `concurrent_session` error specifying which session holds the lock (FR9)

**Given** `pkg/session/types.go`
**When** session state is reviewed
**Then** it includes: ID, collector ref, environment, backup config, injected pipelines, captured signals, findings, suggested fixes, timestamps, and state enum (Created, Capturing, Analyzing, Closed)

### Story 3.5: start_analysis Tool

As an observability engineer,
I want a `start_analysis` MCP tool that initiates a safe analysis session on a dev/staging collector,
So that I can begin the dynamic analysis workflow with full safety guarantees.

**Acceptance Criteria:**

**Given** `start_analysis` is called with `namespace`, `name`, `environment=staging`, and optional `pipelines`
**When** the tool executes
**Then** it: (1) validates environment is not production, (2) creates a session, (3) backs up the config, (4) injects debug exporter, (5) applies config, (6) triggers rollout, (7) waits for health, (8) returns session_id and status `ready_for_capture` (FR3, FR14)

**Given** `environment=production`
**When** `start_analysis` is called
**Then** it refuses with `production_refused` error — no override, no force flag (FR4)
**And** the refusal is absolute with no bypass mechanism (NFR9)

**Given** successful session start
**When** the response is returned
**Then** it includes `session_id`, `environment`, `backup_id`, `collector` (name, namespace, deploymentMode), `injected_pipelines`, and `status` in markdown table format

**Given** `start_analysis` completes config mutation + rollout + health check
**When** timing is measured
**Then** total execution completes within 30 seconds (NFR4)

**Given** health check fails after debug exporter injection
**When** CrashLoopBackOff is detected
**Then** automatic rollback restores the original config and session is closed with error

### Story 3.6: rollback_config Tool

As an observability engineer,
I want a `rollback_config` MCP tool that restores the backed-up config at any time during an analysis session,
So that I can manually undo all changes if needed.

**Acceptance Criteria:**

**Given** an active session with a backup config
**When** `rollback_config` is called with `session_id`
**Then** the backup config is restored via the mutator, rollout is triggered, and health is verified (FR7)
**And** response includes `restored_from`, `health_check` status, and `status: rollback_complete`

**Given** `rollback_config` is called with an invalid or expired session ID
**When** the tool executes
**Then** it returns `session_not_found` or `session_expired` error

**Given** rollback is executed
**When** timing is measured
**Then** config restore + rollout trigger completes within 10 seconds (NFR3)

---

## Epic 4: Dynamic Signal Capture

Engineers can inject a debug exporter into live collector pipelines, capture real signal data (metrics, logs, traces) for 30-120 seconds, and clean up — with session TTL auto-cleanup and orphan recovery on server restart.

### Story 4.1: Debug Exporter Injection & Removal

As an observability engineer,
I want the MCP server to inject and remove debug exporters from collector pipeline configs,
So that signal capture can be enabled temporarily without affecting existing pipeline components.

**Acceptance Criteria:**

**Given** a collector config YAML with existing pipelines
**When** `pkg/mutator/inject.go` `InjectDebugExporter()` is called with target pipeline names
**Then** a `debug` exporter with `verbosity: basic` is appended to the `exporters` section
**And** `debug` is appended to the exporter lists of the targeted pipelines
**And** no existing receivers, processors, or exporters are modified (FR14, append-only per ADR-017)

**Given** no specific pipelines are specified
**When** `InjectDebugExporter()` is called
**Then** the debug exporter is added to ALL pipelines in the config

**Given** a collector config with an injected debug exporter and approved fixes
**When** `RemoveDebugExporter()` is called
**Then** the `debug` exporter is removed from exporters section and all pipeline references
**And** any approved fixes applied during the session are preserved (FR19)

**Given** the debug exporter already exists in the config
**When** `InjectDebugExporter()` is called
**Then** it detects the existing debug exporter and skips injection (idempotent)

### Story 4.2: Signal Parser

As an observability engineer,
I want the MCP server to parse debug exporter stdout into structured signal data,
So that runtime detection rules can analyze real metrics, logs, and traces.

**Acceptance Criteria:**

**Given** `pkg/signals/types.go` defines signal data models
**When** the types are reviewed
**Then** they include `CapturedSignals`, `MetricDataPoint` (name, labels, value, type), `LogRecord` (body, attributes, resource, severity), `SpanData` (traceID, spanID, parentSpanID, name, attributes, events, duration)

**Given** debug exporter stdout with metric data (collector v0.90+ format)
**When** `pkg/signals/parser_metrics.go` parses the output
**Then** it produces `MetricDataPoint` structs with metric name, label key/value pairs, and data point values (FR18, NFR24)

**Given** debug exporter stdout with log records
**When** `pkg/signals/parser_logs.go` parses the output
**Then** it produces `LogRecord` structs with body, attributes, resource attributes, and severity

**Given** debug exporter stdout with trace spans
**When** `pkg/signals/parser_traces.go` parses the output
**Then** it produces `SpanData` structs with traceID, spanID, parentSpanID, attributes, and parent/child relationships

**Given** up to 100,000 data points captured in a 60-second window
**When** parsing completes
**Then** it finishes within 5 seconds (NFR5)
**And** `pkg/signals/` has no internal dependencies — leaf package using only stdlib

### Story 4.3: capture_signals Tool

As an observability engineer,
I want a `capture_signals` MCP tool that streams and parses debug exporter output for a specified duration,
So that I can capture a representative sample of live signal data for analysis.

**Acceptance Criteria:**

**Given** an active session in `ready_for_capture` state
**When** `capture_signals` is called with `session_id` and `duration_seconds=60`
**Then** it streams collector pod stdout for 60 seconds, parses the output via `pkg/signals/parser.go`, and stores the `CapturedSignals` in the session (FR17)

**Given** `duration_seconds` is not specified
**When** `capture_signals` is called
**Then** it defaults to 60 seconds

**Given** `duration_seconds` is outside the 30-120 range
**When** `capture_signals` is called
**Then** it rejects with a validation error

**Given** signal capture completes
**When** the response is returned
**Then** it includes signal counts (`metrics.data_points`, `metrics.unique_metric_names`, `logs.records`, `traces.spans`, `traces.unique_trace_ids`) and `status: capture_complete` in markdown table format

**Given** captured signal data for a single session
**When** memory usage is measured
**Then** it stays under 100MB (NFR16)

**Given** an invalid or expired session ID
**When** `capture_signals` is called
**Then** it returns `session_not_found` or `session_expired` error

### Story 4.4: cleanup_debug Tool

As an observability engineer,
I want a `cleanup_debug` MCP tool that removes the debug exporter and closes the analysis session,
So that the collector is returned to a clean state after analysis with a summary of what was found and fixed.

**Acceptance Criteria:**

**Given** an active session with a debug exporter injected
**When** `cleanup_debug` is called with `session_id`
**Then** the debug exporter is removed from the config (preserving any approved fixes), config is applied, rollout triggered, and health verified (FR19)

**Given** cleanup completes
**When** the response is returned
**Then** it includes `removed_from_pipelines`, `health_check` status, and a `session_summary` with findings_count, fixes_applied, rollbacks, and duration_seconds (FR44)
**And** response uses markdown table format

**Given** cleanup completes
**When** session state is released
**Then** all captured signal data is freed from memory immediately (NFR17)

**Given** an already-closed session
**When** `cleanup_debug` is called
**Then** it returns `session_expired` error

### Story 4.5: Session TTL Auto-Cleanup & Orphan Recovery

As a platform engineer,
I want abandoned analysis sessions to auto-cleanup and orphaned debug exporters to be recovered on server restart,
So that debug exporters never get permanently left in collector pipelines.

**Acceptance Criteria:**

**Given** a session with no activity for longer than `SessionTTL` (default 10 minutes)
**When** the session manager cleanup loop detects the expiry
**Then** it automatically removes the debug exporter from the collector, restores clean config, and closes the session (FR20)

**Given** the MCP server restarts while an analysis session was active
**When** the server starts up
**Then** `pkg/session/recovery.go` scans all ConfigMaps and CRDs for `mcp.otel.dev/session-id` annotations (FR21)
**And** for each orphaned session: removes the debug exporter, restores the backup config, cleans up annotations

**Given** a recovered orphaned session
**When** recovery completes
**Then** a WARN-level log entry is emitted with the collector name, namespace, and session ID

---

## Epic 5: Runtime Issue Detection

Engineers get automated detection of runtime anti-patterns in live signal data — high-cardinality metrics, PII leaking in logs, orphan spans, bloated attributes, missing resource attributes, duplicate signals, missing sampling, and resource sizing mismatches.

### Story 5.1: Runtime Analyzer Framework & High-Cardinality Detection

As an observability engineer,
I want the MCP server to detect high-cardinality metric dimensions in captured signal data,
So that I can identify which metrics are inflating my observability costs before they reach the backend.

**Acceptance Criteria:**

**Given** `pkg/analysis/runtime/analyzer.go`
**When** the framework is reviewed
**Then** it defines `RuntimeAnalyzer` function type, `RuntimeAnalysisInput` (containing `CapturedSignals`, collector config, deployment mode), and `AllRuntimeAnalyzers()` registry function
**And** runtime analyzers return `[]types.DiagnosticFinding` (same output type as v1 static analyzers)

**Given** captured metric data with a metric having >100 unique label value combinations
**When** `analyzer_high_cardinality.go` runs
**Then** it flags the metric with severity `warning`, category `cardinality`, lists the high-cardinality label keys, and reports the unique combination count (FR22)

**Given** captured metric data with all metrics under the threshold
**When** the high-cardinality analyzer runs
**Then** it returns no findings (no false positives)

**Given** the cardinality threshold
**When** it is checked
**Then** the default is >100 unique combinations in the capture window

### Story 5.2: PII Detection with False Positive Exclusion

As a platform engineer,
I want the MCP server to detect PII patterns in captured log and span data while excluding known false positives,
So that I can identify compliance risks without being overwhelmed by irrelevant matches.

**Acceptance Criteria:**

**Given** captured log records containing email addresses in attributes
**When** `analyzer_pii.go` runs
**Then** it detects email pattern matches using RFC 5322 simplified regex and reports the attribute key and PII type — but NOT the actual PII value (FR23, NFR12)

**Given** captured data containing IPv4 and IPv6 addresses
**When** the PII analyzer runs
**Then** it detects both address formats (FR23)

**Given** captured data containing 13-19 digit sequences
**When** the PII analyzer runs
**Then** it validates with Luhn algorithm before flagging as credit card numbers (FR23)

**Given** captured data containing trace IDs, span IDs, metric names, or Kubernetes resource names that match PII patterns
**When** the PII analyzer runs
**Then** it excludes these known false positives (FR24)
**And** OTel semantic convention values (e.g., `http.url` containing IPs) are also excluded

**Given** phone numbers in international format
**When** the PII analyzer runs
**Then** it detects them but with a note about high false positive risk

### Story 5.3: Orphan Span & Bloated Attribute Detection

As an observability engineer,
I want the MCP server to detect orphan spans and bloated attributes in captured signal data,
So that I can identify broken instrumentation and attribute value bloat before they impact my backend.

**Acceptance Criteria:**

**Given** captured span data where a span has no parent AND no children in the observation window
**When** `analyzer_orphan_spans.go` runs
**Then** it flags the span as orphaned with severity `warning`, category `orphan_spans`, and the span name/service (FR25)

**Given** captured span data where all spans have either a parent or children
**When** the orphan span analyzer runs
**Then** it returns no findings

**Given** captured signal data where an attribute value exceeds 1KB
**When** `analyzer_bloated_attrs.go` runs
**Then** it flags the attribute with severity `warning`, category `bloated_attrs`, the attribute key, and the approximate size (FR26)

**Given** captured signal data where an attribute has extremely high unique value count
**When** the bloated attribute analyzer runs
**Then** it flags the attribute with a recommendation to evaluate cardinality

### Story 5.4: Missing Resource Attributes & Duplicate Signal Detection

As an observability engineer,
I want the MCP server to detect missing resource attributes and duplicate signals,
So that I can ensure proper service identification and eliminate redundant data collection.

**Acceptance Criteria:**

**Given** captured signal data where `service.name` is missing, set to `unknown`, or empty
**When** `analyzer_missing_resource.go` runs
**Then** it flags the finding with severity `warning`, category `missing_resource`, and lists which attributes are missing or invalid (FR27)
**And** checks for `service.name`, `service.version`, and `deployment.environment`

**Given** captured metric data with identical metric names from different sources
**When** `analyzer_duplicate_signals.go` runs
**Then** it flags the duplicates with severity `info`, category `duplicates`, and the conflicting sources (FR28)

**Given** captured metric data with semantically equivalent metrics under different names
**When** the duplicate signal analyzer runs
**Then** it flags them as potential duplicates for review

### Story 5.5: Missing Sampling & Resource Sizing Detection

As an observability engineer,
I want the MCP server to detect missing sampling configuration and compare throughput against resource limits,
So that I can proactively address trace volume and collector sizing before problems occur.

**Acceptance Criteria:**

**Given** a collector config with no probabilistic or tail sampling processor
**When** `analyzer_missing_sampling.go` runs
**Then** it flags with severity `info`, category `sampling`, and prompts the user to confirm if this is intentional (FR29)

**Given** a collector config with sampling already configured
**When** the missing sampling analyzer runs
**Then** it returns no findings

**Given** captured signal data throughput (data points/sec, spans/sec, log records/sec) and the collector's CPU/memory resource limits
**When** `analyzer_resource_sizing.go` runs
**Then** it compares observed throughput against resource limits and flags if throughput suggests the collector is undersized or oversized (FR30)

### Story 5.6: detect_issues Tool

As an observability engineer,
I want a `detect_issues` MCP tool that runs all runtime detection rules against captured signal data,
So that I get a prioritized list of findings in one call.

**Acceptance Criteria:**

**Given** an active session with captured signal data
**When** `detect_issues` is called with `session_id`
**Then** all 8 runtime analyzers execute against the captured data and return findings sorted by severity (critical → warning → info)

**Given** detection completes
**When** the response is returned
**Then** each finding includes `rule`, `severity`, `category`, `summary`, `detail`, `affected_signals`, and `fix_available` boolean in markdown table format

**Given** all 8 detection rules
**When** timing is measured
**Then** execution completes within 10 seconds (NFR6)

**Given** one detection rule panics or errors
**When** the other 7 rules are unaffected
**Then** they still execute and return their findings (NFR21)
**And** the failed rule is reported as an error in the response

**Given** captured signal samples in findings
**When** the response is built
**Then** sample values are truncated to prevent leaking full payloads (NFR13)

---

## Epic 6: Fix Suggestion & Application

Engineers receive actionable fix suggestions (complete YAML config blocks) for detected issues and can apply them one-by-one with automatic health checks — OTTL transforms, filter rules, attributes processors, resource processors, each with risk assessment.

### Story 6.1: Fix Generator Framework & Cardinality Fix

As an observability engineer,
I want the MCP server to generate OTTL transform and attributes processor fixes for high-cardinality metrics,
So that I get ready-to-apply config blocks that reduce cardinality at the collector level.

**Acceptance Criteria:**

**Given** `pkg/fixes/types.go`
**When** the framework is reviewed
**Then** it defines `FixGenerator` interface, `FixSuggestion` struct (finding_index, fix_type, description, processor_config YAML, pipeline_changes, risk level)

**Given** a high-cardinality finding identifying specific label keys
**When** `fix_cardinality.go` generates a fix
**Then** it produces an attributes processor config that drops the high-cardinality label keys (FR33)
**And** the fix includes a complete YAML config block ready to apply, the target pipeline, and risk assessment (FR35)

**Given** a high-cardinality finding with OTTL-addressable dimensions
**When** the fix generator runs
**Then** it can also produce an OTTL transform processor statement as an alternative (FR31)

### Story 6.2: PII Redaction & Missing Resource Fixes

As an observability engineer,
I want the MCP server to generate OTTL redaction transforms for PII and resource processor configs for missing attributes,
So that I can fix compliance issues and service identification gaps with ready-to-apply config blocks.

**Acceptance Criteria:**

**Given** a PII finding with email addresses in a specific attribute
**When** `fix_pii.go` generates a fix
**Then** it produces an OTTL `replace_pattern` statement that redacts the PII (FR31)
**And** the fix includes the complete transform processor config block with the OTTL statement

**Given** a PII finding with credit card numbers
**When** the fix generator runs
**Then** it produces a `delete_key` statement to remove the attribute entirely

**Given** a PII finding with IP addresses
**When** the fix generator runs
**Then** it produces an OTTL `replace_pattern` that redacts the IP

**Given** a missing resource attribute finding (e.g., `service.version` missing)
**When** `fix_missing_resource.go` generates a fix
**Then** it produces a resource processor config that adds the missing attribute with a placeholder value (FR34)

### Story 6.3: Bloated Attribute & Duplicate Signal Fixes

As an observability engineer,
I want the MCP server to generate fixes for bloated attributes and duplicate signals,
So that I can reduce storage costs and eliminate redundant data collection.

**Acceptance Criteria:**

**Given** a bloated attribute finding (value >1KB)
**When** `fix_bloated_attrs.go` generates a fix
**Then** it produces an OTTL transform processor statement to truncate or delete the attribute (FR31)

**Given** a duplicate signal finding
**When** `fix_duplicate_signals.go` generates a fix
**Then** it produces a filter processor rule to drop the duplicate metric (FR32)

**Given** any generated fix
**When** the fix suggestion is reviewed
**Then** it includes `fix_type` (ottl|filter|attribute|resource|config), `description`, complete `processor_config` YAML, `pipeline_changes`, and `risk` (low|medium|high) (FR35)

### Story 6.4: suggest_fixes Tool

As an observability engineer,
I want a `suggest_fixes` MCP tool that generates fix suggestions for detected issues,
So that I can review proposed config changes before applying them.

**Acceptance Criteria:**

**Given** an active session with detection findings that have `fix_available=true`
**When** `suggest_fixes` is called with `session_id`
**Then** fix generators run for all fixable findings and return suggestions in markdown table format

**Given** `suggest_fixes` is called with `finding_index` specified
**When** the tool executes
**Then** it generates a fix only for the specified finding

**Given** a finding with no fix available
**When** `suggest_fixes` runs
**Then** it skips that finding and reports it as "no automated fix available"

**Given** generated suggestions
**When** the response is reviewed
**Then** each suggestion includes the finding it addresses, complete YAML config block, target pipeline, and risk assessment

### Story 6.5: apply_fix Tool

As an observability engineer,
I want an `apply_fix` MCP tool that applies a single user-approved fix with automatic health checking,
So that I can iteratively improve my collector config with safety guarantees.

**Acceptance Criteria:**

**Given** an active session with fix suggestions
**When** `apply_fix` is called with `session_id` and `suggestion_index`
**Then** it applies ONLY the specified fix to the collector config, triggers rollout, and runs health check (FR36)
**And** presents each fix individually — no batch auto-apply (FR37)

**Given** a fix is applied and health check passes
**When** the response is returned
**Then** it includes the applied fix details and `status: fix_applied` with health check results

**Given** a fix is applied and health check detects CrashLoopBackOff
**When** automatic rollback triggers
**Then** the fix is reverted, collector restored to pre-fix state, and response includes `status: rolled_back` with failure reason (FR8)

**Given** `apply_fix` is called with an out-of-range `suggestion_index`
**When** the tool validates
**Then** it returns a clear error listing available suggestion indices

---

## Epic 7: Sampling & Sizing Recommendations

Engineers get data-driven sampling strategy recommendations (tail/probabilistic/hybrid with complete config) and resource sizing recommendations based on observed throughput.

### Story 7.1: recommend_sampling Tool

As an observability engineer,
I want a `recommend_sampling` MCP tool that analyzes captured trace data and recommends a sampling strategy,
So that I can reduce trace volume with confidence based on actual traffic patterns.

**Acceptance Criteria:**

**Given** an active session with captured trace data
**When** `recommend_sampling` is called with `session_id`
**Then** it analyzes: total spans, error spans, error rate, p99 duration, unique services (FR38)
**And** recommends a strategy: `tail_sampling`, `probabilistic`, or `hybrid` with rationale

**Given** a recommendation for tail sampling
**When** the response is returned
**Then** it includes a complete tail sampling processor config block based on observed patterns — error-biased, latency-biased, or probabilistic (FR39)
**And** estimated volume reduction percentage

**Given** captured data with very low trace volume
**When** the tool analyzes
**Then** it may recommend no sampling with an explanation

**Given** the response format
**When** reviewed
**Then** it includes `trace_analysis` stats and `recommendation` with strategy, config YAML, estimated reduction, and rationale in markdown table format

### Story 7.2: recommend_sizing Tool

As an observability engineer,
I want a `recommend_sizing` MCP tool that estimates recommended resource limits based on observed throughput,
So that I can right-size my collector resources instead of guessing.

**Acceptance Criteria:**

**Given** an active session with captured signal data
**When** `recommend_sizing` is called with `session_id`
**Then** it calculates observed throughput: metrics/sec, logs/sec, spans/sec (FR40)
**And** retrieves current resource requests/limits from the collector's pod spec

**Given** observed throughput data
**When** sizing recommendations are generated
**Then** recommended CPU and memory requests/limits include headroom for traffic spikes
**And** the response includes rationale explaining the recommendation basis

**Given** the collector has no resource limits set
**When** the tool runs
**Then** it recommends initial limits based on observed throughput and flags the missing limits as a concern

**Given** the response format
**When** reviewed
**Then** it includes `observed_throughput`, `current_resources`, and `recommendation` sections in markdown table format

---

## Epic 8: v2 Self-Instrumentation

Operations teams can observe v2 analysis workflows in their observability backend — session durations, signal capture counts, detection hits, fix outcomes, rollback events, health check results, and active backup counts, all correlated via trace IDs.

### Story 8.1: v2 Metrics Instruments

As a platform engineer,
I want the MCP server to emit v2-specific OTel metrics for analysis workflows,
So that I can monitor v2 usage patterns, detection effectiveness, and operational health in my observability backend.

**Acceptance Criteria:**

**Given** `pkg/telemetry/metrics.go` is extended
**When** the new metric instruments are reviewed
**Then** 8 new instruments are registered:
- `mcp.analysis.duration_seconds` (histogram, labels: collector_name, namespace)
- `mcp.analysis.sessions_total` (counter, labels: environment, outcome)
- `mcp.capture.signals_total` (counter, labels: signal_type)
- `mcp.detection.hits_total` (counter, labels: rule, severity)
- `mcp.fixes.applied_total` (counter, labels: fix_type, outcome)
- `mcp.rollbacks.total` (counter, labels: trigger, reason)
- `mcp.health_checks.total` (counter, labels: result)
- `mcp.backup.active` (gauge, labels: collector_name, namespace) (FR50)

**Given** a complete analysis workflow (start → capture → detect → fix → cleanup)
**When** the workflow completes
**Then** all relevant metrics are incremented at each step

**Given** the existing v1 telemetry framework
**When** v2 metrics are added
**Then** they are additive to the existing `Metrics` struct — no v1 metrics are modified

### Story 8.2: v2 Spans, Child Spans & Structured Mutation Logging

As a platform engineer,
I want v2 tool calls to produce OTel spans with child spans for multi-step operations and structured mutation logging,
So that I can trace the full analysis workflow and audit all config changes.

**Acceptance Criteria:**

**Given** any v2 tool call
**When** it executes
**Then** it produces a span following GenAI/MCP semantic conventions inherited from v1 instrumentation (FR49)

**Given** `start_analysis` executes
**When** its span is examined
**Then** it contains child spans for: environment validation, config backup, debug exporter injection, config apply, and health check

**Given** `capture_signals` completes
**When** its span attributes are examined
**Then** they include `capture.duration_seconds`, `capture.metrics_count`, `capture.logs_count`, `capture.spans_count`

**Given** `apply_fix` executes
**When** its span is examined
**Then** it contains child spans for: config mutation, rollout trigger, and health check

**Given** `rollback_config` executes
**When** its span attributes are examined
**Then** they include `rollback.trigger` (manual|automatic) and `rollback.reason`

**Given** any config mutation (apply, rollback)
**When** the operation completes
**Then** an INFO-level structured log entry is emitted with `trace_id`, `span_id`, session_id, collector name, namespace, and mutation type (FR51)

**Given** a production environment refusal
**When** the refusal occurs
**Then** a WARN-level structured log entry is emitted with full context (FR52)

---

## Epic 9: v2 Documentation

Engineers have comprehensive guides to adopt v2 — tool reference for all 10 new tools, safety model explainer, v1→v2 migration guide, and detection rules reference with thresholds and examples.

### Story 9.1: v2 Tool Reference & Safety Model Guide

As an observability engineer,
I want comprehensive documentation for all v2 tools and the safety model,
So that I understand how to use v2 features and trust the safety guarantees.

**Acceptance Criteria:**

**Given** `docs/docs/tools/index.md` is updated
**When** the v2 tool reference is reviewed
**Then** all 10 v2 tools are documented with: description, input parameters, output fields, example invocations, and sample output in markdown table format (FR53)

**Given** `docs/docs/v2/safety-model.md` is created
**When** the safety model guide is reviewed
**Then** it explains: environment gate (user-declared, no heuristics), config backup (ConfigMap annotations, survives restarts), automatic health checks (30s timeout, 2s poll), automatic rollback (on CrashLoopBackOff or readiness failure), concurrent session rejection, and session TTL cleanup (FR54)
**And** includes a diagram showing the safety chain flow

### Story 9.2: Migration Guide & Detection Rules Reference

As a platform engineer,
I want a migration guide from v1 to v2 and a detection rules reference,
So that I can upgrade existing deployments and understand what v2 detects.

**Acceptance Criteria:**

**Given** `docs/docs/v2/migration.md` is created
**When** the migration guide is reviewed
**Then** it covers: RBAC changes (new write permissions), Helm values (`v2.enabled`, `v2.sessionTTL`, `v2.maxConcurrentSessions`), backward compatibility (v1 tools unchanged), and step-by-step upgrade instructions (FR55)

**Given** `docs/docs/v2/detection-rules.md` is created
**When** the detection rules reference is reviewed
**Then** all 8 runtime detection rules are documented with: rule name, description, threshold values, example findings, false positive mitigation strategies, and which fix generators address each rule (FR56)

**Given** `docs/mkdocs.yml` is updated
**When** the nav structure is reviewed
**Then** v2 documentation pages are included in the site navigation

**Given** existing docs (`getting-started.md`, `contributing.md`, `troubleshooting.md`)
**When** they are updated
**Then** they reference v2 enablement, adding runtime rules, and v2-specific troubleshooting
