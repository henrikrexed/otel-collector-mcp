# Response Format

All tools return **compact markdown tables** optimized for LLM token efficiency.

## Format

```markdown
| St | Resource | Summary | Detail |
|----|----------|---------|--------|
| ✅ | OpenTelemetryCollector default/otel | 3 receivers, 2 processors, 2 exporters | - |
| ⚠️ | - | Missing memory_limiter processor | OOM risk → add memory_limiter |
| ❗ | ConfigMap default/otel-config | Hardcoded API token in exporter | security risk → use env var |
```

## Severity Icons

| Icon | Level | Meaning |
|------|-------|---------|
| ✅ | OK | No issues found |
| ℹ️ | Info | Informational finding |
| ⚠️ | Warning | Potential issue, investigate |
| ❗ | Critical | Requires immediate action |

## get_config Response

The `get_config` tool returns a **structured summary** of the collector configuration,
not the full raw YAML. This includes:

- List of receiver, processor, exporter, and connector names
- Pipeline definitions (which components are in each pipeline)
- Extension names

This is sufficient for LLM diagnosis while keeping token count low.

## Design Decisions

### Why markdown tables instead of JSON?

MCP tools are primarily consumed by LLM agents. JSON responses are **3x more token-expensive**:

| Format | Tokens per finding |
|--------|-------------------|
| Pretty JSON | ~80 |
| **Markdown table** | ~20 |

### 12 Misconfiguration Detectors

The `check_config` tool runs 12 analyzers. Each finding appears as a row in the table
with severity, the specific misconfiguration, and a remediation suggestion.
