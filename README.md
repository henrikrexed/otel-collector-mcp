# otel-collector-mcp

A Model Context Protocol (MCP) server for OpenTelemetry Collector troubleshooting, runtime analysis, and design guidance in Kubernetes.

[![Build](https://github.com/henrikrexed/otel-collector-mcp/actions/workflows/ci.yaml/badge.svg)](https://github.com/henrikrexed/otel-collector-mcp/actions)
[![Release](https://img.shields.io/github/v/release/henrikrexed/otel-collector-mcp)](https://github.com/henrikrexed/otel-collector-mcp/releases)
[![License](https://img.shields.io/badge/license-Apache%202.0-blue)](LICENSE)

## Overview

`otel-collector-mcp` runs in-cluster and gives AI agents the ability to diagnose, analyze, and fix OpenTelemetry Collector deployments. It combines static analysis (12 misconfiguration patterns) with **runtime dynamic analysis** (live signal capture, issue detection, automated fix generation).

**v1 — Static Analysis:**
- 🔍 **Triage scan**: Run all 12 analyzers in one call, get prioritized issues
- 📋 **12 detection rules**: Missing batch processor, memory limiter gaps, hardcoded tokens, wrong port bindings, tail sampling anti-patterns, and more
- 🏗️ **Design skills**: Architecture recommendations and OTTL expression generation

**v2 — Dynamic Pipeline Analyzer (NEW):**
- 🔬 **Runtime analysis**: Inject debug exporters, capture live signals, detect issues from actual data
- 🛡️ **Safety model**: Environment gate (refuses production), config backup, health checks, auto-rollback on CrashLoopBackOff
- 🔎 **8 runtime detectors**: High cardinality, PII leaks, orphan spans, bloated attributes, missing resources, duplicate signals, sampling gaps, resource sizing
- 🔧 **Automated fixes**: OTTL transforms, filter processors, attribute drops — suggested with user approval
- 📊 **Sampling & sizing recommendations**: Based on observed traffic patterns

**Common:**
- 📊 **OpenTelemetry instrumented**: Full traces, metrics, and logs following GenAI + MCP semantic conventions
- 🔒 **RBAC-aware**: v1 read-only (safe for production), v2 requires write permissions (dev/staging only)
- 🌐 **Multi-cluster**: Every response includes cluster identity
- 🚀 **CRD-aware**: Auto-discovers OpenTelemetry Operator CRDs

## Quick Start

### Helm (recommended)

```bash
helm repo add henrikrexed https://henrikrexed.github.io/otel-collector-mcp
helm install otel-collector-mcp henrikrexed/otel-collector-mcp \
  --namespace tools --create-namespace
```

Enable v2 features:
```bash
helm install otel-collector-mcp henrikrexed/otel-collector-mcp \
  --namespace tools --create-namespace \
  --set v2.enabled=true
```

### Docker

```bash
docker run -p 8080:8080 ghcr.io/henrikrexed/otel-collector-mcp:latest
```

## MCP Tools

### v1 — Static Analysis (read-only, production-safe)

| Tool | Description |
|------|-------------|
| `triage_scan` | Run all 12 analyzers, return prioritized issue list |
| `detect_deployment_type` | Identify collector deployment type (DaemonSet/Deployment/StatefulSet/Operator) |
| `list_collectors` | Discover all collector instances in the cluster |
| `get_config` | Retrieve running collector configuration |
| `parse_collector_logs` | Analyze collector logs for OTTL errors, exporter failures, OOM |
| `parse_operator_logs` | Check OTel Operator logs for rejected CRDs, reconciliation issues |
| `check_config` | Full misconfiguration detection suite |

### v2 — Dynamic Pipeline Analyzer (requires v2.enabled)

| Tool | Description |
|------|-------------|
| `check_health` | Verify collector pods are healthy before analysis |
| `start_analysis` | Begin an analysis session (environment gate, config backup, debug injection) |
| `capture_signals` | Capture live traces/metrics/logs flowing through the collector |
| `detect_issues` | Run 8 runtime analyzers on captured signals |
| `suggest_fixes` | Generate OTTL/filter fixes for detected issues |
| `apply_fix` | Apply a suggested fix to the collector config (with user approval) |
| `rollback_config` | Restore the original config from backup |
| `cleanup_debug` | Remove debug exporters, close the analysis session |
| `recommend_sampling` | Generate tail sampling config based on observed traces |
| `recommend_sizing` | Recommend CPU/memory based on observed signal volume |

### MCP Skills

| Skill | Description |
|-------|-------------|
| `design_architecture` | Get architecture recommendations (DaemonSet vs Deployment, Gateway pattern, etc.) |
| `generate_ottl` | Generate OTTL expressions for common transformations |

## v2 Analysis Workflow

The v2 tools follow a structured workflow with built-in safety:

```
check_health → start_analysis → capture_signals → detect_issues
                                                        ↓
                                                  suggest_fixes
                                                        ↓
                                                   apply_fix
                                                        ↓
                                                 cleanup_debug
```

**Safety guarantees:**
- Asks for environment type — refuses production
- Backs up config before any changes
- Health checks after modifications
- Auto-rollback on CrashLoopBackOff
- Session TTL with automatic cleanup

## Connect Your AI Agent

```json
{
  mcpServers: {
    otel-collector-mcp: {
      url: http://otel-collector-mcp.tools.svc:8080/mcp,
      transport: streamable-http
    }
  }
}
```

Works with **Claude Desktop**, **VS Code**, **GitHub Copilot**, **kagent**, **HolmesGPT**, **Sympozium**, and any MCP-compatible client.

See the [integration guides](https://henrikrexed.github.io/otel-collector-mcp/integrations/) for platform-specific setup.

## Detection Rules

### v1 — Static (from config)

12 rules including: missing batch processor, no memory limiter, hardcoded auth tokens, wrong port bindings, tail sampling anti-patterns, connector misconfiguration, resource detector conflicts, and more.

### v2 — Runtime (from live signals)

| Rule | What it detects |
|------|-----------------|
| High Cardinality | Metric series with >100 unique label values |
| PII Detection | Email, phone, SSN, credit card patterns in attributes |
| Orphan Spans | Traces with disconnected spans (missing parent) |
| Bloated Attributes | Attributes >1KB that waste storage |
| Missing Resources | Spans/metrics without service.name or service.namespace |
| Duplicate Signals | Identical metrics/logs from multiple sources |
| Sampling Check | High-volume traces with no sampling configured |
| Resource Sizing | CPU/memory over/under-provisioning |

## Observability

The server produces OpenTelemetry traces, metrics, and logs following the [GenAI](https://opentelemetry.io/docs/specs/semconv/gen-ai/) and [MCP](https://opentelemetry.io/docs/specs/semconv/gen-ai/mcp/) semantic conventions.

```yaml
# Helm values
otel:
  enabled: true
  endpoint: otel-collector.observability.svc.cluster.local:4317
```

## Documentation

📖 Full documentation: [https://henrikrexed.github.io/otel-collector-mcp](https://henrikrexed.github.io/otel-collector-mcp)

- [Getting Started (v2)](https://henrikrexed.github.io/otel-collector-mcp/guides/getting-started-v2/)
- [Safety Model](https://henrikrexed.github.io/otel-collector-mcp/guides/safety-model/)
- [Detection Rules](https://henrikrexed.github.io/otel-collector-mcp/guides/detection-rules/)
- [Use Cases](https://henrikrexed.github.io/otel-collector-mcp/guides/use-cases/)
- [Integration Guides](https://henrikrexed.github.io/otel-collector-mcp/integrations/) (kagent, HolmesGPT, Claude, Copilot, VS Code, Sympozium)

## Deployment

### Kubernetes (plain manifests)

```bash
kubectl apply -k deploy/kubernetes/
```

### Sympozium (SkillPack)

```bash
kubectl apply -f deploy/sympozium/skillpack.yaml
```

## Part of IsItObservable

This project is part of the [IsItObservable](https://youtube.com/@IsItObservable) ecosystem — open-source tools for Kubernetes observability.

- [mcp-k8s-networking](https://github.com/henrikrexed/mcp-k8s-networking) — Kubernetes networking diagnostics
- [mcp-otel-proxy](https://github.com/henrikrexed/mcp-proxy) — Universal OTel sidecar proxy for any MCP server

## License

Apache License 2.0
