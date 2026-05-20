# CCE API & SDK Usage — Huawei Cloud Cloud Container Engine

## SDK Import and Client Initialization

```go
import (
    "github.com/huaweicloud/huaweicloud-sdk-go-v3/core/config"
    "github.com/huaweicloud/huaweicloud-sdk-go-v3/core/auth/basic"
    "github.com/huaweicloud/huaweicloud-sdk-go-v3/services/cce/v3"
    "github.com/huaweicloud/huaweicloud-sdk-go-v3/services/cce/v3/model"
)

func newCceClient(region, ak, sk string) *v3.CceClient {
    cfg := config.DefaultHttpConfig()
    return v3.NewCceClient(
        v3.CceClientBuilder().
            WithEndpoint(fmt.Sprintf("cce.%s.myhuaweicloud.com", region)).
            WithCredential(basic.NewCredentialsBuilder().
                WithAk(ak).WithSk(sk).Build()).
            WithHttpConfig(cfg).Build())
}
```

## Operation Map

### Cluster Operations

| Operation | HTTP | API Path | SDK Method | Key Request Fields | Key Response Fields |
|-----------|------|----------|-----------|-------------------|-------------------|
| CreateCluster | POST | `/api/v3/projects/{project_id}/clusters` | `CreateCluster` | `metadata.name`, `spec.hostType`, `spec.network.vpc`, `spec.network.subnet`, `spec.network.securityGroup`, `billingMode` | `metadata.uid`, `status.phase` |
| ListClusters | GET | `/api/v3/projects/{project_id}/clusters` | `ListClusters` | Query params only | `items[]` (Cluster array) |
| GetCluster | GET | `/api/v3/projects/{project_id}/clusters/{cluster_id}` | `ShowCluster` | `cluster_id` path param | Full Cluster object |
| DeleteCluster | DELETE | `/api/v3/projects/{project_id}/clusters/{cluster_id}` | `DeleteCluster` | `cluster_id` path param | Empty on success |
| UpdateCluster | PUT | `/api/v3/projects/{project_id}/clusters/{cluster_id}` | `UpdateCluster` | `metadata.update`, `spec.description` | Updated Cluster object |

### Node Operations

| Operation | HTTP | API Path | SDK Method | Key Request Fields | Key Response Fields |
|-----------|------|----------|-----------|-------------------|-------------------|
| CreateNode | POST | `/api/v3/projects/{project_id}/clusters/{cluster_id}/nodes` | `CreateNode` | `spec.flavor`, `spec.os`, `spec.volume[].size`, `spec.volume[].volumetype`, `spec.login.sshKey` | `metadata.uid`, `status.serverIP` |
| ListNode | GET | `/api/v3/projects/{project_id}/clusters/{cluster_id}/nodes` | `ListNode` | `phase` filter | `items[]` (Node array) |
| GetNode | GET | `/api/v3/projects/{project_id}/clusters/{cluster_id}/nodes/{node_id}` | `ShowNode` | `cluster_id`, `node_id` | Full Node object |
| DeleteNode | DELETE | `/api/v3/projects/{project_id}/clusters/{cluster_id}/nodes/{node_id}` | `DeleteNode` | `cluster_id`, `node_id` | Empty on success |

### Node Pool Operations

| Operation | HTTP | API Path | SDK Method | Key Request Fields | Key Response Fields |
|-----------|------|----------|-----------|-------------------|-------------------|
| CreateNodePool | POST | `/api/v3/projects/{project_id}/clusters/{cluster_id}/nodepools` | `CreateNodePool` | `metadata.name`, `spec.initialNodeCount`, `spec.autoscaling.enable`, `spec.autoscaling.minNodeCount`, `spec.autoscaling.maxNodeCount`, `spec.nodeTemplate` | `metadata.uid`, `status.currentNodeCount` |
| ListNodePool | GET | `/api/v3/projects/{project_id}/clusters/{cluster_id}/nodepools` | `ListNodePool` | Query params only | `items[]` (NodePool array) |
| GetNodePool | GET | `/api/v3/projects/{project_id}/clusters/{cluster_id}/nodepools/{nodepool_id}` | `ShowNodePool` | `cluster_id`, `nodepool_id` | Full NodePool object |
| UpdateNodePool | PUT | `/api/v3/projects/{project_id}/clusters/{cluster_id}/nodepools/{nodepool_id}` | `UpdateNodePool` | `spec.initialNodeCount`, `spec.autoscaling` | Updated NodePool object |
| DeleteNodePool | DELETE | `/api/v3/projects/{project_id}/clusters/{cluster_id}/nodepools/{nodepool_id}` | `DeleteNodePool` | `cluster_id`, `nodepool_id` | Empty on success |

### Addon Operations

| Operation | HTTP | API Path | SDK Method | Key Request Fields | Key Response Fields |
|-----------|------|----------|-----------|-------------------|-------------------|
| CreateAddonInstance | POST | `/api/v3/projects/{project_id}/clusters/{cluster_id}/addons` | `CreateAddonInstance` | `spec.version`, `spec.clusterID`, `values` | `metadata.uid`, `status.status` |
| ListAddonInstances | GET | `/api/v3/projects/{project_id}/clusters/{cluster_id}/addons` | `ListAddonInstances` | Query params only | `items[]` (Addon array) |
| GetAddonInstance | GET | `/api/v3/projects/{project_id}/clusters/{cluster_id}/addons/{addon_id}` | `ShowAddonInstance` | `cluster_id`, `addon_id` | Full Addon object |
| DeleteAddonInstance | DELETE | `/api/v3/projects/{project_id}/clusters/{cluster_id}/addons/{addon_id}` | `DeleteAddonInstance` | `cluster_id`, `addon_id` | Empty on success |
| ListAddonTemplates | GET | `/api/v3/projects/{project_id}/clusters/{cluster_id}/addon-templates` | `ListAddonTemplates` | `addon_name`, `addon_version` | `items[]` (AddonTemplate array) |

### Certificate Operations

| Operation | HTTP | API Path | SDK Method | Key Request Fields | Key Response Fields |
|-----------|------|----------|-----------|-------------------|-------------------|
| GetCert | GET | `/api/v3/projects/{project_id}/clusters/{cluster_id}/certificate` | `ShowClusterCertificate` | `cluster_id` | `certificate`, `clusterExternalEndpoint`, `clusterInternalEndpoint` |

## Async Behavior

### Cluster Creation

Cluster creation returns immediately but the cluster is **not yet available**. You must poll:

```
GET /api/v3/projects/{project_id}/clusters/{cluster_id}
```

| Field | Terminal States | Description |
|-------|----------------|-------------|
| `$.status.phase` | `Available` (success), `Error` (failure) | Primary status indicator |
| `$.status.statusReason` | Error message text | Failure reason when phase = Error |

**Polling:** Every 5 seconds, max 300 seconds (5 minutes).

### Node Creation

Similar to cluster creation, node addition is async:

| Field | Terminal States |
|-------|----------------|
| `$.status.phase` | `Active` (success), `Error` (failure), `Deleting` |
| `$.status.statusReason` | Error message text |

**Polling:** Every 5 seconds, max 300 seconds.

### Node Pool Creation

| Field | Terminal States |
|-------|----------------|
| `$.status.phase` | `Active` (success), `Error` (failure) |

**Polling:** Every 5 seconds, max 300 seconds.

## Pagination

- `ListClusters`, `ListNode`, `ListNodePool` support `limit` and `page` query parameters.
- Default limit: varies per operation; set explicitly for batch operations.
- When `items` count equals `limit`, there may be more results.

## Request Body Patterns

### Cluster Create (Key Fields)

```json
{
  "metadata": {
    "name": "my-cluster",
    "labels": {"env": "production"}
  },
  "spec": {
    "hostType": "VM",
    "flavor": "cce.s1.small",
    "version": "1.28",
    "kubernetesSvcIpRange": "10.247.0.0/16",
    "network": {
      "vpc": "vpc-uuid",
      "subnet": "subnet-uuid",
      "securityGroup": "sg-uuid",
      "containerNetwork": {"mode": "vpc"},
      "eniNetwork": {"eniSubnetId": "eni-subnet-uuid"}
    }
  },
  "billingMode": 0
}
```

### Node Create (Key Fields)

```json
{
  "metadata": {
    "name": "my-node"
  },
  "spec": {
    "flavor": "s6.large.2",
    "os": "EulerOS 2.9",
    "volume": [
      {"size": 40, "volumetype": "SATA", "metadata": {"": ""}}
    ],
    "login": {"sshKey": "my-ssh-key"},
    "count": 1
  }
}
```
