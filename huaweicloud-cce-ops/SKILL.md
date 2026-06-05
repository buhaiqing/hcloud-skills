---
name: huaweicloud-cce-ops
description: >-
  Use when the user needs to deploy, configure, troubleshoot, or monitor Huawei
  Cloud Cloud Container Engine (CCE / 云容器引擎) — cluster, node, node pool,
  and addon lifecycle. User mentions CCE, 云容器引擎, K8s集群, 节点管理,
  插件安装, or describes scenarios (e.g., cluster creation failure, node
  NotReady, addon installation error, node pool autoscaling issue) even without
  naming the product directly. Not for billing, IAM, or related products that
  have their own ops skills.
license: MIT
compatibility: >-
  Official Huawei Cloud CLI (`hcloud` / `openstack`), Go 1.21+ runtime
  (for JIT SDK fallback via huaweicloud-sdk-go-v3), valid API credentials,
  network access to Huawei Cloud endpoints.
metadata:
  author: huaweicloud
  version: "1.0.0"
  last_updated: "2026-05-20"
  runtime: Harness AI Agent, Claude Code, Cursor, or compatible Agent runtimes
  go_version_minimum: "1.21"
  go_version_jit: "1.24+"
  api_profile: "CCE API v3.0 - https://support.huaweicloud.com/api-cce/cce_02_0001.html"
  cli_applicability: "dual-path"
  cli_support_evidence: >-
    CCE product supported by hcloud CLI. Use `hcloud cce --help` to verify
    available commands for cluster, node, nodepool, and addon operations.
  environment:
    - HW_ACCESS_KEY_ID
    - HW_SECRET_ACCESS_KEY
    - HW_REGION_ID
    - HW_PROJECT_ID
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
        change: "GCL Phase 2 rollout: added references/rubric.md (v1, 5-dim, S1–S17 CCE/Kubernetes-specific Safety rules, including drain-before-delete, PDB guard, StatefulSet scale, privileged manifest, master cordon) and references/prompt-templates.md (Generator + Critic + Orchestrator). SKILL.md gains 'Quality Gate (GCL)' chapter."
      - version: "1.0.0"
        date: "2026-05-20"
        change: "Initial skill release."
---

> This skill follows the [Agent Skill Open Specification](https://agentskills.io/specification).

# Huawei Cloud Cloud Container Engine (CCE) Operations Skill

## Overview

Huawei Cloud Cloud Container Engine (CCE / 云容器引擎) is a managed Kubernetes service for deploying, managing, and scaling containerized applications. This skill is an **operational runbook** for agents: explicit scope, credential rules, pre-flight checks, **dual-path execution** (official **SDK/API** and **`hcloud` CLI**), response validation, and failure recovery.

### CLI applicability (repository policy)

- **`cli_applicability: dual-path`:** Official CLI supports CCE product. You **MUST** ship **`references/cli-usage.md`** and, in **each** execution flow, document **both** the SDK step **and** the CLI step.

## Five Core Standards (Quality Gates)

| # | Standard | How This Skill Fulfills It |
|---|----------|---------------------------|
| 1 | **Clear Boundaries** | SHOULD/SHOULD NOT Use conditions with precise triggers and delegation rules |
| 2 | **Structured I/O** | Placeholder conventions with type and source documented; `{{user.*}}` for interactive, `{{env.*}}` for runtime |
| 3 | **Explicit Actionable Steps** | Every operation: Pre-flight → Execute (CLI + SDK) → Validate → Recover |
| 4 | **Complete Failure Strategies** | Error taxonomy ≥ 15 codes; HALT vs retry per error type; async polling for cluster/node lifecycle |
| 5 | **Absolute Single Responsibility** | One product (CCE); cross-product networking/storage delegation to VPC/ECS/EVS skills |

### Three-Pillar Ops Integration (FinOps + SecOps + AIOps)

| Pillar | Skill Integration | Reference |
|--------|-------------------|-----------|
| **FinOps** | Node billing optimization, idle cluster detection, spot vs subscription guidance | `references/well-architected-assessment.md` §3 (Cost Pillar) |
| **SecOps** | IAM minimum permissions for cluster/node ops, network isolation, RBAC enforcement | `references/well-architected-assessment.md` §4 (SecOps section) |
| **AIOps** | ≥ 6 anomaly patterns (Node NotReady, OOM, pod crash-loop, addon failure), cross-skill diagnosis, alarm storm suppression | `references/knowledge-base.md` and `references/monitoring.md` |

### Well-Architected Framework Integration (卓越架构)

| Pillar | Skill Integration | Reference |
|--------|-------------------|-----------|
| **安全 (Security)** | IAM permissions for cluster/nodes, credential isolation, RBAC, security group patterns | `references/well-architected-assessment.md` §1 |
| **稳定 (Stability)** | Multi-AZ clusters, node pool redundancy, backup/restore, DR runbook | `references/well-architected-assessment.md` §2 |
| **成本 (Cost)** | Node billing models (pay-per-use/substitution/spot), idle detection, right-sizing | `references/well-architected-assessment.md` §3 |
| **效率 (Efficiency)** | Batch node/nodepool operations, CI/CD integration, addon automation | `references/well-architected-assessment.md` §4 |
| **性能 (Performance)** | Auto-scaling thresholds, node resource baselines, API server latency | `references/well-architected-assessment.md` §5 |

## Trigger & Scope (Agent-Readable)

### SHOULD Use This Skill When

- User mentions "Huawei Cloud CCE", "Cloud Container Engine", "云容器引擎", "K8s集群", "Kubernetes集群"
- Task involves cluster lifecycle: create, list, describe, delete, upgrade, resize
- Task involves node management: add, remove, list, describe, update nodes
- Task involves node pool management: create, list, update, delete, scale node pools
- Task involves addon management: install, list, update, delete addons (coredns, everest, metrics-server)
- Task keywords: CCE集群, 节点池, 插件, 容器引擎, K8s节点, 容器网络
- User asks to deploy, configure, troubleshoot, or monitor CCE resources via API, SDK, CLI, or automation

### SHOULD NOT Use This Skill When

- Task is purely billing / account management → delegate to: `huaweicloud-billing-ops`
- Task is IAM / permission model only → delegate to: `huaweicloud-iam-ops`
- Task is creating/deleting the **underlying compute** itself (e.g., ECS instance outside CCE) → delegate to: `huaweicloud-ecs-ops`
- Task is VPC/subnet/security group configuration → delegate to: `huaweicloud-vpc-ops`
- Task is ELB/LoadBalancer configuration → delegate to: `huaweicloud-elb-ops`
- Task is container image management → delegate to `huaweicloud-swr-ops` (when present)

### Delegation Rules

- Cluster creation requires VPC, subnet, and security group — verify via `huaweicloud-vpc-ops` before CCE create.
- Node specification depends on ECS flavor — reference `huaweicloud-ecs-ops` for flavor details.
- Persistent storage in CCE nodes requires EVS — delegate volume questions to `huaweicloud-evs-ops`.
- CCE monitoring and alarms use CES — delegate metric/alarm questions to `huaweicloud-ces-ops`.
- CCE log collection uses LTS — delegate log stream questions to `huaweicloud-lts-ops`.
- Multi-product requests: handle each product with its skill; do not merge unrelated APIs.

## Variable Convention (Agent-Readable)

| Placeholder | Meaning | Agent Action |
|-------------|---------|--------------|
| `{{env.HW_ACCESS_KEY_ID}}` | From runtime environment | NEVER ask the user; fail if unset |
| `{{env.HW_SECRET_ACCESS_KEY}}` | From runtime environment | NEVER ask the user; fail if unset |
| `{{env.HW_REGION_ID}}` | From runtime environment | Use documented default only if skill explicitly allows |
| `{{env.HW_PROJECT_ID}}` | From runtime environment | Use documented default only if skill explicitly allows |
| `{{user.region}}` | User-supplied region | Ask once; reuse |
| `{{user.cluster_name}}` | User-supplied cluster name | Ask once; reuse |
| `{{user.cluster_id}}` | User-supplied cluster ID | Parse from `{{output.cluster_id}}` after creation |
| `{{user.vpc_id}}` | User-supplied VPC ID for cluster networking | Ask if not specified |
| `{{user.subnet_id}}` | User-supplied subnet ID for cluster nodes | Ask if not specified |
| `{{user.node_name}}` | User-supplied node name | Ask once; reuse |
| `{{user.node_id}}` | User-supplied node ID | Parse from `{{output.node_id}}` after creation |
| `{{user.nodepool_name}}` | User-supplied node pool name | Ask once; reuse |
| `{{user.nodepool_id}}` | User-supplied node pool ID | Parse from `{{output.nodepool_id}}` after creation |
| `{{user.addon_name}}` | User-supplied addon name (e.g., coredns) | Ask once; reuse |
| `{{user.addon_version}}` | User-supplied addon version | Ask if not specified |
| `{{user.flavor}}` | User-supplied node flavor (e.g., s6.large.2) | Ask once; reuse |
| `{{output.cluster_id}}` | From cluster create response | Parse per OpenAPI: `$.metadata.uid` |
| `{{output.node_id}}` | From node create response | Parse per OpenAPI: `$.metadata.uid` |
| `{{output.nodepool_id}}` | From nodepool create response | Parse per OpenAPI: `$.metadata.uid` |
| `{{output.addon_id}}` | From addon create response | Parse per OpenAPI: `$.metadata.uid` |

> **`{{env.*}}` MUST NOT** be collected from the user. **`{{user.*}}`** MUST be collected interactively when missing.

> **Security Warning (Credential Masking — MANDATORY):** **NEVER** log, print, or expose `HW_SECRET_ACCESS_KEY` or any credential field value.

## API and Response Conventions

- **OpenAPI is canonical** for path, query, body fields, enums, and response shapes.
- **Errors:** Map SDK/HTTP errors to `error_code` / `error_msg` fields per spec.
- **Timestamps:** ISO 8601; cluster/node creation is async — poll until terminal status.
- **Idempotency:** Cluster names must be unique per project; duplicate names return `CCE.0002`.
- **Async operations:** Cluster/node creation return a job ID — poll `ShowCluster`/`ShowNode` with 5s interval, 300s max wait until `status.phase` = `Available` or terminal failure.

## Quick Start

### What This Skill Does
Manages Huawei Cloud CCE (Cloud Container Engine / 云容器引擎) cluster, node, node pool, and addon lifecycle operations.

### Prerequisites
- [ ] Huawei Cloud CLI installed (or Go runtime for JIT fallback)
- [ ] Credentials configured: `HW_ACCESS_KEY_ID`, `HW_SECRET_ACCESS_KEY`
- [ ] Region and Project ID set: `HW_REGION_ID`, `HW_PROJECT_ID`
- [ ] VPC and subnet available for cluster networking

### Verify Setup
```bash
hcloud cce list-clusters --region {{env.HW_REGION_ID}}
```

### Your First Command
```bash
# List all CCE clusters in region
hcloud cce list-clusters --region {{env.HW_REGION_ID}}
```

### Next Steps
- [Core Concepts](references/core-concepts.md) — Understand CCE architecture and cluster types
- [Execution Flows](#execution-flows) — Cluster, node, nodepool, addon operations
- [Troubleshooting](references/troubleshooting.md) — Fix common CCE issues

## Capabilities at a Glance

| Operation | Description | Complexity | Risk Level |
|-----------|-------------|------------|------------|
| CreateCluster | Create K8s cluster with VPC/network config | High | Low |
| ListClusters | List clusters with filters | Low | None |
| DescribeCluster | View cluster details and status | Low | None |
| DeleteCluster | Remove a cluster | Medium | **High** — irreversible |
| CreateNode | Add a node to an existing cluster | Medium | Low |
| ListNode | List nodes in a cluster | Low | None |
| DeleteNode | Remove a node from cluster | Medium | **High** — data loss risk |
| CreateNodePool | Create auto-scaling node pool | High | Low |
| ListNodePool | List node pools in a cluster | Low | None |
| UpdateNodePool | Scale or resize a node pool | Medium | Low |
| DeleteNodePool | Remove a node pool | Medium | **High** — data loss risk |
| InstallAddon | Install a CCE addon (coredns, everest, etc.) | Medium | Low |
| ListAddons | List installed addons per cluster | Low | None |

## Execution Flows

### Operation: Create Cluster

#### Pre-flight Checks

| Check | Method | Expected | On Failure |
|-------|--------|----------|------------|
| CLI / deps | `hcloud --version` | Exit code 0 | Document CLI install |
| Credentials | Env var existence check | Non-empty AK/SK | HALT; user configures env |
| VPC | Verify VPC exists via `hcloud vpc describe-vpc` | VPC ACTIVE | HALT; create VPC via `huaweicloud-vpc-ops` |
| Subnet | Verify subnet exists in target VPC | Subnet ACTIVE | HALT; create subnet via `huaweicloud-vpc-ops` |
| Security Group | Verify security group exists | SG ACTIVE | HALT; create SG via `huaweicloud-vpc-ops` |
| Quota | Call quota API | Sufficient CCE cluster quota | HALT; raise quota request |

#### Execution — CLI (Primary Path)

```bash
hcloud cce create-cluster \
  --region "{{user.region}}" \
  --name "{{user.cluster_name}}" \
  --flavor "{{user.cluster_flavor:cce.s1.small}}" \
  --version "{{user.k8s_version:1.28}}" \
  --vpc-id "{{user.vpc_id}}" \
  --subnet-id "{{user.subnet_id}}" \
  --security-group-id "{{user.security_group_id}}" \
  --container-network-mode "{{user.network_mode:vpc}}" \
  --billing-mode "{{user.billing_mode:0}}" \
  --enable-authentication true
```

#### Execution — JIT Go SDK (Fallback Path)

```go
package main

import (
	"fmt"
	"os"

	"github.com/huaweicloud/huaweicloud-sdk-go-v3/core/config"
	"github.com/huaweicloud/huaweicloud-sdk-go-v3/core/auth/basic"
	"github.com/huaweicloud/huaweicloud-sdk-go-v3/services/cce/v3"
	"github.com/huaweicloud/huaweicloud-sdk-go-v3/services/cce/v3/model"
)

func main() {
	ak := os.Getenv("HW_ACCESS_KEY_ID")
	sk := os.Getenv("HW_SECRET_ACCESS_KEY")
	region := os.Getenv("HW_REGION_ID")

	cfg := config.DefaultHttpConfig()
	client := v3.NewCceClient(
		v3.CceClientBuilder().
			WithEndpoint(fmt.Sprintf("cce.%s.myhuaweicloud.com", region)).
			WithCredential(basic.NewCredentialsBuilder().
				WithAk(ak).WithSk(sk).Build()).
			WithHttpConfig(cfg).Build())

	network := model.Clusterspec{
		Network: &model.Network{
			Vpc:           os.Getenv("VPC_ID"),
			Subnet:        os.Getenv("SUBNET_ID"),
			SecurityGroup: os.Getenv("SECURITY_GROUP_ID"),
		},
		ContainerNetwork: &model.ContainerNetwork{
			Mode: "vpc",
		},
	}

	request := &model.CreateClusterRequest{
		Body: &model.Cluster{
			Metadata: &model.ClusterMetadata{
				Name: os.Getenv("CLUSTER_NAME"),
			},
			Spec:        &network,
			BillingMode: 0,
		},
	}

	response, err := client.CreateCluster(request)
	if err != nil {
		fmt.Fprintf(os.Stderr, "CreateCluster failed: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("Cluster Uid: %s\n", response.Metadata.Uid)
}
```

#### Post-execution Validation (Async Polling)

Cluster creation is **asynchronous**. Poll until terminal state:

1. Read `{{output.cluster_id}}` from response path `$.metadata.uid`.
2. Poll **ShowCluster** every 5s (max 300s) until `$.status.phase` = `Available`.
3. Terminal failure states: `Error`, `Deleting`, `ShuttingDown`.
4. On success, report `{{output.cluster_id}}`, version, and VPC details.

#### Failure Recovery

| Error | Max retries | Backoff | Agent Action | UX Feedback |
|-------|-------------|---------|--------------|-------------|
| `CCE.0002` ClusterAlreadyExists | 0 | — | Ask reuse vs new name | `[ERROR] Cluster already exists. Use different name.` |
| `CCE.0003` InvalidParameter | 0–1 | — | Fix args from OpenAPI | `[ERROR] Invalid parameter: Check VPC/subnet IDs.` |
| `CCE.0016` ProjectNotAuthorized | 0 | — | Verify IAM perms | `[ERROR] Unauthorized. Check IAM permissions.` |
| `CCE.0020` QuotaExceeded | 0 | — | HALT | `[ERROR] CCE cluster quota exceeded.` |
| `CCE.0030` ResourceNotFound | 0 | — | Fix VPC/subnet ID | `[ERROR] VPC/subnet not found. Verify IDs.` |
| `InsufficientBalance` | 0 | — | HALT | `[ERROR] Insufficient balance. Recharge account.` |
| `429` / Throttling | 3 | exponential | Back off | `⚠️ Rate limited. Retrying in {backoff}s...` |
| `5xx` InternalError | 3 | 2s, 4s, 8s | Retry; then HALT | `[ERROR] Server error. Retry or escalate with RequestId.` |

### Operation: List / Describe Cluster

#### Execution — CLI

```bash
# List all clusters
hcloud cce list-clusters \
  --region "{{user.region}}"

# Describe specific cluster
hcloud cce describe-cluster \
  --region "{{user.region}}" \
  --cluster-id "{{user.cluster_id}}"
```

#### Execution — SDK

```
GET  /api/v3/projects/{project_id}/clusters           — List clusters
GET  /api/v3/projects/{project_id}/clusters/{cluster_id}  — Describe cluster
```

#### Post-execution Validation

- Report cluster details: name, version, phase (Available/Error), VPC, status, node count.

### Operation: Delete Cluster

#### Pre-flight (Safety Gate)

- **MUST** obtain explicit confirmation: irreversible delete of cluster `{{user.cluster_name}}` (`{{user.cluster_id}}`).
- **MUST NOT** proceed without clear user assent.
- **MUST** warn user: all workloads, node pools, and addons will be lost.
- **MUST** remind user to backup critical data (PV snapshots, config exports).

#### Execution

```bash
hcloud cce delete-cluster \
  --region "{{user.region}}" \
  --cluster-id "{{user.cluster_id}}"
```

#### Post-execution Validation

- Poll **ShowCluster** until 404 or `$.status.phase` = `Deleting` → `NotFound`.
- Confirm deletion within 600 seconds (cluster deletion is slow).

### Operation: Create Node

#### Pre-flight Checks

| Check | Method | Expected | On Failure |
|-------|--------|----------|------------|
| Cluster exists | Describe cluster | Phase = Available | HALT; create cluster |
| Credentials | Env var existence check | Non-empty AK/SK | HALT |
| Flavor | Verify ECS flavor exists | Valid flavor | Suggest valid flavors |
| OS Image | Verify OS image is compatible | Valid image | List compatible images |

#### Execution — CLI

```bash
hcloud cce create-node \
  --region "{{user.region}}" \
  --cluster-id "{{user.cluster_id}}" \
  --name "{{user.node_name}}" \
  --flavor "{{user.flavor}}" \
  --os "{{user.os:EulerOS 2.9}}" \
  --volume-type "{{user.volume_type:SATA}}" \
  --volume-size "{{user.volume_size:40}}" \
  --ssh-key "{{user.ssh_key_name}}"
```

#### Post-execution Validation (Async Polling)

1. Read `{{output.node_id}}` from `$.metadata.uid`.
2. Poll **ShowNode** every 5s (max 300s) until `$.status.phase` = `Active`.
3. Report node IP, flavor, and phase.

### Operation: List / Describe Nodes

#### Execution — CLI

```bash
# List nodes
hcloud cce list-nodes \
  --region "{{user.region}}" \
  --cluster-id "{{user.cluster_id}}"

# Describe specific node
hcloud cce describe-node \
  --region "{{user.region}}" \
  --cluster-id "{{user.cluster_id}}" \
  --node-id "{{user.node_id}}"
```

#### Post-execution Validation

- Report node details: name, IP, flavor, phase, OS, SSH key, labels.

### Operation: Delete Node

#### Pre-flight (Safety Gate)

- **MUST** confirm: irreversible remove node `{{user.node_name}}` from cluster `{{user.cluster_id}}`.
- **MUST** warn: pods on this node will be evicted; ensure replicas exist.
- **MUST** check node pool — if node belongs to a pool, warn about pool reconciliation.

#### Execution

```bash
hcloud cce delete-node \
  --region "{{user.region}}" \
  --cluster-id "{{user.cluster_id}}" \
  --node-id "{{user.node_id}}"
```

#### Post-execution Validation

- Poll **ShowNode** until 404 or phase = `Deleted`.
- Confirm deletion within 300 seconds.

### Operation: Create Node Pool

#### Pre-flight Checks

| Check | Method | Expected | On Failure |
|-------|--------|----------|------------|
| Cluster exists | Describe cluster | Phase = Available | HALT; cluster not ready |
| Subnet | Verify node subnet exists | Subnet ACTIVE | HALT; create subnet |
| Autoscaling config | Validate min ≤ desired ≤ max | Valid range | Fix values |

#### Execution — CLI

```bash
hcloud cce create-nodepool \
  --region "{{user.region}}" \
  --cluster-id "{{user.cluster_id}}" \
  --name "{{user.nodepool_name}}" \
  --flavor "{{user.flavor}}" \
  --initial-node-count "{{user.desired_count:1}}" \
  --min-node-count "{{user.min_count:0}}" \
  --max-node-count "{{user.max_count:10}}" \
  --volume-type "{{user.volume_type:SATA}}" \
  --volume-size "{{user.volume_size:40}}" \
  --os "{{user.os:EulerOS 2.9}}" \
  --ssh-key "{{user.ssh_key_name}}" \
  --enable-autoscaling true
```

#### Post-execution Validation

1. Read `{{output.nodepool_id}}` from `$.metadata.uid`.
2. Poll **ShowNodePool** every 5s (max 300s) until `$.status.phase` = `Active`.
3. Report nodepool ID, initial node count, and scaling range.

### Operation: List / Describe Node Pool

#### Execution — CLI

```bash
# List node pools
hcloud cce list-nodepools \
  --region "{{user.region}}" \
  --cluster-id "{{user.cluster_id}}"

# Describe specific node pool
hcloud cce describe-nodepool \
  --region "{{user.region}}" \
  --cluster-id "{{user.cluster_id}}" \
  --nodepool-id "{{user.nodepool_id}}"
```

### Operation: Update Node Pool

#### Pre-flight Checks

- Collect new desired node count (must be within min/max).
- Warn if scaling down: pods will be evicted.

#### Execution — CLI

```bash
# Scale node pool
hcloud cce update-nodepool \
  --region "{{user.region}}" \
  --cluster-id "{{user.cluster_id}}" \
  --nodepool-id "{{user.nodepool_id}}" \
  --initial-node-count "{{user.desired_count}}" \
  --min-node-count "{{user.min_count}}" \
  --max-node-count "{{user.max_count}}"
```

#### Post-execution Validation

- Poll **ShowNodePool** until node count matches desired.
- Report current vs desired node count.

### Operation: Delete Node Pool

#### Pre-flight (Safety Gate)

- **MUST** confirm: irreversible delete nodepool `{{user.nodepool_name}}` (`{{user.nodepool_id}}`).
- **MUST** warn: all nodes in this pool will be removed from cluster.

#### Execution

```bash
hcloud cce delete-nodepool \
  --region "{{user.region}}" \
  --cluster-id "{{user.cluster_id}}" \
  --nodepool-id "{{user.nodepool_id}}"
```

#### Post-execution Validation

- Poll **ShowNodePool** until 404.
- Confirm deletion within 300 seconds.

### Operation: Install Addon

#### Pre-flight Checks

| Check | Method | Expected | On Failure |
|-------|--------|----------|------------|
| Cluster exists | Describe cluster | Phase = Available | HALT |
| Addon supported | Check addon version compatibility | Valid version for K8s version | Suggest compatible versions |
| Cluster type | Verify addon is compatible with cluster type | Match type | HALT; addon not supported |

#### Execution — CLI

```bash
hcloud cce install-addon \
  --region "{{user.region}}" \
  --cluster-id "{{user.cluster_id}}" \
  --addon-name "{{user.addon_name}}" \
  --addon-version "{{user.addon_version}}" \
  --values '{{user.addon_values_json:{}}}'
```

#### Execution — SDK

```go
package main

import (
	"fmt"
	"os"

	"github.com/huaweicloud/huaweicloud-sdk-go-v3/core/config"
	"github.com/huaweicloud/huaweicloud-sdk-go-v3/core/auth/basic"
	"github.com/huaweicloud/huaweicloud-sdk-go-v3/services/cce/v3"
	"github.com/huaweicloud/huaweicloud-sdk-go-v3/services/cce/v3/model"
)

func main() {
	ak := os.Getenv("HW_ACCESS_KEY_ID")
	sk := os.Getenv("HW_SECRET_ACCESS_KEY")
	region := os.Getenv("HW_REGION_ID")

	cfg := config.DefaultHttpConfig()
	client := v3.NewCceClient(
		v3.CceClientBuilder().
			WithEndpoint(fmt.Sprintf("cce.%s.myhuaweicloud.com", region)).
			WithCredential(basic.NewCredentialsBuilder().
				WithAk(ak).WithSk(sk).Build()).
			WithHttpConfig(cfg).Build())

	values := &model.AddonValues{
		Values: make(map[string]interface{}),
		Basic: &model.AddonValuesBasic{
			Name:       os.Getenv("ADDON_NAME"),
			Version:    os.Getenv("ADDON_VERSION"),
			Alias:      os.Getenv("ADDON_NAME"),
			ClusterID:  os.Getenv("CLUSTER_ID"),
			ClusterType: os.Getenv("CLUSTER_TYPE"),
		},
	}

	request := &model.CreateAddonInstanceRequest{
		ClusterId: os.Getenv("CLUSTER_ID"),
		Body:      values,
	}

	response, err := client.CreateAddonInstance(request)
	if err != nil {
		fmt.Fprintf(os.Stderr, "InstallAddon failed: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("Addon Uid: %s\n", response.Metadata.Uid)
}
```

#### Post-execution Validation

1. Read `{{output.addon_id}}` from `$.metadata.uid`.
2. Poll **ShowAddonInstance** every 5s (max 120s) until `$.status.status` = `running`.
3. Report addon name, version, and running status.

### Operation: List Addons

#### Execution — CLI

```bash
# List installed addons
hcloud cce list-addons \
  --region "{{user.region}}" \
  --cluster-id "{{user.cluster_id}}"

# List supported addon templates
hcloud cce list-addon-templates \
  --region "{{user.region}}" \
  --cluster-id "{{user.cluster_id}}" \
  --addon-version "{{user.k8s_version}}"
```

## Prerequisites

1. **Install KooCLI** (official binary):

    ```bash
    curl -sSL https://cn-north-4.myhuaweicloud.com/cli/latest/hcloud_install.sh -o ./hcloud_install.sh && bash ./hcloud_install.sh -y
    hcloud version
    ```

2. **Bootstrap Go runtime** (JIT SDK fallback):

    ```bash
    if ! command -v go &> /dev/null; then
        OS=$(uname -s | tr '[:upper:]' '[:lower:]')
        ARCH=$(uname -m)
        [ "$ARCH" = "x86_64" ] && ARCH="amd64"
        [ "$ARCH" = "aarch64" ] && ARCH="arm64"
        mkdir -p /tmp/go-runtime
        curl -fsSL "https://go.dev/dl/go1.25.0.${OS}-${ARCH}.tar.gz" | tar -xz -C /tmp/go-runtime
        export PATH="/tmp/go-runtime/go/bin:$PATH"
        export GOPROXY="https://goproxy.cn,direct"
    fi
    ```

3. **Configure Credentials**:

    ```bash
    export HW_ACCESS_KEY_ID="{{env.HW_ACCESS_KEY_ID}}"
    export HW_SECRET_ACCESS_KEY="{{env.HW_SECRET_ACCESS_KEY}}"
    export HW_REGION_ID="{{env.HW_REGION_ID}}"
    export HW_PROJECT_ID="{{env.HW_PROJECT_ID}}"
    test -n "$HW_SECRET_ACCESS_KEY" && echo "✅ Credentials configured"
    ```

4. **Verify Configuration**: `hcloud cce list-clusters --region {{env.HW_REGION_ID}}`

## Quality Gate (GCL)

This skill is **GCL-required** (per `AGENTS.md` §8). Every CCE (Kubernetes) mutating operation
— cluster create / delete, node create / delete / drain / cordon, node-pool create / update /
delete, `kubectl apply` / `delete` — runs through the **Generator-Critic-Loop** before its
result is returned. Read-only are GCL-**exempt**.

| Field | Value |
|-------|-------|
| Rubric version | v1 (Phase 2, 2026-06-04) |
| `max_iter` | **2** |
| Rubric instance | [`references/rubric.md`](references/rubric.md) |
| Prompt templates | [`references/prompt-templates.md`](references/prompt-templates.md) |
| Trace path | `./audit-results/gcl-trace-YYYYMMDD-HHMMSS.json` |
| Independence | Generator and Critic in **isolated** sub-agent / session contexts |

### Five-Dimension Rubric (summary)

| # | Dimension | Threshold | Notes |
|---|-----------|-----------|-------|
| 1 | Correctness | ≥ 0.5 (1.0 for `delete-cluster` / `delete-node` / `drain` / `delete-pool`) | `ShowCluster` / `ShowNode` / `kubectl get` post-state |
| 2 | Safety | **= 1** (any S-rule hit → ABORT) | S1–S17 in rubric §2 |
| 3 | Idempotency | ≥ 0.5 | Pre-check before create; see also `references/idempotency-checklist.md` |
| 4 | Traceability | ≥ 0.5 | kubeconfig token / `Authorization: Bearer` MUST be `<masked>` |
| 5 | Spec Compliance | ≥ 0.5 | K8s version / network mode / CIDR / name regex |

### Per-Operation Safety Anchors (binding)

- **S1 / S2 / S3** — `delete-cluster` confirmation / workloads present / prePaid refund
- **S4** — `delete-node` without first running `kubectl drain` (graceful)
- **S5** — `delete-node` in ASG pool without checking `desired_size`
- **S6** — `drain` without PDB / DaemonSet check
- **S7** — `delete-node-pool` with running non-replicated workloads
- **S8** — `scale` DOWN forcing StatefulSet `replicas: 0`
- **S9** — `delete-namespace` with running workloads, no force, no confirm
- **S11** — `delete-pod` in `kube-system` / `cce-system` / monitoring
- **S12** — `apply-yaml` with `privileged: true` / `hostNetwork: true` / `hostPID: true`
- **S16** — `cordon` / `drain` on control-plane node when cluster has < 3 masters
- **S17** — `delete-cluster` while `status == Available` AND HA is degraded

### Termination Contract (per `AGENTS.md` §5)

| Condition | Status | Returned |
|-----------|--------|----------|
| All dimensions pass | **PASS** | Generator result + scores + trace path |
| `iter == max_iter` (2) and any dim < threshold | **MAX_ITER** | best-so-far + unresolved rubric items |
| `Safety == 0` | **SAFETY_FAIL** | violated S-rule id; **never** return partial |

### Trace Persistence (mandatory)

Every GCL run writes `./audit-results/gcl-trace-YYYYMMDD-HHMMSS.json` (schema in
`references/prompt-templates.md` §3). Trace is **append-only**; sanitize secrets before write
(see `prompt-templates.md` §4). The path `./audit-results/` is in root `.gitignore`.

### See also

- [`references/rubric.md`](references/rubric.md) — full rubric, S1–S17 rules, per-op thresholds
- [`references/prompt-templates.md`](references/prompt-templates.md) — Generator / Critic / Orchestrator skeletons
- Repository root [`AGENTS.md`](../AGENTS.md) §3, §5, §7, §8 — GCL specification

## Reference Directory

- [Core Concepts](references/core-concepts.md)
- [API & SDK Usage](references/api-sdk-usage.md)
- [CLI Usage](references/cli-usage.md)
- [Troubleshooting Guide](references/troubleshooting.md)
- [Monitoring & Alerts](references/monitoring.md)
- [Integration](references/integration.md)
- [Knowledge Base](references/knowledge-base.md)
- [Observability Integration](references/advanced/observability.md)
- [Idempotency Checklist](references/idempotency-checklist.md)
- [FinOps Cost Optimization](references/well-architected-assessment.md#3-finops-)
- [SecOps Security Operations](references/well-architected-assessment.md#4-secops-)
- [Well-Architected Assessment](references/well-architected-assessment.md)
- [GCL Rubric](references/rubric.md) — Adversarial quality gate (v1, 5-dim, S1–S17 CCE/Kubernetes-specific Safety rules)
- [GCL Prompt Templates](references/prompt-templates.md) — Generator / Critic / Orchestrator skeletons

## Well-Architected + Three-Pillar Assessment

This skill's operations are evaluated against Huawei Cloud's Well-Architected Framework (卓越架构) five pillars plus FinOps, SecOps, and AIOps integration:
- [Security Assessment](references/well-architected-assessment.md)
- [Stability Assessment](references/well-architected-assessment.md)
- [Cost Assessment](references/well-architected-assessment.md)
- [Efficiency Assessment](references/well-architected-assessment.md)
- [Performance Assessment](references/well-architected-assessment.md)
- [FinOps Integration](references/well-architected-assessment.md)
- [SecOps Integration](references/well-architected-assessment.md)
- [AIOps Integration](references/knowledge-base.md)
