# Chaos Engineering — CCE

> **Purpose**: Document fault injection experiments for CCE (Cloud Container Engine) resilience verification.
> **Extends**: `huaweicloud-skill-generator/references/chaos-engineering-template.md`
> **Version**: 1.0.0
> **Last Updated**: 2026-07-18

---

## 1. Fault Injection Experiment Design

| Experiment Type | Injection Method | Observed Metrics | Expected Behavior | Termination Condition |
|----------------|-----------------|-----------------|------------------|---------------------|
| Node failure | Stop worker node VM | Pod rescheduling time, availability | Pods rescheduled ≤ 120s, zero downtime | Pod pending >5min |
| AZ failure | Cordon node in AZ, evict pods | Cross-AZ pod distribution, request success | Pods redistribute across AZs | Request success rate <90% for >3min |
| Pod failure | Kill pod via kubectl | Pod restart time, service availability | Kubelet restarts pod ≤ 60s | Service unavailable >2min |
| Network partition | Isolate node via SG rule | Pod-to-pod connectivity, service mesh | Circuit breaker triggers, degraded service | Packet loss >50% for >2min |
| Resource pressure | Stress CPU/memory on node | Pod eviction time, OOM events | Guaranteed pods survive, best-effort evicted | Node becomes unresponsive >3min |
| API Server failure | Simulate kube-apiserver unavailable | Client request success rate | In-cluster cache serves reads | Write failure >1min |

## 2. Resilience Score Model

### Scoring Dimensions (0-10 each)

| Dimension | Scoring Criteria | Weight |
|-----------|----------------|--------|
| Fault detection speed | Time from fault to alarm/tracing | 20% |
| Fault isolation ability | Pod explosion radius, namespace impact | 20% |
| Recovery automation | Auto-restart, reschedule, HPA effectiveness | 25% |
| Degradation quality | Service availability during node/pod failure | 15% |
| Data consistency | PVC data integrity after pod rescheduling | 20% |

### Resilience Grades

| Score | Grade | Recommendation |
|-------|-------|---------------|
| 8-10 | A (Excellent) | Regular chaos validation, maintain |
| 6-8 | B (Good) | Supplement missing fault scenarios |
| 4-6 | C (Fair) | Increase self-healing, improve degradation |
| 0-4 | D (Weak) | Prioritize critical resilience gaps |

## 3. Chaos Experiment Workflow

```yaml
chaos_experiment:
  name: "cce-node-failure"
  objective: "Verify pod rescheduling within 120s when node fails"

  preconditions:
    - "Workload deployed with ≥2 replicas across AZs"
    - "PodDisruptionBudget configured"
    - "CES alarm on node health"

  steps:
    - inject_fault: "Stop worker node via ECS console"
    - observe_metrics: "Monitor pod rescheduling via kubectl get events"
    - verify_behavior: "Confirm pods rescheduled ≤ 120s"
    - rollback_fault: "Restart node, uncordon after recovery"

  success_criteria:
    - "Pod rescheduled ≤ 120s"
    - "Zero downtime for clustered workloads"
    - "PVC remounted correctly on new node"

  emergency_rollback:
    - "Uncordon node immediately"
    - "Delete pods stuck in Pending if node unrecoverable"
    - "Restore PVC binding if detachment failed"
```

## 4. CCE-Specific Experiment Details

### 4.1 Node Failure (Primary Scenario)

**Objective**: Verify workload survives worker node failure.

**Injection**:
```bash
# Cordon node first to prevent new pods
kubectl cordon <node-name>
# Delete pods to trigger reschedule
kubectl delete pod <pod-name> --grace-period=30
```

**Metrics to Monitor**:
- `container_node_cpu_usage` via CES
- `container_memory_usage` per pod
- Pod lifecycle events: `kubectl get events --sort-by='.lastTimestamp'`

**Expected**: Pods rescheduled on healthy nodes, service remains available.

### 4.2 Pod Failure & Restart

**Objective**: Verify kubelet auto-restart and liveness probe effectiveness.

**Injection**:
```bash
# Kill specific container inside pod
kubectl exec <pod-name> -c <container-name> -- kill -9 1
```

**Metrics**: Pod restart count, restart time, service availability.

### 4.3 Resource Pressure & Eviction

**Objective**: Verify pod priority and QoS during resource contention.

**Injection**:
```bash
# Apply stress to worker node
kubectl label node <node-name> stress=true
# Deploy guaranteed pod and best-effort pod, stress node
```

**Metrics**: Eviction timestamp, pod priority, QoS class survival.

### 4.4 Network Partition

**Objective**: Verify service mesh circuit breaker and in-cluster resilience.

**Injection**:
```bash
# Block pod-to-pod traffic via network policy (simulate)
kubectl label pod <pod-name> chaos=isolated
# Apply NetworkPolicy to isolate namespace
```

**Metrics**: Request success rate, circuit breaker state, fallback behavior.

## 5. Emergency Rollback Procedures

| Scenario | Rollback Action |
|----------|----------------|
| Node unrecoverable | Delete node from cluster, provision replacement |
| Pod stuck in Pending | Force delete pod, check PVC binding |
| Network partition persists | Remove isolation labels, restart CNI plugin |
| PVC detachment failed | Manually detach volume, verify data integrity |
| API Server unavailable | Use in-cluster cache, wait for apiserver recovery |

## 6. Compliance Checklist

- [x] ≥5 fault injection experiments designed (6 experiments)
- [x] Resilience scoring model defined (5 dimensions, 20% weights)
- [x] Experiment workflow documented (YAML format)
- [x] Emergency rollback procedures defined (5 scenarios)
