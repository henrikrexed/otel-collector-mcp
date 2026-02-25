# otel-collector-mcp

MCP server for OpenTelemetry Collector troubleshooting and design guidance in Kubernetes.

## Features

- Automatic collector discovery across DaemonSets, Deployments, StatefulSets, and OTel Operator CRDs
- 12 built-in misconfiguration detection rules with specific remediation
- Collector and operator log parsing and classification
- Architecture design recommendations (topology, skeleton configs)
- OTTL transform generation from natural language
- Multi-cluster support with cluster identity in every response
- Gateway API exposure via Helm chart

## Quick Start

```bash
helm install otel-mcp deploy/helm/otel-collector-mcp \
  --namespace observability \
  --set config.clusterName=my-cluster
```

See the [full documentation](https://hrexed.github.io/otel-collector-mcp) for detailed setup instructions.

## Observability

otel-collector-mcp instruments itself with OpenTelemetry following the [GenAI + MCP semantic conventions](https://opentelemetry.io/docs/specs/semconv/gen-ai/mcp/). When enabled, it exports:

- **Traces**: Every tool call produces a span with `mcp.method.name`, `gen_ai.tool.name`, context propagation from `params._meta`, and diagnostic findings as span events
- **Metrics**: `gen_ai.server.request.duration`, `gen_ai.server.request.count`, `mcp.findings.total`, `mcp.collectors.discovered`, `mcp.errors.total`
- **Logs**: Structured JSON logs bridged to OTel with `trace_id`/`span_id` correlation

Enable via Helm:

```yaml
otel:
  enabled: true
  endpoint: "otel-collector.observability.svc.cluster.local:4317"
```

See the [Observability documentation](https://hrexed.github.io/otel-collector-mcp/observability/) for full details on spans, metrics, logs, and backend configuration examples.

## Documentation

- [Getting Started](https://hrexed.github.io/otel-collector-mcp/getting-started/)
- [Tools Reference](https://hrexed.github.io/otel-collector-mcp/tools/)
- [Skills Reference](https://hrexed.github.io/otel-collector-mcp/skills/)
- [Architecture Guide](https://hrexed.github.io/otel-collector-mcp/architecture/)
- [Observability](https://hrexed.github.io/otel-collector-mcp/observability/)
- [Contributing](https://hrexed.github.io/otel-collector-mcp/contributing/)
- [Troubleshooting](https://hrexed.github.io/otel-collector-mcp/troubleshooting/)

## License

See [LICENSE](LICENSE) for details.
