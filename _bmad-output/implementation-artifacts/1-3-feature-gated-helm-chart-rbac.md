# Story 1.3: Feature-Gated Helm Chart RBAC

Status: done

## Story

As a platform engineer,
I want the Helm chart to conditionally include v2 write RBAC permissions based on `v2.enabled`,
So that clusters only grant write access when v2 is explicitly opted into.

## Acceptance Criteria

1. v2.enabled=false (default) → ClusterRole has only read-only RBAC verbs (get, list, watch)
2. v2.enabled=true → ClusterRole adds write RBAC: update/patch on ConfigMaps, update/patch on OpenTelemetryCollector CRs, patch on Deployments/DaemonSets/StatefulSets
3. v2.sessionTTL and v2.maxConcurrentSessions are configurable in values.yaml and passed as env vars
4. All v1 tools produce identical outputs with v2 enabled (zero regressions)

## Tasks / Subtasks

- [ ] Task 1: Add v2 section to values.yaml
- [ ] Task 2: Add conditional RBAC rules to clusterrole.yaml
- [ ] Task 3: Add v2 env vars to deployment.yaml
- [ ] Task 4: Verify helm template renders correctly

## Dev Agent Record

### Agent Model Used

### Completion Notes List

### File List
