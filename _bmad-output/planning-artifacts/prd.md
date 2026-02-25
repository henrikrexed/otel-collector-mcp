---
stepsCompleted: [step-01-init, step-02-discovery, step-02b-vision, step-02c-executive-summary, step-03-success, step-04-journeys, step-05-domain, step-06-innovation, step-07-project-type, step-08-scoping, step-09-functional, step-10-nonfunctional, step-11-polish, step-12-complete]
inputDocuments: ['product-brief-otel-collector-mcp-2026-02-25.md', 'conversation-brainstorming-context', 'conversation-party-mode-debate']
workflowType: 'prd'
classification:
  projectType: developer_tool
  domain: observability_infrastructure
  complexity: high
  projectContext: greenfield
---

# Product Requirements Document - otel-collector-mcp

**Author:** Henrik.rexed
**Date:** 2026-02-25

## Executive Summary

otel-collector-mcp is an MCP server (Go, in-cluster Kubernetes deployment) that provides context-aware troubleshooting and design guidance for OpenTelemetry Collector pipelines. It targets the critical gap between the OTel Collector's power and its usability: engineers today iterate on pipeline configs through painful trial-and-error — change YAML, restart collector, read cryptic logs, guess what's wrong, repeat. This MCP eliminates that loop with instant, context-aware diagnosis and remediation.

The tool serves three personas: Observability Engineers writing pipeline configs (80% of usage — design-time iteration), Platform Engineers managing collector infrastructure (Operator CRDs, Helm, scaling, security), and SREs troubleshooting production incidents (less frequent but time-critical). It communicates via MCP protocol (SSE transport), integrating directly into MCP-compatible AI assistants and IDEs.

### What Makes This Special

- **Collector-native context**: Understands OTel Collector concepts (pipelines, receivers, processors, exporters, connectors) as first-class entities, not generic YAML
- **Kubernetes-aware**: Auto-detects deployment mode (DaemonSet/Deployment/StatefulSet/Operator CRD) and applies mode-specific best practices
- **Detection + remediation in one flow**: Diagnoses problems and provides the specific config change to fix them in a single interaction
- **Opinionated architecture guidance**: Encodes production-proven patterns (Agent→Gateway hybrid, DaemonSet for logs, StatefulSet for Target Allocator) rather than presenting all options as equal
- **Design-time focus**: Optimized for the 80% case — engineers iterating on configs that don't work yet

## Project Classification

- **Project Type:** Developer Tool (MCP Server)
- **Domain:** Observability Infrastructure (Kubernetes + OpenTelemetry)
- **Complexity:** High — deep domain knowledge required (OTel Collector internals, Kubernetes deployment patterns, OTel Operator, OTTL syntax, Prometheus ecosystem)
- **Project Context:** Greenfield

## Success Criteria

### User Success

- Observability Engineers resolve collector config issues in under 1 minute (down from 15-30 minutes of log reading)
- Config iterations to a working pipeline reduced by 50%+
- Triage scan surfaces 3-5 actionable findings on a typical collector deployment
- Remediation suggestions are directly applicable — >70% applied as-is without modification
- Platform Engineers get architecture recommendations that match documented OTel best practices
- SREs can triage a collector incident at 3 AM without deep collector expertise

### Business Success

- Become the go-to MCP for OTel Collector troubleshooting in the Kubernetes community within 12 months
- Build an active contributor community around detection rules and remediation patterns
- Reduce barrier to OTel Collector adoption — fewer teams abandon OTel due to config complexity
- Establish as a reference implementation for domain-specific MCP servers in the observability space

### Technical Success

- All detection rules produce zero false positives on known-good configurations
- OTTL generation produces syntactically valid transform processor configurations
- MCP server starts and responds within 5 seconds of pod readiness
- Tool correctly identifies deployment type for all four modes (DaemonSet, Deployment, StatefulSet, Operator CRD)
- In-cluster Kubernetes API calls complete within 2 seconds under normal load

### Measurable Outcomes

| Metric | 3-Month Target | 12-Month Target |
|--------|---------------|-----------------|
| GitHub stars | 200+ | 1,000+ |
| Monthly active clusters (opt-in telemetry) | 50+ | 500+ |
| Community-contributed detection rules | 5+ | 25+ |
| External contributor issues/PRs | 10+ | 50+ |
| Detection rule accuracy (true positive rate) | >90% | >95% |

## Product Scope

### MVP - Minimum Viable Product

**MVP Strategy:** Problem-solving MVP — ship the smallest set of tools that transforms the collector config iteration loop from "guess and check" to "instant diagnosis and fix."

**Core Capabilities:**
1. Auto-detect collector deployment type (DaemonSet/Deployment/StatefulSet/Operator CRD)
2. Triage scan — single entry point running all detectors, returning prioritized issue list
3. Collector log parsing — OTTL syntax errors, exporter failures, OOM events, receiver issues
4. Operator log parsing — rejected CRDs, reconciliation failures
5. Misconfig detection suite (10+ detection rules)
6. Exporter backpressure / silent data loss detection
7. Detection → remediation as a single flow
8. Architecture design guidance (opinionated deployment topology recommendations)
9. OTTL transform generation via transform processor

### Growth Features (Post-MVP)

**Phase 2 — Expanded Proactive Skills:**
- Tail sampling configuration (errors + slow transactions + probabilistic)
- Cumulative-to-delta conversion guidance for delta backends (Dynatrace, etc.)
- Security hardening skill (env vars / K8s secrets patterns)
- High-cardinality remediation with OTTL transform generation
- Target Allocator setup for Prometheus scraping at scale
- zpages integration for live pipeline health

**Phase 3 — Instrumentation & Beyond:**
- Instrumentation CRD configuration (auto-instrumentation: Java, Python, Node.js, .NET, Go)
- Pod annotation suggestions for SDK injection
- OBI (OpenTelemetry eBPF Instrumentation) deployment guidance
- SDK vs OBI vs hybrid instrumentation guidance
- Pre-deployment YAML validation

### Vision (Future)

- Community-driven detection rule library (pluggable patterns contributed by OTel community)
- CI/CD pipeline integration for automated collector config validation
- Cross-MCP orchestration with k8s-networking-mcp for full-stack observability troubleshooting
- Support for non-Kubernetes deployments (Docker, bare metal, cloud-managed collectors)

## User Journeys

### Journey 1: Alex the Observability Engineer — Design-Time Config Iteration (Primary)

Alex is a mid-level observability engineer at a mid-size company. They've been tasked with setting up a new pipeline to collect application logs from Kubernetes pods, parse them with OTTL transforms, and send them to their Loki backend. They've been fighting with the config for two hours.

**Opening Scene:** Alex has a collector config with a transform processor that keeps failing. The collector logs show `error: failed to parse OTTL statement` followed by an incomprehensible Go stack trace. They've tried three variations of the OTTL syntax and none work. They're about to open a GitHub issue.

**Rising Action:** Alex connects their IDE to otel-collector-mcp and asks "What's wrong with my collector?" The MCP auto-detects the DaemonSet deployment, pulls the collector logs, and identifies the exact OTTL syntax error — a missing parenthesis in a `replace_pattern` call. But it doesn't stop there: it also flags that the pipeline has no memory limiter, the exporter has no retry/queue config, and there's a hardcoded API token in the exporter section.

**Climax:** The MCP returns all four issues in a prioritized list with the exact config changes for each. Alex applies the OTTL fix and the pipeline starts working. They then address the other three issues before the config reaches production review.

**Resolution:** What was a two-hour frustration becomes a 5-minute fix-and-improve cycle. Alex starts using the MCP as part of their standard config development workflow. When they need to add a new pipeline for trace sampling, they ask the MCP to generate the OTTL transforms instead of guessing at syntax.

### Journey 2: Jordan the Platform Engineer — Architecture Design (Primary)

Jordan is a senior platform engineer responsible for the company's observability infrastructure. They're migrating from a single monolithic collector Deployment to a scalable multi-pipeline architecture to handle 500+ microservices. They need to decide: how should collectors be deployed?

**Opening Scene:** Jordan knows they need DaemonSets for log collection but isn't sure about the metrics and traces topology. Should they use a gateway pattern? StatefulSet for Target Allocator? How does the Agent→Gateway hybrid work in practice?

**Rising Action:** Jordan asks the MCP to design a collector architecture for their use case. They provide context: 500 microservices, need log collection from nodes, Prometheus metric scraping via Target Allocator, and trace collection forwarded to Tempo. The MCP returns an opinionated architecture: DaemonSet agents for logs (needs /var/log access), a StatefulSet gateway with Target Allocator for Prometheus scraping, and a Deployment gateway for traces. It includes the rationale for each decision.

**Climax:** The MCP generates the skeleton CRD configs for each component and flags that the current Operator version has a known issue with StatefulSet scaling that requires a specific annotation. Jordan would have hit this issue two days into implementation.

**Resolution:** Jordan has a production-ready architecture plan in 20 minutes instead of 2 days of research. They submit it for team review with confidence, and the architecture survives peer review without major changes.

### Journey 3: Sam the SRE — 3 AM Production Triage (Secondary)

Sam is an SRE who got paged because Grafana dashboards show a gap in metrics for the last 30 minutes. Sam is not a collector expert — they know Kubernetes, but the collector pipeline is Jordan's domain. Jordan is on PTO.

**Opening Scene:** Sam sees metrics gaps in Grafana. They check the collector pods — all running, no restarts, no OOM kills. Logs show no obvious errors. Something is silently failing.

**Rising Action:** Sam connects to the MCP and runs a triage scan across all collector instances in the monitoring namespace. The MCP detects that the gateway collector's exporter queue is full — backpressure from the Datadog endpoint is causing silent data loss. It also detects that the retry configuration has `max_elapsed_time: 0` which disables retries entirely.

**Climax:** The MCP provides two remediation options: (1) immediate — increase queue size to buffer the backpressure, and (2) proper fix — configure exponential backoff retry with a 5-minute max elapsed time. Sam applies the immediate fix and the metrics gap closes within minutes.

**Resolution:** Sam resolves the incident without waking Jordan. They apply the proper fix the next morning and add a detection alert for exporter queue saturation. Total incident time: 12 minutes instead of the usual 1-2 hours of log archaeology.

### Journey 4: Platform Engineer — Operator Troubleshooting (Edge Case)

Jordan applies a new OpenTelemetryCollector CRD but the collector pods don't come up. `kubectl get otelcol` shows the resource exists but no pods are created.

**Opening Scene:** The CRD appears accepted by Kubernetes (no API error) but the Operator isn't reconciling it into running pods. `kubectl describe otelcol` shows no events.

**Rising Action:** Jordan asks the MCP to check Operator status. The MCP parses the OTel Operator logs and identifies a reconciliation failure: the CRD specifies a `targetAllocator` section but the Operator was deployed without the Target Allocator feature flag enabled. The error is buried in the Operator's structured logs at debug level.

**Climax:** The MCP provides the fix: either remove the targetAllocator section or redeploy the Operator with `--feature-gates=operator.targetallocator.enabled`. It includes the Helm values change needed.

**Resolution:** Jordan enables the feature gate, the Operator reconciles, and the collector pods start. A problem that could have taken hours of Operator log debugging was resolved in minutes.

### Journey Requirements Summary

| Journey | Capabilities Revealed |
|---------|----------------------|
| Alex (Config Iteration) | Log parsing, OTTL error detection, misconfig detection, remediation generation, OTTL transform generation |
| Jordan (Architecture) | Deployment type detection, architecture design skill, CRD awareness, opinionated guidance |
| Sam (Production Triage) | Triage scan, exporter backpressure detection, remediation generation, silent failure detection |
| Jordan (Operator) | Operator log parsing, CRD reconciliation diagnosis, feature gate awareness |

## Domain-Specific Requirements

### OpenTelemetry Collector Domain Knowledge

The MCP must encode deep knowledge of:

- **Collector component taxonomy**: receivers (OTLP, Prometheus, filelog, hostmetrics), processors (batch, memory_limiter, transform, filter, tail_sampling, resourcedetection), exporters (OTLP, Prometheus, Datadog, Dynatrace, Loki, Tempo), connectors (spanmetrics, count, routing)
- **Pipeline semantics**: signal types (logs, metrics, traces), pipeline topology (single-pipeline, multi-pipeline, fan-out, fan-in), connector role in cross-signal pipelines
- **Deployment patterns**: DaemonSet (agent mode, node-level access), Deployment (gateway mode, stateless), StatefulSet (gateway mode with Target Allocator, stateful), sidecar (edge cases only)
- **OTel Operator**: OpenTelemetryCollector CRD, Instrumentation CRD, Target Allocator, OpAMP Bridge, feature gates, reconciliation lifecycle
- **OTTL (OpenTelemetry Transformation Language)**: syntax, functions, contexts (log, span, metric, resource), common patterns

### Anti-Pattern Knowledge Base

The MCP must detect and explain these anti-patterns:

| Anti-Pattern | Why It's Bad | Detection Method |
|-------------|-------------|-----------------|
| Tail sampling on DaemonSet | Sampling decisions split across nodes produce inconsistent results | Config analysis: tail_sampling processor + DaemonSet deployment |
| Missing memory_limiter processor | Collector OOMs under load spikes | Config analysis: processor pipeline missing memory_limiter |
| Missing batch processor | Inefficient exporter utilization, increased network overhead | Config analysis: processor pipeline missing batch |
| Hardcoded tokens in config | Security vulnerability, tokens visible in CRDs and ConfigMaps | Config analysis: string pattern matching in exporter configs |
| No retry/queue on exporters | Transient backend failures cause permanent data loss | Config analysis: exporter section missing retry_on_failure and sending_queue |
| Invalid regex in filter processor | Collector crashes or silently drops all data | Log analysis: regex compilation errors |
| Resource detector conflicts | Detectors overwrite each other's resource attributes | Config analysis: multiple resourcedetection processors with overlapping detectors |
| Exporter queue full | Silent data loss under sustained backpressure | Log analysis: queue capacity warnings |
| Cumulative→delta without stateful storage | Metric conversion produces incorrect results after pod restart | Config analysis: cumulativetodelta processor + Deployment (non-StatefulSet) |
| Wrong receiver port bindings | Collector appears healthy but receives no data | Config analysis: receiver endpoint vs service port mismatch |

### Kubernetes Integration Requirements

- Read-only access to Kubernetes API via client-go (pods, deployments, daemonsets, statefulsets, configmaps, CRDs)
- RBAC: ClusterRole with read access to otel-related resources across namespaces
- Log access: ability to stream/tail pod logs for collectors and the OTel Operator
- CRD awareness: parse OpenTelemetryCollector CRD spec and status

## Innovation & Novel Patterns

### Detected Innovation Areas

**MCP as a Domain-Specific Diagnostic Engine:** While MCP servers exist for general Kubernetes operations, applying the MCP protocol specifically as a diagnostic + remediation engine for a single complex system (OTel Collector) is novel. The innovation is in the depth of domain encoding — not just "read logs" but "understand what this OTTL error means and generate the fix."

**Detection-Remediation Coupling:** Existing tools separate detection (monitoring/alerting) from remediation (runbooks/documentation). otel-collector-mcp couples them into a single interaction: detect the issue, understand the context, generate the specific fix. This is closer to a domain expert's thought process than a traditional tool chain.

### Validation Approach

- Validate detection accuracy against a corpus of known-bad collector configurations
- Measure remediation acceptance rate — do engineers apply the suggested fix or modify it?
- Track time-to-resolution before and after MCP adoption
- Gather qualitative feedback from OTel community early adopters

### Risk Mitigation

- **Risk:** Detection rules produce false positives → **Mitigation:** Conservative detection thresholds, clearly label confidence levels, allow user to disable rules
- **Risk:** Generated OTTL/configs have syntax errors → **Mitigation:** Validate generated configs against OTel Collector config schema before presenting
- **Risk:** Kubernetes API access is too broad → **Mitigation:** Minimal RBAC, read-only operations only, namespace-scoped option for restrictive clusters

## MCP Server Specific Requirements

### Project-Type Overview

This is an MCP server — it exposes capabilities as MCP tools (reactive, invoked by the AI assistant) and MCP skills/prompts (proactive, offering guided workflows). It does not have a UI. All interaction happens through the MCP protocol.

### Technical Architecture Considerations

- **Transport:** SSE (Server-Sent Events) for MCP communication
- **SDK:** Official Go MCP SDK
- **Deployment:** In-cluster Kubernetes pod (Deployment), Helm chart for installation
- **Kubernetes Client:** client-go with in-cluster config
- **Logging:** slog (structured logging)
- **Observability:** Self-instrumented with OTel SDK (dogfooding)
- **Configuration:** Environment variables and/or ConfigMap for MCP server settings

### MCP Tool Design

Each reactive capability is exposed as an MCP tool with:
- Clear tool name and description (consumed by AI assistants for tool selection)
- Typed input parameters (namespace, collector name, etc.)
- Structured output (issues list with severity, description, remediation)

### MCP Prompt/Skill Design

Each proactive capability is exposed as an MCP prompt/resource with:
- Guided workflow that asks clarifying questions
- Structured output (config snippets, architecture recommendations)
- Context-aware suggestions based on detected cluster state

### Implementation Considerations

- **Stateless design:** No persistent storage required; all state comes from Kubernetes API and collector logs at query time
- **Concurrency:** Multiple tool invocations may run in parallel; Kubernetes client must be safe for concurrent use
- **Error handling:** Graceful degradation when Kubernetes API is unreachable or RBAC is insufficient — report what access is missing rather than crashing
- **Sibling MCP:** k8s-networking-mcp handles connectivity concerns; otel-collector-mcp can recommend invoking it for network-related issues but shares no state

## Project Scoping & Phased Development

### MVP Strategy & Philosophy

**MVP Approach:** Problem-solving MVP — validate the core assumption that an MCP server with deep OTel Collector context can meaningfully accelerate config troubleshooting and design.

**Resource Requirements:** Single developer. Go backend, Kubernetes client-go, MCP SDK. No external dependencies beyond the Kubernetes API.

### MVP Feature Set (Phase 1)

**Core User Journeys Supported:**
- Alex: Config iteration with instant diagnosis + fix (primary)
- Jordan: Architecture design guidance (primary)
- Sam: Production triage scan (secondary)
- Jordan: Operator troubleshooting (edge case)

**Must-Have Capabilities:**
1. Deployment type auto-detection
2. Triage scan (unified entry point)
3. Collector log parsing and error classification
4. Operator log parsing for CRD/reconciliation issues
5. 10+ misconfig detection rules with remediation
6. Exporter backpressure detection
7. Architecture design skill
8. OTTL transform generation skill
9. Helm chart for in-cluster deployment

### Post-MVP Features

**Phase 2 (v2):**
- Tail sampling config generation
- Cumulative→delta guidance
- Security hardening skill
- High-cardinality remediation
- Target Allocator setup skill
- zpages health checking
- Community detection rule plugin system

**Phase 3 (v3):**
- Instrumentation CRD configuration
- OBI deployment guidance
- SDK vs OBI guidance
- Pre-deployment YAML validation
- CI/CD integration
- Cross-MCP orchestration with k8s-networking-mcp

### Risk Mitigation Strategy

**Technical Risks:**
- OTel Collector config schema changes between versions → Pin to supported version range, test against multiple collector versions
- Go MCP SDK is relatively new → Contribute upstream fixes, abstract SDK usage behind internal interfaces
- Kubernetes RBAC varies widely across clusters → Provide clear RBAC requirements, graceful degradation when access is restricted

**Market Risks:**
- OTel community may build similar functionality into the Operator → Differentiate on design-time iteration speed and MCP integration
- Low adoption if discovery is poor → Engage with OTel community early (CNCF Slack, KubeCon talks, blog posts)

**Resource Risks:**
- Single developer bottleneck → Prioritize community contributions, make detection rules easy to add
- Scope creep into instrumentation/OBI → Hard boundary at v1 scope, explicit v2/v3 gating

## Functional Requirements

### Collector Discovery & Detection

- FR1: MCP server can auto-detect the deployment type (DaemonSet, Deployment, StatefulSet, OTel Operator CRD) of any OTel Collector instance in the cluster
- FR2: MCP server can list all OTel Collector instances across all namespaces (or a specified namespace)
- FR3: MCP server can retrieve the running configuration of a detected collector instance
- FR4: MCP server can identify the OTel Collector version running in each instance

### Triage & Diagnosis

- FR5: MCP server can execute a triage scan that runs all detection rules against a specified collector and returns a prioritized issue list
- FR6: Each issue in the triage scan result includes severity (critical/warning/info), description, affected config section, and specific remediation
- FR7: MCP server can parse collector pod logs and classify errors into categories (OTTL syntax, exporter failure, OOM, receiver issue, processor error)
- FR8: MCP server can parse OTel Operator pod logs and detect rejected CRDs and reconciliation failures
- FR9: MCP server can detect missing batch processor in any pipeline
- FR10: MCP server can detect missing memory_limiter processor in any pipeline
- FR11: MCP server can detect hardcoded tokens or credentials in exporter configurations
- FR12: MCP server can detect missing retry_on_failure and sending_queue on exporters
- FR13: MCP server can detect receiver endpoint / port binding mismatches
- FR14: MCP server can detect tail_sampling processor configured on a DaemonSet deployment
- FR15: MCP server can detect invalid regex patterns in filter processor configurations
- FR16: MCP server can detect connector misconfigurations (e.g., referencing non-existent pipelines)
- FR17: MCP server can detect conflicting resource detection processors that overwrite each other
- FR18: MCP server can detect cumulative-to-delta conversion issues (e.g., non-stateful deployment without persistent storage)
- FR19: MCP server can detect high-cardinality metric labels by analyzing metric metadata or config patterns
- FR20: MCP server can detect exporter queue saturation and backpressure conditions from collector logs

### Remediation

- FR21: Every detection (FR9-FR20) includes a specific remediation: the exact config change, addition, or removal needed to fix the issue
- FR22: Remediation output includes the corrected YAML config block that can be directly applied
- FR23: MCP server can explain why a detected issue is problematic (impact description) alongside the fix

### Architecture Design Guidance

- FR24: MCP server can recommend a collector deployment topology based on user-described workload (signal types, scale, backend targets)
- FR25: Architecture recommendations include opinionated rationale for each decision (why DaemonSet for logs, why StatefulSet for Target Allocator, etc.)
- FR26: Architecture recommendations include the hybrid Agent→Gateway pattern when appropriate
- FR27: MCP server can generate skeleton collector configuration for a recommended architecture

### OTTL Transform Generation

- FR28: MCP server can generate OTTL transform processor statements for log parsing use cases described in natural language
- FR29: MCP server can generate OTTL transform processor statements for span attribute manipulation
- FR30: MCP server can generate OTTL transform processor statements for metric label operations (rename, drop, aggregate)
- FR31: Generated OTTL statements include the complete transform processor config block (not just the OTTL expression)

### Infrastructure & Deployment

- FR32: MCP server can be deployed in-cluster via Helm chart with configurable RBAC, namespace scope, resource limits, and replica count
- FR33: MCP server exposes health and readiness endpoints for Kubernetes liveness/readiness probes
- FR34: MCP server emits structured logs (slog) with configurable log level
- FR35: MCP server instruments itself with OpenTelemetry SDK (traces and metrics) for observability
- FR36: Helm chart exposes the MCP server via Gateway API HTTPRoute resource
- FR37: Helm chart supports configurable gateway provider (Istio, Envoy Gateway, Cilium, NGINX, kgateway) as a chart variable
- FR38: Helm chart includes configurable SecurityContext (runAsNonRoot, readOnlyRootFilesystem, allowPrivilegeEscalation: false)

### Multi-Cluster Identity

- FR39: Every MCP tool response includes cluster identification (cluster name, Kubernetes context, MCP server namespace) so an AI agent connected to multiple otel-collector-mcp instances can distinguish which cluster data comes from
- FR40: Cluster identification fields are present in every `StandardResponse` and `DiagnosticFinding` output, not just top-level metadata
- FR41: Cluster name is configurable via Helm chart value and environment variable (`CLUSTER_NAME`)

### CI/CD Pipeline

- FR42: GitHub Actions workflow builds the Go binary for linux/amd64 and linux/arm64
- FR43: GitHub Actions workflow runs unit tests and reports results
- FR44: GitHub Actions workflow runs integration tests against a Kubernetes test cluster (or mocks)
- FR45: GitHub Actions workflow builds multi-arch Docker image (amd64 + arm64) and publishes as GitHub artifact / container registry
- FR46: GitHub Actions workflow runs security scanning: Trivy (container image), gosec (Go source), govulncheck (Go dependencies)
- FR47: GitHub Actions workflow runs golangci-lint for code quality
- FR48: GitHub Actions workflow builds MkDocs documentation site and deploys to GitHub Pages

### User-Facing Documentation

- FR49: MkDocs documentation site includes a Getting Started guide covering deployment (Helm), connecting to an AI assistant, and running first triage scan
- FR50: MkDocs documentation site includes a Tool Reference documenting each MCP tool with parameters, examples, and sample output
- FR51: MkDocs documentation site includes a Skills Reference documenting each MCP skill with use cases and generated output examples
- FR52: MkDocs documentation site includes an Architecture Guide covering deployment patterns, multi-cluster setup, and Gateway API exposure
- FR53: MkDocs documentation site includes a Contributing Guide for adding detection rules, skills, and documentation
- FR54: MkDocs documentation site includes a Troubleshooting Guide for common installation and connectivity issues

## Non-Functional Requirements

### Performance

- MCP tool responses complete within 10 seconds for single-collector operations under normal cluster load
- Triage scan (all detectors) completes within 30 seconds for a single collector instance
- Log parsing processes the most recent 1000 log lines within 5 seconds
- MCP server cold start to ready state within 15 seconds

### Security

- MCP server operates with read-only Kubernetes RBAC — no write operations to cluster resources
- No secrets or credentials stored in the MCP server's own configuration or memory beyond what client-go uses for in-cluster auth
- All detection rules that flag hardcoded credentials do not include the credential values in their output
- MCP SSE transport supports TLS when fronted by a Kubernetes Service or Ingress

### Scalability

- Single MCP server instance supports up to 50 concurrent collector instances in the cluster
- Tool invocations are stateless — horizontal scaling via additional replicas if needed
- Memory footprint remains under 256MB for clusters with up to 100 collector pods

### Reliability

- MCP server gracefully handles Kubernetes API timeouts and returns partial results with clear indication of what couldn't be checked
- Detection rules fail independently — one rule failure does not prevent other rules from executing
- MCP server recovers automatically from transient Kubernetes API errors without requiring pod restart

### Integration

- MCP server supports MCP protocol via SSE transport as defined by the MCP specification
- Compatible with any MCP-capable client (Claude Desktop, IDE extensions, custom clients)
- k8s-networking-mcp integration: MCP server can suggest invoking the sibling MCP for network-related diagnosis but does not depend on it being present
