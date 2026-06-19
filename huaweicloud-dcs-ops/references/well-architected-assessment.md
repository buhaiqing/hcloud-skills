# Well-Architected Assessment — Huawei Cloud DCS

> This document maps DCS operations to Huawei Cloud's Well-Architected Framework (卓越架构) five pillars plus FinOps, SecOps, and AIOps.

## 1. Framework Overview

| Pillar | DCS Focus | Status |
|--------|----------|--------|
| 安全 (Security) | IAM, Redis AUTH, TLS, VPC isolation, whitelist | Required |
| 稳定 (Stability) | Backup/recovery, multi-AZ, auto-failover, DR runbook | Required |
| 成本 (Cost) | Billing models, idle detection, right-sizing | Required |
| 效率 (Efficiency) | Batch operations, CLI automation, CI/CD | Recommended |
| 性能 (Performance) | Scaling, hot/big key detection, latency thresholds | Required |

## 2. Five Pillar Skill Integration

### 2.1 安全 (Security)

#### IAM Minimum Permissions

| API Operation | IAM Action | Resource Scope |
|--------------|-----------|---------------|
| ListInstances | dcs:*List* | acs:dcs:*:*:*/* |
| ShowInstance | dcs:*Get* | acs:dcs:*:*:instance/* |
| CreateInstance | dcs:*Create* | acs:dcs:*:*:instance/* |
| UpdateInstance | dcs:*Update* | acs:dcs:*:*:instance/* |
| DeleteInstance | dcs:*Delete* | acs:dcs:*:*:instance/* |
| ResetPassword | dcs:*Update* | acs:dcs:*:*:instance/* |
| CreateBackup | dcs:*Create* | acs:dcs:*:*:instance/* |

```json
{
  "Version": "1.1",
  "Statement": [
    {
      "Effect": "Allow",
      "Action": ["dcs:*List*", "dcs:*Get*", "dcs:*Create*", "dcs:*Update*", "dcs:*Delete*"],
      "Resource": ["*"]
    },
    {
      "Effect": "Allow",
      "Action": ["vpc:*Get*", "ces:*List*", "ces:*Create*"],
      "Resource": ["*"]
    }
  ]
}
```

#### Redis Authentication & TLS
- **AUTH password**: mandatory for production, 8-64 chars with letters+digits+symbols
- **TLS**: available for Redis 6.0, recommended for sensitive data
- **IP whitelist**: additional access control layer beyond security groups
- **Credential management**: AK/SK rotation every 90 days, use IAM agency for cross-account

### 2.2 稳定 (Stability)

#### Backup & Recovery

| Operation | RTO | RPO | Notes |
|----------|-----|-----|-------|
| Manual backup + restore (new instance) | 5-15 min | Backup age | Non-destructive |
| Automatic failover (HA mode) | <30s | <1s (AOF) | Transparent to app |
| Resize with data migration | 2-10 min | <1s | Brief connectivity dip |
| Restore to existing instance | 5-15 min | Backup age | Overwrites current data |

#### DR Runbook

**Phase 1: Backup Verification**
1. Confirm latest backup exists and `status = SUCCESS`
2. Verify backup size > 0
3. Check backup age within RPO window

**Phase 2: Recovery Execution**
1. If instance is running, create pre-recovery backup
2. Execute restore via API or CLI
3. Poll status until `RUNNING`

**Phase 3: Post-Recovery Verification**
1. PING test: `redis-cli -h {ip} -p {port} -a {pwd} PING`
2. Data integrity: `INFO keyspace` to verify key count
3. Application smoke test: run representative queries
4. Document recovery duration vs RTO target

#### Multi-AZ Deployment
- HA mode: master in AZ-a, standby in AZ-b (automatic failover)
- Cluster mode: shards distributed across multiple AZs
- Single-node: no cross-AZ, NOT for production

### 2.3 成本 (Cost)

#### Billing Model Comparison

| 计费类型 | 最佳场景 | 节省幅度 | RTO |
|---------|---------|---------|-----|
| 按需计费 (Pay-per-use) | Dev/test, short-term | N/A | — |
| 包年包月 (Subscription) | Production stable load | up to 85% vs 按需 | — |

#### Idle Resource Detection
- CPU < 10% for 7 consecutive days → over-provisioned
- Memory < 20% for 7 consecutive days → right-size candidate
- Zero commands for 3+ days → decommission/reuse candidate

### 2.4 效率 (Efficiency)

- **Batch operations**: ListInstances with pagination, batch reset/start/stop
- **CLI JSON output**: compatible with jq for pipeline parsing
- **CI/CD integration**: Terraform/OpenTofu for DCS provisioning, JSON output for health checks
- **Automation patterns**: script-based health check, backup verification, capacity planning

### 2.5 性能 (Performance)

| Metric | Scale-up Threshold | Scale-down Threshold | Window |
|--------|-------------------|---------------------|--------|
| CPU usage | > 80% sustained 5 min | < 30% sustained 15 min | 300s |
| Memory usage | > 85% sustained 5 min | < 50% sustained 15 min | 300s |
| Connected clients | > 70% of max | < 40% of max | 300s |
| Latency (P95) | > 50 ms sustained | < 5 ms sustained | 300s |

- **Hot key detection**: via `INFO COMMANDSTATS` or Redis 6.0 CLIENT TRACKING
- **Big key detection**: `redis-cli --bigkeys`, keys > 10MB
- **Performance baseline**: capture first 24h metrics, alert on >2σ deviation

## 3. FinOps (财务运营)

### 3.1 成本可见性

| Tool | Use Case | Integration |
|------|----------|-------------|
| 费用中心 BSS | Bill query, cost trends | `hcloud bss query-bill` |
| Cost Center (CCS) | Multi-dimension analysis | Tag-based filtering |
| Budget (BUD) | Budget alerts | Alert at 80%/90%/100% |

**Cost Tagging Strategy**:
- `cost_center`: team/department owning the Redis instance
- `environment`: prod/staging/dev
- `application`: app name using Redis

### 3.2 成本优化

| CPU Utilization | Memory Usage | Recommendation | Expected Savings |
|----------------|-------------|-----------------|-----------------|
| < 20% | < 30% | Downgrade capacity | 30-60% |
| < 20% | > 80% | Change to memory-optimized spec | 10-20% |
| > 80% | < 50% | Change to compute-optimized spec | — |
| > 80% | > 80% | Upgrade capacity | — |
| High variance (peak > 3x avg) | — | Pay-per-use + auto-scale | 20-50% |

**Lifecycle Cost Management**:
- Dev/test → auto-scale down during off-hours
- Production stable → subscription (包年包月)
- Temporary events → pay-per-use, release after

### 3.3 成本问责

| Budget Level | Threshold | Action | Notify |
|-------------|-----------|--------|--------|
| Warning | 80% | Notify cost owner | Email + SMS |
| Approval required | 90% | Approve new resources | Approval flow |
| Freeze | 100% | Block non-critical resource creation | Auto-control |

## 4. SecOps (安全运营)

### 4.1 身份安全

- **No root AK/SK** for daily operations — create dedicated IAM user per project
- **MFA mandatory** for console access to DCS instances
- **AK/SK rotation**: 90-day cycle
- **IAM agency** (委托) for cross-account DCS access management

### 4.2 网络安全

- **VPC isolation**: DCS instances only accessible within VPC (no public endpoint)
- **Security groups**: inbound port 6379 from app servers only, explicit deny default
- **IP whitelist**: additional layer — whitelist app server IPs
- **VPC Endpoint**: for API calls from within VPC (avoid public API endpoint)

### 4.3 数据安全

| Data State | Encryption Method | Configuration |
|-----------|-------------------|--------------|
| In transit | TLS 1.2+ (Redis 6.0) | Enable SSL at creation |
| At rest (backup) | KMS (SSE-KMS) | Automatic for backups stored in OBS |
| AUTH password | Strong password policy | 8-64 chars, letters+digits+symbols |

- **Backup data**: encrypted at rest via OBS SSE-KMS
- **Log sanitization**: NEVER log Redis AUTH passwords or key values containing PII
- **Audit trail**: CTS (Cloud Trace Service) records all DCS API calls

### 4.4 威胁检测

| Trigger | Detection | Action |
|---------|-----------|--------|
| Unauthorized access attempt | CES failed auth count spike | Block IP via whitelist, alert security team |
| Key deletion via CONFIG | Audit log shows CONFIG command | Alert, review access policy |
| Mass key deletion | DBSIZE drops suddenly | Investigate, restore from backup if needed |
| Connection from non-whitelisted IP | Audit log | Block, review security group + whitelist |

## 5. AIOps Integration

### 5.1 Multi-Metric Correlation (≥ 4 patterns)

| Pattern | Metrics Correlated | Detection Logic | Severity |
|---------|-------------------|----------------|----------|
| Memory exhaustion | memory_usage > 95% AND evicted_keys > 0 | Immediate alert | Critical |
| Connection storm | connected_clients > 80% AND latency > 2x baseline | Immediate alert | High |
| Cache miss avalanche | hit_rate < 50% AND expired_keys > 3x normal AND commands surge | Detect within 5 min | Critical |
| Latency degradation | latency P95 > 50ms AND cpu_usage > 80% | Detect within 5 min warning, 15 min critical | High |

### 5.2 Cross-Skill Delegation Matrix

| Alarm | Primary Diagnostic Skill | Secondary Skill |
|-------|-------------------------|----------------|
| Memory/ CPU spike | DCS skill (this skill) | CES for detailed metrics |
| Connection timeout | DCS + VPC skill | CES for network metrics |
| High cost | DCS cost section | Billing skill |
| Security alert | DCS + IAM skill | HSS for ECS host security |

### 5.3 Knowledge Base

Refer to `references/knowledge-base.md` for:
- 5 product-specific fault patterns
- 2 cross-product cascade failure scenarios
- Historical diagnosis notes with resolution times

### 5.4 Alarm Storm Handling

When CPU + memory + connected_clients all spike simultaneously:
1. **Aggregate**: treat as single incident, not separate alarms
2. **Root resource**: DCS instance is the root cause
3. **Action**: resize immediately, then investigate root cause
4. **Suppress**: secondary alarms (individual metric breaches) during resize

### 5.5 Proactive Inspection Workflow

```
Schedule: Daily automated check
1. Discover all DCS instances in account
2. Collect: status, CPU, memory, connections, hit_rate, latency
3. Detect anomalies per pattern definitions above
4. Generate report: instance health score per instance
5. Alert: only on instances with score < 70
6. Archive: daily reports for 30 days, trend analysis
```
---

## Worker Output Contract (Read-Only Assessment Mode)

> Invoked when Well-Architected review sets `{{user.mode}}=well-architected-readonly`.
> Return **`{{output.product_assessment}}`** — field names MUST match the canonical schema.

**Canonical schema:** [worker-output-schema.md](../../huaweicloud-skill-generator/references/worker-output-schema.md)

| Constant | Value |
|----------|-------|
| `skill_id` | `huaweicloud-dcs-ops` |
| `product` | `dcs` |
| Finding `id` pattern | `dcs-{rel|sec|cost|eff}-NNN` |

### Pillar → checklist map

| `pillars` key | Checklist source in this document |
|---------------|-------------------------------------|
| `reliability` | Stability / DR / backup sections |
| `security` | IAM / network / encryption sections |
| `cost` | FinOps / billing / idle detection sections |
| `efficiency` | Automation / batch / CI/CD sections |

### Example `{{output.product_assessment}}`

```json
{
  "skill_id": "huaweicloud-dcs-ops",
  "product": "dcs",
  "region": "cn-north-4",
  "scope": "account-wide",
  "assessment_date": "2026-06-19T10:00:00+08:00",
  "status": "OK",
  "partial": false,
  "resource_count": 1,
  "pillars": {
    "cost": {
      "score": 80,
      "status": "assessed",
      "findings": []
    },
    "efficiency": {
      "score": 80,
      "status": "assessed",
      "findings": []
    },
    "reliability": {
      "score": 80,
      "status": "assessed",
      "findings": []
    },
    "security": {
      "score": 80,
      "status": "assessed",
      "findings": []
    }
  },
  "recommendations": [],
  "trace": {
    "commands": [
      "hcloud dcs read-only-list --region cn-north-4 (HW_SECRET_ACCESS_KEY=<masked>)"
    ],
    "request_ids": [
      "0123456789abcdef0123456789abcdef"
    ]
  },
  "errors": []
}
```
