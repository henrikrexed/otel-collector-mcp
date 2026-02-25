# Observability

otel-collector-mcp instruments itself with OpenTelemetry, producing traces, metrics, and logs following the **OTel GenAI and MCP semantic conventions**.

## Enabling OTel Self-Instrumentation

### Helm Chart Values

```yaml
otel:
  enabled: true
  endpoint: "otel-collector.observability.svc.cluster.local:4317"
  insecure: true
  serviceName: "otel-collector-mcp"
```

| Value | Env Var | Default | Description |
|-------|---------|---------|-------------|
| `otel.enabled` | `OTEL_ENABLED` | `false` | Enable OTel export |
| `otel.endpoint` | `OTEL_EXPORTER_OTLP_ENDPOINT` | `otel-collector.observability.svc.cluster.local:4317` | OTLP gRPC endpoint |
| `otel.insecure` | `OTEL_EXPORTER_OTLP_INSECURE` | `true` | Use insecure gRPC connection |
| `otel.serviceName` | `OTEL_SERVICE_NAME` | Chart fullname | Service name in telemetry |

### Environment Variables

If running outside Helm, set environment variables directly:

```bash
export OTEL_ENABLED=true
export OTEL_EXPORTER_OTLP_ENDPOINT=localhost:4317
export OTEL_EXPORTER_OTLP_INSECURE=true
export OTEL_SERVICE_NAME=otel-collector-mcp
```

## Spans

Every MCP tool call produces a span following the [OTel MCP semantic conventions](https://opentelemetry.io/docs/specs/semconv/gen-ai/mcp/).

### Span Name Format

```
{mcp.method.name} {gen_ai.tool.name}
```

Examples: `tools/call triage_scan`, `tools/call list_collectors`, `tools/call check_config`

### Span Kind

All tool call spans use `SpanKind: SERVER`.

### Attributes

**Required:**

| Attribute | Example | Description |
|-----------|---------|-------------|
| `mcp.method.name` | `tools/call` | MCP JSON-RPC method |
| `gen_ai.tool.name` | `triage_scan` | Tool being invoked |
| `gen_ai.operation.name` | `execute_tool` | Always "execute_tool" |
| `mcp.protocol.version` | `2025-06-18` | MCP protocol version |

**Recommended:**

| Attribute | Example | Description |
|-----------|---------|-------------|
| `jsonrpc.request.id` | `1` | JSON-RPC request ID |
| `jsonrpc.protocol.version` | `2.0` | JSON-RPC version |
| `network.transport` | `tcp` | Network transport |
| `server.address` | `otel-mcp-pod-xyz` | Server hostname |
| `server.port` | `8080` | Server port |

**Opt-in:**

| Attribute | Description |
|-----------|-------------|
| `gen_ai.tool.call.arguments` | Sanitized tool arguments (max 1KB) |
| `gen_ai.tool.call.result` | Truncated tool result (max 1KB) |

### Error Handling

On tool execution errors:

- `error.type` is set to `tool_error`
- Span status is set to `ERROR`
- The error is recorded via `span.RecordError()`

### Span Events

Diagnostic findings from triage and analysis tools are recorded as span events:

```
Event: "diagnostic_finding"
  severity: "critical"
  category: "missing_batch_processor"
  summary: "Pipeline 'traces' is missing the batch processor"
```

### Context Propagation

The server extracts `traceparent` and `tracestate` from the MCP request `params._meta` field, enabling end-to-end distributed tracing from AI agent → MCP server → Kubernetes API.

If your AI agent includes trace context in tool calls:

```json
{
  "method": "tools/call",
  "params": {
    "name": "triage_scan",
    "arguments": {"namespace": "default"},
    "_meta": {
      "traceparent": "00-0af7651916cd43dd8448eb211c80319c-b7ad6b7169203331-01"
    }
  }
}
```

The MCP server span becomes a child of the upstream trace.

## Metrics

### GenAI Semconv Metrics

| Metric | Type | Unit | Description |
|--------|------|------|-------------|
| `gen_ai.server.request.duration` | Histogram | `s` | Tool execution latency |
| `gen_ai.server.request.count` | Counter | — | Tool invocations (by `gen_ai.tool.name`, `error.type`) |

### Custom Domain Metrics

| Metric | Type | Description |
|--------|------|-------------|
| `mcp.findings.total` | Counter | Diagnostic findings (by `severity`, `analyzer`) |
| `mcp.collectors.discovered` | Gauge | Number of discovered OTel Collector instances |
| `mcp.errors.total` | Counter | Tool execution errors (by `error.type`) |

## Logs

When OTel is enabled, application logs are exported via the OTel log pipeline **in addition** to stdout.

### Format

Logs are structured JSON with automatic trace correlation:

```json
{
  "time": "2026-02-25T10:30:00Z",
  "level": "INFO",
  "msg": "tool invoked",
  "tool": "triage_scan",
  "trace_id": "0af7651916cd43dd8448eb211c80319c",
  "span_id": "b7ad6b7169203331"
}
```

The `trace_id` and `span_id` fields are automatically injected when the log is emitted within a span context, enabling log-to-trace correlation in your backend.

### Log Bridge

The slog→OTel bridge uses a tee handler:

- **stdout**: JSON format (always active, for `kubectl logs`)
- **OTel export**: OTLP gRPC to the configured endpoint (when `OTEL_ENABLED=true`)

## Example: OTel Collector in the Same Cluster

Deploy an OTel Collector in the `observability` namespace that receives OTLP and exports to your backend:

```yaml
# values.yaml for otel-collector-mcp
otel:
  enabled: true
  endpoint: "otel-collector.observability.svc.cluster.local:4317"
  insecure: true
```

The MCP server will send traces, metrics, and logs to the collector, which can then fan out to any backend.

## Example: Connecting to Dynatrace

```yaml
otel:
  enabled: true
  endpoint: "<your-dynatrace-otlp-endpoint>:4317"
  insecure: false
  serviceName: "otel-collector-mcp-prod"
```

Configure your OTel Collector or Dynatrace ActiveGate to accept OTLP gRPC. Dynatrace will automatically map MCP spans and GenAI metrics.

## Example: Connecting to Grafana (Tempo + Mimir + Loki)

Route all 3 signals through an OTel Collector with Grafana-compatible exporters:

```yaml
# OTel Collector config
exporters:
  otlphttp/tempo:
    endpoint: http://tempo.monitoring:4318
  prometheusremotewrite:
    endpoint: http://mimir.monitoring/api/v1/push
  loki:
    endpoint: http://loki.monitoring:3100/loki/api/v1/push
```

Then point the MCP server at that collector:

```yaml
otel:
  enabled: true
  endpoint: "otel-collector.monitoring.svc.cluster.local:4317"
```

## Example: Connecting to Jaeger

For traces only via Jaeger's OTLP receiver:

```yaml
otel:
  enabled: true
  endpoint: "jaeger-collector.observability.svc.cluster.local:4317"
  insecure: true
```

Jaeger natively accepts OTLP gRPC on port 4317. Metrics and logs will be sent but ignored by Jaeger (no error produced).
