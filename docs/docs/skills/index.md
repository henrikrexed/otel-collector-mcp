# Skills Reference

In addition to reactive diagnostic tools, otel-collector-mcp provides 2 proactive MCP skills that generate design recommendations and configuration snippets. Skills differ from tools in that they produce forward-looking guidance rather than inspecting existing state.

All skill responses use the same `StandardResponse` envelope as tools, with the `tool` field set to the skill name.

---

## design_architecture

Recommend an OTel Collector deployment topology based on a description of your workload requirements.

This skill evaluates your signal types, expected scale, backend targets, and sampling needs to recommend one of four deployment topologies:

- **DaemonSet Only** -- For log-only collection requiring node-level `/var/log` access.
- **DaemonSet Agent** -- For simple per-node signal collection at scale.
- **Gateway (Deployment/StatefulSet)** -- For centralized processing with tail sampling or multi-backend fan-out.
- **Hybrid Agent-to-Gateway** -- For mixed signal types that need both node-level collection and centralized processing.

The skill also produces a list of recommended components with their deployment modes, rationale for the design decisions, and a skeleton collector configuration.

### Use Cases

- Planning a greenfield OTel Collector deployment
- Evaluating whether to add a gateway tier to an existing agent-only setup
- Determining the right deployment mode for Prometheus target scraping with the Target Allocator
- Designing multi-backend fan-out architectures

### Parameters

| Parameter | Type | Required | Description |
|---|---|---|---|
| `signal_types` | array of strings | No | Signal types to collect: `traces`, `metrics`, `logs` |
| `scale` | string | No | Expected scale: `small` (<50 pods), `medium` (50-500), `large` (500+) |
| `backends` | array of strings | No | Backend targets, e.g. `jaeger`, `prometheus`, `datadog`, `dynatrace`, `otlp` |
| `needs_sampling` | boolean | No | Whether tail sampling is needed |
| `needs_prometheus_scraping` | boolean | No | Whether Prometheus target scraping is needed |

### Example Invocation

```json
{
  "method": "tools/call",
  "params": {
    "name": "design_architecture",
    "arguments": {
      "signal_types": ["traces", "metrics", "logs"],
      "scale": "large",
      "backends": ["datadog", "jaeger"],
      "needs_sampling": true,
      "needs_prometheus_scraping": true
    }
  }
}
```

### Example Output

```json
{
  "cluster": "production-us-east",
  "namespace": "observability",
  "timestamp": "2025-01-15T10:35:00Z",
  "tool": "design_architecture",
  "data": {
    "topology": "Hybrid Agent\u2192Gateway",
    "components": [
      {
        "name": "otel-agent-logs",
        "deploymentMode": "DaemonSet",
        "role": "Log collection agent",
        "reason": "Logs require node-level /var/log access, which only DaemonSets provide"
      },
      {
        "name": "otel-gateway",
        "deploymentMode": "StatefulSet",
        "role": "Centralized gateway",
        "reason": "StatefulSet required for Target Allocator to assign scrape targets to specific pods"
      }
    ],
    "rationale": [
      "Mixed signal types with gateway requirements call for the hybrid Agent\u2192Gateway pattern",
      "Prometheus scraping with Target Allocator requires StatefulSet",
      "Tail sampling requires all spans for a trace to reach the same collector, necessitating a centralized gateway"
    ],
    "configSkeleton": "# Skeleton collector configuration\nreceivers:\n  otlp:\n    protocols:\n      grpc:\n        endpoint: \"0.0.0.0:4317\"\n      http:\n        endpoint: \"0.0.0.0:4318\"\n\nprocessors:\n  memory_limiter:\n    check_interval: 1s\n    limit_mib: 512\n    spike_limit_mib: 128\n  batch:\n    send_batch_size: 8192\n    timeout: 200ms\n\nexporters:\n  datadog:\n    endpoint: \"<configure-datadog-endpoint>\"\n  jaeger:\n    endpoint: \"<configure-jaeger-endpoint>\"\n\nservice:\n  pipelines:\n    traces:\n      receivers: [otlp]\n      processors: [memory_limiter, batch]\n      exporters: [datadog, jaeger]\n    metrics:\n      receivers: [otlp]\n      processors: [memory_limiter, batch]\n      exporters: [datadog, jaeger]\n    logs:\n      receivers: [otlp]\n      processors: [memory_limiter, batch]\n      exporters: [datadog, jaeger]\n"
  }
}
```

---

## generate_ottl

Generate OTTL (OpenTelemetry Transformation Language) transform processor statements for log parsing, span manipulation, or metric operations.

This skill takes a signal type and a natural-language description of the desired transformation, then produces the appropriate OTTL statements and a ready-to-use processor configuration snippet.

### Use Cases

- Parsing JSON-structured log bodies into attributes
- Extracting fields from log bodies using regex patterns
- Setting or mapping severity levels from log attributes
- Adding, renaming, or removing span attributes
- Renaming or dropping metric labels to reduce cardinality

### Parameters

| Parameter | Type | Required | Description |
|---|---|---|---|
| `signal_type` | string | Yes | Signal type: `logs`, `traces`, or `metrics` |
| `operation` | string | Yes | Natural language description of the desired transformation |

### Supported Operations by Signal Type

**Logs:**

- JSON parsing (`"parse json"`, `"json parse"`)
- Severity mapping (`"severity"`, `"log level"`)
- Field extraction with regex (`"extract"`, `"regex"`)
- Custom transformations (any other description)

**Traces:**

- Add/set attributes (`"add attribute"`, `"set attribute"`)
- Delete/remove attributes (`"delete"`, `"remove"`)
- Rename attributes (`"rename"`)
- Custom transformations (any other description)

**Metrics:**

- Rename labels (`"rename"`, `"label"`)
- Drop labels (`"drop"`, `"delete"`)
- Aggregation guidance (`"aggregate"`, `"sum"`)
- Custom transformations (any other description)

### Example Invocation

```json
{
  "method": "tools/call",
  "params": {
    "name": "generate_ottl",
    "arguments": {
      "signal_type": "logs",
      "operation": "parse json body and extract fields into attributes"
    }
  }
}
```

### Example Output

```json
{
  "cluster": "production-us-east",
  "namespace": "observability",
  "timestamp": "2025-01-15T10:36:00Z",
  "tool": "generate_ottl",
  "data": {
    "skill": "generate_ottl",
    "recommendation": {
      "signalType": "logs",
      "operation": "parse json body and extract fields into attributes",
      "statements": [
        "merge_maps(cache, ParseJSON(body), \"insert\")",
        "set(attributes[\"parsed\"], cache)"
      ],
      "context": "log"
    },
    "configSnippet": "processors:\n  transform/logs_transform:\n    log_statements:\n      - context: log\n        statements:\n          - 'merge_maps(cache, ParseJSON(body), \"insert\")'\n          - 'set(attributes[\"parsed\"], cache)'\n\nservice:\n  pipelines:\n    logs:\n      processors: [transform/logs_transform]\n"
  }
}
```

### Another Example: Renaming a Span Attribute

```json
{
  "method": "tools/call",
  "params": {
    "name": "generate_ottl",
    "arguments": {
      "signal_type": "traces",
      "operation": "rename http.method to http.request.method"
    }
  }
}
```

Output:

```json
{
  "cluster": "production-us-east",
  "namespace": "observability",
  "timestamp": "2025-01-15T10:37:00Z",
  "tool": "generate_ottl",
  "data": {
    "skill": "generate_ottl",
    "recommendation": {
      "signalType": "traces",
      "operation": "rename http.method to http.request.method",
      "statements": [
        "set(attributes[\"new.name\"], attributes[\"old.name\"]) where attributes[\"old.name\"] != nil",
        "delete_key(attributes, \"old.name\")"
      ],
      "context": "span"
    },
    "configSnippet": "processors:\n  transform/traces_transform:\n    span_statements:\n      - context: span\n        statements:\n          - 'set(attributes[\"new.name\"], attributes[\"old.name\"]) where attributes[\"old.name\"] != nil'\n          - 'delete_key(attributes, \"old.name\")'\n\nservice:\n  pipelines:\n    traces:\n      processors: [transform/traces_transform]\n"
  }
}
```

!!! note
    The generated OTTL statements use template placeholders (like `old.name` and `new.name`) that you should replace with your actual attribute names. The AI assistant will typically do this substitution for you when presenting the results.
