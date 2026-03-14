# Getting Started with v2

This guide walks you through enabling and using the v2 analysis and mutation features.

## Prerequisites

- otel-collector-mcp deployed in your Kubernetes cluster
- Helm 3.x
- An OpenTelemetry Collector running as a Deployment, DaemonSet, StatefulSet, or OTel Operator CRD
- **Non-production environment** (v2 refuses production sessions)

## Step 1: Enable v2 in Helm

Update your Helm values to enable v2:

```yaml
v2:
  enabled: true
  sessionTTL: "10m"
  maxConcurrentSessions: 5
```

Deploy:

```bash
helm upgrade otel-collector-mcp deploy/helm/otel-collector-mcp \
  --set v2.enabled=true \
  --namespace otel-mcp
```

Or via environment variable directly:

```bash
V2_ENABLED=true
V2_SESSION_TTL=10m
V2_MAX_SESSIONS=5
```

## Step 2: Verify RBAC

When v2 is enabled, additional **write** permissions are required on top of the v1 read-only permissions. The Helm chart adds these automatically when `v2.enabled=true`.

### Full ClusterRole (copy-pasteable)

For non-Helm deployments, apply this ClusterRole directly:

```yaml
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: otel-collector-mcp
rules:
  # --- v1 read-only permissions (always required) ---
  - apiGroups: [""]
    resources:
      - pods
      - pods/log
      - services
      - configmaps
      - namespaces
    verbs: ["get", "list", "watch"]
  - apiGroups: ["apps"]
    resources:
      - deployments
      - daemonsets
      - statefulsets
    verbs: ["get", "list", "watch"]
  - apiGroups: ["opentelemetry.io"]
    resources:
      - opentelemetrycollectors
      - instrumentations
    verbs: ["get", "list", "watch"]
  - apiGroups: ["apiextensions.k8s.io"]
    resources:
      - customresourcedefinitions
    verbs: ["get", "list", "watch"]
  # --- v2 write permissions (required when v2.enabled=true) ---
  - apiGroups: [""]
    resources:
      - configmaps
    verbs: ["update", "patch"]
  - apiGroups: ["opentelemetry.io"]
    resources:
      - opentelemetrycollectors
    verbs: ["update", "patch"]
  - apiGroups: ["apps"]
    resources:
      - deployments
      - daemonsets
      - statefulsets
    verbs: ["patch"]
```

Don't forget the ClusterRoleBinding:

```yaml
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRoleBinding
metadata:
  name: otel-collector-mcp
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: otel-collector-mcp
subjects:
  - kind: ServiceAccount
    name: otel-collector-mcp
    namespace: otel-mcp   # adjust to your namespace
```

### Tool-to-RBAC Mapping

Each v2 tool requires specific Kubernetes API permissions. Tools marked "in-memory only" operate on data already captured in the session and make no API calls.

| Tool | API Group | Resource | Verbs | Why |
|------|-----------|----------|-------|-----|
| `check_health` | `""` | pods | get, list | List pods by label selector and check phase/readiness |
| `start_analysis` | `""` | configmaps | get | Read ConfigMap to detect GitOps annotations |
| `start_analysis` | `opentelemetry.io` | opentelemetrycollectors | get | Read CRD to detect GitOps annotations |
| `capture_signals` | `""` | configmaps | get, update | Inject debug exporter into ConfigMap config |
| `capture_signals` | `opentelemetry.io` | opentelemetrycollectors | get, update | Inject debug exporter into CRD spec |
| `capture_signals` | `""` | pods/log | get | Stream pod logs to capture signal data |
| `detect_issues` | — | — | — | In-memory only: analyzes captured signal data |
| `suggest_fixes` | — | — | — | In-memory only: generates fix configs from findings |
| `apply_fix` | `""` | configmaps | get, update | Backup config to annotation, apply new config |
| `apply_fix` | `opentelemetry.io` | opentelemetrycollectors | get, update | Backup spec to annotation, apply new config |
| `apply_fix` | `apps` | deployments, daemonsets, statefulsets | patch | Trigger rollout restart via annotation patch |
| `apply_fix` | `""` | pods | list | Post-apply health check (poll pod readiness) |
| `recommend_sampling` | — | — | — | In-memory only: analyzes captured trace data |
| `recommend_sizing` | — | — | — | In-memory only: analyzes captured throughput |
| `rollback_config` | `""` | configmaps | get, update | Read backup annotation, restore original config |
| `rollback_config` | `opentelemetry.io` | opentelemetrycollectors | get, update | Read backup annotation, restore original spec |
| `rollback_config` | `apps` | deployments, daemonsets, statefulsets | patch | Trigger rollout restart after rollback |
| `cleanup_debug` | `""` | configmaps | get, update | Remove debug exporter, clear session annotations |
| `cleanup_debug` | `opentelemetry.io` | opentelemetrycollectors | get, update | Remove debug exporter, clear session annotations |

!!! note
    ConfigMap-based collectors use core API permissions. OTel Operator CRD collectors use `opentelemetry.io` permissions. You only need the permissions matching your deployment mode, but the ClusterRole above includes both for flexibility.

### Verify RBAC

```bash
# Check v2 write permissions
kubectl auth can-i update configmaps \
  --as=system:serviceaccount:otel-mcp:otel-collector-mcp
# Should return: yes

kubectl auth can-i patch deployments.apps \
  --as=system:serviceaccount:otel-mcp:otel-collector-mcp
# Should return: yes

kubectl auth can-i update opentelemetrycollectors.opentelemetry.io \
  --as=system:serviceaccount:otel-mcp:otel-collector-mcp
# Should return: yes
```

## Step 3: Verify v2 Tools Are Registered

Check the server logs:

```bash
kubectl logs -l app.kubernetes.io/name=otel-collector-mcp -n otel-mcp | grep "v2"
```

You should see:

```
"msg":"v2 tools registered","tool_count":10
```

## Step 4: Run Your First Analysis

Using any MCP client, execute the full analysis loop:

### 4a. Check collector health

```json
{"tool": "check_health", "arguments": {"name": "my-collector", "namespace": "observability"}}
```

### 4b. Start an analysis session

```json
{"tool": "start_analysis", "arguments": {"collector_name": "my-collector", "namespace": "observability", "environment": "dev"}}
```

Save the returned `session_id` — all subsequent calls need it.

### 4c. Capture live signals

```json
{"tool": "capture_signals", "arguments": {"session_id": "<session_id>", "duration_seconds": 60}}
```

This injects a debug exporter, captures for 60 seconds, then returns a summary.

### 4d. Detect issues

```json
{"tool": "detect_issues", "arguments": {"session_id": "<session_id>"}}
```

Runs all 8 analyzers on the captured data.

### 4e. Get fix suggestions

```json
{"tool": "suggest_fixes", "arguments": {"session_id": "<session_id>"}}
```

### 4f. Apply a fix (optional)

```json
{"tool": "apply_fix", "arguments": {"session_id": "<session_id>", "suggestion_index": 0}}
```

The fix is applied with automatic backup, health check, and rollback on failure.

### 4g. Clean up

```json
{"tool": "cleanup_debug", "arguments": {"session_id": "<session_id>"}}
```

Removes the debug exporter and closes the session.

## Configuration Reference

| Setting | Env Var | Default | Description |
|---------|---------|---------|-------------|
| `v2.enabled` | `V2_ENABLED` | `false` | Enable v2 tools |
| `v2.sessionTTL` | `V2_SESSION_TTL` | `10m` | Session inactivity timeout |
| `v2.maxConcurrentSessions` | `V2_MAX_SESSIONS` | `5` | Max concurrent analysis sessions |

## What's Next

- [Safety Model](safety-model.md) — Understand the backup, health check, and rollback chain
- [Detection Rules](detection-rules.md) — All 8 analyzers with thresholds and examples
- [Use Cases](use-cases.md) — Real-world workflows
- [v2 Tools Reference](../tools/v2-tools.md) — Full parameter and schema documentation
