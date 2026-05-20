# CCE CLI Usage — Huawei Cloud Cloud Container Engine

## CLI Command Map

### Cluster Commands

| Operation | CLI Command | Key Flags |
|-----------|-------------|-----------|
| List clusters | `hcloud cce list-clusters` | `--region`, `--phase` |
| Create cluster | `hcloud cce create-cluster` | `--region`, `--name`, `--flavor`, `--version`, `--vpc-id`, `--subnet-id`, `--security-group-id` |
| Describe cluster | `hcloud cce describe-cluster` | `--region`, `--cluster-id` |
| Delete cluster | `hcloud cce delete-cluster` | `--region`, `--cluster-id` |
| Get cluster cert | `hcloud cce get-cluster-cert` | `--region`, `--cluster-id` |

### Node Commands

| Operation | CLI Command | Key Flags |
|-----------|-------------|-----------|
| List nodes | `hcloud cce list-nodes` | `--region`, `--cluster-id`, `--phase` |
| Create node | `hcloud cce create-node` | `--region`, `--cluster-id`, `--name`, `--flavor`, `--os`, `--volume-type`, `--volume-size`, `--ssh-key` |
| Describe node | `hcloud cce describe-node` | `--region`, `--cluster-id`, `--node-id` |
| Delete node | `hcloud cce delete-node` | `--region`, `--cluster-id`, `--node-id` |

### Node Pool Commands

| Operation | CLI Command | Key Flags |
|-----------|-------------|-----------|
| List node pools | `hcloud cce list-nodepools` | `--region`, `--cluster-id` |
| Create node pool | `hcloud cce create-nodepool` | `--region`, `--cluster-id`, `--name`, `--flavor`, `--initial-node-count`, `--min-node-count`, `--max-node-count`, `--enable-autoscaling` |
| Describe node pool | `hcloud cce describe-nodepool` | `--region`, `--cluster-id`, `--nodepool-id` |
| Update node pool | `hcloud cce update-nodepool` | `--region`, `--cluster-id`, `--nodepool-id`, `--initial-node-count`, `--min-node-count`, `--max-node-count` |
| Delete node pool | `hcloud cce delete-nodepool` | `--region`, `--cluster-id`, `--nodepool-id` |

### Addon Commands

| Operation | CLI Command | Key Flags |
|-----------|-------------|-----------|
| List addons | `hcloud cce list-addons` | `--region`, `--cluster-id` |
| Install addon | `hcloud cce install-addon` | `--region`, `--cluster-id`, `--addon-name`, `--addon-version`, `--values` |
| Describe addon | `hcloud cce describe-addon` | `--region`, `--cluster-id`, `--addon-id` |
| Delete addon | `hcloud cce delete-addon` | `--region`, `--cluster-id`, `--addon-id` |
| List addon templates | `hcloud cce list-addon-templates` | `--region`, `--cluster-id`, `--addon-version` |

## Coverage Gap Table

| Operation | CLI Support | SDK Support | Notes |
|-----------|------------|-------------|-------|
| Cluster CRUD | ✅ | ✅ | Full coverage |
| Node CRUD | ✅ | ✅ | Full coverage |
| NodePool CRUD | ✅ | ✅ | Full coverage |
| Addon install/delete | ✅ | ✅ | Full coverage |
| Addon update | ✅ | ✅ | Via `--values` flag |
| Cluster upgrade | ✅ | ✅ | Version upgrade |
| Cluster resize | ❌ | ✅ | SDK-only via UpdateCluster |
| Batch node creation | ✅ | ✅ | Via `--count` flag |
| Certificate retrieval | ✅ | ✅ | Full coverage |

## JSON Output Parsing (jq Patterns)

```bash
# Extract cluster ID from create response
hcloud cce create-cluster ... | jq -r '.metadata.uid'

# Extract cluster status
hcloud cce describe-cluster --cluster-id xxx | jq -r '.status.phase'

# List cluster IDs that are Available
hcloud cce list-clusters | jq -r '.items[] | select(.status.phase == "Available") | .metadata.uid'

# Count nodes in a cluster
hcloud cce list-nodes --cluster-id xxx | jq '.items | length'

# Get node pool autoscaling config
hcloud cce describe-nodepool --cluster-id xxx --nodepool-id yyy | jq '.spec.autoscaling'

# Find addon version
hcloud cce describe-addon --cluster-id xxx --addon-id yyy | jq -r '.spec.version'

# List NotReady nodes
hcloud cce list-nodes --cluster-id xxx | jq -r '.items[] | select(.status.phase != "Active") | .metadata.name'
```

## Batch Operation Patterns

```bash
# Create 3 nodes in a cluster
hcloud cce create-node \
  --region "{{user.region}}" \
  --cluster-id "{{user.cluster_id}}" \
  --flavor "{{user.flavor}}" \
  --count 3

# Delete all nodes from a node pool (drain first)
for NODE_ID in $(hcloud cce list-nodes --cluster-id "{{user.cluster_id}}" | jq -r '.items[] | select(.metadata.labels["nodepool"] == "{{user.nodepool_id}}") | .metadata.uid'); do
  hcloud cce delete-node --region "{{user.region}}" --cluster-id "{{user.cluster_id}}" --node-id "$NODE_ID"
done

# Scale multiple node pools
for POOL in "${USER_POOL_IDS[@]}"; do
  hcloud cce update-nodepool --region "{{user.region}}" --cluster-id "{{user.cluster_id}}" --nodepool-id "$POOL" --initial-node-count 3
done
```
