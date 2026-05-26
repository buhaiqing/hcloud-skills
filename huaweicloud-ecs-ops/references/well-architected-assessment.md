# Well-Architected + Three-Pillar Assessment — Huawei Cloud ECS

This file contains ECS-specific well-architessed assessment patterns. The generator-level specification is in `../huaweicloud-skill-generator/references/well-architected-assessment.md`.

## 1. Security (安全) — ECS

### IAM Minimum Permissions

| Operation | IAM Action | Resource Scope |
|-----------|-----------|---------------|
| ListServers | `ecs:*List*`, `ecs:*Describe*` | `*` |
| CreateServer | `ecs:*Create*`, `evs:*Create*`, `vpc:*CreateSecurityGroupRule` | `*` |
| DeleteServer | `ecs:*Delete*` | `ecs:server:*` |
| ResizeServer | `ecs:*Resize*` | `ecs:server:*` |
| CloudCell Execute | `ecs:*CloudCell*` | `ecs:server:*` |

### Credential Management
- Use IAM agency (委托) for ECS→EVS/CS cross-service access
- Do NOT embed AK/SK in cloud-init scripts or user_data
- Rotate AK/SK every 90 days

### Network Security
- Security group: deny all inbound by default, allow only required ports
- Cloud-Cell Agent: outbound HTTPS (443) + ECS service domain whitelist
- EIP: only bind when public access required, use WAF for HTTPS traffic

## 2. Stability (稳定) — ECS

### Backup & Recovery

| Operation | API | Description |
|-----------|-----|-------------|
| Create Snapshot | `CreateServerGroupSnapshot` / EVS API | Snapshot system + data disks |
| Create Backup (CSBS) | CBS backup API | Consistent backup via agent |
| Cross-region copy | `CopySnapshot` with `dest_region_id` | DR snapshot replication |
| Restore from image | `CreateServer` with backup image ID | Full restore |

**RTO target:** < 30 minutes for ECS instance restore
**RPO target:** < 24 hours for daily backup, < 1 hour for CBS continuous backup

### DR Runbook

**Phase 1: Backup Verification**
1. Confirm snapshot exists and status = `available`
2. Verify snapshot covers all required volumes
3. Confirm snapshot age within RPO window

**Phase 2: Recovery Execution**
1. Create instance from snapshot image
2. Verify instance reaches `ACTIVE` state
3. Update DNS/ELB backend to new instance

**Phase 3: Post-Recovery Validation**
1. SSH/CloudCell connectivity test
2. Application health check (HTTP 200, DB connection)
3. Verify data integrity (file checksums, DB row counts)

### Multi-AZ
- Deploy production across ≥ 2 AZs via AS or multiple instances
- Single AZ instance = SPOF (risk score: HIGH)
- ELB + multi-AZ ECS = recommended pattern

## 3. Cost (成本) — ECS (FinOps)

### Billing Model Comparison

| Billing | Best For | Savings | Risk |
|---------|----------|---------|------|
| 按需 (Pay-per-use) | Dev/test, short-term | N/A | Highest ongoing cost |
| 包年包月 (Subscription) | Production 24/7 | Up to 83% vs 按需 | No flexibility |
| 竞价 (Spot) | Batch, stateless, AS | Up to 90% vs 按需 | May be reclaimed |

### Idle Detection

| Resource Type | Idle Condition | Detection Method | Action |
|--------------|---------------|-----------------|--------|
| ECS instance | `cpu_util` < 10% for 7+ days | CES DescribeMetricData | Stop or delete |
| Stopped instance | `status: SHUTOFF` > 7 days | ListServersDetail | Delete (keeps billing EVS) |
| Unattached EVS | No `server_id` association | EVS ListVolumes | Delete or snapshot+delete |
| Zombie EIP | Not bound to any resource | VPC ListPublicIps | Release |
| Orphaned snapshot | No associated image/volume | EVS ListSnapshots | Delete if > retention |

### Idle Instance Financial Impact

**Critical**: Stopping an ECS instance does NOT stop EVS billing!

| Billing Model | ECS Stop Impact | EVS Billing | Recommended Action |
|---------------|-----------------|-------------|-------------------|
| Pay-per-use | CPU cost = 0 | EVS continues | Delete if > 30 days idle |
| Subscription | No savings until term ends | Included | Resize immediately |
| Spot | Terminated, no billing | Released | Already reclaimed |

**Cost Calculation**:
- Monthly savings from stop: `flavor_hourly_rate × 24 × 30`
- EVS monthly cost continues: `disk_size_gb × gb_monthly_rate`
- Net savings = ECS savings - EVS cost (may be negative!)

**Recommendation**:
- Pay-per-use idle (< 30 days): Stop (temporary savings)
- Pay-per-use idle (> 30 days): Delete (full savings, snapshot first)
- Subscription idle: Resize immediately for term savings

### Right-Sizing Matrix

| CPU avg(7d) | MEM avg(7d) | Action | Savings |
|-------------|------------|--------|---------|
| < 20% | < 30% | Downgrade to smaller flavor | 30-60% |
| < 20% | > 80% | Switch to memory-optimized (m-series) | 10-20% |
| > 80% | < 50% | Switch to compute-optimized (c-series) | — |
| > 80% | > 80% | Upgrade flavor or scale out | — |
| Spiky (max > 3× avg) | — | Use AS with按需 + Spot | 20-50% |

### Cost Tagging Strategy
- Tag new instances with: `cost_center`, `project`, `environment`, `owner`, `ttl`
- Auto-decommission instances tagged with `ttl=30d` on day 30
- Monthly cost report grouped by tag

## 4. Security Operations (SecOps) — ECS

### HSS (Host Security Service) Integration

| HSS Trigger | Action | Delegation |
|-------------|--------|-----------|
| Vulnerability detected | Scan → Prioritize → Patch via CloudShell | Auto-create patch task |
| Intrusion detected | Isolate instance (security group block all but admin) | Notify security team |
| Baseline violation | Generate compliance report | Schedule remediation |
| Brute-force SSH | Block source IP via security group rule | Alert + auto-block |

### Encryption Requirements

| Data State | Method | Service |
|-----------|--------|---------|
| EVS system disk | KMS key encryption | CreateServer with `volumes[].metadata.__system__cmkid` |
| EVS data disk | KMS key encryption | AttachVolume with encryption flag |
| Cloud init user_data | Avoid secrets; use vault references | IAM Vault service |
| Backup/snapshot | Inherits source disk encryption | Automatic |

### Compliance Alignment
- 等保2.0 Level 3: ECS deployment requires ≥ 2 AZs, HSS enabled, daily backup
- Audit trail: Enable CTS (Cloud Trace Service) for all ECS operations

## 5. Performance (性能) — ECS

### Scaling Triggers

| Metric | Scale Up | Scale Down | Window |
|--------|----------|-----------|--------|
| `cpu_util` | > 80% for 5min | < 30% for 15min | 300s |
| `mem_usedPercent` | > 85% for 5min | < 50% for 15min | 300s |
| `load1` | > 0.8 × vCPU count | < 0.3 × vCPU count | 300s |

### Performance Baseline Establishment

1. Deploy test instance with target flavor
2. Run benchmark: `sysbench cpu --cpu-max-prime=20000 run`
3. Record baseline: CPU ops/sec, disk IOPS, network throughput
4. Set alert threshold at 2σ deviation from baseline

## 6. AIOps — ECS

### Multi-Metric Inspection

Anomaly patterns supported:
1. `cpu_mem_dual_high` — CPU>80% AND memory>85% → Resource pressure
2. `disk_io_bottleneck` — IOPS peak + diskUtil>90% → Storage saturation
3. `mem_leak_trend` — Memory slope > 0.5%/min → Application memory leak
4. `sudden_cpu_spike` — CPU delta > 50% in 5min → Process anomaly
5. `network_storm` — Network pps > 10× baseline → DDoS or scan
6. `disk_fill_acceleration` — Disk fill rate increasing → Imminent full disk

### Cross-Skill Diagnosis Decision Tree

```
ECS Alarm → Verify metric is real → Check instance state
    │
    ├── CPU High? → CloudShell: top → If Java → AOM trace
    │                                  → If unknown → HSS scan
    ├── Memory High? → CloudShell: free -m → If trend ↑ → Knowledge-base: Memory leak
    ├── Disk Full? → CloudShell: du -sh /var/log/* → Clean → LTS log config
    └── Unreachable? → Check security group → Check VPC route → Check CloudCell agent
```
