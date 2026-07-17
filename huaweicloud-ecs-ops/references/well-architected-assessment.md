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

### Credential & Network Security

| Rule | Detail |
|------|--------|
| IAM agency | Use 委托 for ECS→EVS/CS cross-service access |
| AK/SK | Never embed in cloud-init/user_data; rotate every 90 days |
| Security group | Deny all inbound by default; allow only required ports |
| Cloud-Cell Agent | Outbound HTTPS (443) + ECS service domain whitelist |
| EIP | Bind only when public access required; use WAF for HTTPS |

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

| Phase | Steps |
|-------|-------|
| Backup Verification | Snapshot status=`available`, covers all volumes, age within RPO |
| Recovery Execution | Create instance from snapshot → verify `ACTIVE` → update DNS/ELB backend |
| Post-Recovery | SSH/CloudCell connectivity → app health check (HTTP 200, DB) → data integrity |

### Multi-AZ
- Deploy production across ≥ 2 AZs via AS or multiple instances
- Single AZ instance = SPOF (risk score: HIGH)
- ELB + multi-AZ ECS = recommended pattern

## 3. Cost (成本) — ECS (FinOps)

### Billing & Idle Detection

| Billing | Savings | Risk | Idle Action |
|---------|---------|------|-------------|
| 按需 | N/A | Highest cost | Stop <30d; Delete >30d |
| 包年包月 | Up to 83% | No flexibility | Resize immediately |
| 竞价 | Up to 90% | May reclaim | Already reclaimed |

> **⚠️ Stopping ECS does NOT stop EVS billing.** Net savings = ECS saved − EVS continued (may be negative).

| Idle Resource | Condition | Action |
|--------------|-----------|--------|
| ECS instance | `cpu_util` <10% for 7+ days | Stop or delete |
| Stopped >7d | `SHUTOFF` >7 days | Delete (snapshot first) |
| Unattached EVS | No `server_id` | Delete or snapshot+delete |
| Zombie EIP | Not bound | Release |
| Orphaned snapshot | No image/volume | Delete if >retention |

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
| `load1` | > 0.8 × vCPU | < 0.3 × vCPU | 300s |

### Performance Baseline

Benchmark: `sysbench cpu --cpu-max-prime=20000 run`. Set alert at 2σ deviation from baseline.

## 6. AIOps — ECS

### Multi-Metric Inspection

Anomaly patterns supported:
1. `cpu_mem_dual_high` — CPU>80% AND memory>85% → Resource pressure
2. `disk_io_bottleneck` — IOPS peak + diskUtil>90% → Storage saturation
3. `mem_leak_trend` — Memory slope > 0.5%/min → Application memory leak
4. `sudden_cpu_spike` — CPU delta > 50% in 5min → Process anomaly
5. `network_storm` — Network pps > 10× baseline → DDoS or scan
6. `disk_fill_acceleration` — Disk fill rate increasing → Imminent full disk

### Cross-Skill Diagnosis

| Symptom | First Action | Delegate If |
|---------|-------------|-------------|
| CPU high | CloudShell: `top` | Java → AOM; unknown → HSS scan |
| Memory high | CloudShell: `free -m` | Trend ↑ → knowledge-base ECS-003 |
| Disk full | CloudShell: `du -sh /var/log/*` | → LTS log config |
| Unreachable | Check SG → VPC route → CloudCell agent | — |
---

## Worker Output Contract

> Read-only assessment mode: `{{user.mode}}=well-architected-readonly` → return `{{output.product_assessment}}`.

**Schema:** [worker-output-schema.md](../../huaweicloud-skill-generator/references/worker-output-schema.md)

| Constant | Value |
|----------|-------|
| `skill_id` | `huaweicloud-ecs-ops` |
| `product` | `ecs` |
| Finding `id` | `ecs-{rel|sec|cost|eff}-NNN` |

| `pillars` key | Source sections |
|---------------|---------------|
| `reliability` | Stability / DR / backup |
| `security` | IAM / network / encryption |
| `cost` | FinOps / billing / idle |
| `efficiency` | Automation / batch / CI/CD |

### Example `{{output.product_assessment}}`

```json
{
  "skill_id": "huaweicloud-ecs-ops",
  "product": "ecs",
  "region": "cn-north-4",
  "scope": "account-wide",
  "assessment_date": "2026-06-19T10:00:00+08:00",
  "status": "PARTIAL",
  "partial": false,
  "resource_count": 42,
  "pillars": {
    "reliability": {
      "score": 65,
      "status": "assessed",
      "findings": [
        {
          "id": "ecs-rel-001",
          "severity": "High",
          "confidence": "HIGH",
          "title": "ECS instance i-0a1b2c3d has no backup or CBR policy",
          "evidence": "describe-server returned 'backup_policy: none'",
          "recommendation": "Create a CBR vault and attach a daily backup policy with 30-day retention",
          "effort": "quick"
        }
      ]
    },
    "security": {
      "score": 70,
      "status": "assessed",
      "findings": [
        {
          "id": "ecs-sec-001",
          "severity": "Medium",
          "confidence": "HIGH",
          "title": "Security group sg-web allows 0.0.0.0/0 on port 22",
          "evidence": "rule: ingress tcp/22 from 0.0.0.0/0",
          "recommendation": "Restrict SSH ingress to the bastion or office CIDR",
          "effort": "quick"
        }
      ]
    },
    "cost": {
      "score": 55,
      "status": "assessed",
      "findings": [
        {
          "id": "ecs-cost-001",
          "severity": "Medium",
          "confidence": "MEDIUM",
          "title": "3 ECS instances idle (CPU < 5% over 30 days)",
          "evidence": "ces metric cpu_util avg 3.2% last 30d",
          "recommendation": "Stop or downsize idle instances; consider scheduled shutdown",
          "effort": "medium"
        }
      ]
    },
    "efficiency": {
      "score": 80,
      "status": "assessed",
      "findings": []
    }
  },
  "recommendations": [
    {
      "pillar": "reliability",
      "text": "Attach CBR backup policy to all production ECS instances"
    },
    {
      "pillar": "security",
      "text": "Tighten SSH ingress security groups to known CIDRs"
    }
  ],
  "trace": {
    "commands": [
      "hcloud ecs read-only-list --region cn-north-4 (HW_SECRET_ACCESS_KEY=<masked>)",
      "hcloud ces show-metric --metric cpu_util --resource i-0a1b2c3d (HW_SECRET_ACCESS_KEY=<masked>)"
    ],
    "request_ids": [
      "0123456789abcdef0123456789abcdef"
    ]
  },
  "errors": []
}
```
