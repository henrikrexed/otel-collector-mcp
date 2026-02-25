---
stepsCompleted: [1, 2, 3, 4, 5, 6, 7, 8]
inputDocuments: ['product-brief-otel-collector-mcp-2026-02-25.md', 'prd.md', 'sibling-project-mcp-k8s-networking']
workflowType: 'architecture'
project_name: 'otel-collector-mcp'
user_name: 'Henrik.rexed'
date: '2026-02-25'
lastStep: 8
status: 'complete'
completedAt: '2026-02-25'
updatedAt: '2026-02-25'
updateNotes: 'Added ADR-013 (OTel GenAI + MCP Semantic Conventions). Updated project structure: tracer.go → telemetry.go + metrics.go. Updated FR coverage to 62 FRs (added FR-OTel-1 to FR-OTel-8).'
classification:
  projectType: developer_tool
  domain: observability_infrastructure
  complexity: high
  projectContext: greenfield
---

# Architecture Decision Document

_otel-collector-mcp — An MCP server for OpenTelemetry Collector troubleshooting and design guidance in Kubernetes._

## Project Context Analysis

### Requirements Overview

**Functional Requirements:**
62 FRs organized into 11 capability areas:
- **Collector Discovery & Detection** (FR1-FR4): Auto-detect deployment types, list collectors, retrieve configs, identify versions
- **Triage & Diagnosis** (FR5-FR20): Triage scan, log parsing, 12 misconfig detection rules
- **Remediation** (FR21-FR23): Config fix generation, corrected YAML output, impact explanations
- **Architecture Design Guidance** (FR24-FR27): Topology recommendations, opinionated rationale, skeleton config generation
- **OTTL Transform Generation** (FR28-FR31): Log parsing, span manipulation, metric operations, complete config blocks
- **Infrastructure & Deployment** (FR32-FR38): Helm deployment, health endpoints, structured logging, self-instrumentation, Gateway API HTTPRoute, configurable gateway provider, SecurityContext
- **OTel Self-Instrumentation** (FR-OTel-1 to FR-OTel-8): MCP semantic convention spans, context propagation from _meta, GenAI metrics, custom domain metrics, slog→OTel log bridge, sanitized span attributes, error.type handling, OTLP gRPC export of all 3 signals
- **Multi-Cluster Identity** (FR39-FR41): Cluster identification in every response, configurable cluster name
- **CI/CD Pipeline** (FR42-FR48): Go build (multi-arch), unit/integration tests, Docker multi-arch image, security scanning (Trivy/gosec/govulncheck), linting, MkDocs deployment
- **User-Facing Documentation** (FR49-FR54): MkDocs site with Getting Started, Tool Reference, Skills Reference, Architecture Guide, Contributing Guide, Troubleshooting Guide

**Non-Functional Requirements:**
- **Performance**: Tool responses <10s, triage scan <30s, log parsing <5s for 1000 lines, cold start <15s
- **Security**: Read-only RBAC, no credential storage, credential values never in output
- **Scalability**: 50 concurrent collector instances, <256MB memory for 100 collector pods
- **Reliability**: Graceful Kubernetes API timeouts, independent rule failure, auto-recovery from transient errors
- **Integration**: MCP SSE transport, any MCP client, loose coupling with k8s-networking-mcp

**Scale & Complexity:**
- Primary domain: Go backend, Kubernetes-native MCP server
- Complexity level: High (deep OTel domain knowledge, Kubernetes API integration, CRD parsing, OTTL syntax understanding)
- Estimated architectural components: 8 packages

### Technical Constraints & Dependencies

- Must run in-cluster with client-go in-cluster config
- Read-only Kubernetes API access (no write operations)
- Must parse OpenTelemetryCollector CRDs (both v1alpha1 and v1beta1)
- Must understand OTel Collector config YAML schema (receivers, processors, exporters, connectors, pipelines)
- MCP protocol via SSE transport (same as sibling project pattern)
- Must self-instrument with OpenTelemetry SDK (dogfooding)

### Cross-Cutting Concerns Identified

- **Error handling**: Graceful degradation when RBAC is insufficient or API is unreachable
- **Structured logging**: slog JSON handler throughout, consistent with sibling project
- **Telemetry**: OTel SDK traces, metrics, and logs on all tool invocations following GenAI + MCP semantic conventions (ADR-013)
- **Config parsing**: OTel Collector YAML config must be parsed into structured types for analysis
- **Log streaming**: Pod log access via client-go for both collector and Operator pods

## Technology Stack & Starter Foundation

### Primary Technology Domain

Go backend — in-cluster Kubernetes MCP server. No starter template needed; follows the proven architecture established by the sibling project `mcp-k8s-networking`.

### Technology Stack

| Technology | Version | Purpose |
|-----------|---------|---------|
| Go | 1.25.0 | Language |
| MCP Go SDK | v1.3.1 | `github.com/modelcontextprotocol/go-sdk` — MCP protocol implementation |
| client-go | v0.35.1 | `k8s.io/client-go` — Kubernetes API access |
| k8s API | v0.35.1 | `k8s.io/api` — Kubernetes API types |
| k8s apimachinery | v0.35.1 | `k8s.io/apimachinery` — Kubernetes type machinery |
| OTel Go SDK | v1.40.0 | `go.opentelemetry.io/otel` — Self-instrumentation |
| OTel OTLP exporter | v1.40.0 | `go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc` |
| slog | stdlib | Structured JSON logging |

### Rationale for Following Sibling Project Patterns

The sibling project `mcp-k8s-networking` has established production-proven patterns for:
- MCP server lifecycle (StreamableHTTP handler, tool registration, health checks)
- CRD-based feature discovery with watch loop
- Dynamic tool registration/deregistration
- Kubernetes client setup (in-cluster → kubeconfig fallback)
- DiagnosticFinding types for standardized diagnostic output
- Helm chart structure with RBAC, health probes, OTel integration

Reusing these patterns ensures consistency across the MCP server family, reduces architectural risk, and allows knowledge transfer between projects.

## Core Architectural Decisions

### ADR-001: MCP Transport — Streamable HTTP (SSE)

**Decision:** Use MCP Streamable HTTP handler with SSE transport, exposed on a single `/mcp` endpoint.

**Rationale:** Matches the MCP specification, the Go MCP SDK's native support, and the sibling project's pattern. SSE provides real-time streaming for long-running operations (triage scan) without WebSocket complexity.

**Consequences:** Requires HTTP service exposure in-cluster. Health checks on a separate port (port+1).

### ADR-002: CRD-Based Discovery for OTel Operator Detection

**Decision:** Use CRD watch loop (same pattern as mcp-k8s-networking) to detect whether the OTel Operator is installed by watching for `opentelemetry.io` API group CRDs.

**Rationale:** The Operator may or may not be installed. Tools that analyze Operator CRDs and Operator logs should only be registered when the Operator is present. Dynamic discovery avoids startup failures and provides runtime adaptability.

**Features to detect:**
- `opentelemetry.io` API group → HasOTelOperator (enables Operator log parsing, CRD analysis tools)
- Target Allocator CRD presence → HasTargetAllocator (enables TA-related detection rules)

**Consequences:** Tool set varies based on cluster state. MCP clients see only relevant tools. Discovery must expose a `Features` struct and `IsReady()` state for readiness probes.

### ADR-003: Collector Config Parsing via YAML Unmarshaling

**Decision:** Parse OTel Collector configuration YAML into Go structs for programmatic analysis. Do NOT use the upstream collector config package (too heavy, too tightly coupled). Define lightweight structs that capture the fields we need for detection rules.

**Rationale:** Detection rules need to inspect pipeline topology (which receivers/processors/exporters are in which pipeline), processor configuration (is memory_limiter present?), and exporter configuration (are retry/queue settings present?). String-based YAML parsing is fragile. Structured unmarshaling enables reliable, testable detection logic.

**Struct design:**
```go
type CollectorConfig struct {
    Receivers  map[string]interface{} `yaml:"receivers"`
    Processors map[string]interface{} `yaml:"processors"`
    Exporters  map[string]interface{} `yaml:"exporters"`
    Connectors map[string]interface{} `yaml:"connectors"`
    Service    ServiceConfig          `yaml:"service"`
}

type ServiceConfig struct {
    Pipelines map[string]PipelineConfig `yaml:"pipelines"`
}

type PipelineConfig struct {
    Receivers  []string `yaml:"receivers"`
    Processors []string `yaml:"processors"`
    Exporters  []string `yaml:"exporters"`
}
```

**Consequences:** Must be maintained as collector config schema evolves. Keep structs minimal — only parse what detection rules need.

### ADR-004: Log Parsing via Streaming Pod Logs (client-go)

**Decision:** Use `client-go`'s `GetLogs()` with `Follow: false` and `TailLines` parameter to fetch recent collector and Operator pod logs for analysis.

**Rationale:** Collector logs are the primary diagnostic signal. client-go provides native, authenticated log access without additional infrastructure. Limiting to tail lines (default: 1000) keeps memory bounded and response times fast.

**Consequences:** Cannot access logs from crashed/deleted pods (only current pod logs). Historical log analysis requires external log aggregation (out of scope).

### ADR-005: Detection Rules as Composable Analyzers

**Decision:** Each detection rule is a standalone analyzer function with a common signature:

```go
type Analyzer func(ctx context.Context, input *AnalysisInput) []types.DiagnosticFinding

type AnalysisInput struct {
    Config       *CollectorConfig        // Parsed collector config
    DeployMode   DeploymentMode          // DaemonSet, Deployment, StatefulSet, Sidecar
    Logs         []string                // Recent collector log lines
    OperatorLogs []string                // Recent Operator log lines (if available)
    PodInfo      *corev1.Pod             // Collector pod metadata
}
```

**Rationale:** Composable analyzers allow: (1) running all analyzers in the triage scan, (2) running individual analyzers for targeted diagnosis, (3) independent failure — one analyzer crash doesn't affect others, (4) easy addition of new detection rules without modifying existing code.

**Consequences:** Each analyzer must be self-contained. Shared analysis utilities go in a `pkg/analysis/helpers.go` package.

### ADR-006: Remediation as Part of DiagnosticFinding

**Decision:** Extend the sibling project's `DiagnosticFinding` type with a `Remediation` field containing the specific config change:

```go
type DiagnosticFinding struct {
    Severity    string       // "critical", "warning", "info", "ok"
    Category    string       // pipeline, config, security, performance, operator
    Resource    *ResourceRef // Kind, Namespace, Name, APIVersion
    Summary     string       // Short description
    Detail      string       // Detailed explanation
    Suggestion  string       // Human-readable fix description
    Remediation string       // Specific YAML config block to apply (optional)
}
```

**Rationale:** The PRD requires detection and remediation as a single flow. Including the corrected config in the finding eliminates a separate remediation tool call and matches the "instant diagnosis + fix" value proposition.

**Consequences:** Analyzers must generate the corrected YAML, not just describe the problem. The `Remediation` field is optional (some findings like "exporter queue full" don't have a config fix).

### ADR-007: zpages Access via Kubernetes Port-Forward or Service

**Decision:** Defer zpages integration to v2. When implemented, access zpages via a temporary Kubernetes port-forward to the collector pod's zpages port (default 55679), or via the collector's zpages Service if exposed.

**Rationale:** zpages requires either port-forwarding (adds complexity and is ephemeral) or a pre-exposed Service (depends on user configuration). The MVP value proposition focuses on config and log analysis, which don't require zpages. Adding it in v2 allows time to design the right UX.

**Consequences:** v1 does not have live pipeline health data from zpages. Detection relies on config analysis and log parsing.

### ADR-008: MCP Tools vs Prompts/Skills Boundary

**Decision:**
- **MCP Tools** (reactive): All detection, diagnosis, and triage capabilities. These are invoked by the AI assistant when the user asks "what's wrong with my collector?"
- **MCP Prompts** (proactive): Architecture design guidance and OTTL generation. These are guided workflows where the MCP asks clarifying questions and generates structured output.

**Rationale:** Tools are fire-and-forget diagnostic operations. Prompts are multi-turn conversations that need context. The MCP protocol supports both. This maps naturally to the PRD's reactive/proactive split.

**Consequences:** Skills use the same registry pattern as the sibling project (`pkg/skills/`). Prompts are registered as MCP resources/prompts, not tools.

### ADR-009: Read-Only RBAC — No Write Operations

**Decision:** The MCP server performs only read operations against the Kubernetes API. No CRD creation, no config modifications, no pod management.

**Rationale:** Minimizes blast radius. The MCP diagnoses and suggests fixes; the user or their CI/CD pipeline applies them. This matches the security principle of least privilege and avoids the risk of the MCP accidentally modifying production collector configs.

**RBAC scope:**
```yaml
rules:
  - apiGroups: [""]
    resources: ["pods", "pods/log", "services", "configmaps", "namespaces"]
    verbs: ["get", "list", "watch"]
  - apiGroups: ["apps"]
    resources: ["deployments", "daemonsets", "statefulsets"]
    verbs: ["get", "list", "watch"]
  - apiGroups: ["opentelemetry.io"]
    resources: ["opentelemetrycollectors", "instrumentations"]
    verbs: ["get", "list", "watch"]
  - apiGroups: ["apiextensions.k8s.io"]
    resources: ["customresourcedefinitions"]
    verbs: ["get", "list", "watch"]
```

**Consequences:** Cannot apply fixes automatically. Remediation output is YAML that the user must apply.

### ADR-010: Multi-Cluster Identity in Every Response

**Decision:** Every MCP tool response includes cluster identification fields (`cluster`, `namespace`, `context`) in the `StandardResponse` envelope. The cluster name is configured via the `CLUSTER_NAME` environment variable (set via Helm chart value `config.clusterName`).

**Rationale:** A single AI agent may connect to multiple otel-collector-mcp instances across different clusters. Without cluster identification in every response, the agent cannot disambiguate which cluster a diagnostic finding belongs to. This is critical for multi-cluster environments where the same collector names may exist in different clusters.

**Implementation:**
```go
type StandardResponse struct {
    Cluster   string      `json:"cluster"`    // From CLUSTER_NAME env var
    Namespace string      `json:"namespace"`  // MCP server's own namespace
    Context   string      `json:"context"`    // Kubernetes context (if available)
    Timestamp string      `json:"timestamp"`
    Tool      string      `json:"tool"`
    Data      interface{} `json:"data"`
}
```

The `Cluster` field is populated from `config.ClusterName` which reads `CLUSTER_NAME` env var. The Helm chart sets this via `config.clusterName` value, defaulting to `""` (empty string indicates unconfigured).

**Consequences:** Every tool must use `StandardResponse` — no direct JSON output. The cluster name must be set explicitly; there is no reliable auto-detection of cluster identity from within a pod.

### ADR-011: Gateway API Exposure via HTTPRoute

**Decision:** The Helm chart includes an optional Gateway API HTTPRoute resource to expose the MCP server externally. The gateway provider (Istio, Envoy Gateway, Cilium, NGINX, kgateway) is configurable via chart values.

**Rationale:** MCP clients (AI assistants, IDE extensions) outside the cluster need HTTP access to the `/mcp` endpoint. Gateway API is the Kubernetes-standard successor to Ingress, providing portable routing across gateway implementations. The sibling project `mcp-k8s-networking` uses the same pattern.

**Helm values structure:**
```yaml
gateway:
  enabled: false
  className: ""           # e.g., "istio", "envoy-gateway", "cilium", "nginx", "kgateway"
  hostname: ""            # e.g., "otel-mcp.example.com"
  port: 8080
  tls:
    enabled: false
    certificateRef:
      name: ""
      namespace: ""
  annotations: {}         # Gateway-provider-specific annotations
```

**HTTPRoute template:** `deploy/helm/otel-collector-mcp/templates/httproute.yaml` — conditionally rendered when `gateway.enabled` is true. References the parent gateway by `className`.

**Consequences:** Users must have a Gateway API-compatible controller installed. The chart does not install or manage the gateway controller itself. TLS termination is handled at the gateway level, not by the MCP server.

### ADR-012: CI/CD Pipeline Architecture (GitHub Actions)

**Decision:** Use GitHub Actions with a multi-stage pipeline: lint → test → build → security scan → Docker build → docs deploy. Workflows are defined in `.github/workflows/`.

**Rationale:** GitHub Actions is the standard CI/CD for open-source Go projects hosted on GitHub. Multi-arch Docker builds (amd64 + arm64) are native via `docker/build-push-action` with QEMU. Security scanning (Trivy, gosec, govulncheck) provides supply-chain and code security assurance.

**Workflow design:**

```
.github/workflows/
├── ci.yml              # Triggered on push/PR to main
│   ├── lint            # golangci-lint
│   ├── test            # go test ./... with coverage
│   ├── build           # go build for linux/amd64, linux/arm64
│   └── security        # gosec, govulncheck
├── docker.yml          # Triggered on tag push (vX.Y.Z)
│   └── docker-build    # Multi-arch Docker image (amd64+arm64), push to GHCR
├── docs.yml            # Triggered on push to main (docs/ changes)
│   └── mkdocs-deploy   # Build MkDocs site, deploy to GitHub Pages
└── release.yml         # Triggered on tag push (vX.Y.Z)
    └── goreleaser      # Binary releases with checksums
```

**Key CI decisions:**
- **Lint before test:** Fail fast on code quality issues
- **Security scan on every PR:** gosec for source code, govulncheck for dependency vulnerabilities
- **Trivy on Docker image:** Container image vulnerability scanning on release builds
- **MkDocs deploy on main:** Documentation updates publish automatically
- **Multi-arch Docker:** Both amd64 and arm64 for production Kubernetes clusters (including ARM-based nodes)

**Consequences:** Contributors need no local CI tooling — all checks run in GitHub Actions. Docker image is published to GitHub Container Registry (ghcr.io). MkDocs site is hosted on GitHub Pages.

### ADR-013: OTel GenAI + MCP Semantic Conventions for Self-Instrumentation

**Decision:** Implement full self-instrumentation following the OTel GenAI and MCP semantic conventions (`opentelemetry.io/docs/specs/semconv/gen-ai/mcp/`). Replace `pkg/telemetry/tracer.go` with `pkg/telemetry/telemetry.go` that initializes all 3 OTel signals (traces, metrics, logs). Add `pkg/telemetry/metrics.go` for metric instruments. Instrument the MCP server's tool dispatch path in `pkg/mcp/server.go`.

**Rationale:** The MCP server is an OpenTelemetry tool — it should dogfood OTel. The OTel spec now defines MCP-specific semantic conventions under the GenAI namespace. Following these conventions ensures that traces, metrics, and logs produced by the MCP server are interoperable with any OTel-compatible backend and can be correlated with upstream AI agent telemetry via context propagation from `params._meta`.

**Span conventions:**
- **Span name**: `{mcp.method.name} {gen_ai.tool.name}` (e.g., `tools/call triage_scan`)
- **SpanKind**: `SERVER`
- **Required attributes**: `mcp.method.name`, `gen_ai.tool.name`, `gen_ai.operation.name="execute_tool"`, `mcp.protocol.version="2025-06-18"`
- **Recommended attributes**: `mcp.session.id`, `jsonrpc.request.id`, `jsonrpc.protocol.version="2.0"`, `network.transport="tcp"`, `server.address`, `server.port`
- **Opt-in attributes**: `gen_ai.tool.call.arguments` (sanitized, 1KB max), `gen_ai.tool.call.result` (truncated)
- **Error handling**: `error.type` set to JSON-RPC error code or `tool_error`; `span.SetStatus(codes.Error)`

**Context propagation:**
Extract `traceparent`/`tracestate` from MCP request `params._meta` using `propagation.MapCarrier`, enabling end-to-end traces from AI agent → MCP server → K8s API.

**Metrics:**
- `gen_ai.server.request.duration` (Float64Histogram) — tool execution latency
- `gen_ai.server.request.count` (Int64Counter) — tool invocations by tool name and error type
- `mcp.findings.total` (Int64Counter) — diagnostic findings by severity and analyzer
- `mcp.collectors.discovered` (Int64Gauge) — number of discovered collectors
- `mcp.errors.total` (Int64Counter) — error count by type

**Logs:**
- Bridge `slog` to OTel via `otelslog.NewHandler` with a tee handler (stdout JSON + OTel export)
- Automatic `trace_id`/`span_id` correlation in exported logs

**Implementation:**
- `pkg/telemetry/telemetry.go`: `InitTelemetry(ctx, enabled, endpoint) (func(), error)` — sets up TracerProvider, MeterProvider, LoggerProvider, W3C propagator
- `pkg/telemetry/metrics.go`: `Metrics` struct with all instruments, `NewMetrics() *Metrics`
- `pkg/mcp/server.go`: Instrumented `handleMCP` with span creation, metric recording, context extraction from `_meta`
- `pkg/config/config.go`: `SlogLevel() slog.Level` method for log bridge setup
- `cmd/server/main.go`: Replace `InitTracer` with `InitTelemetry`, add log bridge setup

**Consequences:** Self-instrumentation adds ~5 new Go dependencies (metric/log exporters, log bridge). Traces are only exported when `OTEL_ENABLED=true`. The telemetry overhead is negligible for the MCP server's request volume. All existing tool behavior is unchanged — instrumentation is purely additive.

### Decision Priority Analysis

**Critical Decisions (Block Implementation):**
- ADR-001: MCP Transport (SSE via StreamableHTTP)
- ADR-003: Config parsing approach (YAML unmarshaling)
- ADR-005: Detection rule architecture (composable analyzers)
- ADR-009: Read-only RBAC
- ADR-010: Multi-cluster identity in every response

**Important Decisions (Shape Architecture):**
- ADR-002: CRD discovery for Operator detection
- ADR-004: Log parsing via client-go streaming
- ADR-006: Remediation in DiagnosticFinding
- ADR-008: Tools vs Prompts boundary
- ADR-011: Gateway API exposure via HTTPRoute
- ADR-012: CI/CD pipeline architecture (GitHub Actions)
- ADR-013: OTel GenAI + MCP semantic conventions for self-instrumentation

**Deferred Decisions (Post-MVP):**
- ADR-007: zpages access strategy (v2)
- Plugin system for community detection rules (v2)
- Cross-MCP communication with k8s-networking-mcp (v3)

## Implementation Patterns & Consistency Rules

### Naming Patterns

**Go Package Naming:**
- Package names: lowercase, single word (`tools`, `analysis`, `discovery`, `config`)
- No underscores or mixed case in package names
- Internal packages use `internal/` if needed for encapsulation

**Go Code Naming:**
- Types: PascalCase (`DiagnosticFinding`, `CollectorConfig`, `AnalysisInput`)
- Functions: PascalCase for exported, camelCase for unexported
- Variables: camelCase (`collectorConfig`, `podLogs`)
- Constants: PascalCase for exported (`SeverityCritical`), camelCase for unexported
- Interfaces: PascalCase, named by behavior (`Analyzer`, `Tool`)

**MCP Tool Naming:**
- Format: `verb_noun` with snake_case (`detect_deployment_type`, `triage_scan`, `parse_collector_logs`)
- Consistent verb vocabulary: `detect_`, `parse_`, `check_`, `list_`, `get_`, `design_`, `generate_`

**File Naming:**
- Go files: snake_case (`collector_config.go`, `triage_scan.go`)
- Test files: `*_test.go` co-located with source
- Tool files: `tool_<name>.go` (e.g., `tool_triage.go`, `tool_detect_deployment.go`)
- Analyzer files: `analyzer_<name>.go` (e.g., `analyzer_missing_batch.go`)

### Structure Patterns

**Tool Implementation:**
All tools embed `BaseTool` and implement the `Tool` interface:
```go
type BaseTool struct {
    Cfg     *config.Config
    Clients *k8s.Clients
}

type TriageScanTool struct {
    BaseTool
    Analyzers []analysis.Analyzer
}

func (t *TriageScanTool) Name() string { return "triage_scan" }
func (t *TriageScanTool) Description() string { return "..." }
func (t *TriageScanTool) InputSchema() map[string]interface{} { return ... }
func (t *TriageScanTool) Run(ctx context.Context, args map[string]interface{}) (*StandardResponse, error) { ... }
```

**Analyzer Implementation:**
Each analyzer is a function, not a struct:
```go
// pkg/analysis/analyzer_missing_batch.go
func AnalyzeMissingBatch(ctx context.Context, input *AnalysisInput) []types.DiagnosticFinding {
    // Check each pipeline for batch processor
    // Return findings if missing
}
```

**Error Handling:**
- Use `types.MCPError` for structured MCP errors with error codes
- Error codes: `ErrCodeCollectorNotFound`, `ErrCodeConfigParseFailed`, `ErrCodeRBACInsufficient`, `ErrCodeLogAccessFailed`
- Tools return partial results + error findings when possible (never fail silently)
- Log errors with `slog.Error(...)` including structured context

**Logging:**
```go
slog.Info("tool invoked", "tool", toolName, "namespace", ns, "collector", name)
slog.Error("failed to parse collector config", "error", err, "namespace", ns, "collector", name)
slog.Debug("analyzer complete", "analyzer", "missing_batch", "findings", len(findings))
```

### Format Patterns

**MCP Tool Response:**
All tools return `*StandardResponse`:
```go
type StandardResponse struct {
    Cluster   string      `json:"cluster"`    // From CLUSTER_NAME env var (ADR-010)
    Namespace string      `json:"namespace"`  // MCP server's own namespace
    Context   string      `json:"context"`    // Kubernetes context (if available)
    Timestamp string      `json:"timestamp"`
    Tool      string      `json:"tool"`
    Data      interface{} `json:"data"`
}
```

Where `Data` is typically `*types.ToolResult`:
```go
type ToolResult struct {
    Findings []DiagnosticFinding `json:"findings"`
    Metadata ClusterMetadata     `json:"metadata"`
}
```

**Timestamp format:** RFC3339 (`time.Now().UTC().Format(time.RFC3339)`)

### Process Patterns

**Tool Execution Flow:**
1. Parse and validate input args
2. Resolve collector (by name/namespace or auto-detect)
3. Gather data (config, logs, pod info)
4. Run analysis
5. Return `StandardResponse` with findings

**Graceful Degradation:**
- If RBAC denies a specific resource: return a finding with `SeverityWarning` explaining what couldn't be checked
- If collector config can't be parsed: return error finding, don't crash
- If Operator not installed: Operator-specific tools are not registered (CRD discovery handles this)

### Enforcement Guidelines

**All AI Agents MUST:**
- Follow the `BaseTool` embedding pattern for all new tools
- Use the `Analyzer` function signature for all detection rules
- Return `StandardResponse` from all tool `Run()` methods
- Log with `slog` using structured fields (no fmt.Printf or log.Println)
- Include `Remediation` field in findings when a config fix is possible
- Register tools in `cmd/server/main.go` following the registration pattern
- Write `_test.go` files co-located with every new analyzer

## Project Structure & Boundaries

### Complete Project Directory Structure

```
otel-collector-mcp/
├── cmd/
│   └── server/
│       └── main.go                     # Server initialization, tool registration, lifecycle
├── pkg/
│   ├── analysis/                       # Detection rule analyzers
│   │   ├── analyzer.go                 # Analyzer type definition, AnalysisInput
│   │   ├── helpers.go                  # Shared analysis utilities
│   │   ├── analyzer_missing_batch.go
│   │   ├── analyzer_missing_memory_limiter.go
│   │   ├── analyzer_hardcoded_tokens.go
│   │   ├── analyzer_missing_retry_queue.go
│   │   ├── analyzer_receiver_bindings.go
│   │   ├── analyzer_tail_sampling_daemonset.go
│   │   ├── analyzer_invalid_regex.go
│   │   ├── analyzer_connector_misconfig.go
│   │   ├── analyzer_resource_detector_conflicts.go
│   │   ├── analyzer_cumulative_delta.go
│   │   ├── analyzer_high_cardinality.go
│   │   ├── analyzer_exporter_backpressure.go
│   │   └── *_test.go                   # Co-located tests for each analyzer
│   ├── collector/                      # Collector config parsing and discovery
│   │   ├── config.go                   # CollectorConfig YAML types and parsing
│   │   ├── config_test.go
│   │   ├── detect.go                   # Deployment type detection (DS/Deploy/SS/CRD)
│   │   ├── detect_test.go
│   │   ├── logs.go                     # Collector and Operator log streaming/parsing
│   │   └── logs_test.go
│   ├── config/                         # Server configuration
│   │   └── config.go                   # Config struct, env vars, SetupLogging
│   ├── discovery/                      # CRD-based feature discovery
│   │   └── discovery.go                # CRD watcher, Features struct, onChange callback
│   ├── k8s/                            # Kubernetes client setup
│   │   └── client.go                   # Clients struct (Dynamic, Discovery, Clientset)
│   ├── mcp/                            # MCP server wrapper
│   │   └── server.go                   # MCP server, StreamableHTTP handler, tool sync
│   ├── skills/                         # Proactive skills (MCP prompts)
│   │   ├── registry.go                 # Skill registry with feature sync
│   │   ├── types.go                    # SkillDefinition, SkillResult
│   │   ├── skill_architecture.go       # Architecture design guidance
│   │   ├── skill_ottl.go              # OTTL transform generation
│   │   └── *_test.go
│   ├── telemetry/                      # OpenTelemetry self-instrumentation
│   │   ├── telemetry.go                # InitTelemetry: TracerProvider, MeterProvider, LoggerProvider, slog bridge
│   │   └── metrics.go                  # Metrics struct: GenAI + MCP metric instruments
│   ├── tools/                          # MCP tool implementations
│   │   ├── registry.go                 # Thread-safe tool registry
│   │   ├── types.go                    # Tool interface, BaseTool, StandardResponse
│   │   ├── tool_triage.go             # Triage scan (runs all analyzers)
│   │   ├── tool_detect_deployment.go  # Auto-detect collector deployment type
│   │   ├── tool_list_collectors.go    # List all collector instances
│   │   ├── tool_get_config.go         # Retrieve running collector config
│   │   ├── tool_parse_logs.go         # Parse collector logs for errors
│   │   ├── tool_parse_operator_logs.go # Parse Operator logs
│   │   ├── tool_check_config.go       # Run misconfig detection suite
│   │   └── *_test.go
│   └── types/                          # Shared types
│       ├── findings.go                 # DiagnosticFinding, severity/category constants
│       ├── errors.go                   # MCPError with error codes
│       └── metadata.go                 # ToolResult, ClusterMetadata, StandardResponse
├── deploy/
│   ├── helm/
│   │   └── otel-collector-mcp/
│   │       ├── Chart.yaml
│   │       ├── values.yaml
│   │       └── templates/
│   │           ├── deployment.yaml
│   │           ├── service.yaml
│   │           ├── serviceaccount.yaml
│   │           ├── clusterrole.yaml
│   │           ├── clusterrolebinding.yaml
│   │           ├── namespace.yaml
│   │           ├── httproute.yaml      # Gateway API HTTPRoute (conditional on gateway.enabled)
│   │           └── _helpers.tpl
│   └── manifests/                      # Static YAML for non-Helm install
│       └── install.yaml
├── docs/                               # MkDocs documentation source
│   ├── mkdocs.yml                      # MkDocs configuration (material theme)
│   └── docs/
│       ├── index.md                    # Landing page
│       ├── getting-started.md          # Deployment, connection, first triage scan
│       ├── tools/
│       │   └── index.md                # Tool Reference (each MCP tool with params/examples)
│       ├── skills/
│       │   └── index.md                # Skills Reference (each MCP skill with use cases)
│       ├── architecture/
│       │   └── index.md                # Architecture Guide (deployment patterns, multi-cluster, Gateway API)
│       ├── contributing.md             # Contributing Guide (adding detection rules, skills, docs)
│       └── troubleshooting.md          # Troubleshooting Guide (installation, connectivity)
├── go.mod
├── go.sum
├── Makefile
├── Dockerfile
└── .github/
    └── workflows/
        ├── ci.yml                      # Lint, test, build, security scan on push/PR
        ├── docker.yml                  # Multi-arch Docker build + push to GHCR on tag
        ├── docs.yml                    # MkDocs build + deploy to GitHub Pages on main
        └── release.yml                 # goreleaser binary releases on tag
```

### Architectural Boundaries

**API Boundaries:**
- Single MCP endpoint: `/mcp` (StreamableHTTP)
- Health check endpoints: `/healthz` (liveness), `/readyz` (readiness — after CRD discovery complete)
- No REST API — all interaction through MCP protocol

**Package Boundaries:**
- `pkg/tools/` → depends on `pkg/analysis/`, `pkg/collector/`, `pkg/types/`, `pkg/config/`, `pkg/k8s/`
- `pkg/analysis/` → depends on `pkg/collector/`, `pkg/types/` (no dependency on `pkg/tools/`)
- `pkg/collector/` → depends on `pkg/k8s/`, `pkg/config/` (no dependency on `pkg/analysis/`)
- `pkg/skills/` → depends on `pkg/collector/`, `pkg/types/`, `pkg/config/`, `pkg/k8s/`
- `pkg/discovery/` → depends on `pkg/k8s/` only
- `pkg/types/` → no internal dependencies (leaf package)
- `cmd/server/` → depends on all `pkg/` packages (composition root)

**Data Flow:**
```
MCP Client → /mcp endpoint → MCP Server → Tool Registry → Tool.Run()
                                                              ↓
                                              ┌───────────────┴───────────────┐
                                              ↓                               ↓
                                    pkg/collector/              pkg/analysis/
                                    (config parsing,            (detection rules)
                                     log streaming,                   ↓
                                     deployment detection)     DiagnosticFinding[]
                                              ↓                       ↓
                                    Kubernetes API              StandardResponse
                                    (client-go)                       ↓
                                                               MCP Client
```

### Requirements to Structure Mapping

| FR Category | Package / Location | Key Files |
|------------|-------------------|-----------|
| Collector Discovery (FR1-FR4) | `pkg/collector/` | `detect.go`, `config.go` |
| Triage & Diagnosis (FR5-FR20) | `pkg/tools/`, `pkg/analysis/` | `tool_triage.go`, `analyzer_*.go` |
| Remediation (FR21-FR23) | `pkg/analysis/` | Embedded in each `analyzer_*.go` |
| Architecture Design (FR24-FR27) | `pkg/skills/` | `skill_architecture.go` |
| OTTL Generation (FR28-FR31) | `pkg/skills/` | `skill_ottl.go` |
| Infrastructure & Deployment (FR32-FR38) | `cmd/server/`, `deploy/helm/`, `pkg/telemetry/` | `main.go`, Helm chart, `httproute.yaml`, `telemetry.go` |
| OTel Self-Instrumentation (FR-OTel-1 to FR-OTel-8) | `pkg/telemetry/`, `pkg/mcp/`, `cmd/server/` | `telemetry.go`, `metrics.go`, `server.go`, `main.go` |
| Multi-Cluster Identity (FR39-FR41) | `pkg/config/`, `pkg/types/` | `config.go` (ClusterName), `metadata.go` (StandardResponse) |
| CI/CD Pipeline (FR42-FR48) | `.github/workflows/` | `ci.yml`, `docker.yml`, `docs.yml`, `release.yml` |
| User-Facing Documentation (FR49-FR54) | `docs/` | `mkdocs.yml`, `getting-started.md`, `tools/`, `skills/`, `architecture/`, `contributing.md`, `troubleshooting.md` |

### Integration Points

**Internal Communication:**
- Tools invoke analyzers by calling `Analyzer` functions directly (no RPC, no message bus)
- Discovery triggers tool registration via `onChange` callback
- MCP server syncs tools via `SyncTools()` diffing mechanism

**External Integrations:**
- Kubernetes API (client-go): pod logs, deployments, daemonsets, statefulsets, configmaps, CRDs
- OTel Operator API (if present): OpenTelemetryCollector CRDs
- OTLP endpoint (optional): self-instrumentation export

**Sibling MCP (k8s-networking-mcp):**
- No shared state or direct communication
- Architecture design skill can suggest: "For connectivity issues, use k8s-networking-mcp"
- Same Helm chart patterns for consistent deployment experience

## Architecture Validation Results

### Coherence Validation

**Decision Compatibility:**
All technology choices are compatible. Go 1.25, client-go v0.35.1, MCP SDK v1.3.1, and OTel SDK v1.40.0 are all current and work together. The sibling project validates this exact dependency combination in production.

**Pattern Consistency:**
Naming conventions (snake_case for tools, PascalCase for Go types) are consistent with the sibling project. The `BaseTool` embedding pattern, `StandardResponse` format, and `DiagnosticFinding` types create a uniform interface across all tools.

**Structure Alignment:**
The `pkg/` directory structure separates concerns cleanly: `analysis/` (detection logic) is independent from `tools/` (MCP tool wrappers), enabling analyzer reuse and independent testing.

### Requirements Coverage Validation

**Functional Requirements Coverage:**
- FR1-FR4 (Discovery): `pkg/collector/detect.go` + `pkg/collector/config.go`
- FR5-FR20 (Triage/Detection): `pkg/tools/tool_triage.go` + 12 analyzers in `pkg/analysis/`
- FR21-FR23 (Remediation): `Remediation` field in `DiagnosticFinding`, generated by each analyzer
- FR24-FR27 (Architecture): `pkg/skills/skill_architecture.go`
- FR28-FR31 (OTTL): `pkg/skills/skill_ottl.go`
- FR32-FR38 (Infrastructure & Deployment): `cmd/server/main.go`, Helm chart (incl. `httproute.yaml`), `pkg/telemetry/`, `pkg/config/`
- FR-OTel-1 to FR-OTel-8 (OTel Self-Instrumentation): `pkg/telemetry/telemetry.go`, `pkg/telemetry/metrics.go`, `pkg/mcp/server.go`, `cmd/server/main.go`
- FR39-FR41 (Multi-Cluster Identity): `pkg/config/config.go` (ClusterName), `pkg/types/metadata.go` (StandardResponse with cluster fields)
- FR42-FR48 (CI/CD Pipeline): `.github/workflows/ci.yml`, `docker.yml`, `docs.yml`, `release.yml`
- FR49-FR54 (Documentation): `docs/` MkDocs site with all required pages

All 62 FRs are covered (54 original + 8 OTel self-instrumentation).

**Non-Functional Requirements Coverage:**
- Performance: Bounded log parsing (TailLines), stateless tools, no heavy computation
- Security: Read-only RBAC (ADR-009), no credential storage, credentials redacted in output
- Scalability: Stateless design, <256MB memory target via bounded analysis inputs
- Reliability: Independent analyzer failure, graceful RBAC degradation, auto-recovery
- Integration: MCP SSE transport, standard StreamableHTTP handler

### Implementation Readiness Validation

**Decision Completeness:** All critical and important decisions are documented with ADRs, rationale, and Go code examples.

**Structure Completeness:** Complete project tree with all files, packages, and their responsibilities defined.

**Pattern Completeness:** Naming, structure, format, and process patterns are documented with examples and anti-patterns.

### Architecture Completeness Checklist

**Requirements Analysis**
- [x] Project context thoroughly analyzed (PRD + product brief + sibling project patterns)
- [x] Scale and complexity assessed (high complexity, 8 packages)
- [x] Technical constraints identified (read-only RBAC, in-cluster, CRD compatibility)
- [x] Cross-cutting concerns mapped (error handling, logging, telemetry, config parsing)

**Architectural Decisions**
- [x] 13 ADRs documented with rationale and consequences
- [x] Technology stack fully specified with versions
- [x] Integration patterns defined (Kubernetes API, MCP, OTel, sibling MCP, Gateway API)
- [x] Performance considerations addressed (bounded inputs, stateless design)
- [x] Multi-cluster identity pattern defined (ADR-010)
- [x] CI/CD pipeline architecture defined (ADR-012)

**Implementation Patterns**
- [x] Naming conventions established (Go packages, tools, files, MCP tools)
- [x] Structure patterns defined (BaseTool, Analyzer, StandardResponse)
- [x] Communication patterns specified (tool → analyzer → findings)
- [x] Process patterns documented (tool execution flow, graceful degradation)

**Project Structure**
- [x] Complete directory structure defined with all files
- [x] Package boundaries established with dependency rules
- [x] Integration points mapped (Kubernetes API, MCP, OTel)
- [x] Requirements to structure mapping complete (all 54 FRs)

### Architecture Readiness Assessment

**Overall Status:** READY FOR IMPLEMENTATION

**Confidence Level:** High — architecture reuses proven patterns from the sibling project, all technology choices are validated in production, and the scope is well-defined.

**Key Strengths:**
- Proven patterns from sibling project reduce architectural risk
- Composable analyzer architecture enables easy addition of detection rules
- Read-only RBAC minimizes blast radius
- Clear package boundaries prevent circular dependencies
- Detection + remediation coupling matches the core value proposition

**Areas for Future Enhancement:**
- zpages integration (v2) — needs UX design for port-forward lifecycle
- Community detection rule plugin system (v2) — needs plugin API design
- Cross-MCP orchestration (v3) — needs protocol for MCP-to-MCP recommendations

### Implementation Handoff

**AI Agent Guidelines:**
- Follow all ADRs exactly as documented
- Use `BaseTool` embedding for all new tools
- Use `Analyzer` function signature for all detection rules
- Return `StandardResponse` from all tool `Run()` methods
- Register tools in `cmd/server/main.go` following the sibling project's pattern
- Write `_test.go` co-located with every analyzer and tool
- Log with `slog` using structured fields

**First Implementation Priority:**
1. Scaffold project: `go mod init`, create `pkg/` directory structure, copy type definitions from sibling project
2. Implement `pkg/config/`, `pkg/k8s/`, `pkg/types/` (foundation packages)
3. Implement `pkg/collector/detect.go` (deployment type detection)
4. Implement `pkg/collector/config.go` (collector config YAML parsing)
5. Implement first analyzer: `analyzer_missing_batch.go` (simplest detection rule)
6. Implement `pkg/tools/tool_triage.go` (triage scan wiring all analyzers)
7. Implement `cmd/server/main.go` (MCP server lifecycle)
8. Create Helm chart from sibling project template
