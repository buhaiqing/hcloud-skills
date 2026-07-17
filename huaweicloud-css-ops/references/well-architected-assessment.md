# CSS Well-Architected Assessment

## 卓越架构框架评估 (Well-Architected Framework)

### 2.1 安全支柱 (Security)

| Best Practice | Implementation | Status |
|---------------|----------------|--------|
| Data encryption at rest | KMS encryption for cluster disks | ✅ Required |
| Data encryption in transit | HTTPS enabled for all endpoints | ✅ Required |
| VPC isolation | Private subnet deployment | ✅ Required |
| Security groups | Least privilege ingress/egress | ✅ Required |
| IAM least privilege | Service-specific permissions | ✅ Required |
| Password policy | Strong admin password, regular rotation | ✅ Recommended |
| Snapshot encryption | OBS bucket encryption | ✅ Recommended |
| Audit logging | CTS integration for API calls | ⚪ Optional |

**Security Configuration Example**:
```yaml
security:
  https_enabled: true
  disk_encryption:
    enabled: true
    kms_key_id: "{{user.kms_key_id}}"
  network:
    vpc_isolation: true
    security_group_rules:
      - protocol: tcp
        port: 9200
        source: "vpc_cidr_only"
  iam:
    policy: "css-minimum-permissions"
```

### 2.2 稳定支柱 (Stability)

| Best Practice | Implementation | Status |
|---------------|----------------|--------|
| Multi-AZ deployment | Nodes across 3 AZs | ✅ Required |
| Automated snapshots | Daily backup with 7-day retention | ✅ Required |
| Health monitoring | CES alerts for cluster health | ✅ Required |
| Auto-recovery | Node failure auto-replacement | ✅ Platform |
| Master node HA | 3 master nodes minimum | ✅ Required |
| Data replication | Replica shards for redundancy | ✅ Required |
| Snapshot testing | Periodic restore verification | ⚪ Recommended |
| DR runbook | Documented recovery procedures | ⚪ Recommended |

**Availability Targets**:
- Single AZ: 99.9% SLA
- Multi AZ: 99.95% SLA

### 2.3 成本支柱 (Cost)

| Best Practice | Implementation | Savings |
|---------------|----------------|---------|
| Right-sizing | Match flavor to workload | 20-40% |
| Storage tiering | Cold nodes for archive data | 30-50% |
| Snapshot lifecycle | Automated cleanup | 10-20% |
| Client nodes | Offload coordination | 15-25% |
| Reserved instances | 1-year commitment | 20-30% |
| Auto-scaling | Scale based on demand | Variable |

**Cost Optimization Strategies**:

1. **Instance Right-Sizing**
   ```bash
   # Monitor CPU/memory for 7 days
   # Target: 60-70% average utilization
   # Scale up if >80% sustained
   # Scale down if <30% sustained
   ```

2. **Storage Optimization**
   - Hot data: ULTRAHIGH SSD
   - Warm data: HIGH SSD
   - Cold data: Cold nodes + COMMON storage

3. **Snapshot Cost Control**
   - Daily: 7 days retention
   - Weekly: 4 weeks retention
   - Monthly: 12 months retention

### 2.4 效率支柱 (Efficiency)

| Best Practice | Implementation | Benefit |
|---------------|----------------|---------|
| Infrastructure as Code | Terraform/CloudFormation templates | Repeatability |
| CI/CD integration | Automated deployment pipeline | Speed |
| Parameter templates | Reusable configuration presets | Consistency |
| Batch operations | Bulk API for management tasks | Throughput |
| Monitoring automation | Auto-alerting and response | Reduced MTTR |

**Efficiency Metrics**:
- Deployment time: < 15 minutes
- Recovery time: < 30 minutes
- Scale-out time: < 10 minutes

### 2.5 性能支柱 (Performance)

| Best Practice | Implementation | Target |
|---------------|----------------|--------|
| Shard strategy | 20-50GB per shard | Optimal sizing |
| Replica count | 1 for prod, 0 for dev | Balance HA/cost |
| Query optimization | Use filters, avoid deep pagination | <100ms p99 |
| Index lifecycle | Rollover at 30GB or 1 day | Manageable size |
| Client nodes | Add for query-heavy workloads | Reduced latency |
| Bulk indexing | Batch size 5-15MB | Max throughput |

**Performance Baselines**:

| Metric | Target | Warning | Critical |
|--------|--------|---------|----------|
| Search latency (p99) | <100ms | 100-500ms | >500ms |
| Indexing throughput | >10k/s | 5-10k/s | <5k/s |
| CPU utilization | 40-70% | 70-80% | >80% |
| JVM heap usage | 50-70% | 70-85% | >85% |
| Disk I/O wait | <5% | 5-10% | >10% |

---

## 3. FinOps Integration (财务运营)

### 3.1 Cost Visibility

| Resource | Billing Model | Cost Driver |
|----------|---------------|-------------|
| Data Nodes | Hourly (实例规格) | vCPU + Memory + Storage |
| Master Nodes | Hourly | vCPU + Memory |
| Client Nodes | Hourly | vCPU + Memory |
| Storage | GB-month | Allocated disk |
| Snapshots | GB-month + API calls | OBS storage |
| Data Transfer | GB | Cross-region/AZ traffic |

**Cost Attribution Tags**:
```yaml
tags:
  - key: Project
    value: "{{user.project}}"
  - key: Environment
    value: "{{user.environment}}"
  - key: CostCenter
    value: "{{user.cost_center}}"
  - key: Owner
    value: "{{user.owner}}"
```

### 3.2 Cost Optimization

#### Right-Sizing Decision Matrix

| Current Usage | Recommendation | Expected Savings |
|---------------|----------------|------------------|
| CPU <30%, Memory <40% | Downgrade one tier | 20-30% |
| CPU 30-60%, Memory 40-70% | Keep current | - |
| CPU >80%, Memory >80% | Upgrade one tier | Avoid bottleneck |
| Disk >85% | Extend storage | Prevent outage |

#### Idle Resource Detection

```python
def detect_idle_clusters(days=7):
    """Identify clusters with low activity"""
    clusters = list_css_clusters()
    
    for cluster in clusters:
        metrics = query_ces(
            cluster_id=cluster.id,
            metrics=['search_rate', 'indexing_rate', 'cpu_usage'],
            days=days
        )
        
        # Idle if no searches and CPU <10%
        if metrics['search_rate'].sum() == 0 and metrics['cpu_usage'].avg() < 10:
            flag_for_review(cluster)
```

### 3.3 Budget Alerting

```yaml
budget_alerts:
  - threshold: 50%
    action: notify
    recipients: [team-lead]
  - threshold: 80%
    action: warn
    recipients: [team-lead, finance]
  - threshold: 100%
    action: block_new_resources
    recipients: [team-lead, finance, manager]
```

---

## 4. SecOps Integration (安全运营)

### 4.1 IAM Security

**Minimum Permissions for CSS Operations**:

```json
{
  "Version": "1.1",
  "Statement": [
    {
      "Sid": "CSSReadOnly",
      "Effect": "Allow",
      "Action": [
        "css:cluster:get",
        "css:cluster:list",
        "css:snapshot:get",
        "css:snapshot:list",
        "css:dict:get",
        "css:dict:list"
      ],
      "Resource": "*"
    },
    {
      "Sid": "CSSWrite",
      "Effect": "Allow",
      "Action": [
        "css:cluster:create",
        "css:cluster:delete",
        "css:cluster:modify",
        "css:snapshot:create",
        "css:snapshot:delete",
        "css:snapshot:restore"
      ],
      "Resource": "*",
      "Condition": {
        "StringEquals": {
          "vpc:id": "{{user.vpc_id}}"
        }
      }
    }
  ]
}
```

### 4.2 Network Security

| Layer | Control | Implementation |
|-------|---------|----------------|
| VPC | Isolation | Private subnets only |
| Subnet | Segmentation | Dedicated CSS subnet |
| Security Group | Port access | 9200, 9300 from VPC only |
| NACL | Additional filtering | Deny external ingress |
| Endpoint | HTTPS only | TLS 1.2+ required |

### 4.3 Data Security

| Data State | Protection | Mechanism |
|------------|------------|-----------|
| At rest | KMS encryption | AES-256 |
| In transit | TLS | TLS 1.2+ |
| In snapshot | OBS encryption | SSE-KMS |
| In backup | Client-side | Application-level |

### 4.4 Threat Detection

| Threat | Detection | Response |
|--------|-----------|----------|
| Unauthorized access | Failed auth logs | Alert + IP block |
| Privilege escalation | IAM policy changes | Alert + audit |
| Data exfiltration | Unusual query patterns | Alert + rate limit |
| Crypto mining | High CPU anomaly | Alert + investigate |

---

## 5. AIOps Integration (智能运营)

### 5.1 Anomaly Detection

| Pattern | Detection Method | Severity |
|---------|------------------|----------|
| Cluster health degradation | Health status + shard allocation | Critical |
| Query latency spike | P99 latency > 3σ | Warning |
| Storage growth anomaly | Growth rate > 2x baseline | Warning |
| JVM pressure | Heap usage > 85% | Critical |
| Node failure | Node count < expected | Critical |
| Snapshot failure | Status = FAILED | Warning |

### 5.2 Self-Healing Capabilities

| Issue | Automated Response | Manual Escalation |
|-------|-------------------|-------------------|
| Node failure | Auto-replacement | If > 1 node fails |
| Shard unassigned | Auto-reallocation | If persists > 10min |
| High JVM | GC optimization | If OOM occurs |
| Snapshot fail | Auto-retry (3x) | If all retries fail |

### 5.3 Knowledge Base

**Known Issues and Patterns**:

| Issue | Root Cause | Solution |
|-------|------------|----------|
| Red cluster health | Node failure | Replace node, reallocate shards |
| Yellow cluster health | Replica unassigned | Add node or reduce replicas |
| High search latency | Hot shard | Reindex with more shards |
| Slow indexing | Bulk queue full | Reduce batch size, add nodes |
| Snapshot slow | OBS throttling | Reduce frequency, check OBS limits |

---

## Cross-Pillar Trade-off Matrix

| Decision | Security | Stability | Cost | Efficiency | Performance |
|----------|----------|-----------|------|------------|-------------|
| Enable encryption | + | 0 | - | 0 | - |
| Add AZ | + | ++ | -- | 0 | 0 |
| Reduce replicas | - | - | ++ | 0 | 0 |
| Cold nodes | 0 | 0 | + | 0 | - |
| Client nodes | 0 | 0 | - | + | ++ |
| Larger shards | 0 | - | + | + | - |

Legend: ++ Strong positive, + Positive, 0 Neutral, - Negative, -- Strong negative
### SLO/SLI Definition — CSS

#### SLI (Service Level Indicator) Metrics

| SLI Name | Formula | Data Source | Collection Frequency |
|----------|---------|-------------|---------------------|
| Availability | Successful requests / Total requests × 100% | CES + ELB | 1min |
| Latency P99 | 99th percentile response time (ms) | AOM Trace | 1min |
| Error Rate | 5xx responses / Total requests × 100% | ELB + AOM | 1min |
| Saturation | CPU utilization / Connection utilization / Disk utilization | CES | 5min |

#### SLO Targets

| SLI | SLO Target | Error Budget (Monthly) | Alert Threshold |
|-----|------------|-----------------------|-----------------|
| Availability | ≥ 99.9% | 43.2 min/month | < 99.95% triggers Warning |
| Latency P99 | ≤ 200ms | — | > 300ms triggers Warning |
| Error Rate | ≤ 0.1% | — | > 0.5% triggers Critical |
| Saturation | ≤ 80% | — | > 85% triggers Warning |

#### Error Budget Burn Rate Alerts

| Burn Rate | Consumption Speed | Alert Level | Meaning |
|-----------|------------------|-------------|---------|
| 1× | Normal consumption (43.2 min/month) | — | Normal |
| 2× | 21.6 min exhausted | Info | Attention needed |
| 5× | 8.6 min exhausted | Warning | Intervention needed |
| 14.4× | 3h exhausted | Critical | Immediate action required |


---

## Worker Output Contract (Read-Only Assessment Mode)

> Invoked when Well-Architected review sets `{{user.mode}}=well-architected-readonly`.
> Return **`{{output.product_assessment}}`** — field names MUST match the canonical schema.

**Canonical schema:** [worker-output-schema.md](../../huaweicloud-skill-generator/references/worker-output-schema.md)

| Constant | Value |
|----------|-------|
| `skill_id` | `huaweicloud-css-ops` |
| `product` | `css` |
| Finding `id` pattern | `css-{rel|sec|cost|eff}-NNN` |

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
  "skill_id": "huaweicloud-css-ops",
  "product": "css",
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
      "hcloud css read-only-list --region cn-north-4 (HW_SECRET_ACCESS_KEY=<masked>)"
    ],
    "request_ids": [
      "0123456789abcdef0123456789abcdef"
    ]
  },
  "errors": []
}
```
