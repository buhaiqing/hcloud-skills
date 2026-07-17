# CCE Well-Architected Assessment — Huawei Cloud Cloud Container Engine

## 1. Security Assessment

### 1.1 IAM Minimum Permissions

| Operation | IAM Action | Resource Scope |
|-----------|-----------|----------------|
| List/Describe clusters, nodes, nodepools | `cce:cluster:list`, `cce:node:list` | `*` or specific project |
| Create cluster | `cce:cluster:create`, `vpc:vpcs:get`, `vpc:subnets:get` | Project level |
| Delete cluster | `cce:cluster:delete` | Specific cluster |
| Create/Delete node | `cce:node:create`, `cce:node:delete`, `ecs:servers:create` | Specific cluster |
| Create/Delete node pool | `cce:nodepool:create`, `cce:nodepool:delete` | Specific cluster |
| Install/Delete addon | `cce:addon:create`, `cce:addon:delete` | Specific cluster |
| Get cluster cert | `cce:cluster:get` | Specific cluster |

### Recommended IAM Policy Groups

| Role | Permissions | Use Case |
|------|------------|----------|
| CCE Administrator | Full CCE access | Cluster administrators |
| CCE Viewer | Read-only CCE access | Monitoring and auditing |
| CCE Developer | Cluster list, node list, pod operations | Developers accessing clusters |

### 1.2 Credential Management

- Credentials MUST use `{{env.*}}` placeholders — NEVER ask user for secrets
- AK/SK rotation: recommend 90-day cycle
- Use IAM agency (委托) for cross-account CCE operations
- Prefer VPC endpoints over public endpoints for API calls

### 1.3 Network Isolation

- CCE API calls: Use `https://cce.<region>.myhuaweicloud.com` (HTTPS mandatory)
- Pod network: Configure security groups to restrict inter-service traffic
- API server: Enable authentication tokens; disable anonymous access
- Service exposure: Use internal ELB for within-VPC services; EIP only for public-facing

### 1.4 RBAC Enforcement

- Cluster RBAC: Configure RoleBinding per namespace
- Default service accounts: Disable default `default` SA from mounting tokens where possible
- Pod security: Use restricted pod security standards; avoid privileged containers

## 2. Stability Assessment

### 2.1 Built-in Resilience

- Every operation follows Pre-flight → Execute → Validate → Recover
- Non-retryable errors (QuotaExceeded, InsufficientBalance) trigger HALT
- Async operations (cluster/node creation) poll with timeout and explicit failure states
- Node pool autoscaling: automatic node replacement for unhealthy nodes

### 2.2 Multi-AZ Deployment

| Strategy | Description | CLI/SDK Guidance |
|----------|-------------|------------------|
| Multi-AZ node pools | Spread node pools across AZs | Create multiple node pools with AZ-specific subnet |
| Pod anti-affinity | Prevent all replicas on same AZ | Use `topologySpreadConstraints` with `topology.kubernetes.io/zone` |
| Cluster HA | CCE control plane is multi-AZ by default | Managed by Huawei Cloud; no user action needed |

### 2.3 Backup and Recovery

| Resource | Backup Method | RPO | RTO |
|----------|--------------|-----|-----|
| Cluster (etcd snapshots) | CCE backup addon | 1 hour | 30 minutes |
| PersistentVolume (EVS snapshots) | EVS snapshot via everest CSI | 1 hour | 15 minutes |
| Application manifests | Git backup | 0 (version-controlled) | 5 minutes |
| ConfigMaps/Secrets | etcd snapshot or backup script | 1 hour | 10 minutes |

### 2.4 DR Runbook

#### Phase 1: Backup Verification

1. Confirm etcd backup exists and is recent (within RPO window)
2. Verify EVS snapshots are healthy for all PVCs
3. Export critical manifests: `kubectl get all -A -o yaml > backup-$(date +%Y%m%d).yaml`
4. Confirm backup can be accessed from target region

#### Phase 2: Recovery Execution

1. Create new cluster in target region/DR zone
2. Restore etcd from backup
3. Reinstall addons (coredns, everest, metrics-server, CA)
4. Restore PVCs from EVS snapshots
5. Apply application manifests from backup

#### Phase 3: Post-Recovery Validation

1. Verify all nodes Active and resources allocated
2. Verify all pods Running and passing health checks
3. Verify external endpoints (ELB, EIP) pointing to new cluster
4. Run smoke tests against critical services
5. Document recovery duration vs RTO target

### 2.5 Auto-Scaling

| Trigger | Scale-Out Threshold | Scale-In Threshold | Cooldown |
|---------|-------------------|-------------------|----------|
| CPU > 80% | Average CPU > 80% for 5min across node pool | Average CPU < 30% for 15min | 300s |
| Memory > 85% | Average memory > 85% for 5min | Average memory < 50% for 15min | 300s |
| Pending POD | Any pod Pending due to Insufficient CPU/Memory | No Pending pods, nodes underutilized | 300s |

## 3. Cost Assessment

### 3.1 Billing Model Comparison

| Billing Type | Best For CCE | Savings vs Pay-per-use |
|-------------|--------------|----------------------|
| Pay-per-use (按需) | Dev/test, short-lived workloads, burst capacity | N/A |
| Subscription (包年包月) | Production clusters with stable node count | Up to 85% savings |
| Spot instances (竞价) | Fault-tolerant, batch processing, HPA scale-out pods | Up to 90% savings |

### 3.2 Node Cost Optimization Matrix

| CPU Usage | Memory Usage | Recommendation | Expected Savings |
|-----------|-------------|---------------|-----------------|
| < 10% (7+ days) | < 10% (7+ days) | Scale down node pool or decommission | 60-100% |
| < 10% | > 70% | Switch to memory-optimized flavor (m7) | 10-20% |
| > 70% | < 30% | Switch to compute-optimized flavor (c7) | 10-20% |
| > 70% | > 70% | Scale up flavor or add nodes | — |
| High variance | — | Use spot instances + HPA for burst | 30-60% |

### 3.3 Idle Cluster Detection

- Nodes with CPU < 5% and memory < 10% for 7+ consecutive days → consider decommissioning cluster or scaling pool to 0
- Node pools with 0 running pods across all nodes → candidate for deletion
- Addons installed but no workloads using them → consider removal

### 3.4 Cost Tagging Strategy

| Tag Key | Value Example | Purpose |
|---------|--------------|---------|
| `env` | production / staging / dev | Environment classification |
| `team` | platform / product-data | Cost center attribution |
| `cost-center` | CC-2024-001 | Finance allocation |
| `project` | ecommerce-v2 | Project cost tracking |
| `ttl` | 30d | Auto-decommission deadline for dev clusters |

## 4. Efficiency Assessment

### 4.1 Batch Operations

- Creating ≥ 3 nodes: use `--count` flag for batch node creation
- Updating ≥ 2 node pools: parallel UpdateNodePool calls
- Bulk addon installation: script loop over addon list
- CI/CD integration: CLI JSON output compatible with `jq` for pipeline parsing

### 4.2 Event Response Integration

- CCE cluster events: `kubectl get events -n kube-system`
- CCE API errors can trigger CES alarm rules
- Escalation: automated → skill-assisted → human
- Node pool autoscaling events: log and monitor for pattern detection

### 4.3 Addon Automation

- Pre-install required addons during cluster creation (coredns, everest)
- Use addon templates for standardized installations
- Validate addon version compatibility before installation
- Automate addon updates through CI/CD pipelines

## 5. Performance Assessment

### 5.1 Cluster-Scale Limits

| Resource | Limit | Notes |
|----------|-------|-------|
| Pods per cluster | 10,000 | Varying by cluster type |
| Pods per node | 128 | Limit is flavor-dependent; VPC mode = subnet IPs |
| Nodes per cluster | 1,000 | Hard limit |
| Services per cluster | 10,000 | Depends on kube-proxy mode |

### 5.2 Performance Tuning

| Component | Setting | Recommended Value | Impact |
|-----------|---------|-------------------|--------|
| kube-proxy mode | `ipvs` vs `iptables` | `ipvs` | Better performance for > 1000 services |
| Pod CIDR size | /16 per cluster or /24 per node | Match workload density | Larger CIDR = more pods per node |
| ETCD disk type | Ultra-high I/O SSD | For clusters with high etcd throughput | Lower write latency |

### 5.3 API Server Performance

| Metric | Healthy Threshold | Degraded | Critical |
|--------|------------------|----------|----------|
| API latency (P50) | < 50ms | 50-200ms | > 200ms |
| API latency (P99) | < 500ms | 500ms-2s | > 2s |
| API error rate | < 0.1% | 0.1%-1% | > 1% |
### SLO/SLI Definition — CCE

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
| `skill_id` | `huaweicloud-cce-ops` |
| `product` | `cce` |
| Finding `id` pattern | `cce-{rel|sec|cost|eff}-NNN` |

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
  "skill_id": "huaweicloud-cce-ops",
  "product": "cce",
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
      "hcloud cce read-only-list --region cn-north-4 (HW_SECRET_ACCESS_KEY=<masked>)"
    ],
    "request_ids": [
      "0123456789abcdef0123456789abcdef"
    ]
  },
  "errors": []
}
```
