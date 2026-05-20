# CCE Observability — Huawei Cloud Cloud Container Engine

## Metrics → Logs → Traces Linkage

### Linkage Architecture

```
Metrics (CES) → Detect anomaly
    ↓
Logs (LTS) → Diagnose root cause
    ↓
Traces (APM) → Pinpoint code-level issue
```

| Layer | Service | CCE Integration | Data Flow |
|-------|---------|----------------|-----------|
| Metrics | CES | SYS.CCE + SYS.ECS + AGT.* namespaces | kubelet → CES Agent → CES API → Dashboards |
| Logs | LTS | Cluster logs, node logs, pod logs | CCE log collection → LTS log groups → Search/analysis |
| Traces | APM | Application-level distributed tracing | SDK instrumentation → APM Collector → Traces view |

## LTS (Log Tank Service) Integration

### Log Collection Setup

| Log Type | Source | LTS Log Group | Collection Method |
|----------|--------|---------------|-------------------|
| Kubelet log | Node: /var/log/messages | cce-cluster-logs | CCE log addon (ICAgent) |
| Kube-apiserver audit log | Control plane | cce-cluster-logs | Enabled via cluster config |
| Node system log | Node: journalctl | cce-node-logs | CCE log addon (ICAgent) |
| Pod stdout/stderr | Container | cce-pod-logs | Container stdout/stderr collection |

### Log Collection via CCE Addon

```bash
# Install log-dis addon for LTS integration
hcloud cce install-addon \
  --region "{{user.region}}" \
  --cluster-id "{{user.cluster_id}}" \
  --addon-name "cce-logs-agent" \
  --addon-version "{{user.addon_version}}"
```

### Log Query Patterns

| Query | Purpose | LTS LogQL Pattern |
|-------|---------|-------------------|
| All error logs from a namespace | Error tracking | `namespace=prod` AND `level=ERROR` |
| OOM kill events | Out-of-memory detection | `oom_killed` OR `Out of memory` |
| kubelet restart events | Node instability | `systemctl.*kubelet` |
| Pod crash-loop | App instability | `Back-off restarting failed container` |
| Network policy drops | Network debugging | `policy drop` |

## AOM (Application Operations Management) Integration

| Feature | Purpose | CCE Integration |
|---------|---------|----------------|
| Application topology | Visualize service dependencies | Auto-discovered via service mesh |
| Performance monitoring | Response time, throughput | SDK-instrumented apps |
| Error tracking | Exception rates, slow endpoints | SDK-instrumented apps |
| Infrastructure monitoring | Node metrics, cluster health | AOM agent on CCE nodes |

### AOM Agent Installation

```bash
hcloud cce install-addon \
  --region "{{user.region}}" \
  --cluster-id "{{user.cluster_id}}" \
  --addon-name "aom-agent" \
  --addon-version "{{user.addon_version}}"
```

## Cross-Service Observability Chain

### CCE → CES → LTS → SMN Chain

```
1. CES detects anomaly (e.g., node CPU > 90%)
    ↓
2. CES triggers alarm rule
    ↓
3. SMN notification sent (email/SMS/webhook)
    ↓
4. Agent receives alert → runs this skill
    ↓
5. Skill queries LTS for correlated logs
    ↓
6. Skill queries CES for additional metrics
    ↓
7. Skill diagnoses root cause
    ↓
8. Skill executes remediation or recommends next steps
```

### Alarm-to-Log Correlation

| CES Alarm | Relevant LTS Log Source | Correlation Key |
|-----------|------------------------|-----------------|
| Node CPU > 90% | kubelet + pod logs | node_id, timestamp |
| Node memory > 95% | OOM kill events | namespace, pod_name |
| API server error rate > 1% | api-server audit logs | cluster_id, API request path |
| Pod crash-loop | Pod stdout/stderr | namespace, pod_name |
| Disk > 90% | Node system logs | node_id, disk_path |

## Observability Cost Considerations

| Service | Billing Factor | Cost Optimization |
|---------|---------------|-------------------|
| CES metrics | Number of metrics × retention days | Use basic monitoring (5-min) unless high-res needed |
| LTS storage | Log volume (GB) × retention days | Set retention: 7 days for debug logs, 30 days for audit |
| APM traces | Trace volume × storage | Sample traces (10-20%) for production |
| SMN notifications | Number of notifications | Aggregate alarms to reduce notification volume |
