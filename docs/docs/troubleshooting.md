# Troubleshooting

This guide covers common issues encountered when installing and using otel-collector-mcp, along with diagnostic steps for each.

## Installation Issues

### Pod Not Starting

**Symptom:** The otel-collector-mcp pod is in `CrashLoopBackOff` or `Error` state.

**Diagnostic steps:**

1. Check pod logs:
   ```bash
   kubectl logs -n observability -l app.kubernetes.io/name=otel-collector-mcp
   ```

2. Check pod events:
   ```bash
   kubectl describe pod -n observability -l app.kubernetes.io/name=otel-collector-mcp
   ```

**Common causes:**

- **Image pull failure** -- Verify the image repository and tag in your Helm values. Check that your cluster has access to `ghcr.io/hrexed/otel-collector-mcp`:
  ```bash
  kubectl get events -n observability --field-selector reason=Failed
  ```

- **Insufficient resources** -- The default resource requests are 100m CPU and 128Mi memory. If your cluster has strict resource quotas, adjust the Helm values:
  ```yaml
  resources:
    requests:
      cpu: 50m
      memory: 64Mi
  ```

- **Security context issues** -- The chart sets `runAsNonRoot: true` and `readOnlyRootFilesystem: true`. If your cluster has a restrictive PodSecurityPolicy or PodSecurityStandard, ensure these settings are compatible.

### RBAC Errors

**Symptom:** Logs show `forbidden` errors when the server tries to list pods, configmaps, or other resources.

**Diagnostic steps:**

1. Verify the ClusterRole exists:
   ```bash
   kubectl get clusterrole -l app.kubernetes.io/name=otel-collector-mcp
   ```

2. Verify the ClusterRoleBinding exists and references the correct ServiceAccount:
   ```bash
   kubectl get clusterrolebinding -l app.kubernetes.io/name=otel-collector-mcp -o yaml
   ```

3. Test permissions directly:
   ```bash
   kubectl auth can-i list pods --as=system:serviceaccount:observability:otel-collector-mcp --all-namespaces
   kubectl auth can-i get pods/log --as=system:serviceaccount:observability:otel-collector-mcp --all-namespaces
   kubectl auth can-i list configmaps --as=system:serviceaccount:observability:otel-collector-mcp --all-namespaces
   kubectl auth can-i list daemonsets.apps --as=system:serviceaccount:observability:otel-collector-mcp --all-namespaces
   ```

**Resolution:**

The Helm chart creates the ClusterRole, ServiceAccount, and ClusterRoleBinding automatically. If they are missing, reinstall the chart:

```bash
helm upgrade --install otel-collector-mcp deploy/helm/otel-collector-mcp \
  --namespace observability
```

If your cluster uses a custom RBAC policy or namespaced roles, ensure the ServiceAccount has read access to the resources listed in the [Getting Started - RBAC permissions](getting-started.md#rbac-permissions) section.

### CRD Discovery Fails

**Symptom:** Logs show `failed to discover server resources, will retry` warnings.

**Diagnostic steps:**

1. Check the log output:
   ```bash
   kubectl logs -n observability -l app.kubernetes.io/name=otel-collector-mcp | grep "discover"
   ```

2. Verify the ServiceAccount can access the discovery API:
   ```bash
   kubectl auth can-i list customresourcedefinitions.apiextensions.k8s.io \
     --as=system:serviceaccount:observability:otel-collector-mcp
   ```

**Resolution:**

CRD discovery failures are not fatal. The server retries every 30 seconds. If the OTel Operator is not installed, these warnings are expected -- the server will still function for standard workload types (DaemonSet, Deployment, StatefulSet). The `hasOTelOperator` feature will remain `false` until the operator CRDs are detected.

If the operator IS installed but discovery fails, ensure the ClusterRole includes:

```yaml
- apiGroups: ["apiextensions.k8s.io"]
  resources: ["customresourcedefinitions"]
  verbs: ["get", "list", "watch"]
```

### Readiness Probe Failing

**Symptom:** Pod is running but the readiness probe fails, and the Service has no endpoints.

**Diagnostic steps:**

1. Check the readiness endpoint directly:
   ```bash
   kubectl exec -n observability -it <pod-name> -- wget -qO- http://localhost:8080/readyz
   ```

2. Check pod readiness status:
   ```bash
   kubectl get pods -n observability -l app.kubernetes.io/name=otel-collector-mcp -o wide
   ```

**Resolution:**

The readiness probe depends on the initial CRD discovery completing. If the Kubernetes API server is slow or overloaded, the initial discovery may take longer than the readiness probe's `initialDelaySeconds` (default: 3 seconds). Increase the delay if needed:

```yaml
# In your Helm values or via --set
readinessProbe:
  initialDelaySeconds: 10
```

## Connectivity Issues

### MCP Client Cannot Connect

**Symptom:** The AI assistant reports it cannot reach the MCP server, or tool calls time out.

**Diagnostic steps:**

1. Verify the Service exists and has endpoints:
   ```bash
   kubectl get svc -n observability otel-collector-mcp
   kubectl get endpoints -n observability otel-collector-mcp
   ```

2. If using port-forward, verify it is running:
   ```bash
   kubectl port-forward -n observability svc/otel-collector-mcp 8080:8080
   ```

3. Test connectivity from your local machine:
   ```bash
   curl http://localhost:8080/healthz
   curl http://localhost:8080/readyz
   curl http://localhost:8080/mcp
   ```

4. Verify the MCP client configuration points to the correct URL. For Claude Desktop, check `claude_desktop_config.json`:
   ```json
   {
     "mcpServers": {
       "otel-collector-mcp": {
         "url": "http://localhost:8080/mcp"
       }
     }
   }
   ```

**Common causes:**

- **Port-forward died** -- `kubectl port-forward` can disconnect silently. Restart it.
- **Wrong port** -- Ensure the port in your MCP client config matches the port-forward source port.
- **Pod not ready** -- If the readiness probe is failing, the Service will have no endpoints and connections will be refused.

### Gateway API Route Not Working

**Symptom:** External requests to the Gateway hostname return 404 or connection refused.

**Diagnostic steps:**

1. Verify the HTTPRoute was created:
   ```bash
   kubectl get httproute -n observability otel-collector-mcp -o yaml
   ```

2. Check the HTTPRoute status for accepted conditions:
   ```bash
   kubectl get httproute -n observability otel-collector-mcp -o jsonpath='{.status.parents}'
   ```

3. Verify the parent Gateway exists and is programmed:
   ```bash
   kubectl get gateway -A
   ```

4. Check that the Gateway class matches your `gateway.className` value:
   ```bash
   kubectl get gatewayclass
   ```

5. If using TLS, verify the certificate Secret exists:
   ```bash
   kubectl get secret -n observability <certificateRef.name>
   ```

**Common causes:**

- **Gateway not provisioned** -- The Gateway resource must exist and be in `Programmed` state before the HTTPRoute can attach.
- **Wrong className** -- The `gateway.className` must match the name of an existing Gateway resource, not the GatewayClass.
- **DNS not configured** -- If you specified a `hostname`, ensure DNS points to the Gateway's external IP or load balancer.
- **TLS mismatch** -- When `tls.enabled=true`, the HTTPRoute references the `https` sectionName. Your Gateway must have a listener named `https`.

### TLS Certificate Issues

**Symptom:** HTTPS connections fail with certificate errors.

**Diagnostic steps:**

1. Verify the TLS secret exists and contains valid data:
   ```bash
   kubectl get secret -n observability <tls-secret-name> -o jsonpath='{.data.tls\.crt}' | base64 -d | openssl x509 -noout -dates
   ```

2. Verify the Gateway references the correct certificate:
   ```bash
   kubectl get gateway -n observability -o yaml | grep -A5 certificateRefs
   ```

**Resolution:**

TLS is terminated at the Gateway, not at the MCP server. Ensure your Gateway has a valid TLS listener configuration. The MCP server always serves plain HTTP on its configured port.

## Tool Execution Issues

### Tool Returns "Collector workload not found"

**Symptom:** `detect_deployment_type` or `triage_scan` reports that the collector workload was not found.

**Diagnostic steps:**

1. Verify the collector exists in the specified namespace:
   ```bash
   kubectl get daemonsets,deployments,statefulsets -n <namespace> | grep otel
   ```

2. If using the OTel Operator, check for CRDs:
   ```bash
   kubectl get opentelemetrycollectors -n <namespace>
   ```

3. Ensure the exact `name` parameter matches the workload name (not the pod name).

**Common causes:**

- **Wrong name** -- The `name` parameter should be the DaemonSet/Deployment/StatefulSet name, not a pod name.
- **Wrong namespace** -- Double-check the namespace.
- **Operator CRD not detected** -- If the collector is operator-managed, the CRD discovery must have completed. Check that `hasOTelOperator` is `true` in the startup logs.

### Tool Returns "Failed to retrieve collector configuration"

**Symptom:** `get_config`, `triage_scan`, or `check_config` cannot retrieve the ConfigMap.

**Diagnostic steps:**

1. Verify the ConfigMap exists:
   ```bash
   kubectl get configmap -n <namespace> <configmap-name>
   ```

2. Check that the `configmap` parameter matches the actual ConfigMap name.

3. Verify RBAC allows ConfigMap access:
   ```bash
   kubectl auth can-i get configmaps -n <namespace> \
     --as=system:serviceaccount:observability:otel-collector-mcp
   ```

### Log Parsing Returns No Findings

**Symptom:** `parse_collector_logs` runs successfully but returns zero classified entries.

This is normal if the collector logs contain no error patterns. The tool only classifies lines that match known error categories (OTTL syntax, exporter failure, OOM, receiver issue, processor error). Informational and debug log lines are not classified.

To increase coverage:

- Increase `tail_lines` to capture more log history.
- Ensure the collector's log level is at least `info` (not `error`-only).

## Environment Variable Reference

| Variable | Default | Description |
|---|---|---|
| `PORT` | `8080` | HTTP server listen port |
| `LOG_LEVEL` | `info` | Log level: `debug`, `info`, `warn`, `error` |
| `CLUSTER_NAME` | `""` | Cluster identifier for multi-cluster setups |
| `OTEL_ENABLED` | `false` | Enable OpenTelemetry tracing for the MCP server itself |
| `OTEL_ENDPOINT` | `""` | OTLP endpoint for the MCP server's own traces |
| `POD_NAMESPACE` | (from Downward API) | Namespace of the MCP server pod, set automatically by the Helm chart |

## Getting Help

If you encounter an issue not covered here:

1. Check the server logs for error messages:
   ```bash
   kubectl logs -n observability -l app.kubernetes.io/name=otel-collector-mcp --tail=100
   ```

2. Enable debug logging for more detail:
   ```bash
   helm upgrade otel-collector-mcp deploy/helm/otel-collector-mcp \
     --namespace observability \
     --set config.logLevel=debug
   ```

3. Open an issue at [github.com/hrexed/otel-collector-mcp](https://github.com/hrexed/otel-collector-mcp/issues) with:
   - Your Kubernetes version
   - Your Helm values (redact any secrets)
   - Relevant log output
   - Steps to reproduce the issue
