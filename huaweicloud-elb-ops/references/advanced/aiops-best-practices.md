# AIOps Best Practices — Huawei Cloud ELB

> Intelligent operations integration patterns for ELB load balancers:
> multi-metric correlation, anomaly detection, and self-healing for
> listener / backend / health-check failures.
> **Version:** 1.0.0

## AIOps Goals for ELB

ELB is the front door of nearly every production workload. AIOps workflows should:

- Correlate ELB metrics (5xx, latency, active connections) with backend
  health-check failures and CES metrics from backend ECS instances
- Detect gradual degradation (e.g., rising p99 latency, growing 5xx ratio)
  before user-visible outages
- Auto-remediate common patterns (drain unhealthy backend, scale up
  backend group, fail over between AZs)
- Feed audit events to CTS for cross-skill correlation

## Recommended AIOps Patterns

### 1. Backend Health Degradation Correlation

| Pattern | Metrics Correlated | Detection Logic | Remediation |
|---------|-------------------|-----------------|-------------|
| `single_backend_down` | `healthy_members` (per backend group) | count < expected AND unchanged > 2 min | Mark backend unhealthy, redistribute traffic |
| `az_imbalance` | `active_connections` per AZ | stddev / mean > 0.5 | Re-weight listener rule |
| `gradual_5xx_rise` | `5xx_ratio` (15-min window) | slope > 0.1%/min | Alert + page on-call |
| `tls_handshake_failures` | `tls_handshake_errors` | rate > 5/min | Check cert expiry, SNI config |
| `listener_unresponsive` | `active_connections` + `new_connections_rate` | new_rate = 0 for 5 min | Restart listener (CLI / SDK) |

### 2. Latency Anomaly Detection

```bash
# Pseudo — fetch p99 latency from CES for each listener
hcloud ces list-metrics \
  --namespace "SYS.ELB" \
  --metric-name "l7_p99_latency" \
  --dimension "loadbalancer_id=$LB_ID,listener_id=$LISTENER_ID"
```

When `l7_p99_latency` exceeds 2× the 7-day baseline for ≥ 5 min,
trigger:

1. Pull backend ECS `cpu_util`, `mem_usedPercent` via CES
2. Pull `unhealthy_members` from ELB show-health
3. Cross-reference with CTS `UpdateListener` / `UpdateBackend` events
4. Either auto-scale backend group (CES alarm → AS action) or page
   on-call if root cause is unknown

### 3. Anomaly Storm Handling

When ≥ 3 ELB listeners trigger Critical alarms within 5 min:

1. Pause non-essential remediation (avoid cascade)
2. Snapshot current `show-health` and `show-loadbalancer` outputs
3. Emit a single consolidated page with the 3 affected LBs
4. Auto-create a CES event for the cluster, tagged `aiops-cluster:elb`

## ML Integration Hooks

ELB AIOps can leverage the following CES metric streams:

| Metric | Aggregation | Use Case |
|--------|-------------|----------|
| `m7_in_Bps`, `m7_out_Bps` | 1-min, 5-min | Traffic surge detection |
| `active_connections` | 1-min | Capacity planning |
| `new_connections` | 1-min | DDoS detection (rate-of-change) |
| `5xx_ratio` (custom) | 1-min | Backend health correlation |
| `unhealthy_hosts` | 1-min | Quorum safety |

## Cross-Skill Delegation Matrix

| Symptom | Delegate To |
|---------|-------------|
| Backend ECS CPU saturated | `huaweicloud-ecs-ops` (right-size / restart) |
| Backend ECS unreachable | `huaweicloud-ecs-ops` (diagnosis) + `huaweicloud-vpc-ops` (SG / subnet) |
| Certificate expired | This skill (rotate cert) + `huaweicloud-iam-ops` (KMS decrypt) |
| ELB quota exceeded | `huaweicloud-billing-ops` (quota raise) + `huaweicloud-iam-ops` (permission) |
| 5xx during RDS failover | `huaweicloud-rds-ops` (failover timing) + `huaweicloud-ces-ops` (metric correlation) |
| DDoS pattern detected | `huaweicloud-waf-ops` (CC rules) + AAD console |

## Self-Healing Playbook

| Trigger | Auto Action | Manual Step |
|---------|------------|-------------|
| Single backend unhealthy > 2 min | Drain from backend group | Investigate ECS after drain |
| 50% backends unhealthy | Open Jira incident (Sev2) | On-call decides whether to fail over AZ |
| Listener cert expires ≤ 7 days | Emit warning ticket | Rotate cert manually |
| LB enters `ERROR` state | Snapshot config + page | Manual restore from snapshot |

## Reference: jq paths for ELB AIOps

```bash
# Backend health map: {ip: {server_id, health, reason}}
hcloud elb show-member-health --member-ids "$IDS" -o json \
  | jq '.members[] | {ip: .address, health: .operating_status, reason: .health_reason}'

# Listener cert expiry (custom — derive from list-certificates)
hcloud elb list-certificates -o json | jq '.certificates[] | {id, not_after}'
```

## Knowledge Base Anchors

- ELB → backend ECS correlation: `references/integration.md` §3
- Listener / cert failure patterns: `references/troubleshooting.md`
- Cost anomaly: `references/well-architected-assessment.md` §3 (FinOps)
