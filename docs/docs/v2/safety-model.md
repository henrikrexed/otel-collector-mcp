# v2 Safety Model

The v2 safety model ensures that dynamic analysis operations never leave a collector in a broken state.

## Safety Chain

Every config mutation follows this safety chain:

```
Environment Gate → Config Backup → Apply Config → Trigger Rollout → Health Check → Success/Rollback
```

### Environment Gate

- Users must declare the environment type: `dev`, `staging`, or `production`
- **Production is always refused** — no override, no force flag, no bypass mechanism
- This is a user-declared gate, not heuristic-based

### Config Backup

- Full ConfigMap `.data` or CRD `.spec` is stored as a `mcp.otel.dev/config-backup` annotation
- Session ID stored as `mcp.otel.dev/session-id` annotation
- Backups survive MCP server pod restarts (stored on the resource itself)
- `resourceVersion` captured for optimistic concurrency

### Automatic Health Check

- After every mutation, `WaitHealthy()` polls pod health at 2-second intervals
- 30-second timeout for health verification
- Checks: pod phase (Running), readiness probes, CrashLoopBackOff detection

### Automatic Rollback

- Triggered automatically when health check detects:
  - CrashLoopBackOff
  - Readiness probe failure after timeout
- Rollback restores the backup config, triggers rollout, and verifies recovery
- Target: 100% rollback success rate

### Concurrent Session Rejection

- Only one session per collector at a time
- Maximum concurrent sessions configurable (default: 5)
- Duplicate collector sessions rejected with clear error identifying the blocking session

### Session TTL Cleanup

- Sessions expire after `SessionTTL` (default: 10 minutes) of inactivity
- Cleanup loop runs every 30 seconds
- Expired sessions: debug exporter removed, config restored, session closed

### Orphan Recovery

- On server restart, scans ConfigMaps and CRDs for `mcp.otel.dev/session-id` annotations
- Orphaned sessions cleaned up automatically with WARN-level logging

## GitOps Awareness

- Detects ArgoCD (`argocd.argoproj.io/managed-by`) and Flux (`fluxcd.io/automated`) annotations
- Returns `gitops_conflict` warning when mutations may be reverted by GitOps controllers
