# otel-collector-mcp

**MCP server for OpenTelemetry Collector troubleshooting and design guidance.**

otel-collector-mcp is a Model Context Protocol (MCP) server that runs inside your Kubernetes cluster and gives AI assistants real-time access to your OpenTelemetry Collector fleet. It discovers collectors, retrieves configurations, parses logs, runs misconfiguration detection rules, and provides architecture design recommendations -- all through a standardized MCP tool interface.

## Key Features

- **Automatic Collector Discovery** -- Finds all OTel Collector instances across namespaces by scanning DaemonSets, Deployments, StatefulSets, and OTel Operator CRDs.
- **Configuration Retrieval and Analysis** -- Pulls live collector configurations from ConfigMaps and runs 12 built-in detection rules covering missing batch processors, hardcoded tokens, tail-sampling on DaemonSets, high-cardinality attributes, and more.
- **Log Classification** -- Parses collector and operator pod logs and classifies errors into actionable categories: OTTL syntax errors, exporter failures, OOM events, receiver issues, and processor errors.
- **Triage Scanning** -- Runs every detection rule against a collector instance and returns a severity-ranked issue list with specific remediation snippets.
- **Architecture Design** -- Recommends deployment topologies (DaemonSet, Gateway, Hybrid Agent-to-Gateway) based on signal types, scale, backend targets, and sampling requirements. Generates skeleton collector configurations.
- **OTTL Generation** -- Produces OpenTelemetry Transformation Language (OTTL) processor statements for log parsing, span manipulation, and metric transformations.
- **Multi-Cluster Support** -- Deploy one MCP server per cluster, connect your AI agent to all of them, and every response includes cluster identity for disambiguation.
- **Gateway API Exposure** -- Built-in Helm support for Kubernetes Gateway API HTTPRoute resources, compatible with Istio, Envoy Gateway, Cilium, NGINX, and kgateway.

## Quick Links

| Section | Description |
|---|---|
| [Getting Started](getting-started.md) | Install the Helm chart, connect your AI client, run your first scan |
| [Tools Reference](tools/index.md) | Complete reference for all 7 MCP tools with parameters and examples |
| [Skills Reference](skills/index.md) | Reference for proactive skills: architecture design and OTTL generation |
| [Architecture Guide](architecture/index.md) | Deployment patterns, multi-cluster setup, Gateway API configuration |
| [Contributing](contributing.md) | Add detection rules, skills, and submit pull requests |
| [Troubleshooting](troubleshooting.md) | Common issues and diagnostic steps |

## How It Works

otel-collector-mcp deploys as a single-replica Deployment in your Kubernetes cluster. On startup it:

1. Initializes Kubernetes clients (in-cluster ServiceAccount or local kubeconfig).
2. Starts a CRD watcher that discovers whether the OTel Operator and Target Allocator are installed, re-checking every 30 seconds.
3. Registers 7 MCP tools and 2 MCP skills in thread-safe registries.
4. Serves an HTTP endpoint at `/mcp` that accepts MCP `tools/list` and `tools/call` requests.
5. Exposes `/healthz` and `/readyz` endpoints for Kubernetes probe integration.

Every tool response is wrapped in a `StandardResponse` envelope that includes cluster name, namespace, timestamp, tool name, and the tool-specific data payload. This consistent structure allows AI assistants to correlate findings across multiple clusters.

## Response Envelope

All tool responses follow this structure:

```json
{
  "cluster": "production-us-east",
  "namespace": "observability",
  "timestamp": "2025-01-15T10:30:00Z",
  "tool": "triage_scan",
  "data": {
    "findings": [...],
    "metadata": {...}
  }
}
```

## Requirements

- Kubernetes 1.24+
- Helm 3.x
- An MCP-compatible AI client (Claude Desktop, or any client implementing the MCP protocol)
