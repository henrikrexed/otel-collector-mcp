# otel-collector-mcp

A Model Context Protocol (MCP) server for OpenTelemetry Collector troubleshooting and design guidance in Kubernetes.

[![Build](https://github.com/henrikrexed/otel-collector-mcp/actions/workflows/ci.yaml/badge.svg)](https://github.com/henrikrexed/otel-collector-mcp/actions)
[![License](https://img.shields.io/badge/license-Apache%202.0-blue)](LICENSE)

## Overview

`otel-collector-mcp` runs in-cluster and gives AI agents the ability to diagnose, analyze, and guide OpenTelemetry Collector deployments. It detects 12 common misconfiguration patterns and provides actionable remediation — because 80% of collector pain happens at design time, not production.

**Key features:**
- 🔍 **Triage scan**: Run all 12 analyzers in one call, get prioritized issues
- 📋 **12 detection rules**: Missing batch processor, memory limiter gaps, hardcoded tokens, wrong port bindings, tail sampling anti-patterns, high cardinality, and more
- 🏗️ **Design skills**: Architecture recommendations and OTTL expression generation
- 📊 **OpenTelemetry instrumented**: Full traces, metrics, and logs following GenAI + MCP semantic conventions
- 🔒 **Read-only RBAC**: Safe to run in production clusters
- 🌐 **Multi-cluster**: Every response includes cluster identity
- 🚀 **CRD-aware**: Auto-discovers OpenTelemetry Operator CRDs

## Quick Start

### Helm (recommended)

```bash
helm repo add isitobservable https://henrikrexed.github.io/otel-collector-mcp
helm install otel-collector-mcp isitobservable/otel-collector-mcp \
  --namespace tools --create-namespace
```

### Docker

```bash
docker run -p 8080:8080 ghcr.io/henrikrexed/otel-collector-mcp:latest
```

## MCP Tools

| Tool | Description |
|------|-------------|
| `triage_scan` | Run all 12 analyzers, return prioritized issue list |
| `detect_deployment_type` | Identify collector deployment type (DaemonSet/Deployment/StatefulSet/Operator) |
| `list_collectors` | Discover all collector instances in the cluster |
| `get_config` | Retrieve running collector configuration |
| `parse_collector_logs` | Analyze collector logs for OTTL errors, exporter failures, OOM |
| `parse_operator_logs` | Check OTel Operator logs for rejected CRDs, reconciliation issues |
| `check_config` | Full misconfiguration detection suite |

## MCP Skills

| Skill | Description |
|-------|-------------|
| `design_architecture` | Get architecture recommendations (DaemonSet vs Deployment, Gateway pattern, etc.) |
| `generate_ottl` | Generate OTTL expressions for common transformations |

## Connect Your AI Agent

Register otel-collector-mcp as an MCP skill in your AI agent:

```json
{
  "mcpServers": {
    "otel-collector-mcp": {
      "url": "http://localhost:8080/mcp",
      "transport": "streamable-http"
    }
  }
}
```

Works with Claude Desktop, Cursor, VS Code, kagent, and any MCP-compatible client. See the [Getting Started guide](https://henrikrexed.github.io/otel-collector-mcp/getting-started/) for full setup instructions including Cursor/VS Code, kagent (Kubernetes-native), and multi-cluster configurations.

## Observability

The server produces OpenTelemetry traces, metrics, and logs following the [GenAI](https://opentelemetry.io/docs/specs/semconv/gen-ai/) and [MCP](https://opentelemetry.io/docs/specs/semconv/gen-ai/mcp/) semantic conventions.

Enable via Helm values:
```yaml
otel:
  enabled: true
  endpoint: "otel-collector.observability.svc.cluster.local:4317"
```

## Documentation

📖 Full documentation: [https://henrikrexed.github.io/otel-collector-mcp](https://henrikrexed.github.io/otel-collector-mcp)

## Part of IsItObservable

This project is part of the [IsItObservable](https://youtube.com/@IsItObservable) ecosystem — open-source tools for Kubernetes observability.

- [mcp-k8s-networking](https://github.com/henrikrexed/mcp-k8s-networking) — Kubernetes networking diagnostics
- [mcp-proxy](https://github.com/henrikrexed/mcp-proxy) — Universal OTel sidecar proxy for any MCP server

## License

Apache License 2.0
