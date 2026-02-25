---
stepsCompleted: [1, 2, 3, 4, 5, 6]
inputDocuments: ['conversation-brainstorming-context', 'conversation-party-mode-debate']
date: 2026-02-25
author: Henrik.rexed
status: complete
---

# Product Brief: otel-collector-mcp

## Executive Summary

**otel-collector-mcp** is an MCP server, written in Go and deployed in-cluster, that helps engineers troubleshoot and design OpenTelemetry Collector pipelines in Kubernetes. It targets the critical gap between the OTel Collector's power and its usability: today, engineers iterate on pipeline configs through painful trial-and-error — change YAML, restart the collector, read cryptic logs, guess what's wrong, repeat. This MCP eliminates that loop by providing instant, context-aware diagnosis and remediation.

The core insight is that most collector issues happen at **design time**, not in production. Engineers aren't debugging rare edge cases — they're fighting configs that don't work *yet*. otel-collector-mcp turns a 30-minute log-reading session into a seconds-long diagnosis + fix cycle. It understands collector deployment context (DaemonSet, Deployment, StatefulSet, Operator CRDs), detects misconfigurations and anti-patterns, and suggests the correct remediation — all without leaving the engineer's workflow.

This is an open-source tool positioned as a companion to the OTel Collector ecosystem. It does not compete with observability platforms — it makes them work correctly by ensuring the collector pipeline feeding them is properly configured.

---

## Core Vision

### Problem Statement

The OpenTelemetry Collector is the backbone of modern observability pipelines, but configuring it correctly in Kubernetes is unnecessarily difficult. Engineers face a painful feedback loop: write YAML config, deploy the collector, check logs for errors, try to interpret cryptic error messages, guess at the fix, and repeat. There is no tool that understands the collector's context — its deployment mode, pipeline topology, receiver/processor/exporter relationships — and can tell the engineer *what's wrong* and *how to fix it*.

This problem is compounded by the collector's complexity: hundreds of components, multiple deployment patterns (DaemonSet, Deployment, StatefulSet, sidecar), the OTel Operator's CRD layer, and subtle anti-patterns that only experienced engineers recognize (e.g., tail sampling on a DaemonSet, missing memory limiters, hardcoded credentials in config).

### Problem Impact

- **Wasted engineering time**: Hours spent on trial-and-error config iteration instead of building observability value
- **Observability gaps**: Misconfigured collectors produce incomplete or incorrect telemetry, leading to blind spots in production monitoring
- **Erosion of OTel adoption**: Teams that can't get the collector working properly question whether OpenTelemetry was the right choice, risking abandonment of the entire observability investment
- **Silent data loss**: Subtle misconfigurations (exporter backpressure, wrong cumulative-to-delta settings) cause data to be silently dropped — engineers don't even know what they're missing
- **Knowledge bottleneck**: Deep collector expertise is rare; teams depend on one or two senior engineers for all pipeline troubleshooting

### Why Existing Solutions Fall Short

- **Collector logs alone** are cryptic and lack actionable guidance — they tell you *something* failed, not *why* or *how to fix it*
- **OTel Operator** validates CRD schema but doesn't understand pipeline semantics or best practices
- **Generic Kubernetes debugging tools** (kubectl, k9s) have no understanding of OTel Collector concepts — they see pods and containers, not pipelines and processors
- **Observability platform documentation** covers their own backends but not the collector configuration that feeds them
- **Community forums and Slack** provide answers, but searching and waiting is slow and context-dependent
- **No tool exists** that combines Kubernetes context awareness with deep OTel Collector domain knowledge in a real-time, interactive workflow

### Proposed Solution

An MCP server deployed in-cluster that provides two categories of capability:

1. **Reactive tools (troubleshooting & diagnosis)**: Auto-detect the collector deployment, parse logs, scan configs, and return a prioritized list of issues with specific remediation instructions. Detection and fix are delivered as a single flow — not "here's what's wrong" in one tool and "here's the fix" in another.

2. **Proactive skills (design guidance)**: Help engineers design collector architectures, generate OTTL transforms, configure sampling strategies, and follow security best practices — all grounded in opinionated, production-proven patterns.

The MCP protocol (SSE transport) means this integrates directly into any MCP-compatible AI assistant or IDE, meeting engineers where they already work.

### Key Differentiators

- **Collector-native context**: Understands OTel Collector concepts (pipelines, receivers, processors, exporters, connectors) as first-class entities, not generic YAML
- **Kubernetes-aware**: Detects deployment mode (DaemonSet/Deployment/StatefulSet/Operator CRD) and applies mode-specific best practices
- **Detection + remediation in one flow**: Doesn't just report problems — provides the specific config change to fix them
- **Opinionated architecture guidance**: Encodes production patterns (e.g., Agent→Gateway hybrid, DaemonSet for logs, StatefulSet for Target Allocator) rather than presenting all options as equal
- **Design-time focus**: Optimized for the 80% case — engineers iterating on configs that don't work yet, not rare production incidents
- **In-cluster deployment**: Runs where the collector runs, with direct access to collector logs, configs, and Kubernetes state via client-go

---

## Target Users

### Primary Users

#### 1. Observability Engineer — "Alex"

**Role & Context**: Mid-to-senior engineer responsible for writing and maintaining OTel Collector pipeline configurations. Works with receivers, processors, and exporters daily. Iterates on OTTL transforms, sampling rules, metric conversions, and log parsing pipelines.

**Day-to-day**: Alex spends most of their time in YAML configs and collector logs. They're trying to get a new pipeline working — parsing application logs through the transform processor, converting cumulative metrics to delta for their Dynatrace backend, or setting up tail sampling that captures errors and slow transactions while keeping costs down.

**Pain points**: Alex changes a config, restarts the collector, waits for logs, and tries to decode what went wrong. OTTL syntax errors are particularly painful — the error messages are unhelpful. They often don't know if they've missed a best practice (no memory limiter, no retry on exporters) until something breaks in production. This is where 80% of the pain lives.

**Success moment**: Alex points the MCP at their collector, gets an instant list of issues ranked by severity, and gets the corrected config block in seconds. What used to take 30 minutes of log archaeology takes 10 seconds.

#### 2. Platform Engineer — "Jordan"

**Role & Context**: Senior/staff engineer responsible for the collector infrastructure itself — deployment mode, scaling, Helm charts, Operator CRDs, Target Allocator, security posture. Cares about architecture decisions that affect the whole fleet.

**Day-to-day**: Jordan decides whether to run collectors as DaemonSets, Deployments, or StatefulSets. They configure the OTel Operator, set up the Target Allocator for Prometheus scraping at scale, and ensure no hardcoded tokens leak into collector configs. They handle collector upgrades across the fleet.

**Pain points**: Architecture decisions are high-stakes and hard to reverse. Should this be a DaemonSet or a Deployment? When do they need a StatefulSet? How should the Agent→Gateway topology work? They also deal with Operator reconciliation failures and rejected CRDs that give opaque error messages.

**Success moment**: Jordan asks the MCP to design a collector architecture for their use case and gets an opinionated, production-proven recommendation with the specific CRD or Helm values to implement it.

#### 3. SRE (On-Call) — "Sam"

**Role & Context**: SRE who gets paged when observability pipelines break in production. Not a daily collector user — interacts with it during incidents. Needs fast triage, not deep design guidance.

**Day-to-day (during incidents)**: Sam gets an alert that metrics are missing or spans are dropping. They need to quickly determine: is this a collector problem? Which collector? What's wrong? Is it OOM, exporter backpressure, a bad config rollout, or a networking issue?

**Pain points**: Sam isn't a collector expert. They need the MCP to run a triage scan and tell them in plain terms what's failing and what to do about it, so they can either fix it or escalate to the right person.

**Success moment**: Sam runs triage scan at 3 AM, gets "Collector `otel-gateway` in namespace `monitoring` has exporter queue full — backpressure from Datadog endpoint. Suggested fix: increase queue size or add retry with backoff. Here's the config change." Incident resolved in minutes, not hours.

### Secondary Users

- **Engineering managers / observability leads**: Don't use the MCP directly but benefit from faster iteration cycles and fewer collector-related incidents. May be decision-makers for adoption.
- **k8s-networking-mcp users**: Engineers already using the sibling networking MCP may discover otel-collector-mcp as a complementary tool for their observability stack.

### User Journey

1. **Discovery**: Engineer finds otel-collector-mcp through OTel community channels, GitHub, or recommendation from a colleague who used it to solve a collector issue
2. **Onboarding**: Deploy the MCP server in-cluster via Helm chart. Connect their MCP-compatible AI assistant. Takes minutes, not hours
3. **First value ("aha" moment)**: Run triage scan on existing collector — immediately see 3-5 issues they didn't know about, with specific fixes. "Where was this tool six months ago?"
4. **Core usage**: Becomes part of the config iteration loop. Instead of change→deploy→read logs→guess, it's change→ask MCP→get instant feedback. Also used for new pipeline design with architecture and OTTL generation skills
5. **Long-term integration**: Standard tool in the team's observability workflow. New team members use it to learn collector best practices through the opinionated guidance. Reduces dependency on senior engineers for collector troubleshooting

---

## Success Metrics

### User Success Metrics

| Metric | Target | Measurement |
|--------|--------|-------------|
| Time to diagnose collector issue | < 1 minute (down from 15-30 min) | User feedback, before/after comparison |
| Config iterations to working pipeline | Reduced by 50%+ | User surveys, usage telemetry |
| Issues detected per triage scan | 3-5 actionable findings on typical collector | Tool output analysis |
| Remediation acceptance rate | > 70% of suggested fixes applied as-is | Usage telemetry (if opted in) |
| User retention (weekly active) | > 60% of installers still active after 30 days | GitHub / telemetry data |

### Business Objectives

- **Adoption**: Become the go-to MCP for OTel Collector troubleshooting in the Kubernetes community
- **Community growth**: Build an active contributor community around detection rules and remediation patterns
- **Ecosystem positioning**: Establish as a reference implementation for domain-specific MCP servers in the observability space
- **OTel ecosystem value**: Reduce the barrier to OTel Collector adoption — fewer teams give up due to config complexity

### Key Performance Indicators

| KPI | 3-Month Target | 12-Month Target |
|-----|---------------|-----------------|
| GitHub stars | 200+ | 1,000+ |
| Monthly active clusters (opt-in telemetry) | 50+ | 500+ |
| Community-contributed detection rules | 5+ | 25+ |
| Issues/PRs from external contributors | 10+ | 50+ |
| Mentions in OTel community channels | Regular presence | Recommended in official OTel docs/guides |

---

## MVP Scope

### Core Features (v1)

**Reactive Tools — Troubleshooting & Diagnosis:**

1. **Auto-detect collector deployment type** — Identify DaemonSet, Deployment, StatefulSet, or OTel Operator CRD automatically from Kubernetes state
2. **Triage scan** — Single entry point that runs all detectors and returns a prioritized issue list with severity rankings
3. **Collector log parsing** — Parse collector logs to detect OTTL syntax errors, exporter failures, OOM events, and receiver issues
4. **Operator log parsing** — Parse OTel Operator logs for rejected CRDs and reconciliation failures
5. **Misconfig detection suite**:
   - Missing batch processor or memory limiter
   - Hardcoded tokens/credentials in config
   - Missing retry/queue on exporters
   - Wrong receiver bindings (port conflicts, missing protocols)
   - Tail sampling on DaemonSet (anti-pattern)
   - Invalid regex in filter processor
   - Connector misconfiguration
   - Resource detector conflicts (overwriting each other)
   - Cumulative-to-delta conversion issues
   - High-cardinality metric labels
6. **Exporter backpressure detection** — Detect queue full / silent data loss from exporter backpressure
7. **Detection → remediation flow** — Every detection includes the specific config change to fix the issue, delivered as a single interaction

**Proactive Skills — Design Guidance (v1):**

8. **Architecture design** — Recommend collector deployment topology (DaemonSet for logs, Deployment/StatefulSet for metrics+traces, Agent→Gateway hybrid pattern) with opinionated rationale
9. **OTTL generation** — Generate OTTL transform instructions for log/span parsing and transformation via the transform processor

### Out of Scope for MVP

| Feature | Reason | Target Version |
|---------|--------|----------------|
| OBI (eBPF Instrumentation) guidance | Separate concern — instrumentation, not collector pipelines | v2 or separate MCP |
| Instrumentation CRD configuration | Operator instrumentation is adjacent but distinct from collector config | v2 |
| Target Allocator setup | Requires write operations to create CRDs, increases blast radius | v2 |
| Pre-deployment YAML validation | Only post-deployment diagnosis in v1; pre-deploy validation adds complexity | v2 |
| zpages health checking | Requires port-forwarding or service exposure — additional infrastructure | v2 |
| Tail sampling configuration skill | Detection is in v1; full config generation is v2 | v2 |
| Cumulative-to-delta conversion guidance skill | Detection is in v1; full guidance skill is v2 | v2 |
| Security hardening skill | Detection of hardcoded tokens is in v1; full hardening guidance is v2 | v2 |
| High-cardinality remediation skill | Detection is in v1; OTTL-based remediation generation is v2 | v2 |
| filelog receiver operators | Focus on transform processor; filelog is a different domain | Not planned |
| Shared state with k8s-networking-mcp | Loose coupling only — recommend the sibling MCP for connectivity issues | Not planned |

### MVP Success Criteria

- Engineers can deploy the MCP in-cluster and run a triage scan within 15 minutes of installation
- Triage scan correctly identifies at least 80% of the common misconfigurations listed in the detection suite
- Remediation suggestions are specific enough to apply directly (not generic "check your config" advice)
- Architecture design skill produces recommendations that match documented OTel best practices
- OTTL generation produces syntactically valid transform processor configurations
- Tool works with all four collector deployment modes (DaemonSet, Deployment, StatefulSet, Operator CRD)

### Future Vision (v2+)

**v2 — Expanded Proactive Skills:**
- Full tail sampling configuration skill (errors + slow transactions + probabilistic)
- Cumulative-to-delta conversion guidance for delta backends (Dynatrace, etc.)
- Security hardening skill (env vars / K8s secrets patterns)
- High-cardinality remediation with OTTL transform generation
- Target Allocator setup for Prometheus scraping at scale
- zpages integration for live pipeline health visualization

**v3 — Instrumentation & Beyond:**
- Instrumentation CRD configuration (auto-instrumentation for Java, Python, Node.js, .NET, Go)
- Pod annotation suggestions for SDK injection
- OBI (OpenTelemetry eBPF Instrumentation) deployment guidance
- SDK vs OBI vs hybrid instrumentation guidance
- Pre-deployment YAML validation ("validate this before I apply it")

**Long-term Vision:**
- Community-driven detection rule library (pluggable detection patterns contributed by the OTel community)
- Integration with CI/CD pipelines for automated collector config validation
- Cross-MCP orchestration with k8s-networking-mcp for full-stack observability troubleshooting
- Support for non-Kubernetes deployments (Docker, bare metal, cloud-managed collectors)

---

## Technical Context

| Aspect | Decision |
|--------|----------|
| **Language** | Go |
| **Deployment** | In-cluster (Kubernetes) |
| **Protocol** | MCP (SSE transport) |
| **MCP SDK** | Official Go MCP SDK |
| **K8s Client** | client-go |
| **Logging** | slog |
| **Observability** | OTel instrumentation (dogfooding) |
| **Sibling Project** | k8s-networking-mcp (loose coupling, no shared state) |

### Architecture Opinions (encoded as opinionated defaults)

- **Logs**: Always DaemonSet (needs node-level `/var/log` access)
- **Metrics + Traces**: Deployment or StatefulSet (gateway mode)
- **StatefulSet**: When using Target Allocator for Prometheus scraping
- **Sidecar**: Edge cases only — not recommended as default
- **Hybrid**: Agent (DaemonSet) → Gateway (Deployment/StatefulSet) is the production pattern
