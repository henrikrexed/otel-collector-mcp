# GitHub Copilot Integration

Use otel-collector-mcp with GitHub Copilot in VS Code for collector analysis directly in your editor.

## Prerequisites

- GitHub Copilot extension installed in VS Code
- GitHub Copilot Chat enabled
- otel-collector-mcp deployed with v2 enabled
- Port-forward or network access to the MCP server

## Expose the MCP Server

```bash
kubectl port-forward svc/otel-collector-mcp 8080:8080 -n otel-mcp
```

## Configuration

### VS Code Settings

Add the MCP server to your VS Code `settings.json`:

```json
{
  "github.copilot.chat.mcpServers": {
    "otel-collector-mcp": {
      "url": "http://localhost:8080/mcp",
      "transport": "streamable-http"
    }
  }
}
```

Or add it to your workspace `.vscode/settings.json` for project-specific configuration.

### Copilot Agent Mode

For Copilot agent mode (agentic chat), add to `.vscode/mcp.json` in your workspace root:

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

## Example Chat Prompts

### Quick health check

In Copilot Chat (`Ctrl+Shift+I` / `Cmd+Shift+I`):

```
@workspace Check the health of the OpenTelemetry Collector "gateway-collector"
in the "observability" namespace
```

### Full analysis

```
@workspace Analyze the collector "gateway-collector" in "observability" namespace
(staging environment). Capture 60 seconds of signals, detect issues, suggest
fixes, and report findings. Clean up when done.
```

### Resource sizing while editing Helm values

While editing your `values.yaml`:

```
@workspace I'm configuring resource limits for this collector. Run a sizing
analysis on "metrics-collector" in "monitoring" (dev environment) and suggest
appropriate CPU/memory values for these Helm values.
```

### PII check during code review

```
@workspace Before we merge this logging config change, run a PII scan on
"log-collector" in "logging" (dev). Capture 120 seconds and check for any
PII in log attributes.
```

### Sampling configuration

```
@workspace Recommend a tail sampling configuration for "trace-gateway" in
"observability" (staging). Capture trace data and provide a config I can
add to the collector.
```

## Copilot Agent Mode Setup

Agent mode allows Copilot to autonomously chain multiple tool calls. Enable it in VS Code:

1. Open Settings (`Ctrl+,` / `Cmd+,`)
2. Search for `github.copilot.chat.agent.enabled`
3. Enable the setting

With agent mode, Copilot will automatically:

- Call `check_health` before starting analysis
- Chain `start_analysis` → `capture_signals` → `detect_issues` → `suggest_fixes`
- Call `cleanup_debug` when finished

## Complete Workflow Example

**In Copilot Chat:**

**You:**
```
@workspace Our gateway-collector in observability is showing high memory usage.
Environment is staging. Can you analyze it and recommend fixes?
```

**Copilot:**
> I'll run a full analysis of your collector.
>
> **Health:** 3/3 pods healthy
>
> **Captured 60s of signals:** 31,200 metrics, 4,100 logs, 18,500 spans
>
> **Issues found:**
>
> | Severity | Category | Finding |
> |----------|----------|---------|
> | warning | cardinality | `http_server_duration` has 195 unique label combos |
> | warning | bloated_attrs | `exception.stacktrace` exceeds 1KB on 89 spans |
> | warning | sizing | Throughput 12,450 pts/sec exceeds 10K threshold |
>
> **Sizing recommendation:**
>
> | Resource | Current | Recommended |
> |----------|---------|-------------|
> | CPU request | 100m | 500m |
> | CPU limit | 500m | 2000m |
> | Memory request | 128Mi | 512Mi |
> | Memory limit | 256Mi | 1Gi |
>
> **Fix suggestions:**
> 1. Drop high-cardinality labels from `http_server_duration` (medium risk)
> 2. Truncate attributes to 1KB with OTTL (low risk)
>
> Would you like me to apply any fixes or update your Helm values?

## Tips

- Use `@workspace` prefix to ensure Copilot uses MCP tools
- Agent mode works best for multi-step analysis workflows
- For quick checks, ask for a single tool call (e.g., just `check_health`)
- Copilot will show tool call confirmations — approve each step
- Keep port-forward running during your session
