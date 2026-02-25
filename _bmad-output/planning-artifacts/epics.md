---
stepsCompleted: [step-01-validate-prerequisites, step-02-design-epics, step-03-create-stories, step-04-final-validation]
inputDocuments: ['prd.md', 'architecture.md']
workflowType: 'epics-and-stories'
project_name: 'otel-collector-mcp'
user_name: 'Henrik.rexed'
date: '2026-02-25'
status: 'complete'
completedAt: '2026-02-25'
---

# otel-collector-mcp - Epic Breakdown

## Overview

This document provides the complete epic and story breakdown for otel-collector-mcp, decomposing the requirements from the PRD and Architecture into implementable stories.

## Requirements Inventory

### Functional Requirements

- FR1: MCP server can auto-detect the deployment type (DaemonSet, Deployment, StatefulSet, OTel Operator CRD) of any OTel Collector instance in the cluster
- FR2: MCP server can list all OTel Collector instances across all namespaces (or a specified namespace)
- FR3: MCP server can retrieve the running configuration of a detected collector instance
- FR4: MCP server can identify the OTel Collector version running in each instance
- FR5: MCP server can execute a triage scan that runs all detection rules against a specified collector and returns a prioritized issue list
- FR6: Each issue in the triage scan result includes severity (critical/warning/info), description, affected config section, and specific remediation
- FR7: MCP server can parse collector pod logs and classify errors into categories (OTTL syntax, exporter failure, OOM, receiver issue, processor error)
- FR8: MCP server can parse OTel Operator pod logs and detect rejected CRDs and reconciliation failures
- FR9: MCP server can detect missing batch processor in any pipeline
- FR10: MCP server can detect missing memory_limiter processor in any pipeline
- FR11: MCP server can detect hardcoded tokens or credentials in exporter configurations
- FR12: MCP server can detect missing retry_on_failure and sending_queue on exporters
- FR13: MCP server can detect receiver endpoint / port binding mismatches
- FR14: MCP server can detect tail_sampling processor configured on a DaemonSet deployment
- FR15: MCP server can detect invalid regex patterns in filter processor configurations
- FR16: MCP server can detect connector misconfigurations (e.g., referencing non-existent pipelines)
- FR17: MCP server can detect conflicting resource detection processors that overwrite each other
- FR18: MCP server can detect cumulative-to-delta conversion issues (e.g., non-stateful deployment without persistent storage)
- FR19: MCP server can detect high-cardinality metric labels by analyzing metric metadata or config patterns
- FR20: MCP server can detect exporter queue saturation and backpressure conditions from collector logs
- FR21: Every detection (FR9-FR20) includes a specific remediation: the exact config change, addition, or removal needed to fix the issue
- FR22: Remediation output includes the corrected YAML config block that can be directly applied
- FR23: MCP server can explain why a detected issue is problematic (impact description) alongside the fix
- FR24: MCP server can recommend a collector deployment topology based on user-described workload (signal types, scale, backend targets)
- FR25: Architecture recommendations include opinionated rationale for each decision (why DaemonSet for logs, why StatefulSet for Target Allocator, etc.)
- FR26: Architecture recommendations include the hybrid Agent→Gateway pattern when appropriate
- FR27: MCP server can generate skeleton collector configuration for a recommended architecture
- FR28: MCP server can generate OTTL transform processor statements for log parsing use cases described in natural language
- FR29: MCP server can generate OTTL transform processor statements for span attribute manipulation
- FR30: MCP server can generate OTTL transform processor statements for metric label operations (rename, drop, aggregate)
- FR31: Generated OTTL statements include the complete transform processor config block (not just the OTTL expression)
- FR32: MCP server can be deployed in-cluster via Helm chart with configurable RBAC, namespace scope, resource limits, and replica count
- FR33: MCP server exposes health and readiness endpoints for Kubernetes liveness/readiness probes
- FR34: MCP server emits structured logs (slog) with configurable log level
- FR35: MCP server instruments itself with OpenTelemetry SDK (traces and metrics) for observability
- FR36: Helm chart exposes the MCP server via Gateway API HTTPRoute resource
- FR37: Helm chart supports configurable gateway provider (Istio, Envoy Gateway, Cilium, NGINX, kgateway) as a chart variable
- FR38: Helm chart includes configurable SecurityContext (runAsNonRoot, readOnlyRootFilesystem, allowPrivilegeEscalation: false)
- FR39: Every MCP tool response includes cluster identification (cluster name, Kubernetes context, MCP server namespace) so an AI agent connected to multiple otel-collector-mcp instances can distinguish which cluster data comes from
- FR40: Cluster identification fields are present in every StandardResponse and DiagnosticFinding output, not just top-level metadata
- FR41: Cluster name is configurable via Helm chart value and environment variable (CLUSTER_NAME)
- FR42: GitHub Actions workflow builds the Go binary for linux/amd64 and linux/arm64
- FR43: GitHub Actions workflow runs unit tests and reports results
- FR44: GitHub Actions workflow runs integration tests against a Kubernetes test cluster (or mocks)
- FR45: GitHub Actions workflow builds multi-arch Docker image (amd64 + arm64) and publishes as GitHub artifact / container registry
- FR46: GitHub Actions workflow runs security scanning: Trivy (container image), gosec (Go source), govulncheck (Go dependencies)
- FR47: GitHub Actions workflow runs golangci-lint for code quality
- FR48: GitHub Actions workflow builds MkDocs documentation site and deploys to GitHub Pages
- FR49: MkDocs documentation site includes a Getting Started guide covering deployment (Helm), connecting to an AI assistant, and running first triage scan
- FR50: MkDocs documentation site includes a Tool Reference documenting each MCP tool with parameters, examples, and sample output
- FR51: MkDocs documentation site includes a Skills Reference documenting each MCP skill with use cases and generated output examples
- FR52: MkDocs documentation site includes an Architecture Guide covering deployment patterns, multi-cluster setup, and Gateway API exposure
- FR53: MkDocs documentation site includes a Contributing Guide for adding detection rules, skills, and documentation
- FR54: MkDocs documentation site includes a Troubleshooting Guide for common installation and connectivity issues

### NonFunctional Requirements

- NFR1: MCP tool responses complete within 10 seconds for single-collector operations under normal cluster load
- NFR2: Triage scan (all detectors) completes within 30 seconds for a single collector instance
- NFR3: Log parsing processes the most recent 1000 log lines within 5 seconds
- NFR4: MCP server cold start to ready state within 15 seconds
- NFR5: MCP server operates with read-only Kubernetes RBAC — no write operations to cluster resources
- NFR6: No secrets or credentials stored in the MCP server's own configuration or memory beyond what client-go uses for in-cluster auth
- NFR7: All detection rules that flag hardcoded credentials do not include the credential values in their output
- NFR8: MCP SSE transport supports TLS when fronted by a Kubernetes Service or Ingress
- NFR9: Single MCP server instance supports up to 50 concurrent collector instances in the cluster
- NFR10: Tool invocations are stateless — horizontal scaling via additional replicas if needed
- NFR11: Memory footprint remains under 256MB for clusters with up to 100 collector pods
- NFR12: MCP server gracefully handles Kubernetes API timeouts and returns partial results with clear indication of what couldn't be checked
- NFR13: Detection rules fail independently — one rule failure does not prevent other rules from executing
- NFR14: MCP server recovers automatically from transient Kubernetes API errors without requiring pod restart
- NFR15: MCP server supports MCP protocol via SSE transport as defined by the MCP specification
- NFR16: Compatible with any MCP-capable client (Claude Desktop, IDE extensions, custom clients)
- NFR17: k8s-networking-mcp integration: MCP server can suggest invoking the sibling MCP for network-related diagnosis but does not depend on it being present

### Additional Requirements

- Architecture specifies no starter template — greenfield project following sibling project patterns from mcp-k8s-networking
- All tools must embed BaseTool and implement the Tool interface (ADR pattern from architecture)
- All detection rules must use the Analyzer function signature (ADR-005)
- All tool responses must use StandardResponse with cluster identity fields (ADR-010)
- CRD-based discovery for OTel Operator detection with dynamic tool registration (ADR-002)
- Collector config YAML parsed into Go structs, not raw string manipulation (ADR-003)
- Pod logs accessed via client-go GetLogs with TailLines parameter (ADR-004)
- Remediation field in DiagnosticFinding for detection+fix coupling (ADR-006)
- Read-only RBAC with specific ClusterRole rules (ADR-009)
- Gateway API HTTPRoute template in Helm chart (ADR-011)
- Multi-stage CI/CD with GitHub Actions (ADR-012)
- MkDocs with Material theme for documentation site
- Dockerfile for multi-arch container image
- Makefile for local development commands

### FR Coverage Map

FR1: Epic 1 — Auto-detect deployment type
FR2: Epic 1 — List all collector instances
FR3: Epic 1 — Retrieve running collector config
FR4: Epic 1 — Identify collector version
FR5: Epic 3 — Triage scan entry point
FR6: Epic 3 — Triage scan prioritized findings with severity
FR7: Epic 2 — Parse collector logs and classify errors
FR8: Epic 2 — Parse Operator logs for CRD/reconciliation failures
FR9: Epic 3 — Detect missing batch processor
FR10: Epic 3 — Detect missing memory_limiter
FR11: Epic 3 — Detect hardcoded tokens/credentials
FR12: Epic 3 — Detect missing retry/queue on exporters
FR13: Epic 3 — Detect receiver binding mismatches
FR14: Epic 3 — Detect tail sampling on DaemonSet
FR15: Epic 3 — Detect invalid regex in filter processor
FR16: Epic 3 — Detect connector misconfigurations
FR17: Epic 3 — Detect resource detector conflicts
FR18: Epic 3 — Detect cumulative-to-delta issues
FR19: Epic 3 — Detect high-cardinality metric labels
FR20: Epic 3 — Detect exporter backpressure
FR21: Epic 3 — Remediation in every detection
FR22: Epic 3 — Corrected YAML config output
FR23: Epic 3 — Impact explanation with fix
FR24: Epic 4 — Architecture design recommendations
FR25: Epic 4 — Opinionated rationale
FR26: Epic 4 — Hybrid Agent→Gateway pattern
FR27: Epic 4 — Skeleton config generation
FR28: Epic 4 — OTTL log parsing generation
FR29: Epic 4 — OTTL span manipulation generation
FR30: Epic 4 — OTTL metric operations generation
FR31: Epic 4 — Complete transform processor config block
FR32: Epic 5 — Helm chart deployment
FR33: Epic 1 — Health and readiness endpoints
FR34: Epic 1 — Structured logging (slog)
FR35: Epic 1 — OTel self-instrumentation
FR36: Epic 5 — Gateway API HTTPRoute
FR37: Epic 5 — Configurable gateway provider
FR38: Epic 5 — SecurityContext in Helm chart
FR39: Epic 1 — Cluster identity in every response
FR40: Epic 1 — Cluster identity in StandardResponse and DiagnosticFinding
FR41: Epic 5 — Cluster name configurable via Helm/env
FR42: Epic 6 — CI build for multi-arch
FR43: Epic 6 — CI unit tests
FR44: Epic 6 — CI integration tests
FR45: Epic 6 — CI Docker multi-arch build
FR46: Epic 6 — CI security scanning
FR47: Epic 6 — CI golangci-lint
FR48: Epic 6 — CI MkDocs deploy
FR49: Epic 7 — Getting Started guide
FR50: Epic 7 — Tool Reference
FR51: Epic 7 — Skills Reference
FR52: Epic 7 — Architecture Guide
FR53: Epic 7 — Contributing Guide
FR54: Epic 7 — Troubleshooting Guide

## Epic List

### Epic 1: Project Foundation & Collector Discovery
Engineers can deploy the MCP server in-cluster and discover all OTel Collector instances, their deployment types, configs, and versions — providing the foundational data layer that all diagnosis and guidance tools depend on.
**FRs covered:** FR1, FR2, FR3, FR4, FR33, FR34, FR35, FR39, FR40

### Epic 2: Collector & Operator Log Analysis
Engineers can parse and analyze collector and Operator pod logs, getting classified error breakdowns that pinpoint what's failing (OTTL syntax errors, exporter failures, OOM events, rejected CRDs, reconciliation issues).
**FRs covered:** FR7, FR8

### Epic 3: Misconfig Detection & Triage Scan
Engineers can run a triage scan that executes all 12 detection rules against a collector, returning a prioritized list of issues with severity, impact explanation, and the exact YAML config fix for each problem.
**FRs covered:** FR5, FR6, FR9, FR10, FR11, FR12, FR13, FR14, FR15, FR16, FR17, FR18, FR19, FR20, FR21, FR22, FR23

### Epic 4: Architecture Design & OTTL Generation Skills
Engineers can get opinionated architecture recommendations for collector deployment topology and generate OTTL transform processor configurations from natural language descriptions.
**FRs covered:** FR24, FR25, FR26, FR27, FR28, FR29, FR30, FR31

### Epic 5: Helm Chart & Production Deployment
Engineers can deploy the MCP server via a production-ready Helm chart with Gateway API exposure, configurable security context, cluster identity, and proper RBAC.
**FRs covered:** FR32, FR36, FR37, FR38, FR41

### Epic 6: CI/CD Pipeline
Contributors have a complete GitHub Actions pipeline that builds, tests, lints, security-scans, and publishes the project on every push and release.
**FRs covered:** FR42, FR43, FR44, FR45, FR46, FR47, FR48

### Epic 7: Documentation Site
Users and contributors have a comprehensive MkDocs documentation site with getting started, tool/skill reference, architecture guide, contributing guide, and troubleshooting.
**FRs covered:** FR49, FR50, FR51, FR52, FR53, FR54

---

## Epic 1: Project Foundation & Collector Discovery

Engineers can deploy the MCP server in-cluster and discover all OTel Collector instances, their deployment types, configs, and versions — providing the foundational data layer that all diagnosis and guidance tools depend on.

### Story 1.1: Initialize Go Project and Core Type Definitions

As a developer,
I want the Go project scaffolded with module, directory structure, and shared type definitions,
So that all subsequent development has a consistent foundation to build on.

**Acceptance Criteria:**

**Given** a fresh checkout of the repository
**When** I run `go build ./...`
**Then** the project compiles successfully with Go 1.25
**And** `go.mod` declares module `github.com/hrexed/otel-collector-mcp`
**And** the `pkg/types/` package contains `DiagnosticFinding`, `MCPError`, `StandardResponse`, `ToolResult`, and `ClusterMetadata` types matching the architecture document
**And** `StandardResponse` includes `Cluster`, `Namespace`, `Context`, `Timestamp`, `Tool`, and `Data` fields (FR39, FR40)
**And** `DiagnosticFinding` includes `Severity`, `Category`, `Resource`, `Summary`, `Detail`, `Suggestion`, and `Remediation` fields
**And** severity constants (`SeverityCritical`, `SeverityWarning`, `SeverityInfo`, `SeverityOk`) and category constants are defined
**And** error codes (`ErrCodeCollectorNotFound`, `ErrCodeConfigParseFailed`, `ErrCodeRBACInsufficient`, `ErrCodeLogAccessFailed`) are defined
**And** the directory structure matches the architecture document (cmd/, pkg/, deploy/, docs/, .github/)

### Story 1.2: Server Configuration and Structured Logging

As a developer,
I want a configuration package that reads environment variables and sets up structured JSON logging,
So that the server can be configured at deploy time and all logs are structured for machine consumption.

**Acceptance Criteria:**

**Given** the `pkg/config/` package exists
**When** the Config struct is initialized
**Then** it reads `PORT` (default 8080), `LOG_LEVEL` (default "info"), `CLUSTER_NAME` (default ""), `OTEL_ENABLED` (default false), and `OTEL_ENDPOINT` environment variables
**And** `SetupLogging()` configures `slog` with a JSON handler at the configured log level (FR34)
**And** the `ClusterName` field is accessible for inclusion in all StandardResponse outputs (FR39, FR41)
**And** unit tests verify default values and environment variable overrides

### Story 1.3: Kubernetes Client Setup

As a developer,
I want a Kubernetes client package that initializes in-cluster or kubeconfig-based clients,
So that all Kubernetes API operations use authenticated, properly configured clients.

**Acceptance Criteria:**

**Given** the `pkg/k8s/` package exists
**When** `NewClients()` is called
**Then** it returns a `Clients` struct containing `Clientset` (typed client), `DynamicClient`, and `DiscoveryClient`
**And** it attempts in-cluster config first, falling back to kubeconfig (`~/.kube/config`) for local development
**And** the clients are safe for concurrent use by multiple goroutines
**And** a meaningful error is returned if neither in-cluster nor kubeconfig is available

### Story 1.4: CRD-Based Feature Discovery

As a developer,
I want a CRD watcher that detects whether the OTel Operator is installed and triggers dynamic tool registration,
So that Operator-specific tools are only available when the Operator is actually present in the cluster.

**Acceptance Criteria:**

**Given** the `pkg/discovery/` package exists
**When** the CRD watcher starts
**Then** it watches for CRDs in the `opentelemetry.io` API group
**And** it exposes a `Features` struct with `HasOTelOperator` and `HasTargetAllocator` booleans
**And** it calls an `onChange` callback when features change (CRD added or removed)
**And** it exposes an `IsReady()` method that returns true after initial discovery completes
**And** the discovery loop recovers gracefully from transient API errors without crashing

### Story 1.5: MCP Server Lifecycle and Health Endpoints

As a developer,
I want the MCP server initialized with StreamableHTTP handler, tool registry, and health/readiness endpoints,
So that the server can accept MCP connections and be monitored by Kubernetes probes.

**Acceptance Criteria:**

**Given** the `pkg/mcp/` and `pkg/tools/` packages exist
**When** the MCP server starts
**Then** it exposes `/mcp` endpoint using MCP SDK's StreamableHTTP handler (ADR-001)
**And** it exposes `/healthz` (liveness) that returns 200 when the server is running (FR33)
**And** it exposes `/readyz` (readiness) that returns 200 only after CRD discovery is complete (FR33)
**And** the tool registry supports thread-safe `Register()`, `Deregister()`, and `SyncTools()` operations
**And** tools implement the `Tool` interface with `Name()`, `Description()`, `InputSchema()`, and `Run()` methods
**And** all tools embed `BaseTool` which provides `Cfg` and `Clients` fields

### Story 1.6: OpenTelemetry Self-Instrumentation

As a developer,
I want the MCP server to instrument itself with OpenTelemetry SDK for traces and metrics,
So that operators can observe the MCP server's own performance and behavior.

**Acceptance Criteria:**

**Given** the `pkg/telemetry/` package exists
**When** `InitTracer()` is called with OTel enabled in config
**Then** it configures an OTLP gRPC trace exporter to the configured endpoint (FR35)
**And** it returns a `TracerProvider` and shutdown function
**And** when OTel is disabled, it returns a no-op tracer provider
**And** tool invocations create spans with tool name, namespace, and collector name attributes

### Story 1.7: Auto-Detect Collector Deployment Type

As an observability engineer,
I want the MCP server to automatically detect whether a collector runs as a DaemonSet, Deployment, StatefulSet, or Operator CRD,
So that mode-specific best practices and detection rules can be applied.

**Acceptance Criteria:**

**Given** a collector pod is running in the cluster
**When** I invoke the `detect_deployment_type` MCP tool with namespace and pod/collector name
**Then** the tool identifies the deployment mode as one of: DaemonSet, Deployment, StatefulSet, or OperatorCRD (FR1)
**And** for Operator CRD mode, it identifies the OpenTelemetryCollector CRD resource
**And** the response is a `StandardResponse` with cluster identity fields populated (FR39, FR40)
**And** if the collector is not found, the response includes an error finding with `ErrCodeCollectorNotFound`

### Story 1.8: List All Collector Instances

As an observability engineer,
I want to list all OTel Collector instances across the cluster or a specific namespace,
So that I can see what collectors are running and choose which to diagnose.

**Acceptance Criteria:**

**Given** multiple collector instances are running in the cluster
**When** I invoke the `list_collectors` MCP tool (optionally with a namespace filter)
**Then** the tool returns all collector instances with name, namespace, deployment type, version, and pod count (FR2, FR4)
**And** it discovers collectors via label selectors (`app.kubernetes.io/component=opentelemetry-collector` or similar)
**And** it also discovers collectors managed by the OTel Operator via OpenTelemetryCollector CRDs (if Operator is present)
**And** the response is a `StandardResponse` with cluster identity fields (FR39, FR40)

### Story 1.9: Retrieve Collector Configuration

As an observability engineer,
I want to retrieve the running configuration of a specific collector instance,
So that I can inspect the pipeline topology and settings.

**Acceptance Criteria:**

**Given** a collector instance is identified by namespace and name
**When** I invoke the `get_config` MCP tool
**Then** the tool retrieves the collector's YAML configuration from the appropriate source:
- From ConfigMap for standard deployments
- From OpenTelemetryCollector CRD `.spec.config` for Operator-managed collectors (FR3)
**And** the configuration is parsed into `CollectorConfig` struct (ADR-003) with receivers, processors, exporters, connectors, and service/pipelines
**And** the raw YAML is also included in the response for display
**And** the response is a `StandardResponse` with cluster identity fields (FR39, FR40)

### Story 1.10: Main Server Entry Point

As a developer,
I want `cmd/server/main.go` to wire together all packages and start the server,
So that the complete server can be built and run as a single binary.

**Acceptance Criteria:**

**Given** all foundation packages are implemented
**When** I run `go run cmd/server/main.go`
**Then** it initializes config, logging, Kubernetes clients, CRD discovery, OTel instrumentation, MCP server, and tool registry
**And** it registers all discovery tools (detect_deployment_type, list_collectors, get_config)
**And** it starts the HTTP server on the configured port
**And** it handles graceful shutdown on SIGTERM/SIGINT
**And** it logs server start/stop events with slog

---

## Epic 2: Collector & Operator Log Analysis

Engineers can parse and analyze collector and Operator pod logs, getting classified error breakdowns that pinpoint what's failing.

### Story 2.1: Collector Log Streaming and Parsing

As an observability engineer,
I want the MCP server to fetch and parse recent collector pod logs,
So that I can quickly understand what errors the collector is experiencing without manually reading logs.

**Acceptance Criteria:**

**Given** a collector pod is running and generating logs
**When** I invoke the `parse_collector_logs` MCP tool with namespace and collector name
**Then** the tool fetches the most recent 1000 log lines using client-go `GetLogs()` with `TailLines: 1000` (ADR-004)
**And** it classifies log entries into categories: OTTL syntax error, exporter failure, OOM event, receiver issue, processor error, and other (FR7)
**And** each classified error is returned as a `DiagnosticFinding` with severity, category, summary, and relevant log excerpt
**And** log parsing completes within 5 seconds for 1000 lines (NFR3)
**And** the response is a `StandardResponse` with cluster identity fields (FR39, FR40)
**And** if pod logs are inaccessible (RBAC), a warning finding explains what couldn't be checked (NFR12)

### Story 2.2: Operator Log Parsing for CRD and Reconciliation Failures

As a platform engineer,
I want the MCP server to parse OTel Operator pod logs and detect rejected CRDs and reconciliation failures,
So that I can quickly identify why my Operator-managed collectors aren't deploying correctly.

**Acceptance Criteria:**

**Given** the OTel Operator is installed (detected via CRD discovery) and its pods are running
**When** I invoke the `parse_operator_logs` MCP tool
**Then** the tool fetches recent Operator pod logs using client-go
**And** it detects rejected CRD errors (validation failures, schema mismatches) (FR8)
**And** it detects reconciliation failures (failed to create deployment, RBAC errors, resource conflicts)
**And** each issue is returned as a `DiagnosticFinding` with the specific CRD resource reference
**And** this tool is only registered when `Features.HasOTelOperator` is true (CRD discovery gating)
**And** the response is a `StandardResponse` with cluster identity fields (FR39, FR40)

---

## Epic 3: Misconfig Detection & Triage Scan

Engineers can run a triage scan that executes all detection rules against a collector, returning a prioritized list of issues with severity, impact explanation, and the exact YAML config fix for each problem.

### Story 3.1: Analyzer Framework and Missing Batch Processor Detection

As a developer,
I want the analyzer framework defined with the first detection rule (missing batch processor),
So that the composable analyzer pattern is established and subsequent rules can follow the same pattern.

**Acceptance Criteria:**

**Given** the `pkg/analysis/` package exists
**When** the analyzer framework is initialized
**Then** the `Analyzer` type is defined as `func(ctx context.Context, input *AnalysisInput) []types.DiagnosticFinding` (ADR-005)
**And** `AnalysisInput` contains `Config`, `DeployMode`, `Logs`, `OperatorLogs`, and `PodInfo` fields
**And** `AnalyzeMissingBatch()` checks each pipeline for the presence of a batch processor (FR9)
**And** when a pipeline is missing batch processor, it returns a finding with severity "warning", explanation of impact, and a `Remediation` field containing the YAML to add batch processor (FR21, FR22, FR23)
**And** `helpers.go` contains shared utilities for pipeline inspection
**And** unit tests cover: pipeline with batch (no finding), pipeline without batch (finding with remediation), multiple pipelines

### Story 3.2: Missing Memory Limiter Detection

As an observability engineer,
I want the MCP server to detect when a pipeline is missing the memory_limiter processor,
So that I can prevent OOM crashes in my collector.

**Acceptance Criteria:**

**Given** a collector config is parsed
**When** `AnalyzeMissingMemoryLimiter()` runs
**Then** it checks each pipeline for memory_limiter processor (FR10)
**And** when missing, it returns a finding with severity "critical" (OOM risk), impact explanation, and remediation YAML showing the memory_limiter config block with recommended settings (FR21, FR22, FR23)
**And** unit tests cover: present (no finding), missing (finding with remediation)

### Story 3.3: Hardcoded Credentials Detection

As a platform engineer,
I want the MCP server to detect hardcoded tokens and credentials in exporter configurations,
So that I can ensure secrets are managed via environment variables or Kubernetes secrets.

**Acceptance Criteria:**

**Given** a collector config is parsed
**When** `AnalyzeHardcodedTokens()` runs
**Then** it scans exporter configurations for patterns matching API keys, tokens, passwords, and bearer tokens (FR11)
**And** when found, it returns a finding with severity "critical" (security risk), without including the actual credential value in the output (NFR7)
**And** the remediation suggests using `${env:VAR_NAME}` syntax or Kubernetes secret references (FR21, FR22)
**And** unit tests cover: clean config (no finding), hardcoded token (finding without leaking value)

### Story 3.4: Missing Retry/Queue and Receiver Binding Detection

As an observability engineer,
I want the MCP server to detect missing retry/queue on exporters and receiver binding issues,
So that I can prevent silent data loss and port conflicts.

**Acceptance Criteria:**

**Given** a collector config is parsed
**When** `AnalyzeMissingRetryQueue()` runs
**Then** it checks each exporter for `retry_on_failure` and `sending_queue` configuration (FR12)
**And** when missing, it returns a warning finding with remediation YAML (FR21, FR22)

**Given** a collector config is parsed
**When** `AnalyzeReceiverBindings()` runs
**Then** it checks for receiver endpoint port conflicts and missing protocol configurations (FR13)
**And** when issues found, it returns findings with specific remediation (FR21, FR22)
**And** unit tests cover both analyzers with positive and negative cases

### Story 3.5: Tail Sampling on DaemonSet and Invalid Regex Detection

As an observability engineer,
I want the MCP server to detect the anti-pattern of tail sampling on a DaemonSet and invalid regex patterns,
So that I can avoid sampling correctness issues and filter processor failures.

**Acceptance Criteria:**

**Given** a collector config and deployment mode are available
**When** `AnalyzeTailSamplingDaemonSet()` runs
**Then** it detects `tail_sampling` processor configured on a DaemonSet deployment (FR14)
**And** it returns a critical finding explaining why tail sampling requires a gateway (all spans for a trace must reach the same collector) with remediation suggesting Deployment/StatefulSet mode (FR21, FR23)

**Given** a collector config is parsed
**When** `AnalyzeInvalidRegex()` runs
**Then** it extracts regex patterns from filter processor configurations and validates them (FR15)
**And** invalid patterns return a warning finding with the specific regex error and corrected pattern if possible (FR21, FR22)
**And** unit tests cover both analyzers

### Story 3.6: Connector Misconfig and Resource Detector Conflict Detection

As an observability engineer,
I want the MCP server to detect connector misconfigurations and resource detector conflicts,
So that I can fix broken pipeline connections and attribute overwriting issues.

**Acceptance Criteria:**

**Given** a collector config is parsed
**When** `AnalyzeConnectorMisconfig()` runs
**Then** it verifies connectors reference existing pipelines on both input and output sides (FR16)
**And** mismatches return findings with the specific misconfigured connector and remediation (FR21, FR22)

**Given** a collector config is parsed
**When** `AnalyzeResourceDetectorConflicts()` runs
**Then** it checks for multiple resource detection processors in the same pipeline that may overwrite each other's attributes (FR17)
**And** conflicts return findings explaining the overwrite order and suggesting merged configuration (FR21, FR22, FR23)
**And** unit tests cover both analyzers

### Story 3.7: Cumulative-to-Delta and High-Cardinality Detection

As an observability engineer,
I want the MCP server to detect cumulative-to-delta conversion issues and high-cardinality metric patterns,
So that I can ensure correct metric conversion and prevent metric explosion.

**Acceptance Criteria:**

**Given** a collector config and deployment mode are available
**When** `AnalyzeCumulativeDelta()` runs
**Then** it detects cumulativetodelta processor on non-stateful deployments without persistent storage (FR18)
**And** it returns a warning finding explaining the risk of counter resets and suggesting StatefulSet or alternative approaches (FR21, FR23)

**Given** a collector config is parsed
**When** `AnalyzeHighCardinality()` runs
**Then** it analyzes metric processor configurations for patterns that may produce high-cardinality labels (unbounded label values, no aggregation) (FR19)
**And** it returns warning findings with remediation suggesting attribute filtering or aggregation (FR21, FR22)
**And** unit tests cover both analyzers

### Story 3.8: Exporter Backpressure Detection

As an SRE,
I want the MCP server to detect exporter queue saturation from collector logs,
So that I can identify and resolve silent data loss from backpressure.

**Acceptance Criteria:**

**Given** collector log lines are available (from log parsing)
**When** `AnalyzeExporterBackpressure()` runs (this analyzer uses logs, not just config)
**Then** it detects patterns indicating exporter queue full, dropped spans/metrics, and sending failures (FR20)
**And** it returns critical findings identifying the specific exporter and suggesting queue size increase, retry configuration, or endpoint investigation (FR21, FR22, FR23)
**And** unit tests cover: clean logs (no finding), queue full logs (finding), mixed logs

### Story 3.9: Triage Scan Tool — Unified Entry Point

As an observability engineer,
I want a single triage scan tool that runs all detection rules and returns a prioritized issue list,
So that I can get a complete health assessment of my collector in one command.

**Acceptance Criteria:**

**Given** all 12 analyzers are implemented
**When** I invoke the `triage_scan` MCP tool with namespace and collector name
**Then** it gathers the collector's config, deployment mode, recent logs, and pod info (FR5)
**And** it runs all registered analyzers against the gathered data
**And** it returns findings sorted by severity (critical → warning → info) (FR6)
**And** each finding includes severity, category, summary, detail, suggestion, and remediation (FR6)
**And** the triage scan completes within 30 seconds (NFR2)
**And** if any individual analyzer fails, the remaining analyzers still execute and the failed analyzer produces an info-level finding explaining the failure (NFR13)
**And** the response is a `StandardResponse` with cluster identity fields (FR39, FR40)

### Story 3.10: Config Check Tool — Targeted Detection

As an observability engineer,
I want a config-check tool that runs the misconfig detection suite without log analysis,
So that I can quickly validate a collector's configuration independently of its runtime state.

**Acceptance Criteria:**

**Given** a collector's config is accessible
**When** I invoke the `check_config` MCP tool with namespace and collector name
**Then** it retrieves the collector config and deployment mode
**And** it runs only the config-based analyzers (FR9-FR19, not FR20 which requires logs)
**And** it returns findings sorted by severity with remediation
**And** the response is a `StandardResponse` with cluster identity fields (FR39, FR40)

---

## Epic 4: Architecture Design & OTTL Generation Skills

Engineers can get opinionated architecture recommendations for collector deployment topology and generate OTTL transform processor configurations from natural language descriptions.

### Story 4.1: Skills Registry and Architecture Design Skill

As a platform engineer,
I want to ask the MCP server for collector architecture recommendations,
So that I can make informed decisions about deployment topology without deep collector expertise.

**Acceptance Criteria:**

**Given** the `pkg/skills/` package exists with registry and type definitions
**When** I invoke the `design_architecture` MCP prompt/skill with workload description (signal types, scale, backend targets)
**Then** the skill recommends a deployment topology (FR24):
- DaemonSet for logs (node-level /var/log access)
- Deployment or StatefulSet for metrics + traces (gateway mode)
- StatefulSet when Target Allocator is needed
- Hybrid Agent→Gateway pattern for mixed workloads (FR26)
**And** each recommendation includes opinionated rationale explaining why (FR25)
**And** the skill generates a skeleton collector configuration for the recommended architecture (FR27)
**And** the response is a `StandardResponse` with cluster identity fields (FR39, FR40)
**And** unit tests verify recommendations for at least 3 workload scenarios

### Story 4.2: OTTL Transform Generation Skill

As an observability engineer,
I want to describe a log parsing, span manipulation, or metric operation in natural language and get a working OTTL transform processor config,
So that I don't need to memorize OTTL syntax to write transform rules.

**Acceptance Criteria:**

**Given** the OTTL generation skill is registered
**When** I invoke the `generate_ottl` MCP prompt/skill with:
- A natural language description of the desired transformation
- The signal type (logs, traces, or metrics)
**Then** for log parsing use cases, it generates valid OTTL statements for the transform processor (FR28)
**And** for span attribute manipulation, it generates valid OTTL statements (FR29)
**And** for metric label operations (rename, drop, aggregate), it generates valid OTTL statements (FR30)
**And** the output includes the complete transform processor config block, not just the OTTL expression (FR31)
**And** the generated OTTL is syntactically valid (parseable by the collector)
**And** unit tests verify generation for at least one case per signal type

---

## Epic 5: Helm Chart & Production Deployment

Engineers can deploy the MCP server via a production-ready Helm chart with Gateway API exposure, configurable security context, cluster identity, and proper RBAC.

### Story 5.1: Dockerfile and Makefile

As a developer,
I want a multi-stage Dockerfile and Makefile for building and running the project,
So that the project can be containerized and built consistently.

**Acceptance Criteria:**

**Given** the Go project compiles successfully
**When** I run `docker build .`
**Then** the Dockerfile uses a multi-stage build (Go builder → distroless/static runtime)
**And** the final image runs as non-root user
**And** the binary is statically linked (CGO_DISABLED=1)
**And** the image supports both amd64 and arm64 architectures

**Given** the Makefile exists
**When** I run `make build`
**Then** it builds the Go binary for the local platform
**And** `make test` runs all unit tests
**And** `make lint` runs golangci-lint
**And** `make docker-build` builds the Docker image

### Story 5.2: Helm Chart with RBAC and Core Resources

As a platform engineer,
I want to deploy the MCP server via Helm chart with proper RBAC and Kubernetes resources,
So that I can install it in my cluster with minimal configuration.

**Acceptance Criteria:**

**Given** the Helm chart exists at `deploy/helm/otel-collector-mcp/`
**When** I run `helm install otel-mcp ./deploy/helm/otel-collector-mcp`
**Then** it creates: Namespace (configurable), ServiceAccount, ClusterRole (read-only per ADR-009), ClusterRoleBinding, Deployment, and Service (FR32)
**And** the ClusterRole includes exactly the RBAC rules from ADR-009 (pods, pods/log, services, configmaps, namespaces, deployments, daemonsets, statefulsets, opentelemetrycollectors, instrumentations, CRDs) (NFR5)
**And** `values.yaml` supports configurable: namespace, replica count, resource limits/requests, image tag, port, log level
**And** the Deployment includes liveness probe on `/healthz` and readiness probe on `/readyz` (FR33)
**And** SecurityContext sets `runAsNonRoot: true`, `readOnlyRootFilesystem: true`, `allowPrivilegeEscalation: false` by default (FR38)
**And** `config.clusterName` value is passed as `CLUSTER_NAME` environment variable to the container (FR41)
**And** `helm template` renders valid Kubernetes YAML
**And** `helm lint` passes without errors

### Story 5.3: Gateway API HTTPRoute Template

As a platform engineer,
I want the Helm chart to optionally expose the MCP server via Gateway API HTTPRoute,
So that external MCP clients can connect to the server through my cluster's gateway.

**Acceptance Criteria:**

**Given** the Helm chart is installed with `gateway.enabled=true`
**When** the chart renders
**Then** it creates an HTTPRoute resource targeting the MCP server Service (FR36)
**And** the HTTPRoute `parentRef` uses the configured `gateway.className` (FR37)
**And** the hostname is configurable via `gateway.hostname`
**And** TLS can be configured via `gateway.tls.enabled` and `gateway.tls.certificateRef`
**And** gateway-provider-specific annotations are configurable via `gateway.annotations`
**And** when `gateway.enabled=false` (default), no HTTPRoute is created
**And** the template supports Istio, Envoy Gateway, Cilium, NGINX, and kgateway providers (FR37)

---

## Epic 6: CI/CD Pipeline

Contributors have a complete GitHub Actions pipeline that builds, tests, lints, security-scans, and publishes the project on every push and release.

### Story 6.1: CI Workflow — Lint and Test

As a contributor,
I want GitHub Actions to automatically lint and test my code on every push and pull request,
So that code quality and correctness are validated before merge.

**Acceptance Criteria:**

**Given** a push or PR to the main branch
**When** the `ci.yml` workflow runs
**Then** it runs `golangci-lint` on the Go source code (FR47)
**And** it runs `go test ./...` with race detector and coverage reporting (FR43)
**And** it runs integration tests (with mocked Kubernetes API or kind cluster) (FR44)
**And** test results are reported in the GitHub Actions summary
**And** the workflow fails if lint or tests fail

### Story 6.2: CI Workflow — Build and Security Scan

As a contributor,
I want GitHub Actions to build multi-arch binaries and run security scans,
So that the project is verified for build correctness and known vulnerabilities.

**Acceptance Criteria:**

**Given** the CI workflow runs
**When** the build and security jobs execute
**Then** it builds Go binaries for `linux/amd64` and `linux/arm64` (FR42)
**And** it runs `gosec` for Go source code security analysis (FR46)
**And** it runs `govulncheck` for Go dependency vulnerability checking (FR46)
**And** security scan results are reported in the workflow summary
**And** the workflow fails on critical/high severity findings

### Story 6.3: Docker Build and Release Workflows

As a maintainer,
I want GitHub Actions to build multi-arch Docker images on release tags and publish binary releases,
So that users can pull the container image and download binaries.

**Acceptance Criteria:**

**Given** a tag matching `v*.*.*` is pushed
**When** the `docker.yml` workflow runs
**Then** it builds a multi-arch Docker image (amd64 + arm64) using docker buildx (FR45)
**And** it pushes the image to GitHub Container Registry (ghcr.io)
**And** it runs Trivy vulnerability scanning on the built image (FR46)
**And** the image is tagged with the version and `latest`

**Given** a tag matching `v*.*.*` is pushed
**When** the `release.yml` workflow runs
**Then** it creates a GitHub Release with compiled binaries and checksums

### Story 6.4: Documentation Deployment Workflow

As a maintainer,
I want GitHub Actions to automatically build and deploy the MkDocs site on documentation changes,
So that the documentation site stays current with the codebase.

**Acceptance Criteria:**

**Given** a push to main branch that modifies files in `docs/`
**When** the `docs.yml` workflow runs
**Then** it builds the MkDocs site using `mkdocs build` (FR48)
**And** it deploys the built site to GitHub Pages
**And** the workflow only triggers on changes to `docs/` directory or `mkdocs.yml`

---

## Epic 7: Documentation Site

Users and contributors have a comprehensive MkDocs documentation site with getting started, tool/skill reference, architecture guide, contributing guide, and troubleshooting.

### Story 7.1: MkDocs Site Setup and Getting Started Guide

As a new user,
I want a documentation site with a Getting Started guide,
So that I can deploy the MCP server and run my first triage scan within 15 minutes.

**Acceptance Criteria:**

**Given** the `docs/` directory exists with `mkdocs.yml`
**When** I run `mkdocs serve`
**Then** the documentation site builds and serves locally
**And** `mkdocs.yml` configures the Material theme with navigation, search, and syntax highlighting
**And** the Getting Started guide covers: prerequisites, Helm chart installation, connecting to an AI assistant (Claude Desktop, IDE), and running the first triage scan (FR49)
**And** the guide includes example commands and expected output

### Story 7.2: Tool Reference Documentation

As an observability engineer,
I want a Tool Reference page documenting every MCP tool,
So that I know what parameters each tool accepts and what output to expect.

**Acceptance Criteria:**

**Given** the documentation site is set up
**When** I navigate to the Tool Reference page
**Then** each MCP tool is documented with: name, description, input parameters (with types and defaults), example invocation, and sample output (FR50)
**And** tools covered include: detect_deployment_type, list_collectors, get_config, parse_collector_logs, parse_operator_logs, triage_scan, check_config

### Story 7.3: Skills Reference Documentation

As an observability engineer,
I want a Skills Reference page documenting each MCP skill,
So that I understand what design guidance capabilities are available and how to use them.

**Acceptance Criteria:**

**Given** the documentation site is set up
**When** I navigate to the Skills Reference page
**Then** each MCP skill is documented with: name, description, use cases, input parameters, and example generated output (FR51)
**And** skills covered include: design_architecture, generate_ottl

### Story 7.4: Architecture Guide Documentation

As a platform engineer,
I want an Architecture Guide covering deployment patterns, multi-cluster setup, and Gateway API exposure,
So that I can understand the operational aspects of running otel-collector-mcp.

**Acceptance Criteria:**

**Given** the documentation site is set up
**When** I navigate to the Architecture Guide page
**Then** it covers collector deployment patterns (DaemonSet, Deployment, StatefulSet, Operator) (FR52)
**And** it explains the multi-cluster pattern (one MCP instance per cluster, AI agent connects to multiple) (FR52)
**And** it documents Gateway API exposure configuration for each supported provider (FR52)
**And** it includes architecture diagrams showing data flow

### Story 7.5: Contributing Guide and Troubleshooting Documentation

As a contributor,
I want a Contributing Guide and Troubleshooting page,
So that I can add detection rules and skills, and debug common installation issues.

**Acceptance Criteria:**

**Given** the documentation site is set up
**When** I navigate to the Contributing Guide
**Then** it documents how to add a new detection rule (analyzer function signature, test requirements, registration) (FR53)
**And** it documents how to add a new skill (skill interface, registration, test requirements) (FR53)
**And** it includes code contribution guidelines (PR process, code style, testing requirements)

**Given** the documentation site is set up
**When** I navigate to the Troubleshooting page
**Then** it covers common installation issues (RBAC errors, pod not starting, CRD discovery failures) (FR54)
**And** it covers connectivity issues (MCP client can't connect, Gateway API routing, TLS) (FR54)
**And** it includes diagnostic steps for each issue
