---
name: huaweicloud-css-ops
description: >-
  Use when the user needs to deploy, configure, troubleshoot, or monitor Huawei
  Cloud CSS (Cloud Search Service) — Elasticsearch/OpenSearch cluster lifecycle,
  snapshot management, dictionary management, and configuration. User mentions
  CSS, Cloud Search Service, Elasticsearch, OpenSearch, 云搜索服务, 搜索集群,
  or describes search-related scenarios (e.g., cluster unavailable, search
  latency high, shard allocation failed, snapshot restore needed) even without
  naming the product directly. Not for billing, IAM, or Log Tank Service
  (LTS) which has its own ops skill.
license: MIT
compatibility: >-
  Official Huawei Cloud CLI (`hcloud`), Go 1.21+ runtime
  (for JIT SDK fallback via huaweicloud-sdk-go-v3), valid API credentials,
  network access to Huawei Cloud CSS endpoints.
metadata:
  author: huaweicloud
  version: "1.0.0"
  last_updated: "2026-05-27"
  runtime: Harness AI Agent, Claude Code, Cursor, or compatible Agent runtimes
  go_version_minimum: "1.21"
  go_version_jit: "1.24+"
  api_profile: "CSS v1/v2 — https://support.huaweicloud.com/api-css/"
  cli_applicability: "dual-path"
  cli_support_evidence: >-
    CSS operations available via `hcloud CSS <operation>` where operation
    matches the API Explorer name: ListClusters, ShowClusterDetail,
    CreateCluster, DeleteCluster, RestartCluster, ExtendCluster,
    ListSnapshots, CreateSnapshot, RestoreSnapshot, etc.
  gcl:
    enabled: true
    required: true
    rubric_version: "v1"
    max_iter: 2
    rubric_ref: "references/rubric.md"
    prompts_ref: "references/prompt-templates.md"
    trace_dir: "./audit-results/"
    changelog:
      - version: "1.1.0"
        date: "2026-06-04"
        change: "GCL Phase 2 rollout: added references/rubric.md (v1, 5-dim, S1–S16 CSS/ES-specific Safety rules, including wildcard index delete / match_all / system index / forcemerge guards) and references/prompt-templates.md (Generator + Critic + Orchestrator). SKILL.md gains 'Quality Gate (GCL)' chapter."
      - version: "1.0.0"
        date: "2026-05-27"
        change: "Initial skill release."
  environment:
    - HW_ACCESS_KEY_ID
    - HW_SECRET_ACCESS_KEY
    - HW_REGION_ID
    - HW_PROJECT_ID
---

> This skill follows the [Agent Skill Open Specification](https://agentskills.io/specification).

# Huawei Cloud CSS (Cloud Search Service) Operations Skill

## Overview

Cloud Search Service (CSS) is Huawei Cloud's fully managed search engine service, fully compatible with open-source **Elasticsearch** and **OpenSearch**. It supports structured and unstructured text search, aggregation, and analytics. This skill is an **operational runbook** for agents: explicit scope, credential rules, pre-flight checks, **dual-path execution** (official `hcloud` CLI and JIT Go SDK fallback), response validation, and failure recovery.

**API Versions**: v1 (legacy), v2 (recommended) — Go SDK `services/css/v1` and `services/css/v2`

**Supported Engines**:
- **Elasticsearch**: 7.6.2, 7.10.2
- **OpenSearch**: 1.3.6, 2.17.1

**Node Types**:
- **ess**: Elasticsearch/OpenSearch data node (存储节点)
- **ess-master**: Master node (管理节点)
- **ess-client**: Client node (协调节点)
- **ess-cold**: Cold data node (冷数据节点)

### CLI applicability (repository policy)

- **`cli_applicability: dual-path`:** Official CLI supports CSS. In each execution flow below, document **both** the SDK step **and** the CLI step.

## Five Core Standards (Quality Gates)

| # | Standard | How This Skill Fulfills It |
|---|----------|---------------------------|
| 1 | **Clear Boundaries** | SHOULD/SHOULD NOT Use conditions below with delegation rules |
| 2 | **Structured I/O** | `{{env.*}}` for credentials, `{{user.*}}` for interactive input, `{{output.*}}` for response capture |
| 3 | **Explicit Actionable Steps** | Every operation: Pre-flight → Execute (CLI + SDK) → Validate → Recover |
| 4 | **Complete Failure Strategies** | 16 service error codes documented; HALT vs retry per error type |
| 5 | **Absolute Single Responsibility** | One product (CSS); cross-product delegation to other skills |

### Three-Pillar Ops Integration (FinOps + SecOps + AIOps)

| Pillar | Skill Integration | Reference |
|--------|-------------------|-----------|
| **FinOps** | Cluster sizing, storage optimization, snapshot lifecycle, idle detection | `references/advanced/cost-optimization.md` |
| **SecOps** | IAM policies, KMS encryption, VPC isolation, HTTPS enforcement | `references/advanced/security-best-practices.md` |
| **AIOps** | 4 anomaly patterns (cluster health, query latency, shard allocation, storage growth) | `references/advanced/aiops-best-practices.md` |

### Well-Architected Framework Integration (卓越架构)

| Pillar | Skill Integration |
|--------|-------------------|
| **安全 (Security)** | IAM minimum permissions, KMS disk encryption, HTTPS/TLS, VPC isolation |
| **稳定 (Stability)** | Multi-AZ deployment, snapshot backup/restore, cluster health monitoring |
| **成本 (Cost)** | Instance type selection, storage tiering, snapshot retention tuning |
| **效率 (Efficiency)** | Batch CLI operations, template reuse, CI/CD integration |
| **性能 (Performance)** | Shard allocation strategy, node scaling, query optimization |

## Trigger & Scope (Agent-Readable)

### SHOULD Use This Skill When

- User mentions "Huawei Cloud CSS" / "Cloud Search Service" / "Elasticsearch" / "OpenSearch" / "云搜索服务" / "搜索集群"
- Task involves CRUD or lifecycle management of CSS clusters (create, list, show, delete, restart, resize)
- Task involves snapshot management (create, list, delete, restore)
- Task involves dictionary management (IK word dictionary, synonym dictionary, stop word dictionary)
- Task involves cluster configuration (parameter modification, password reset, network config)
- Task keywords: **CSS**, **Elasticsearch**, **OpenSearch**, **search cluster**, **索引**, **快照**, **词库**, **分片**, **shard**, **snapshot**, **dictionary**
- User describes symptoms: cluster unavailable, search latency high, shard allocation failed, snapshot restore failed, disk full

### SHOULD NOT Use This Skill When
- Task is purely billing / cost analysis / 费用 / 预算 → delegate to: `huaweicloud-billing-ops`

- Task is IAM / permission model only → delegate to: `huaweicloud-iam-ops`
- Task is about Log Tank Service (LTS) → delegate to: `huaweicloud-lts-ops`
- Task is about VPC/subnet/security group → delegate to: `huaweicloud-vpc-ops`
- Task is about monitoring/alarm rules → delegate to: `huaweicloud-ces-ops`
- Task is about Object Storage Service (OBS) for snapshot storage → delegate to: `huaweicloud-obs-ops`

### Delegation Rules

- Cluster must be in **AVAILABLE** state before snapshot operations → verify with `ShowClusterDetail` first
- Snapshot operations depend on cluster existence and OBS bucket → confirm `cluster_id` and OBS bucket access
- Dictionary updates require cluster restart to take effect → warn user about restart requirement
- For FinOps questions: use this skill's cost section; delegate cross-resource cost to billing skill
- For SecOps questions: use this skill's security section; delegate account-level IAM to `huaweicloud-iam-ops`

## Variables

| Variable | Source | Description | Example |
|----------|--------|-------------|---------|
| `{{env.HW_ACCESS_KEY_ID}}` | Environment | Huawei Cloud AK | `AKIA...` |
| `{{env.HW_SECRET_ACCESS_KEY}}` | Environment | Huawei Cloud SK | `***` (masked) |
| `{{env.HW_REGION_ID}}` | Environment | Region code | `cn-north-4` |
| `{{env.HW_PROJECT_ID}}` | Environment | Project ID | `a1b2c3d4...` |
| `{{user.cluster_id}}` | User | CSS cluster UUID | `a1b2c3d4-...` |
| `{{user.cluster_name}}` | User | CSS cluster name | `prod-es-cluster` |
| `{{user.engine_version}}` | User | ES/OS version | `7.10.2`, `2.17.1` |
| `{{user.node_type}}` | User | Node specification | `ess.spec-4u8g` |
| `{{user.node_num}}` | User | Number of nodes | `3` |
| `{{user.snapshot_name}}` | User | Snapshot name | `daily-snapshot-20260527` |
| `{{user.bucket_name}}` | User | OBS bucket for snapshots | `css-snapshots-bucket` |
| `{{user.password}}` | User | Cluster admin password | `SecurePass123!` |
| `{{output.cluster_id}}` | API Response | Created cluster ID | `.clusters[0].id` from `CreateCluster` |
| `{{output.snapshot_id}}` | API Response | Created snapshot ID | `.snapshots[0].id` from `CreateSnapshot` |
| `{{output.cluster_status}}` | API Response | Cluster status | `.status` from `ShowClusterDetail` |
| `{{output.cluster_endpoint}}` | API Response | Cluster endpoint URL | `.endpoint` from `ShowClusterDetail` |
| `{{output.snapshot_status}}` | API Response | Snapshot status | `.status` from `ListSnapshots` |

> **Security Warning:** NEVER log or expose `{{env.HW_SECRET_ACCESS_KEY}}` or any credential values.

---

## Common Operations

### 1. Cluster Lifecycle

| Operation | CLI Command (KooCLI) | Equivalent Go SDK |
|-----------|---------------------|-------------------|
| List clusters | `hcloud CSS ListClusters --cli-region="{{env.HW_REGION_ID}}"` | `cssClient.ListClusters()` |
| Show cluster detail | `hcloud CSS ShowClusterDetail --cluster_id="{{user.cluster_id}}"` | `cssClient.ShowClusterDetail()` |
| Create cluster | `hcloud CSS CreateCluster` | `cssClient.CreateCluster()` |
| Delete cluster | `hcloud CSS DeleteCluster --cluster_id="{{user.cluster_id}}"` | `cssClient.DeleteCluster()` |
| Restart cluster | `hcloud CSS RestartCluster --cluster_id="{{user.cluster_id}}"` | `cssClient.RestartCluster()` |
| Extend cluster | `hcloud CSS ExtendCluster --cluster_id="{{user.cluster_id}}"` | `cssClient.ExtendCluster()` |
| Scale out nodes | `hcloud CSS ScaleOut` | `cssClient.ScaleOut()` |
| Modify configuration | `hcloud CSS UpdateCluster` | `cssClient.UpdateCluster()` |
| Reset password | `hcloud CSS ResetPassword --cluster_id="{{user.cluster_id}}"` | `cssClient.ResetPassword()` |

### 2. Snapshot Management

| Operation | CLI Command | Equivalent Go SDK |
|-----------|-------------|-------------------|
| List snapshots | `hcloud CSS ListSnapshots --cluster_id="{{user.cluster_id}}"` | `cssClient.ListSnapshots()` |
| Create snapshot | `hcloud CSS CreateSnapshot --cluster_id="{{user.cluster_id}}"` | `cssClient.CreateSnapshot()` |
| Delete snapshot | `hcloud CSS DeleteSnapshot --cluster_id="{{user.cluster_id}}" --snapshot_id="{{user.snapshot_id}}"` | `cssClient.DeleteSnapshot()` |
| Restore snapshot | `hcloud CSS RestoreSnapshot --cluster_id="{{user.cluster_id}}"` | `cssClient.RestoreSnapshot()` |
| Set snapshot policy | `hcloud CSS SetSnapshotPolicy --cluster_id="{{user.cluster_id}}"` | `cssClient.SetSnapshotPolicy()` |

### 3. Dictionary Management

| Operation | CLI Command | Equivalent Go SDK |
|-----------|-------------|-------------------|
| List dictionaries | `hcloud CSS ListIkDicts --cluster_id="{{user.cluster_id}}"` | `cssClient.ListIkDicts()` |
| Add IK dictionary | `hcloud CSS AddIkDict --cluster_id="{{user.cluster_id}}"` | `cssClient.AddIkDict()` |
| Update IK dictionary | `hcloud CSS UpdateIkDict --cluster_id="{{user.cluster_id}}"` | `cssClient.UpdateIkDict()` |
| Delete IK dictionary | `hcloud CSS DeleteIkDict --cluster_id="{{user.cluster_id}}"` | `cssClient.DeleteIkDict()` |

### 4. Cluster Monitoring

| Operation | CLI Command | Equivalent Go SDK |
|-----------|-------------|-------------------|
| List cluster tags | `hcloud CSS ShowClusterTags --cluster_id="{{user.cluster_id}}"` | `cssClient.ShowClusterTags()` |
| Add cluster tag | `hcloud CSS AddClusterTag --cluster_id="{{user.cluster_id}}"` | `cssClient.AddClusterTag()` |
| List tasks | `hcloud CSS ListTasks --cluster_id="{{user.cluster_id}}"` | `cssClient.ListTasks()` |

### 5. Advanced Operations

| Operation | CLI Command | Equivalent Go SDK | Description |
|-----------|-------------|-------------------|-------------|
| Upgrade Cluster | `hcloud CSS UpgradeCluster` | `cssClient.UpgradeCluster()` | Upgrade ES/OS version |
| Migrate Cluster | `hcloud CSS MigrateCluster` | `cssClient.MigrateCluster()` | Cross-region migration |
| Update AZ | `hcloud CSS UpdateAz` | `cssClient.UpdateAz()` | Add/remove AZ |
| Update SG | `hcloud CSS UpdateSecurityGroup` | `cssClient.UpdateSecurityGroup()` | Update security group |
| Bind EIP | `hcloud CSS BindPublicKibana` | `cssClient.BindPublicKibana()` | Enable public Kibana |
| Unbind EIP | `hcloud CSS UnbindPublicKibana` | `cssClient.UnbindPublicKibana()` | Disable public Kibana |
| Start Pipeline | `hcloud CSS StartPipeline` | `cssClient.StartPipeline()` | Start data pipeline |
| Stop Pipeline | `hcloud CSS StopPipeline` | `cssClient.StopPipeline()` | Stop data pipeline |

---

## Execution Flows

### Operation: Create Cluster

#### Pre-flight Checks

| Check | Method | Expected | On Failure |
|-------|--------|----------|------------|
| CLI / deps | `hcloud --version` | Exit code 0 | Document CLI install |
| Credentials | Construct credential from env | Non-empty keys | HALT; user configures env |
| Region | Call **ListClusters** or equivalent | `{{user.region}}` supported | Suggest valid region |
| VPC/Subnet | Verify VPC and subnet exist | Valid network IDs | HALT; create via `huaweicloud-vpc-ops` |
| OBS Bucket | Verify OBS bucket for snapshots | Bucket accessible | Create via `huaweicloud-obs-ops` |
| Quota | Call quota API per OpenAPI | Sufficient quota | HALT; user raises quota |

#### Execution — CLI (Primary Path)

```bash
# CLI invocation
hcloud CSS CreateCluster \
  --name "{{user.cluster_name}}" \
  --datastore {
    "type": "elasticsearch",
    "version": "{{user.engine_version}}"
  } \
  --instance {
    "flavorRef": "{{user.node_type}}",
    "nics": {
      "vpcId": "{{user.vpc_id}}",
      "netId": "{{user.subnet_id}}",
      "securityGroupId": "{{user.security_group_id}}"
    },
    "volume": {
      "type": "ULTRAHIGH",
      "size": 100
    },
    "availabilityZone": "{{user.az}}"
  } \
  --instanceNum {{user.node_num}} \
  --httpsEnable true \
  --diskEncryptionEnabled true \
  --diskEncryptionKey "{{user.kms_key_id}}"
```

#### Execution — JIT Go SDK (Fallback Path)

When CLI does not support a specific operation, **JIT build a Go SDK script**:

```go
package main

import (
    "fmt"
    "os"
    
    "github.com/huaweicloud/huaweicloud-sdk-go-v3/core/config"
    "github.com/huaweicloud/huaweicloud-sdk-go-v3/core/auth/basic"
    "github.com/huaweicloud/huaweicloud-sdk-go-v3/services/css/v2"
    "github.com/huaweicloud/huaweicloud-sdk-go-v3/services/css/v2/model"
)

func main() {
    ak := os.Getenv("HW_ACCESS_KEY_ID")
    sk := os.Getenv("HW_SECRET_ACCESS_KEY")
    region := os.Getenv("HW_REGION_ID")
    
    cfg := config.DefaultHttpConfig()
    client := css.CssClientBuilder().
        WithEndpoint(fmt.Sprintf("css.%s.myhuaweicloud.com", region)).
        WithCredential(basic.NewCredentialsBuilder().
            WithAk(ak).WithSk(sk).Build()).
        WithHttpConfig(cfg).Build()
    
    request := &model.CreateClusterRequest{
        Body: &model.CreateClusterReq{
            Name: "{{user.cluster_name}}",
            Datastore: &model.CreateClusterReqDatastore{
                Type:    "elasticsearch",
                Version: "{{user.engine_version}}",
            },
            Instance: &model.CreateClusterReqInstance{
                FlavorRef: "{{user.node_type}}",
                Nics: &model.CreateClusterReqInstanceNics{
                    VpcId:           "{{user.vpc_id}}",
                    NetId:           "{{user.subnet_id}}",
                    SecurityGroupId: "{{user.security_group_id}}",
                },
                Volume: &model.CreateClusterReqInstanceVolume{
                    Type: "ULTRAHIGH",
                    Size: int32(100),
                },
                AvailabilityZone: "{{user.az}}",
            },
            InstanceNum: int32({{user.node_num}}),
            HttpsEnable: bool(true),
            DiskEncryptionEnabled: bool(true),
            DiskEncryptionKey: "{{user.kms_key_id}}",
        },
    }
    
    response, err := client.CreateCluster(request)
    if err != nil {
        panic(err)
    }
    fmt.Printf("%+v\n", response)
}
```

#### Post-execution Validation

1. Read `{{output.cluster_id}}` from the documented response path.
2. Poll **ShowClusterDetail** until terminal state or timeout.
3. On success, report `{{output.cluster_id}}` and key fields.
4. On terminal failure, go to **Failure Recovery**.

| State | Status | Action |
|-------|--------|--------|
| `AVAILABLE` | Success | Report cluster ready |
| `CREATING` | In progress | Continue polling (interval: 60s) |
| `FAILED` | Terminal failure | Go to Failure Recovery |
| `EXTENDING` | In progress | Wait for completion |

#### Failure Recovery

| Error Code | Description | Max Retries | Backoff | Agent Action | UX Feedback | Recovery Steps |
|------------|-------------|-------------|---------|--------------|-------------|----------------|
| `CSS.0001` | Invalid request parameter | 0 | — | Fix args from OpenAPI | `[ERROR] InvalidParameter: {field} - {reason}` | 1. Check required fields<br>2. Validate parameter formats<br>3. Refer to API docs |
| `CSS.0002` | Insufficient quota | 0 | — | HALT | `[ERROR] QuotaExceeded: {resource_type}` | 1. Check current usage<br>2. Request quota increase<br>3. Or delete unused resources |
| `CSS.0003` | Insufficient balance | 0 | — | HALT | `[ERROR] InsufficientBalance` | 1. Check account balance<br>2. Recharge account<br>3. Retry operation |
| `CSS.0004` | Resource already exists | 0 | — | Ask reuse vs new name | `[ERROR] AlreadyExists: {resource_name}` | 1. Use different name<br>2. Or use existing resource<br>3. Check with List operation |
| `CSS.0005` | Rate limit exceeded | 3 | exponential | Back off | `⚠️ Rate limited. Retrying in {backoff}s...` | 1. Wait for backoff<br>2. Reduce request rate<br>3. Consider batch operations |
| `CSS.0006` | Internal server error | 3 | 2s, 4s, 8s | Retry then HALT | `[ERROR] InternalError: {request_id}` | 1. Retry with backoff<br>2. If persists, escalate<br>3. Include request_id |
| `CSS.0010` | Invalid VPC/Subnet | 0 | — | HALT | `[ERROR] InvalidNetwork: {vpc_id}` | 1. Verify VPC exists<br>2. Check subnet in VPC<br>3. Use huaweicloud-vpc-ops |
| `CSS.0011` | Invalid flavor spec | 0 | — | HALT | `[ERROR] InvalidFlavor: {flavor_ref}` | 1. List available flavors<br>2. Check region availability<br>3. Select valid flavor |
| `CSS.0012` | Invalid engine version | 0 | — | HALT | `[ERROR] InvalidVersion: {version}` | 1. Check supported versions<br>2. Use: 7.6.2, 7.10.2, 1.3.6, 2.17.1 |
| `CSS.0013` | KMS key not found | 0 | — | HALT | `[ERROR] KMSKeyNotFound` | 1. Verify KMS key ID<br>2. Check key is enabled<br>3. Verify IAM permissions |
| `CSS.0014` | OBS bucket not found | 0 | — | HALT | `[ERROR] BucketNotFound: {bucket}` | 1. Create OBS bucket<br>2. Check bucket permissions<br>3. Use huaweicloud-obs-ops |
| `CSS.0015` | Insufficient permissions | 0 | — | HALT | `[ERROR] AccessDenied` | 1. Check IAM policies<br>2. Verify role permissions<br>3. Contact admin |
| `CSS.0020` | Cluster not found | 0 | — | HALT | `[ERROR] ClusterNotFound: {cluster_id}` | 1. Verify cluster ID<br>2. List clusters to confirm<br>3. Check region |
| `CSS.0021` | Cluster not available | 0 | — | Wait | `⏳ Cluster busy: {status}` | 1. Wait for operation complete<br>2. Poll until AVAILABLE<br>3. Retry operation |
| `CSS.0022` | Operation in progress | 0 | — | Wait | `⏳ Operation in progress` | 1. Wait for current op<br>2. Check task status<br>3. Retry after completion |
| `CSS.0030` | Snapshot not found | 0 | — | HALT | `[ERROR] SnapshotNotFound` | 1. List available snapshots<br>2. Verify snapshot ID<br>3. Check cluster |
| `CSS.0031` | Snapshot creation failed | 1 | 5s | Retry once | `[ERROR] SnapshotFailed: {reason}` | 1. Check OBS permissions<br>2. Verify cluster health<br>3. Retry with delay |
| `CSS.0032` | Snapshot restore failed | 0 | — | HALT | `[ERROR] RestoreFailed` | 1. Check snapshot integrity<br>2. Verify version compatibility<br>3. Try different snapshot |
| `CSS.0040` | Node operation failed | 2 | 10s | Retry | `[ERROR] NodeOperationFailed` | 1. Check node status<br>2. Verify capacity<br>3. Retry or escalate |
| `CSS.0041` | Scale out failed | 1 | 30s | Retry | `[ERROR] ScaleOutFailed` | 1. Check quota<br>2. Verify subnet IPs<br>3. Retry or contact support |

---

### Operation: Delete Cluster

#### Pre-flight (Safety Gate)

- **MUST** obtain explicit confirmation: irreversible delete of `{{user.cluster_name}}` (`{{user.cluster_id}}`).
- **MUST NOT** proceed without clear user assent.
- **MUST** remind user to create snapshot before delete if data retention needed.
- **MUST** verify cluster is not in `CREATING` or `EXTENDING` state.

#### Execution

Call delete API per OpenAPI. Capture response indicating success or error per verified output shape.

```bash
hcloud CSS DeleteCluster \
  --cluster_id "{{user.cluster_id}}" \
  --backupOBSPath "obs://{{user.bucket_name}}/backups/"
```

#### Post-execution Validation

Poll **ShowClusterDetail** until **404** or **NotFound** status — per API semantics — within **max wait (300s)**.

---

### Operation: Create Snapshot

#### Pre-flight Checks

| Check | Method | Expected | On Failure |
|-------|--------|----------|------------|
| Cluster state | `ShowClusterDetail` | `AVAILABLE` | HALT; wait for cluster ready |
| OBS bucket | Verify bucket exists | Accessible | Create via `huaweicloud-obs-ops` |
| Snapshot name | Check uniqueness | Not exists | Suggest alternative name |

#### Execution

```bash
hcloud CSS CreateSnapshot \
  --cluster_id "{{user.cluster_id}}" \
  --name "{{user.snapshot_name}}" \
  --description "Manual snapshot created on $(date)" \
  --indices "*,-.kibana*" \
  --backupOBSPath "obs://{{user.bucket_name}}/snapshots/"
```

#### Post-execution Validation

1. Poll **ListSnapshots** until snapshot status is `COMPLETED` or `FAILED`.
2. Report `{{output.snapshot_id}}` and completion status.

| State | Status |
|-------|--------|
| `COMPLETED` | Success |
| `FAILED` | Terminal failure |
| `CREATING` | In progress (continue polling) |

---

### Operation: Restore Snapshot

#### Pre-flight (Safety Gate)

- **MUST** warn user: restore overwrites current data; suggest pre-restore snapshot.
- **MUST** confirm: target cluster, snapshot source, expected data loss window.
- **MUST** verify target cluster is in `AVAILABLE` state.

#### Execution

```bash
hcloud CSS RestoreSnapshot \
  --cluster_id "{{user.cluster_id}}" \
  --snapshot_id "{{user.snapshot_id}}" \
  --targetCluster "{{user.target_cluster_id}}" \
  --indices "index-*" \
  --renamePattern "index-(.+)" \
  --renameReplacement "restored-index-$1"
```

---

## Cost Optimization (FinOps)

### Cost Calculation Formula

```
Monthly Cost = Compute Cost + Storage Cost + Snapshot Cost + Data Transfer Cost

Compute Cost = (Data Nodes × Price/hour + Master Nodes × Price/hour + Client Nodes × Price/hour) × 730 hours

Storage Cost = Total Storage (GB) × Storage Price/GB/month

Snapshot Cost = Snapshot Size (GB) × OBS Price/GB/month + API Call Costs

Data Transfer Cost = Cross-AZ Transfer (GB) × Transfer Price + Cross-Region Transfer (GB) × Transfer Price
```

### Cost Estimation Example

```yaml
cluster_config:
  data_nodes:
    count: 3
    flavor: ess.spec-4u8g        # 4vCPU 8GB
    price_per_hour: ¥1.50        # Example price
  master_nodes:
    count: 3
    flavor: ess.spec-2u4g        # 2vCPU 4GB
    price_per_hour: ¥0.80
  storage:
    per_node: 100GB
    type: ULTRAHIGH              # ¥0.80/GB/month
  snapshots:
    size: 50GB
    retention: 7 days
    obs_price: ¥0.15/GB/month

cost_breakdown:
  compute: (3 × ¥1.50 + 3 × ¥0.80) × 730 = ¥5,037/month
  storage: 300GB × ¥0.80 = ¥240/month
  snapshots: 50GB × 7 × ¥0.15 = ¥52.50/month
  total: ~¥5,330/month
```

### Reserved Capacity Decision Tree

```
Current Spending > ¥5,000/month AND Stable workload?
├─ Yes → Consider 1-year reserved (20% savings)
│   └─ Commitment > 2 years AND Very stable?
│       └─ Yes → 3-year reserved (35% savings)
└─ No → On-demand or Savings Plans

Multiple clusters with variable load?
└─ Yes → Flexible Savings Plans (15-25% savings)
```

### Cost Optimization Strategies

1. **Instance Sizing**: Use `ListFlavors()` (CLI: `hcloud CSS ListFlavors`) to compare available instance specifications before provisioning. Match workload to node type tiers.
2. **Storage Tiering**: Start with `ULTRAHIGH` SSD and scale as needed. Use `ShowClusterDetail()` to monitor storage usage.
3. **Cold Data Nodes**: For infrequently accessed data, use `ess-cold` nodes with lower cost storage.
4. **Snapshot Retention**: Keep snapshots per compliance. Shorter retention reduces OBS storage costs. Use `SetSnapshotPolicy()` to adjust `keep_days`.
5. **Idle Cluster Detection**: Query clusters with no recent queries or low CPU usage via CES metrics.
6. **Client Node Optimization**: Use client nodes (`ess-client`) to offload query coordination from data nodes.
7. **Index Lifecycle Management (ILM)**: Implement hot-warm-cold architecture for cost-effective data tiering.
8. **Cost Tagging**: Use tags for cost attribution: `Project`, `Environment`, `CostCenter`, `Owner`.
9. **Off-Hours Scaling**: For dev/test, consider scheduled scale-down during off-hours.

### Cost Anomaly Detection

| Anomaly | Detection | Response |
|---------|-----------|----------|
| Cost spike > 50% vs last month | Budget alert | Investigation required |
| Unexpected data transfer | Transfer cost alert | Check cross-region access |
| Snapshot cost spike | OBS cost alert | Review retention policy |
| Idle cluster cost | Low utilization alert | Consider deletion |

### Budget Alerting Rules

```yaml
budget_alerts:
  - threshold: 50%
    action: notify
    recipients: [team-lead]
  - threshold: 80%
    action: warning
    recipients: [team-lead, finance]
  - threshold: 100%
    action: block_new_resources
    recipients: [team-lead, finance, manager]
    escalation: immediate
```

---

## Security Best Practices (SecOps)

### Data Protection

1. **HTTPS Enforcement**: Always set `httpsEnable: true` in cluster creation.
2. **TLS Version**: Minimum TLS 1.2, recommend TLS 1.3.
3. **Disk Encryption**: Enable `diskEncryptionEnabled` with valid KMS key for sensitive data.
4. **Encryption at Rest**: AES-256 via Huawei Cloud KMS.
5. **Encryption in Transit**: TLS 1.2+ with certificate validation.
6. **Snapshot Encryption**: Encrypt snapshots in OBS with SSE-KMS.

### Network Security

1. **VPC Isolation**: Place clusters in private subnets with security group restrictions.
2. **Security Group Rules**:
   ```yaml
   ingress:
     - port: 9200
       source: vpc_cidr_only
       protocol: tcp
     - port: 9300
       source: vpc_cidr_only
       protocol: tcp
   egress:
     - destination: obs_endpoint
       protocol: https
   ```
3. **No Public IP**: Clusters should not have public IP addresses.
4. **Bastion Access**: Use bastion hosts for administrative access.

### IAM and Access Control

1. **IAM Least Privilege**: Grant minimum permissions required for CSS operations.
2. **Role-Based Access**: Use predefined roles (Admin, Operator, ReadOnly).
3. **Index-Level Security**: Implement document and field-level security in Elasticsearch.
4. **API Key Rotation**: Rotate API keys every 90 days.
5. **Multi-Factor Authentication**: Enable MFA for console access.

### Audit and Compliance

1. **CTS Logging**: Enable Cloud Trace Service for all API calls.
2. **Audit Log Format**:
   ```json
   {
     "timestamp": "2026-05-27T10:00:00Z",
     "user": "user@domain.com",
     "action": "css:cluster:create",
     "resource": "cluster-xxxx",
     "result": "success",
     "source_ip": "10.0.0.1",
     "request_id": "req-xxxx"
   }
   ```
3. **Log Retention**: 1 year minimum for compliance.
4. **Compliance Certifications**: ISO 27001, SOC 2, PCI DSS (if applicable).

### Security Monitoring

1. **Failed Login Detection**: Alert on 5+ failed logins in 5 minutes.
2. **Privilege Escalation Alert**: Alert on IAM policy changes.
3. **Unusual Access Pattern**: Alert on off-hours access from new IPs.
4. **Data Exfiltration Detection**: Alert on unusual query volume.

### Security Incident Response SLA

| Severity | Response Time | Resolution Target |
|----------|---------------|-------------------|
| Critical (Data breach) | 15 minutes | 4 hours |
| High (Unauthorized access) | 30 minutes | 8 hours |
| Medium (Policy violation) | 2 hours | 24 hours |
| Low (Configuration drift) | 24 hours | 72 hours |

### Compliance Mapping

| Requirement | Control | Verification |
|-------------|---------|--------------|
| ISO 27001 | A.10.1.2 | Network security groups |
| SOC 2 CC6.1 | Logical access controls | IAM policies |
| PCI DSS 3.4 | Data encryption | KMS encryption enabled |
| GDPR Art. 32 | Data protection | Encryption at rest/transit |

---

## AIOps Anomaly Patterns

| Pattern | Detection Logic | Severity | Agent Action |
|---------|-----------------|----------|--------------|
| Cluster Health Red | `status == "red"` AND `unassigned_shards > 0` | Critical | Immediate alert; suggest emergency reallocation |
| Cluster Health Yellow | `status == "yellow"` for > 10m | Warning | Alert; check replica allocation |
| High Query Latency | `search_latency_p99 > 500ms` for 5m | Warning | Alert; suggest scaling or query optimization |
| Extreme Query Latency | `search_latency_p99 > 1000ms` for 3m | Critical | Immediate alert; check cluster resources |
| Disk Usage Warning | `disk_usage > 75%` | Info | Monitor; plan storage extension |
| Disk Usage Critical | `disk_usage > 85%` | Critical | Alert; immediate scaling required |
| Disk Full | `disk_usage > 95%` | Critical | HALT; cluster rejecting writes |
| JVM Heap Warning | `jvm_heap_used > 70%` | Info | Monitor; observe GC patterns |
| JVM Heap Pressure | `jvm_heap_used > 85%` for 5m | Warning | Alert; suggest heap tuning or scaling |
| JVM OOM Risk | `jvm_heap_used > 95%` | Critical | Critical alert; scale immediately |
| CPU Saturation | `cpu_usage > 80%` for 15m | Warning | Alert; consider CPU scaling |
| CPU Critical | `cpu_usage > 95%` for 5m | Critical | Immediate alert; scale or optimize |
| Slow Query Spike | `slow_queries_per_min > 10` | Warning | Alert; analyze slow query log |
| Indexing Backlog | `indexing_queue > 1000` | Warning | Alert; check bulk queue settings |
| Snapshot Failure | `snapshot.status == FAILED` | Warning | Alert; check OBS permissions and retry |
| Repeated Snapshot Fail | 3+ failures in 24h | Critical | Escalate; check infrastructure |
| Node Failure | `nodes < expected_count` | Critical | Immediate alert; auto-replacement triggered |
| Network Partition | `nodes_seen < cluster_size` | Critical | Critical; split-brain risk |
| Shard Imbalance | `shard_cv > 0.3` | Warning | Alert; suggest rebalancing |
| Hot Shard Detected | `shard_query_ratio > 3x avg` | Warning | Alert; suggest shard splitting |
| Storage Growth Anomaly | `growth_rate > 2x baseline` | Warning | Alert; check index lifecycle |
| Query Pattern Anomaly | `query_types` changed > 50% | Info | Log; potential usage change |
| Memory Leak Pattern | `jvm_heap` increasing steadily | Warning | Alert; potential memory leak |
| GC Pause Spike | `gc_pause > 1s` | Warning | Alert; JVM tuning needed |

### Predictive Maintenance

| Predictor | Early Warning | Action Window |
|-----------|---------------|---------------|
| Disk growth trend | Days to full < 30 days | Plan extension |
| Query latency trend | p99 increasing weekly | Capacity planning |
| JVM heap trend | Growth rate > 10%/week | Heap adjustment |
| Index growth | New indices > 100/day | Review retention |

### Auto-Scaling Triggers

| Metric Condition | Scale Action | Cooldown |
|------------------|--------------|----------|
| `cpu_avg > 70%` for 10m | Add data node | 10min |
| `disk_usage > 80%` | Extend storage | 5min |
| `search_latency_p99 > 200ms` for 15m | Add client node | 10min |
| `indexing_queue > 2000` for 5m | Add data node | 10min |
| `jvm_heap > 80%` for 20m | Scale up node spec | 15min |
| `cpu_avg < 30%` for 1h (off-peak) | Consider scale-down | 1h |

---

## Prerequisites

> Full installation scripts (KooCLI + Go runtime + Credentials): see [references/common-prerequisites.md](../references/common-prerequisites.md)

## Quality Gate (GCL)

This skill is **GCL-required** (per `AGENTS.md` §8). Every CSS (Elasticsearch / OpenSearch)
mutating operation — cluster create / delete / scale, snapshot create / restore, ES REST ops
(DELETE/PUT index, forcemerge, reindex, _delete_by_query, _update_by_query, _cluster/settings)
— runs through the **Generator-Critic-Loop** before its result is returned. Read-only are
GCL-**exempt**.

| Field | Value |
|-------|-------|
| Rubric version | v1 (Phase 2, 2026-06-04) |
| `max_iter` | **2** |
| Rubric instance | [`references/rubric.md`](references/rubric.md) |
| Prompt templates | [`references/prompt-templates.md`](references/prompt-templates.md) |
| Trace path | `./audit-results/gcl-trace-YYYYMMDD-HHMMSS.json` |
| Independence | Generator and Critic in **isolated** sub-agent / session contexts |

### Five-Dimension Rubric (summary)

> The five-dimension rubric (Correctness / Safety / Idempotency / Traceability / Spec Compliance),
> default thresholds, termination contract (PASS / MAX_ITER / SAFETY_FAIL), and trace-persistence
> rules are defined in [`docs/gcl-spec.md`](../../docs/gcl-spec.md) and the repo root
> [`AGENTS.md`](../AGENTS.md) §3, §5, §7, §8. This skill overrides only the items below.

| # | Dimension | Threshold | Notes |
|---|-----------|-----------|-------|
| 1 | Correctness | ≥ 0.5 (1.0 for `delete-cluster` / `restore-snapshot` / index delete) | `ShowClusterDetail` / `ShowSnapshot` / ES HEAD post-state |
| 2 | Safety | **= 1** (any S-rule hit → ABORT) | S1–S16 in rubric §2 |
| 3 | Idempotency | ≥ 0.5 | Pre-check before create |
| 4 | Traceability | ≥ 0.5 | `password` / `security_admin` MUST be `<masked>` |
| 5 | Spec Compliance | ≥ 0.5 | ES version / node flavor / index name regex |

### Per-Operation Safety Anchors (binding)

- **S1 / S2 / S3 / S4** — `delete-cluster` confirmation / state check / snapshot pre-check / prePaid refund
- **S5 / S6** — `restore-snapshot` cross-cluster / same-active-source
- **S7** — ES `DELETE /<index>` with wildcard `*` or `*,-.kibana*`
- **S8** — ES `_delete_by_query` with `query: {"match_all": {}}` on non-test index
- **S9** — ES `_update_by_query` with `query: {"match_all": {}}`
- **S10** — ES `_forcemerge` with `max_num_segments: 1` on prod index
- **S11** — ES `_close` / `_delete` on `.kibana*` / `.security*` / `.tasks`
- **S12** — ES `PUT /_cluster/settings` `cluster.routing.allocation.enable: none` without maintenance window
- **S15** — ES `_reindex` on large index with `wait_for_completion: true`
- **S16** — `update-snapshot-policy` `retention.days < 7`

### See also

- [`references/rubric.md`](references/rubric.md) — full rubric, S1–S16 rules, per-op thresholds
- [`references/prompt-templates.md`](references/prompt-templates.md) — Generator / Critic / Orchestrator skeletons
- Repository root [`AGENTS.md`](../AGENTS.md) §3, §5, §7, §8 — GCL specification

## Reference Directory

- [Core Concepts](references/core-concepts.md) — CSS architecture, engine types, node types
- [API & SDK Usage](references/api-sdk-usage.md) — Operation map, request/response snippets
- [CLI Usage](references/cli-usage.md) — CLI command mapping
- [Troubleshooting](references/troubleshooting.md) — Error codes, diagnostics, recovery
- [Monitoring & Alerts](references/monitoring.md) — CES metrics, dashboards
- [Integration](references/integration.md) — Go bootstrap, cross-skill delegation
- [Cost Optimization](references/advanced/cost-optimization.md) — FinOps patterns
- [Security Best Practices](references/advanced/security-best-practices.md) — SecOps guidance
- [AIOps Best Practices](references/advanced/aiops-best-practices.md) — Anomaly detection, self-healing
- [Well-Architected Assessment](references/well-architected-assessment.md)
- [GCL Rubric](references/rubric.md) — Adversarial quality gate (v1, 5-dim, S1–S16 CSS/ES-specific Safety rules)
- [GCL Prompt Templates](references/prompt-templates.md) — Generator / Critic / Orchestrator skeletons

## Well-Architected + Three-Pillar Assessment

This skill's operations are evaluated against Huawei Cloud's Well-Architected Framework:
- [Security Assessment](references/well-architected-assessment.md#21-安全支柱-security)
- [Stability Assessment](references/well-architected-assessment.md#22-稳定支柱-stability)
- [Cost Assessment](references/well-architected-assessment.md#23-成本支柱-cost)
- [Efficiency Assessment](references/well-architected-assessment.md#24-效率支柱-efficiency)
- [Performance Assessment](references/well-architected-assessment.md#25-性能支柱-performance)
