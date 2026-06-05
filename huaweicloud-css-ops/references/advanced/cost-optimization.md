# CSS Cost Optimization (FinOps)

## Billing Model

### Instance Pricing

| Node Type | Specification | Hourly Rate (参考) | Monthly Estimate |
|-----------|---------------|-------------------|------------------|
| ess | 4vCPU 8GB | ¥X.XX | ¥XXX |
| ess | 8vCPU 16GB | ¥X.XX | ¥XXX |
| ess | 16vCPU 32GB | ¥X.XX | ¥XXXX |
| ess-master | 2vCPU 4GB | ¥X.XX | ¥XXX |
| ess-client | 4vCPU 8GB | ¥X.XX | ¥XXX |
| ess-cold | 4vCPU 8GB + SATA | ¥X.XX | ¥XXX |

### Storage Pricing

| Storage Type | Price (GB/月) | Use Case |
|--------------|---------------|----------|
| ULTRAHIGH SSD | ¥X.XX | Production hot data |
| HIGH SSD | ¥X.XX | Development, warm data |
| COMMON SATA | ¥X.XX | Cold data, archive |

### Snapshot Pricing

| Component | Pricing | Notes |
|-----------|---------|-------|
| OBS Storage | ¥X.XX/GB/月 | Compressed snapshot size |
| API Calls | ¥X/万次 | PUT/GET requests |
| Data Transfer | ¥X/GB | Cross-region transfer |

## Cost Optimization Strategies

### 1. Right-Sizing

**Analysis Framework**:
```
Metrics to collect (7-day window):
- Average CPU usage
- Peak CPU usage
- Average memory usage
- Peak memory usage
- Average disk usage
- Search latency p99

Decision matrix:
CPU avg < 30% AND memory < 40% → Downgrade 1 tier
CPU avg 30-70% OR memory 40-70% → Current size optimal
CPU avg > 80% OR memory > 80% → Upgrade 1 tier
```

**Implementation**:
```bash
# Analyze current usage
hcloud CES ShowMetricData \
  --namespace SYS.CSS \
  --metric_name cpu_usage \
  --dimensions cluster_id={{user.cluster_id}} \
  --period 86400 \
  --from $(date -d '7 days ago' +%s)000 \
  --to $(date +%s)000

# If downsizing is recommended
hcloud CSS ExtendCluster \
  --cluster_id {{user.cluster_id}} \
  --flavor_ref ess.spec-2u4g  # Downgrade example
```

### 2. Storage Tiering

**Hot-Warm-Cold Architecture**:

```
┌─────────────────────────────────────────────┐
│              CSS Cluster                     │
├─────────────────────────────────────────────┤
│  Hot Nodes (ess)        - 7 days data       │
│  Warm Nodes (ess)       - 30 days data      │
│  Cold Nodes (ess-cold)  - 1 year data       │
└─────────────────────────────────────────────┘
```

**Index Lifecycle Policy**:
```json
{
  "policy": {
    "phases": {
      "hot": {
        "min_age": "0ms",
        "actions": {
          "rollover": {
            "max_size": "30GB",
            "max_age": "1d"
          }
        }
      },
      "warm": {
        "min_age": "7d",
        "actions": {
          "shrink": {
            "number_of_shards": 1
          },
          "forcemerge": {
            "max_num_segments": 1
          }
        }
      },
      "cold": {
        "min_age": "30d",
        "actions": {
          "allocate": {
            "require": {
              "box_type": "cold"
            }
          }
        }
      },
      "delete": {
        "min_age": "365d",
        "actions": {
          "delete": {}
        }
      }
    }
  }
}
```

### 3. Snapshot Lifecycle Management

**Retention Strategy**:

| Snapshot Type | Frequency | Retention | Cost Impact |
|---------------|-----------|-----------|-------------|
| Automatic | Daily | 7 days | Baseline |
| Manual (pre-change) | On-demand | 30 days | Variable |
| Weekly | Weekly | 4 weeks | Medium |
| Monthly | Monthly | 12 months | Higher |

**Cost-Optimized Policy**:
```bash
hcloud CSS SetSnapshotPolicy \
  --cluster_id {{user.cluster_id}} \
  --period "02:00 GMT+08:00" \
  --prefix "daily" \
  --keepday 7 \
  --bucket "{{user.bucket_name}}" \
  --base-path "css-snapshots/{{user.cluster_name}}"

# For compliance (monthly retention)
# Create additional monthly snapshot via cron
```

### 4. Reserved Capacity

**Savings Comparison**:

| Purchase Type | Discount | Commitment | Best For |
|---------------|----------|------------|----------|
| On-demand | 0% | None | Variable workloads |
| 1-year reserved | 20% | 1 year | Stable production |
| 3-year reserved | 35% | 3 years | Long-term workloads |
| Savings plans | 15-25% | Flexible | Multiple clusters |

### 5. Idle Resource Detection

**Detection Logic**:
```python
def identify_idle_clusters():
    clusters = list_all_clusters()
    
    for cluster in clusters:
        metrics = query_ces(
            cluster_id=cluster.id,
            metrics=['search_rate', 'indexing_rate', 'cpu_usage'],
            window='7d'
        )
        
        # Idle criteria
        if (metrics['search_rate'].sum() == 0 and 
            metrics['indexing_rate'].sum() < 100 and
            metrics['cpu_usage'].avg() < 5):
            
            flag_for_review(cluster, reason="Low activity detected")
```

**Action Options**:
1. **Delete**: If no longer needed
2. **Scale down**: Reduce to minimum size
3. **Snapshot and delete**: Preserve data, remove cluster
4. **Keep**: Document reason for low utilization

## Cost Monitoring

### Tagging Strategy

```yaml
mandatory_tags:
  - key: Project
    value: "{{user.project}}"
  - key: Environment
    value: "{{user.environment}}"  # dev, staging, prod
  - key: CostCenter
    value: "{{user.cost_center}}"
  - key: Owner
    value: "{{user.owner}}"
  - key: AutoShutdown
    value: "{{user.auto_shutdown}}"  # true/false for dev

optional_tags:
  - key: DataClassification
    value: "{{user.data_class}}"  # public, internal, confidential
  - key: ScheduledHours
    value: "{{user.hours}}"  # 24x7, business-hours
```

### Budget Alerts

```yaml
budget_alerts:
  - name: "50% Budget"
    threshold: 50
    action: notification
    recipients: [team-email]
  
  - name: "80% Budget"
    threshold: 80
    action: warning
    recipients: [team-email, team-lead]
  
  - name: "100% Budget"
    threshold: 100
    action: block_new_resources
    recipients: [team-email, team-lead, finance]
```

## Cost Optimization Checklist

### Weekly Review
- [ ] Review clusters with CPU < 30% for downsizing
- [ ] Review clusters with disk > 80% for cleanup
- [ ] Review snapshot storage growth
- [ ] Verify tagging compliance

### Monthly Review
- [ ] Analyze cost by project/environment
- [ ] Review reserved capacity utilization
- [ ] Identify idle resources
- [ ] Optimize snapshot retention

### Quarterly Review
- [ ] Evaluate reserved capacity purchases
- [ ] Assess storage tiering effectiveness
- [ ] Review ILM policy performance
- [ ] Update cost optimization playbook

## Cost Calculator

### Example: Monthly Cost Estimation

```yaml
cluster_config:
  data_nodes:
    count: 3
    flavor: ess.spec-4u8g
    storage: 100GB SSD
  master_nodes:
    count: 3
    flavor: ess.spec-2u4g
  client_nodes:
    count: 0
  
  snapshots:
    frequency: daily
    retention: 7 days
    avg_snapshot_size: 50GB

estimation:
  compute_cost: |
    (3 × data_node_price + 3 × master_node_price) × 730 hours
  storage_cost: |
    (3 × 100GB × ssd_price) + (50GB × 7 × obs_price)
  total: compute_cost + storage_cost
```
