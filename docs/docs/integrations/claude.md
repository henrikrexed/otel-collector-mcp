# Claude Integration

Use otel-collector-mcp with **Claude Desktop** and **Claude Code CLI** for interactive collector analysis.

## Prerequisites

- otel-collector-mcp deployed with v2 enabled
- Network access from your machine to the MCP server (port-forward or ingress)
- Claude Desktop app or Claude Code CLI installed

## Expose the MCP Server

If the MCP server runs in-cluster, port-forward for local access:

```bash
kubectl port-forward svc/otel-collector-mcp 8080:8080 -n otel-mcp
```

Or use a Gateway/Ingress if configured:

```bash
# If gateway is enabled in Helm values
curl http://otel-mcp.example.com/mcp
```

## Claude Desktop

### Configuration

Add the MCP server to `claude_desktop_config.json`:

=== "macOS"

    ```
    ~/Library/Application Support/Claude/claude_desktop_config.json
    ```

=== "Windows"

    ```
    %APPDATA%\Claude\claude_desktop_config.json
    ```

=== "Linux"

    ```
    ~/.config/Claude/claude_desktop_config.json
    ```

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

Restart Claude Desktop after saving the config.

### Verify Connection

In Claude Desktop, you should see the MCP tools icon in the chat input area. Click it to verify `check_health`, `start_analysis`, and the other v2 tools are listed.

### Example Prompts

**Quick health check:**

```
Check the health of the OpenTelemetry Collector named "gateway-collector"
in the "observability" namespace.
```

**Full analysis workflow:**

```
I need to analyze our staging collector for issues. Here's what I need:

1. Check health of "gateway-collector" in "observability"
2. Start a staging analysis session
3. Capture 60 seconds of signals
4. Detect any issues
5. Suggest fixes for anything found
6. Clean up when done

Summarize findings with severity and recommended actions.
```

**PII audit:**

```
Run a PII compliance scan on "log-collector" in the "logging" namespace.
Use a dev environment session, capture 120 seconds of data, and report
any PII detected in log attributes. Include fix suggestions.
```

**Capacity planning:**

```
Help me right-size "metrics-collector" in "monitoring" (dev environment).
Capture signals for 120 seconds, then give me resource sizing and
sampling recommendations.
```

## Claude Code CLI

### Configuration

Add the MCP server to your project's `.mcp.json` or global `~/.claude/mcp.json`:

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

### Usage

Claude Code automatically discovers the MCP tools. Use them in conversation:

```bash
claude
```

Then in the CLI:

```
> Check the health of collector "gateway-collector" in namespace "observability"

> Start a staging analysis for gateway-collector in observability, capture
  60 seconds of signals, detect issues, and suggest fixes. Clean up after.
```

### Scripted Usage

Use Claude Code in non-interactive mode for automated checks:

```bash
claude -p "Check health of collector 'gateway-collector' in namespace 'observability' and report any issues"
```

## Complete Workflow Example

Here's a full interactive session in Claude Desktop or Claude Code:

**You:**
> Analyze gateway-collector in the observability namespace. Environment is staging.

**Claude:**
> Let me run a full analysis.
>
> **Health Check:** All 3 pods healthy, running for 5 days.
>
> **Session started:** `a1b2c3d4-e5f6-...`
>
> **Capturing signals for 60 seconds...**
> Captured 24,120 metrics, 3,450 logs, 15,890 spans.
>
> **Issues detected (3):**
>
> | # | Severity | Category | Finding |
> |---|----------|----------|---------|
> | 1 | warning | cardinality | http_requests_total: 287 unique label combos |
> | 2 | warning | pii | user.email: email pattern in 42 logs |
> | 3 | info | sampling | No sampling processor configured |
>
> **Fix suggestions (2):**
>
> | # | Type | Risk | Description |
> |---|------|------|-------------|
> | 0 | attribute | medium | Drop high-cardinality keys |
> | 1 | ottl | low | Redact email patterns |
>
> Would you like me to apply any of these fixes?

**You:**
> Apply fix 1 (the PII redaction)

**Claude:**
> Applied OTTL transform to redact email patterns. Health check passed — all 3 pods healthy.
> Session cleaned up.

## Tips

- Keep `kubectl port-forward` running while using Claude Desktop/Code
- For persistent access, configure a Gateway or Ingress for the MCP server
- Claude will automatically chain the v2 tools in the right order when you describe what you want
- Always specify the environment (`dev` or `staging`) — production is blocked
- You can ask Claude to explain findings before applying fixes
