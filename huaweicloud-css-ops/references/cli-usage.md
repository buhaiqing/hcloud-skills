# CSS CLI Usage

## CLI Command Map

### Cluster Management

| Operation | CLI Command | Required Flags |
|-----------|-------------|----------------|
| List Clusters | `hcloud CSS ListClusters` | `--cli-region` |
| Show Cluster | `hcloud CSS ShowClusterDetail` | `--cluster_id` |
| Create Cluster | `hcloud CSS CreateCluster` | `--name`, `--datastore`, `--instance` |
| Delete Cluster | `hcloud CSS DeleteCluster` | `--cluster_id` |
| Restart Cluster | `hcloud CSS RestartCluster` | `--cluster_id` |
| Extend Cluster | `hcloud CSS ExtendCluster` | `--cluster_id`, `--grow` |
| Scale Out | `hcloud CSS ScaleOut` | `--cluster_id`, `--count` |
| Reset Password | `hcloud CSS ResetPassword` | `--cluster_id`, `--password` |
| Update Name | `hcloud CSS UpdateClusterName` | `--cluster_id`, `--name` |

### Snapshot Management

| Operation | CLI Command | Required Flags |
|-----------|-------------|----------------|
| List Snapshots | `hcloud CSS ListSnapshots` | `--cluster_id` |
| Create Snapshot | `hcloud CSS CreateSnapshot` | `--cluster_id`, `--name` |
| Delete Snapshot | `hcloud CSS DeleteSnapshot` | `--cluster_id`, `--snapshot_id` |
| Restore Snapshot | `hcloud CSS RestoreSnapshot` | `--cluster_id`, `--snapshot_id` |
| Show Snapshot Policy | `hcloud CSS ShowSnapshotPolicy` | `--cluster_id` |
| Set Snapshot Policy | `hcloud CSS SetSnapshotPolicy` | `--cluster_id`, `--period`, `--keepday` |

### Dictionary Management

| Operation | CLI Command | Required Flags |
|-----------|-------------|----------------|
| List IK Dicts | `hcloud CSS ListIkDicts` | `--cluster_id` |
| Add IK Dict | `hcloud CSS AddIkDict` | `--cluster_id`, `--dict-type`, `--dict-content` |
| Update IK Dict | `hcloud CSS UpdateIkDict` | `--cluster_id`, `--dict-id` |
| Delete IK Dict | `hcloud CSS DeleteIkDict` | `--cluster_id`, `--dict-id` |

## CLI Examples

### Create Cluster

```bash
hcloud CSS CreateCluster \
  --name "prod-es-cluster" \
  --datastore '{"type":"elasticsearch","version":"7.10.2"}' \
  --instance '{
    "flavorRef":"ess.spec-4u8g",
    "nics":{"vpcId":"vpc-xxx","netId":"subnet-xxx","securityGroupId":"sg-xxx"},
    "volume":{"type":"ULTRAHIGH","size":100},
    "availabilityZone":"cn-north-4a"
  }' \
  --instance-num 3 \
  --https-enable true \
  --disk-encryption-enabled true \
  --disk-encryption-key "kms-key-xxx" \
  --cli-region "cn-north-4"
```

### List Clusters

```bash
hcloud CSS ListClusters \
  --cli-region "cn-north-4" \
  --limit 100
```

### Create Snapshot

```bash
hcloud CSS CreateSnapshot \
  --cluster-id "cluster-xxx" \
  --name "manual-snapshot-$(date +%Y%m%d)" \
  --description "Manual backup" \
  --indices "*,-.kibana*" \
  --backup-obs-path "obs://bucket/snapshots/"
```

### Set Snapshot Policy

```bash
hcloud CSS SetSnapshotPolicy \
  --cluster-id "cluster-xxx" \
  --period "00:00 GMT+08:00" \
  --prefix "auto-snapshot" \
  --keepday 7 \
  --bucket "css-backups" \
  --base-path "cluster-backups" \
  --agency "css_obs_agency"
```

## CLI Coverage Gap Table

| API Operation | CLI Support | SDK Fallback | Notes |
|---------------|-------------|--------------|-------|
| ListClusters | ✅ Full | No | All parameters supported |
| ShowClusterDetail | ✅ Full | No | All parameters supported |
| CreateCluster | ✅ Full | No | Complex nested JSON required |
| DeleteCluster | ✅ Full | No | With backup OBS path |
| RestartCluster | ✅ Full | No | Rolling restart |
| ExtendCluster | ✅ Partial | Yes | Limited scaling options |
| ScaleOut | ✅ Full | No | Add nodes |
| ListSnapshots | ✅ Full | No | Pagination supported |
| CreateSnapshot | ✅ Full | No | All parameters supported |
| RestoreSnapshot | ✅ Partial | Yes | Complex restore options |
| ListIkDicts | ✅ Full | No | Dictionary listing |
| AddIkDict | ✅ Full | No | Upload dictionary file |
| UpdateIkDict | ✅ Partial | Yes | Limited update options |

## Output Formatting

```bash
# JSON output (default)
hcloud CSS ListClusters --cli-region "cn-north-4" -o json

# Filter with jq
hcloud CSS ListClusters --cli-region "cn-north-4" -o json | \
  jq '.clusters[] | {id: .id, name: .name, status: .status}'
```

## JSON Output Paths

| Field | JSON Path | Example |
|-------|-----------|---------|
| Cluster ID | `.clusters[0].id` | `a1b2c3d4-...` |
| Cluster Name | `.clusters[0].name` | `prod-es` |
| Status | `.clusters[0].status` | `AVAILABLE` |
| Engine Version | `.clusters[0].datastore.version` | `7.10.2` |
| Node Count | `.clusters[0].instanceNum` | `3` |
| Created At | `.clusters[0].created` | `2026-05-27T10:00:00Z` |
