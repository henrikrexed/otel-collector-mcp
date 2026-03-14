# v1 to v2 Migration Guide

## Overview

v2 is fully backward compatible with v1. All 7 v1 tools continue to work identically regardless of whether v2 is enabled.

## Step-by-Step Upgrade

### 1. Update Helm Chart

```yaml
# values.yaml
v2:
  enabled: true          # Enable v2 features
  sessionTTL: "10m"      # Session timeout (default: 10m)
  maxConcurrentSessions: 5  # Max parallel sessions (default: 5)
```

### 2. RBAC Changes

When `v2.enabled=true`, the ClusterRole adds write permissions:

| Resource | Verbs Added |
|----------|-------------|
| ConfigMaps | `update`, `patch` |
| OpenTelemetryCollector CRs | `update`, `patch` |
| Deployments, DaemonSets, StatefulSets | `patch` |

These are required for config mutation, backup annotations, and rollout triggers.

### 3. Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `V2_ENABLED` | `false` | Enable v2 tools |
| `V2_SESSION_TTL` | `10m` | Session inactivity timeout |
| `V2_MAX_SESSIONS` | `5` | Maximum concurrent sessions |

### 4. Verify Upgrade

After deploying:

1. Call `tools/list` — should show 17 tools (7 v1 + 10 v2)
2. Call any v1 tool — should produce identical output
3. Call `check_health` with a known collector — should return pod status

### 5. v2 Tool List

| Tool | Description |
|------|-------------|
| `check_health` | Check collector pod health |
| `start_analysis` | Start analysis session |
| `rollback_config` | Rollback to backup config |
| `capture_signals` | Capture live signal data |
| `cleanup_debug` | Remove debug exporter and close session |
| `detect_issues` | Run runtime detection rules |
| `suggest_fixes` | Generate fix suggestions |
| `apply_fix` | Apply a single fix with safety checks |
| `recommend_sampling` | Recommend sampling strategy |
| `recommend_sizing` | Recommend resource sizing |

## Rollback to v1

Set `v2.enabled: false` in Helm values. All v2 tools disappear, write RBAC is removed, v1 tools unchanged.
