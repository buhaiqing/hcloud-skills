# AIOps Best Practices — Huawei Cloud VPC

> Intelligent operations integration patterns for VPC networking:
> subnet saturation detection, security-group anomaly correlation, NAT
> gateway / EIP health, and cross-AZ traffic imbalance.
> **Version:** 1.0.0

## AIOps Goals for VPC

VPC is shared infrastructure. AIOps workflows should:

- Detect subnet IP exhaustion before service interruption
- Correlate security-group rule changes (from CTS) with connectivity
  loss events
- Surface NAT gateway / EIP bandwidth saturation
- Detect cross-AZ traffic imbalance for stateful workloads

## Recommended AIOps Patterns

### 1. Subnet Saturation Detection

| Pattern | Metrics Correlated | Detection Logic | Remediation |
|---------|-------------------|-----------------|-------------|
| `subnet_ip_exhaustion` | `subnet_used_ips / subnet_total_ips` | ratio > 0.9 | Open Sev2 ticket for subnet expansion |
| `sg_rule_drift` | CTS `AuthorizeSecurityGroup` / `RevokeSecurityGroup` events | delta over 24h > 20 | Snapshot SG, page security team |
| `nat_saturation` | `nat_egress_bps / nat_bandwidth` | ratio > 0.85 sustained 5 min | Scale up NAT or split EIP pool |
| `eip_unhealthy` | `eip_health_check_failed` | count > 0 for 2 min | Re-bind EIP, alert |
| `cross_az_imbalance` | `vpc_cross_az_traffic` (per AZ pair) | stddev / mean > 0.4 | Re-weight AS group, re-check SG |

### 2. Connectivity Anomaly Correlation

When a service reports `connection refused` or `connection timeout`:

1. Pull recent CTS events for the involved SG:
   ```bash
   hcloud cts list-traces --trace-type=control \
     --resource-type=SecurityGroup \
     --resource-id=$SG_ID \
     --start=$(( $(date +%s) - 3600 ))000
   ```
2. Pull CES metrics for involved subnet / NAT gateway
3. If a SG rule was revoked in the past 1h, surface it as the
   top-ranked hypothesis
4. Auto-create a temporary "diagnostic SG rule" (deny-all to drop
   nothing — manual step only) — DO NOT auto-revert

### 3. Anomaly Storm Handling

When ≥ 3 subnets trigger Critical alarms within 5 min (e.g., region-wide
scale-up event):

1. Pause non-essential remediation
2. Snapshot SG + subnet + NAT state for all 3
3. Emit a single consolidated page
4. Auto-create CES event tagged `aiops-cluster:vpc`

## ML Integration Hooks

VPC AIOps can leverage:

| Source | Aggregation | Use Case |
|--------|-------------|----------|
| `SYS.NAT` CES metrics | 1-min | NAT bandwidth / connection tracking |
| `SYS.VPC` (custom) | 1-min | Subnet utilization |
| CTS `SecurityGroup` events | event | Drift detection |
| CTS `EIP` events | event | EIP churn analysis |
| VPC flow logs (via LTS) | 5-min | Traffic matrix analysis |

## Cross-Skill Delegation Matrix

| Symptom | Delegate To |
|---------|-------------|
| ECS in subnet unreachable | `huaweicloud-ecs-ops` + this skill (SG check) |
| ELB 5xx after SG change | `huaweicloud-elb-ops` (listener) + this skill (SG) |
| RDS slow query (network) | `huaweicloud-rds-ops` (RDS) + this skill (VPC peering) |
| OBS transfer failure | `huaweicloud-obs-ops` + this skill (endpoint / NAT) |
| EIP bandwidth saturated | This skill (resize) + `huaweicloud-billing-ops` (cost) |
| Cross-VPC peering failure | This skill + `huaweicloud-iam-ops` (route permissions) |

## Self-Healing Playbook

| Trigger | Auto Action | Manual Step |
|---------|------------|-------------|
| Subnet 90% full | Open Jira ticket | Manual subnet expansion |
| EIP health-check fails 2 min | Re-bind EIP, log event | Investigate underlying NIC |
| NAT bandwidth > 85% | Open warning ticket | Scale NAT or add EIPs |
| Cross-AZ imbalance > 0.4 stddev | Log metric, do not act | Architecture review |
| SG drift > 20 rules / 24h | Snapshot SG, page security | Compliance review |

## Reference: jq paths for VPC AIOps

```bash
# Subnets above 80% utilization
hcloud vpc list-subnets -o json | jq '.subnets[] | select((.used_ips / .total_ips) > 0.8) | {id, cidr, used: .used_ips, total: .total_ips}'

# NAT gateway bandwidth
hcloud nat list-nat-gateways -o json | jq '.nat_gateways[] | {id, egress_bps, bandwidth: .bandwidth}'

# EIPs in ERROR state
hcloud vpc list-publicips -o json | jq '.publicips[] | select(.status == "ERROR") | {id, address, status}'
```

## Knowledge Base Anchors

- VPC ↔ ECS / ELB / RDS connectivity: `references/integration.md` §3
- SG troubleshooting: `references/troubleshooting.md`
- Cost anomaly: `references/well-architected-assessment.md` §3 (FinOps)
