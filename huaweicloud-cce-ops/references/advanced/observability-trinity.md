# Observability Trinity — Huawei Cloud CCE

> **Purpose**: Metrics → Logs → Traces linkage rules for CCE (Cloud Container Engine).
> **Version**: 1.0.0
> **Last Updated**: 2026-07-18

## 1. Observability Trinity Overview

| Component | Data Source | Purpose |
|-----------|-------------|---------|
| Metrics | CES (SYS.CCE, AGT.CCE) | CPU%, memory%, disk%, pod count, GPU usage |
| Logs | LTS (workload logs, container logs, system events) | Pod events, OOM, CrashLoopBackOff, Evicted |
| Traces | APM (Application Performance Management) | Request flow across microservices in cluster |

## 2. Linkage Rules

### 2.1 Metric → Log Linkage

| When CES metric alerts | Check LTS logs |
|------------------------|----------------|
| `cpu_usage` > 90% | Pod logs, container engine logs (`kubelet`, `docker`) |
| `mem_usedPercent` > 90% | OOM killer logs, container memory usage logs |
| `diskUsage_percent` > 90% | Persistent volume (PVC) logs, log file growth |
| `pod_num` sudden drop | Kubernetes events, pod termination logs |
| `container_memory_working_set_bytes` spike | Application heap/profile, container restart logs |
| `gpu_usage` > 95% | GPU job logs, device plugin logs |
| Pod restart count spike | Container restart logs, `liveness`/`readiness` probe failures |

### 2.2 Log → Metric Linkage

| When LTS log pattern detected | Check CES metrics |
|------------------------------|-------------------|
| OOM errors (`killed`, `oom_reaper`) | `mem_usedPercent`, `container_memory_working_set_bytes` |
| CrashLoopBackOff | Pod restart count, `cpu_usage`, `mem_usedPercent` |
| Evicted | `diskUsage_percent`, pod scheduling failure metrics |
| ImagePullBackOff | Network metrics, VPC `eni_health` |
| OOMKilled (exit code 137) | `mem_usedPercent`, memory limit configuration |
| Pod unschedulable | `pod_num`, node resource metrics |
| Node NotReady | `cpu_usage`, `mem_usedPercent` of affected node |

### 2.3 Trace → Metric/Log Linkage

| When APM trace shows | Check metrics + logs |
|---------------------|---------------------|
| Span duration > 500ms | Pod CPU/memory metrics, node resource metrics |
| Error in trace | Application error logs, container logs |
| Timeout in trace | Downstream service health, pod restart count |
| Database call latency | RDS metrics (if StatefulSet hosts DB) |
| Service mesh latency | Sidecar proxy logs, `istio-proxy` metrics |

## 3. Data Source Mapping

| Observable | CES Namespace | LTS Log Group | APM Trace |
|-----------|--------------|---------------|-----------|
| CCE Cluster (SYS) | SYS.CCE | `{{user.cluster_name}}-cluster-event` | Yes |
| CCE Node (AGT) | AGT.CCE | `{{user.cluster_name}}-node-log` | No |
| CCE Workload | AGT.CCE | `{{user.cluster_name}}-workload-log` | Yes |
| CCE Pod stdout | AGT.CCE | `{{user.cluster_name}}-container-log` | Via workload |
| CCE Storage (PVC) | SYS.CCE | `{{user.cluster_name}}-pvc-log` | No |

## 4. Correlation Query Examples

### 4.1 Metric Alert → Find Related Logs

```bash
# CPU spike on CCE workload
CLUSTER_NAME="{{user.cluster_name}}"
REGION="{{env.HW_REGION_ID}}"
LOG_GROUP="{{user.cce_log_group_id}}"
NAMESPACE="{{user.namespace}}"
WORKLOAD_NAME="{{user.workload_name}}"

# 1. Query CPU metric to confirm alert
hcloud ces query-metric-data \
  --namespace "SYS.CCE" \
  --metric-name "cpu_usage" \
  --dimension "cluster_id=${CLUSTER_NAME}" \
  --period 60 \
  --from "$(( $(date +%s) - 300 ))" \
  --to "$(date +%s)" \
  --output json

# 2. Query LTS for pod logs around alert time
hcloud lts query-log \
  --log-group-id "$LOG_GROUP" \
  --log-stream-name "${NAMESPACE}-${WORKLOAD_NAME}" \
  --start-time "$(( $(date +%s) * 1000 - 30 * 60 * 1000 ))" \
  --end-time "$(date +%s)" \
  --keywords "cpu|process|throttle" \
  --output json
```

### 4.2 Log Pattern → Find Related Metrics

```bash
# OOM detected in container logs
CLUSTER_NAME="{{user.cluster_name}}"
REGION="{{env.HW_REGION_ID}}"

# Query memory metrics for the cluster
hcloud ces query-metric-data \
  --namespace "SYS.CCE" \
  --metric-name "mem_usedPercent" \
  --dimension "cluster_id=${CLUSTER_NAME}" \
  --period 60 \
  --from "$(( $(date +%s) - 600 ))" \
  --to "$(date +%s)" \
  --output json

# Query container-level memory working set
hcloud ces query-metric-data \
  --namespace "AGT.CCE" \
  --metric-name "container_memory_working_set_bytes" \
  --dimension "cluster_id=${CLUSTER_NAME},namespace={{user.namespace}}" \
  --period 60 \
  --from "$(( $(date +%s) - 600 ))" \
  --to "$(date +%s)" \
  --output json
```

### 4.3 Trace → Metric Correlation

```bash
# High latency trace → check pod metrics
TRACE_ID="{{user.trace_id}}"
CLUSTER_NAME="{{user.cluster_name}}"

# Get span details from APM
hcloud apm query-trace \
  --trace-id "$TRACE_ID" \
  --output json

# Query CCE pod metrics for that workload
hcloud ces query-metric-data \
  --namespace "AGT.CCE" \
  --metric-name "cpu_usage" \
  --dimension "cluster_id=${CLUSTER_NAME},namespace={{user.namespace}},workload_name={{user.workload_name}}" \
  --period 60 \
  --from "$(( $(date +%s) - 300 ))" \
  --to "$(date +%s)" \
  --output json
```

## 5. Trinity-Driven Diagnosis Workflow

```
[CES Metric Alert: cpu_usage > 90% on CCE workload]
    │
    ├── 1. Query LTS: pod/container logs during alert window
    │   └── hcloud lts query-log --keywords "cpu|throttle|process"
    │
    ├── 2. Query APM: traces with high span duration on this workload
    │   └── hcloud apm query-trace --cluster-id "$CLUSTER_NAME" --namespace "$NAMESPACE"
    │
    └── 3. Correlate:
        ├── If application bug → heap/profile analysis + code fix
        ├── If resource limit too low → adjust container spec limits
        └── If OOM → memory metric confirm + increase memory limit / scale replicas
```

```
[CES Metric Alert: mem_usedPercent > 90% on CCE node]
    │
    ├── 1. Query LTS: kubelet and container engine logs
    │   └── hcloud lts query-log --log-stream-name "node-log" --keywords "oom|kill|memory"
    │
    ├── 2. Query LTS: pod logs for OOMKilled events
    │   └── hcloud lts query-log --keywords "OOMKilled|exit_code_137"
    │
    └── 3. Correlate:
        ├── If node memory pressure → scale up node or move workloads
        └── If pod-level OOM → adjust memory limits or optimize application
```

## 6. Cross-Service Linkage

| CCE symptom | Downstream check |
|-------------|------------------|
| CCE pod CPU spike | RDS: `rds001_cpu_util` (if DB StatefulSet) |
| CCE pod memory leak | DCS: `redis_memory_usage` (if cache StatefulSet) |
| CCE disk pressure | OBS: bucket usage (if OBS mount) |
| CCE network latency | ELB: `l7e_listener_qps`, backend health |
| CCE pod eviction | VPC: `eni_health`, node resource metrics |
| CCE GPU exhaustion | ModelServe: inference latency, GPU memory |

## 7. CCE-Specific Metric Dimensions

| Metric Name | Namespace | Dimensions | Unit |
|-------------|-----------|------------|------|
| `cpu_usage` | SYS.CCE | cluster_id | % |
| `mem_usedPercent` | SYS.CCE | cluster_id | % |
| `diskUsage_percent` | SYS.CCE | cluster_id | % |
| `gpu_usage` | SYS.CCE | cluster_id | % |
| `pod_num` | SYS.CCE | cluster_id, namespace | count |
| `container_memory_working_set_bytes` | AGT.CCE | cluster_id, namespace, pod_name | bytes |
| `cpu_usage` | AGT.CCE | cluster_id, namespace, pod_name | % |
| `gpu_memory_used_bytes` | AGT.CCE | cluster_id | bytes |

## 8. Compliance Checklist

- [x] Metrics → Logs linkage defined (SYS.CCE + AGT.CCE)
- [x] Logs → Metrics linkage defined (OOM, CrashLoopBackOff, Evicted)
- [x] Trace → Metric/Log linkage defined (span duration → pod metrics)
- [x] Data source mapping documented (CES namespace → LTS group → APM)
- [x] Correlation query examples provided (3 CLI examples)
- [x] Cross-service linkage defined (CCE → RDS/OBS/ELB/DCS)
- [x] CCE-specific metric dimensions documented
