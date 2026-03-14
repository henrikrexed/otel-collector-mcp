# VS Code Integration

Use otel-collector-mcp directly in VS Code through MCP-compatible extensions for collector analysis alongside your code.

## Prerequisites

- VS Code with an MCP-compatible extension (e.g., Copilot, Cline, Continue, or Roo Code)
- otel-collector-mcp deployed with v2 enabled
- Port-forward or network access to the MCP server

## Expose the MCP Server

```bash
kubectl port-forward svc/otel-collector-mcp 8080:8080 -n otel-mcp
```

## MCP Extension Setup

### Option 1: Workspace MCP Configuration

Create `.vscode/mcp.json` in your workspace root:

```json
{
  "servers": {
    "otel-collector-mcp": {
      "url": "http://localhost:8080/mcp",
      "transport": "streamable-http"
    }
  }
}
```

This is picked up by VS Code's built-in MCP support and by extensions like Copilot agent mode.

### Option 2: User Settings

Add to your global `settings.json` (`Ctrl+,` / `Cmd+,` → Open Settings JSON):

```json
{
  "mcp.servers": {
    "otel-collector-mcp": {
      "url": "http://localhost:8080/mcp",
      "transport": "streamable-http"
    }
  }
}
```

### Option 3: Cline / Continue Configuration

For Cline, add to your MCP settings:

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

For Continue, add to `~/.continue/config.yaml`:

```yaml
mcpServers:
  - name: otel-collector-mcp
    url: "http://localhost:8080/mcp"
    transport: streamable-http
```

## settings.json Configuration

Full configuration options for VS Code integration:

```json
{
  "mcp.servers": {
    "otel-collector-mcp": {
      "url": "http://localhost:8080/mcp",
      "transport": "streamable-http"
    }
  },
  "mcp.defaultTimeout": 180000
}
```

| Setting | Value | Description |
|---------|-------|-------------|
| `url` | `http://localhost:8080/mcp` | MCP server endpoint |
| `transport` | `streamable-http` | MCP transport protocol |
| `mcp.defaultTimeout` | `180000` | Timeout in ms (3 min for capture operations) |

## Example Development Workflows

### Workflow 1: Validate Config Changes Before Commit

You're editing an OTel Collector config and want to verify the current collector state before pushing:

```
Check the health of "gateway-collector" in "observability" and tell me if
there are any issues I should know about before deploying my config change.
```

### Workflow 2: Debug a Failing Pipeline

You see errors in your collector logs and want to diagnose:

```
Start a dev analysis session for "log-collector" in "logging". Capture
60 seconds of signals and detect issues. I think there might be bloated
attributes causing memory pressure.
```

### Workflow 3: Generate Processor Config

You need to add a processor but aren't sure of the exact config:

```
Analyze "trace-gateway" in "observability" (staging) and recommend a tail
sampling configuration. Give me the YAML I can paste into my collector config.
```

### Workflow 4: Pre-Deployment Compliance Check

Before deploying to staging, verify no PII leaks:

```
Run a PII scan on "log-collector" in "logging" (dev environment). Capture
120 seconds and report any PII found. I need this for our compliance checklist.
```

### Workflow 5: Capacity Planning

You're planning infrastructure for next quarter:

```
Analyze "metrics-collector" in "monitoring" (dev). Capture 120 seconds during
peak hours and give me sizing recommendations. Also check if we need sampling.
```

## Complete Workflow Example

**Full analysis loop from VS Code chat:**

**You:**
```
Our metrics-collector in the monitoring namespace has been using more memory
than expected. Environment is dev. Can you do a full analysis?
```

**Assistant:**
> Starting full analysis of metrics-collector...
>
> **Health Check:** 2/2 pods healthy, running 3 days, 0 restarts.
>
> **Signal Capture (60s):** 38,400 metrics, 890 logs, 0 spans.
>
> **Issues Detected:**
>
> | # | Severity | Category | Finding |
> |---|----------|----------|---------|
> | 1 | warning | cardinality | `process_cpu_seconds_total` has 156 unique label combos |
> | 2 | warning | sizing | Throughput 640 points/sec — within normal range |
> | 3 | info | duplicates | 8 metrics with multiple data points |
>
> **Sizing Recommendation:**
>
> | Resource | Recommended |
> |----------|-------------|
> | CPU req/limit | 100m / 500m |
> | Mem req/limit | 128Mi / 256Mi |
>
> The high cardinality on `process_cpu_seconds_total` could be contributing to memory growth. Want me to generate a fix to drop the unbounded labels?

**You:**
```
Yes, show me the fix config
```

**Assistant:**
> Here's the attributes processor config:
>
> ```yaml
> processors:
>   attributes/drop-cardinality:
>     actions:
>       - key: instance_id
>         action: delete
>       - key: pod_uid
>         action: delete
> ```
>
> Risk: **medium**. Want me to apply it with the safety chain (backup → apply → health check → auto-rollback)?

## Tips

- Increase `mcp.defaultTimeout` if you're doing long captures (120s)
- Use `.vscode/mcp.json` for team-shared configuration
- Port-forward must stay active during your session
- For persistent access, consider deploying with a Gateway or Ingress
- All v2 tools are available regardless of which VS Code MCP extension you use
