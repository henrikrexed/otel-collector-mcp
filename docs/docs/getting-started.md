# Getting Started

This guide walks through installing otel-collector-mcp in your Kubernetes cluster, connecting it to an AI assistant, and running your first triage scan.

## Prerequisites

Before you begin, ensure you have:

- **Kubernetes cluster** (v1.24 or later) with `kubectl` configured
- **Helm 3** installed locally
- **An MCP-compatible client** such as Claude Desktop, or any application implementing the Model Context Protocol
- **RBAC permissions** to create ClusterRoles, ClusterRoleBindings, ServiceAccounts, Deployments, and Services in the target namespace

## Installation via Helm

### Add the chart and install with defaults

```bash
helm install otel-collector-mcp deploy/helm/otel-collector-mcp \
  --namespace observability \
  --create-namespace
```

### Install with a cluster name (recommended for multi-cluster)

```bash
helm install otel-collector-mcp deploy/helm/otel-collector-mcp \
  --namespace observability \
  --create-namespace \
  --set config.clusterName=production-us-east
```

### Install with custom values

Create a `values.yaml` file:

```yaml
image:
  repository: ghcr.io/henrikrexed/otel-collector-mcp
  tag: latest
  pullPolicy: IfNotPresent

replicaCount: 1
port: 8080

config:
  clusterName: "production-us-east"
  logLevel: info

otel:
  enabled: false
  endpoint: "otel-collector.observability.svc.cluster.local:4317"
  insecure: true
  serviceName: ""

resources:
  limits:
    cpu: 500m
    memory: 256Mi
  requests:
    cpu: 100m
    memory: 128Mi
```

Then install:

```bash
helm install otel-collector-mcp deploy/helm/otel-collector-mcp \
  --namespace observability \
  --create-namespace \
  -f values.yaml
```

### Verify the installation

```bash
kubectl get pods -n observability -l app.kubernetes.io/name=otel-collector-mcp
kubectl logs -n observability -l app.kubernetes.io/name=otel-collector-mcp
```

You should see logs indicating the server started and CRD discovery completed:

```
{"level":"INFO","msg":"starting otel-collector-mcp","port":8080,"clusterName":"production-us-east","otelEnabled":false}
{"level":"INFO","msg":"CRD discovery complete","hasOTelOperator":true,"hasTargetAllocator":false}
{"level":"INFO","msg":"MCP server starting","addr":":8080"}
```

### RBAC permissions

The Helm chart creates a ClusterRole with the following permissions:

| API Group | Resources | Verbs |
|---|---|---|
| `""` (core) | pods, pods/log, services, configmaps, namespaces | get, list, watch |
| `apps` | deployments, daemonsets, statefulsets | get, list, watch |
| `opentelemetry.io` | opentelemetrycollectors, instrumentations | get, list, watch |
| `apiextensions.k8s.io` | customresourcedefinitions | get, list, watch |

These are read-only permissions. The MCP server never modifies cluster resources.

## Installing as an MCP Skill

otel-collector-mcp exposes its tools via the MCP protocol over Streamable HTTP. Register it in your AI agent or IDE to give it access to OTel Collector diagnostics.

### Port-Forward (for local access)

If the MCP server is running in-cluster, set up port-forwarding:

```bash
kubectl port-forward -n observability svc/otel-collector-mcp 8080:8080
```

For production use, consider [Gateway API exposure](#gateway-api-exposure) instead.

### Claude Desktop

Add the following to your `claude_desktop_config.json`:

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

For multiple clusters, add one entry per cluster:

```json
{
  "mcpServers": {
    "otel-mcp-us-east": {
      "url": "http://localhost:8080/mcp",
      "transport": "streamable-http"
    },
    "otel-mcp-eu-west": {
      "url": "http://localhost:8081/mcp",
      "transport": "streamable-http"
    }
  }
}
```

### Cursor / VS Code

Add to your workspace or user `settings.json`:

```json
{
  "mcp": {
    "servers": {
      "otel-collector-mcp": {
        "url": "http://localhost:8080/mcp",
        "transport": "streamable-http"
      }
    }
  }
}
```

### kagent (Kubernetes-native)

Deploy as a Kubernetes-native MCP server resource:

```yaml
apiVersion: kagent.dev/v1alpha1
kind: MCPServer
metadata:
  name: otel-collector-mcp
spec:
  url: "http://otel-collector-mcp.observability.svc:8080/mcp"
  transport: streamable-http
```

### OpenClaw

Add otel-collector-mcp as an MCP tool in your OpenClaw configuration, pointing to the `/mcp` endpoint with `streamable-http` transport.

### Other MCP Clients

Any client that implements the MCP protocol can connect by sending HTTP requests to the `/mcp` endpoint:

- **GET `/mcp`** -- Returns the list of available tools and their schemas.
- **POST `/mcp`** with `{"method": "tools/call", "params": {"name": "<tool>", "arguments": {...}}}` -- Invokes a tool.

## Running Your First Triage Scan

Once connected, ask your AI assistant to scan a collector. The assistant will use the MCP tools automatically.

### Step 1: List collectors

Ask: *"List all OTel Collectors in my cluster."*

The assistant calls `list_collectors`, which returns:

```json
{
  "cluster": "production-us-east",
  "namespace": "default",
  "timestamp": "2025-01-15T10:30:00Z",
  "tool": "list_collectors",
  "data": {
    "collectors": [
      {
        "name": "otel-collector-agent",
        "namespace": "observability",
        "deploymentMode": "DaemonSet",
        "version": "0.96.0",
        "podCount": 5
      },
      {
        "name": "otel-collector-gateway",
        "namespace": "observability",
        "deploymentMode": "Deployment",
        "version": "0.96.0",
        "podCount": 2
      }
    ],
    "count": 2
  }
}
```

### Step 2: Run a triage scan

Ask: *"Run a triage scan on the otel-collector-gateway in the observability namespace."*

The assistant calls `triage_scan` with the appropriate parameters. The response includes severity-ranked findings with remediation:

```json
{
  "cluster": "production-us-east",
  "namespace": "default",
  "timestamp": "2025-01-15T10:31:00Z",
  "tool": "triage_scan",
  "data": {
    "findings": [
      {
        "severity": "warning",
        "category": "performance",
        "summary": "Pipeline \"traces\" is missing the batch processor",
        "detail": "The batch processor groups data before sending to exporters...",
        "suggestion": "Add the batch processor to this pipeline",
        "remediation": "processors:\n  batch:\n    send_batch_size: 8192\n    timeout: 200ms\n..."
      },
      {
        "severity": "warning",
        "category": "security",
        "summary": "Hardcoded authentication token found in exporter configuration",
        "detail": "Exporter 'otlp/backend' contains a hardcoded bearer token...",
        "suggestion": "Use environment variable substitution or a Kubernetes Secret"
      }
    ],
    "metadata": {
      "deploymentMode": "Deployment",
      "configSource": "otel-collector-gateway-config"
    }
  }
}
```

The AI assistant will interpret these findings and guide you through the remediation steps.

## Multi-Cluster Setup

For organizations running multiple Kubernetes clusters, deploy one otel-collector-mcp instance per cluster. Set a unique `CLUSTER_NAME` for each:

```bash
# Cluster 1
helm install otel-collector-mcp deploy/helm/otel-collector-mcp \
  --namespace observability \
  --set config.clusterName=production-us-east

# Cluster 2
helm install otel-collector-mcp deploy/helm/otel-collector-mcp \
  --namespace observability \
  --set config.clusterName=production-eu-west
```

Every response from the MCP server includes the `cluster` field, allowing your AI assistant to distinguish which cluster a finding belongs to when you connect to multiple MCP servers simultaneously.

## Enable Observability

otel-collector-mcp can export its own traces, metrics, and logs via OTLP gRPC, following the [OTel GenAI + MCP semantic conventions](https://opentelemetry.io/docs/specs/semconv/gen-ai/mcp/).

### Via Helm values

```yaml
otel:
  enabled: true
  endpoint: "otel-collector.observability.svc.cluster.local:4317"
  insecure: true
  serviceName: "otel-collector-mcp"
```

```bash
helm install otel-collector-mcp deploy/helm/otel-collector-mcp \
  --namespace observability \
  --set otel.enabled=true \
  --set otel.endpoint=otel-collector.observability.svc.cluster.local:4317
```

### Via environment variables

```bash
export OTEL_ENABLED=true
export OTEL_EXPORTER_OTLP_ENDPOINT=otel-collector.observability.svc.cluster.local:4317
export OTEL_EXPORTER_OTLP_INSECURE=true
export OTEL_SERVICE_NAME=otel-collector-mcp
```

### Minimal OTel Collector config to receive from this MCP server

If you have an OTel Collector running in-cluster that will receive telemetry from otel-collector-mcp, here is a minimal config:

```yaml
receivers:
  otlp:
    protocols:
      grpc:
        endpoint: "0.0.0.0:4317"

processors:
  batch:
    send_batch_size: 8192
    timeout: 200ms

exporters:
  # Replace with your backend exporter
  debug:
    verbosity: detailed

service:
  pipelines:
    traces:
      receivers: [otlp]
      processors: [batch]
      exporters: [debug]
    metrics:
      receivers: [otlp]
      processors: [batch]
      exporters: [debug]
    logs:
      receivers: [otlp]
      processors: [batch]
      exporters: [debug]
```

### Supported backends

The OTLP gRPC output works with any OTel-compatible backend:

- **Dynatrace** -- via Dynatrace OTLP endpoint or ActiveGate
- **Grafana** -- Tempo (traces), Mimir (metrics), Loki (logs) via an OTel Collector
- **Jaeger** -- native OTLP gRPC receiver on port 4317
- **Datadog** -- via Datadog Agent OTLP ingestion
- **New Relic** -- via OTLP endpoint
- **Elastic / OpenSearch** -- via OTel Collector with appropriate exporters

See the [Observability documentation](observability.md) for full details on spans, metrics, logs, and backend-specific configuration examples.

## Gateway API Exposure

To expose the MCP server through a Kubernetes Gateway API HTTPRoute (instead of port-forwarding), enable the gateway in your Helm values:

```yaml
gateway:
  enabled: true
  className: "istio"
  hostname: "otel-mcp.example.com"
  port: 8080
  tls:
    enabled: true
    certificateRef:
      name: "otel-mcp-tls"
      namespace: "observability"
```

Install with the gateway enabled:

```bash
helm install otel-collector-mcp deploy/helm/otel-collector-mcp \
  --namespace observability \
  --create-namespace \
  -f values.yaml
```

This creates an HTTPRoute resource that routes traffic from the specified Gateway to the MCP service. Supported Gateway API providers include:

- Istio
- Envoy Gateway
- Cilium
- NGINX Gateway Fabric
- kgateway (formerly Gloo Gateway)

See the [Architecture Guide](architecture/index.md) for more details on Gateway API configuration.
