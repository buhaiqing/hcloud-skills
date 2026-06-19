# Well-Architected Assessment — Huawei Cloud RDS

> **Purpose:** Five pillars + FinOps + SecOps + AIOps assessment for RDS operations.
> **Version:** 1.0.0
> **Last Updated:** 2026-05-20

---

## Table of Contents

1. [Framework Overview](#1-framework-overview)
2. [Five Pillar Assessment](#2-five-pillar-assessment)
3. [FinOps Integration](#3-finops-财务运营)
4. [SecOps Integration](#4-secops-安全运营)
5. [AIOps Integration](#5-aiops-integration)
6. [Compliance Checklists](#6-compliance-checklists)

---

## 1. Framework Overview

Huawei Cloud Well-Architected Framework (卓越架构) for RDS with integrated FinOps, SecOps, and AIOps.

| Pillar | RDS Relevance | Key Operations |
|--------|---------------|-----------------|
| **安全 (Security)** | Critical | IAM, VPC, encryption, SSL |
| **稳定 (Stability)** | Critical | HA, backup/restore, DR |
| **成本 (Cost)** | High | Billing, right-sizing, waste |
| **效率 (Efficiency)** | Medium | Automation, batch ops |
| **性能 (Performance)** | Critical | Monitoring, scaling |

---

## 2. Five Pillar Assessment

### 2.1 安全 (Security)

#### IAM Minimum Permissions

| Operation | IAM Action | Resource Scope |
|-----------|-----------|----------------|
| Create Instance | rds:instance:create | acs:rds:*:*:instance/* |
| Describe Instance | rds:instance:get | acs:rds:*:*:instance/* |
| List Instances | rds:instance:list | acs:rds:*:*:instance/* |
| Delete Instance | rds:instance:delete | acs:rds:*:*:instance/${instance_id} |
| Create Backup | rds:backup:create | acs:rds:*:*:instance/${instance_id} |
| Restore Instance | rds:instance:restore | acs:rds:*:*:instance/${instance_id} |
| Modify Parameters | rds:parameter:update | acs:rds:*:*:instance/${instance_id} |
| Create Database | rds:database:create | acs:rds:*:*:instance/${instance_id} |
| Create User | rds:dbuser:create | acs:rds:*:*:instance/${instance_id} |

#### IAM Policy Example

```json
{
  "Version": "1.1",
  "Statement": [
    {
      "Effect": "Allow",
      "Action": [
        "rds:instance:get",
        "rds:instance:list",
        "rds:backup:list",
        "rds:database:list"
      ],
      "Resource": ["acs:rds:*:*:instance/*"]
    },
    {
      "Effect": "Allow",
      "Action": [
        "rds:instance:create",
        "rds:backup:create"
      ],
      "Resource": ["acs:rds:*:*:instance/*"],
      "Condition": {
        "StringEquals": {
          "hw:project": "{{env.HW_PROJECT_ID}}"
        }
      }
    }
  ]
}
```

#### Network Security

- **VPC Endpoint**: Use VPC endpoint for RDS API access (private connectivity)
- **Security Groups**: Restrict inbound to application subnets only
  - MySQL: Port 3306
  - PostgreSQL: Port 5432
  - SQL Server: Port 1433
- **SSL/TLS**: Enforce SSL connections for all database access
- **TDE**: Enable Transparent Data Encryption for sensitive data

#### Data Security

| Data State | Encryption Method | Implementation |
|------------|-------------------|-----------------|
| In Transit | TLS 1.2+ | All API calls use HTTPS |
| At Rest | TDE | Instance-level encryption |
| Backup | AES-256 | OBS server-side encryption |

### 2.2 稳定 (Stability)

#### High Availability Architecture

| Deployment | RTO | RPO | Configuration |
|------------|-----|-----|---------------|
| Single-AZ | N/A | N/A | Not recommended for production |
| Primary/Standby | < 60s | < 5min | HA mode enabled |
| Multi-AZ | < 30s | < 1min | Cross-AZ replication |

#### Backup Strategy

| Backup Type | Frequency | Retention | Use Case |
|-------------|-----------|-----------|----------|
| Automated | Daily | 7 days | Default protection |
| Manual | On-demand | 30 days | Before changes |
| Cross-Region | Weekly | 30 days | DR protection |

#### Disaster Recovery Runbook

```markdown
## Phase 1: Backup Verification
1. Confirm backup exists with status=COMPLETED
2. Verify backup size > 0 (not corrupted)
3. Confirm backup age within RPO window (< 5 min)

## Phase 2: Restore Execution
1. Warn user: data overwrite risk
2. Obtain explicit confirmation
3. Execute restore: hcloud rds restore --instance-id X --backup-id Y
4. Monitor progress: poll status every 60s

## Phase 3: Post-Restore Verification
1. Verify instance status = ACTIVE
2. Verify database integrity
3. Verify application connectivity
4. Update stakeholders

## Phase 4: Post-Mortem
1. Document incident timeline
2. Identify root cause
3. Implement preventive measures
```

### 2.3 成本 (Cost)

#### Billing Model Comparison

| Model | Best For | Cost | Savings vs On-Demand |
|------|----------|------|---------------------|
| 按需计费 | Dev/Test, short-term | Variable | N/A |
| 包年包月 | Production, stable | Fixed | Up to 85% |
| 竞价实例 | Not applicable | N/A | N/A |

#### Right-Sizing Matrix

| CPU Utilization | Memory Utilization | Recommendation | Expected Savings |
|-----------------|-------------------|----------------|-----------------|
| < 20% | < 30% | Downgrade 2 sizes | 30-60% |
| < 20% | > 80% | Change flavor type | 10-20% |
| > 80% | < 50% | Upgrade CPU | — |
| > 80% | > 80% | Upgrade both | — |
| 波动大 (>3x mean) | — | On-demand + auto-scale | 20-50% |

#### Waste Detection Patterns

| Pattern | Detection | Action |
|---------|-----------|--------|
| Idle instance | CPU < 5% for 7 days | Delete or hibernate |
| Unused storage | Disk usage < 20% for 30 days | Shrink (if supported) or migrate |
| Orphaned backup | No active instance reference | Clean up |
| Over-provisioned | CPU < 20%, Mem < 30% for 14 days | Downgrade |

### 2.4 效率 (Efficiency)

#### Automation Patterns

| Operation | CLI | Automation Benefit |
|-----------|-----|---------------------|
| Instance provisioning | hcloud rds create | IaC integration |
| Backup scheduling | hcloud rds create-manual-backup | Cron automation |
| Parameter tuning | hcloud rds modify-parameter | Config management |
| Instance scaling | hcloud rds resize | Dynamic scaling |

#### Batch Operations

```bash
# Batch instance management
for instance_id in $(hcloud rds list --region cn-north-4 --output json | jq -r '.instances[].id'); do
  echo "Checking: $instance_id"
  hcloud rds show --instance-id "$instance_id" --region cn-north-4
done
```

### 2.5 性能 (Performance)

#### Auto-Scaling Thresholds

| Metric | Scale Up | Scale Down | Cooldown |
|--------|----------|-----------|----------|
| CPU | > 80% for 5 min | < 30% for 15 min | 5 min |
| Memory | > 85% for 5 min | < 50% for 15 min | 5 min |
| Connections | > 80% for 5 min | < 40% for 15 min | 5 min |
| Storage | > 90% | N/A | Immediate |

#### Performance Baselines

| Instance Type | Target CPU | Target Memory | Target QPS |
|--------------|-----------|---------------|------------|
| rds.mysql.s1.large | 60-70% | 70-80% | 2,000-3,000 |
| rds.mysql.s1.xlarge | 60-70% | 70-80% | 4,000-6,000 |
| rds.mysql.m1.2xlarge | 60-70% | 70-80% | 8,000-12,000 |

---

## 3. FinOps (财务运营)

### 3.1 Cost Visibility

#### Resource Cost Attribution

| Tag Key | Description | Example |
|---------|-------------|---------|
| CostCenter | Cost center ID | CC-001 |
| Environment | Environment type | prod, staging, dev |
| Owner | Resource owner | user@example.com |
| Project | Project name | e-commerce |
| CreatedDate | Creation date | 2026-05-20 |

#### Cost Analysis Queries

```bash
# Query monthly cost by instance
hcloud bss query-bill --bill_cycle=2026-05 --resource_id=<instance_id>

# Query cost by tag
hcloud bss query-cost-by-tag --tag_key=Environment --tag_value=prod
```

### 3.2 Cost Optimization

#### Cost Optimization Actions

| Action | Trigger | Expected Savings |
|--------|---------|------------------|
| Downgrade | CPU < 20%, Mem < 30% for 14 days | 30-60% |
| Delete idle | CPU < 5% for 7 days | 100% |
| Switch billing | Stable usage > 6 months | 40-85% |
| Enable auto-scaling | Variable load patterns | 20-50% |

### 3.3 Cost Accountability

#### Budget Alert Configuration

| Threshold | Action | Notification |
|-----------|--------|--------------|
| > 80% budget | Warning | Email to owner |
| > 90% budget | Alert | Email + SMS to owner |
| > 100% budget | Block | Email to owner + finance |

---

## 4. SecOps (安全运营)

### 4.1 Identity Security

#### Credential Management

- **AK/SK Rotation**: 90-day rotation recommended
- **MFA Enforcement**: Enable MFA for interactive access
- **IAM Agency**: Use for cross-service access
- **Temporary Credentials**: Use STS for third-party access

#### Security Configuration

```bash
# Enable SSL
hcloud rds modify-parameter \
  --instance-id xxx \
  --name require_secure_transport \
  --value ON \
  --region cn-north-4

# Enable TDE (requires instance recreation)
# Note: TDE must be enabled at instance creation
```

### 4.2 Network Security

#### Security Group Rules

```bash
# Create security group for RDS
hcloud vpc create-security-group \
  --name rds-mysql-sg \
  --vpc-id <vpc_id>

# Allow MySQL from app subnet only
hcloud vpc create-security-group-rule \
  --security-group-id <sg_id> \
  --direction ingress \
  --protocol TCP \
  --port 3306 \
  --source_group_id <app_sg_id>
```

### 4.3 Threat Detection

#### HSS Integration

| Trigger | HSS Action | Skill Response |
|---------|-----------|----------------|
| Vulnerability | Scan RDS instance | Alert security team |
| Intrusion | Detect attack | Isolate instance |
| Compliance | Audit configurations | Generate report |

---

## 5. AIOps Integration

### 5.1 Multi-Metric Correlation

| Pattern | Metrics | Severity |
|---------|---------|----------|
| CPU-Memory Dual High | rds001, rds002 | Critical |
| Connection Saturation | rds003, rds001 | Critical |
| Storage Pressure | rds004, rds045 | Warning |
| Slow Query Spike | rds043 | Warning |

### 5.2 Cross-Skill Diagnosis

| Alert | Primary Skill | Secondary Skill |
|-------|---------------|-----------------|
| High CPU | huaweicloud-rds-ops | huaweicloud-ces-ops |
| Connection Issues | huaweicloud-rds-ops | huaweicloud-vpc-ops |
| Backup Failures | huaweicloud-rds-ops | huaweicloud-obs-ops |

### 5.3 Knowledge Base

#### Fault Pattern: RDS-01 — Connection Exhaustion

| Field | Content |
|-------|---------|
| Trigger | rds003_connections_usage > 90% |
| Symptoms | Connection errors, application timeouts |
| Correlated Metrics | rds001_cpu_usage (may spike due to connection handling) |
| Root Cause | Connection pool leak, too many connections, max_connections too low |
| Diagnosis | Check SHOW PROCESSLIST, check wait_timeout setting, check application pool config |
| Fix | Increase max_connections, restart application, scale up instance |
| Prevention | Set connection timeout, monitor connection trends, implement circuit breaker |

#### Fault Pattern: RDS-02 — Slow Query Degradation

| Field | Content |
|-------|---------|
| Trigger | rds043_slow_queries > 10/s |
| Symptoms | High latency, user complaints |
| Correlated Metrics | rds001_cpu_usage, rds045_iops |
| Root Cause | Missing indexes, large table scans, outdated statistics |
| Diagnosis | Analyze EXPLAIN output, check query plans |
| Fix | Add indexes, optimize queries, update statistics |
| Prevention | Code review for queries, regular EXPLAIN analysis |

---

## 6. Compliance Checklists

### P0 — Must Pass

#### Security
- [ ] IAM minimum permissions documented
- [ ] Credential masking enforced
- [ ] VPC isolation recommended
- [ ] SSL/TLS enforcement documented
- [ ] TDE configuration documented

#### Stability
- [ ] Backup/restore documented with RTO/RPO
- [ ] HA configuration options documented
- [ ] DR runbook phase 1/2/3 structure
- [ ] All destructive ops require confirmation

#### Cost
- [ ] Billing model comparison table
- [ ] Idle resource detection pattern
- [ ] Right-sizing guidance documented
- [ ] Cost tagging strategy documented

#### Performance
- [ ] Key metrics with thresholds
- [ ] Auto-scaling trigger thresholds
- [ ] Performance baseline documented

### P1 — Should Pass

- [ ] Multi-AZ deployment recommendation
- [ ] Cross-region backup strategy
- [ ] Budget alert integration
- [ ] MFA requirement documented
- [ ] Proactive inspection workflow
- [ ] Self-healing patterns documented

---

*This document defines the Well-Architected assessment for RDS operations. Refer to official Huawei Cloud documentation for the latest specifications.*
---

## Worker Output Contract (Read-Only Assessment Mode)

> Invoked when Well-Architected review sets `{{user.mode}}=well-architected-readonly`.
> Return **`{{output.product_assessment}}`** — field names MUST match the canonical schema.

**Canonical schema:** [worker-output-schema.md](../../huaweicloud-skill-generator/references/worker-output-schema.md)

| Constant | Value |
|----------|-------|
| `skill_id` | `huaweicloud-rds-ops` |
| `product` | `rds` |
| Finding `id` pattern | `rds-{rel|sec|cost|eff}-NNN` |

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
  "skill_id": "huaweicloud-rds-ops",
  "product": "rds",
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
      "hcloud rds read-only-list --region cn-north-4 (HW_SECRET_ACCESS_KEY=<masked>)"
    ],
    "request_ids": [
      "0123456789abcdef0123456789abcdef"
    ]
  },
  "errors": []
}
```
