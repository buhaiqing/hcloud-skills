# CSS Troubleshooting Guide

## Error Code Reference

| Error Code | Description | Severity | Recovery Action |
|------------|-------------|----------|-----------------|
| `CSS.0001` | Invalid request parameter | Error | Fix parameter format |
| `CSS.0002` | Insufficient quota | Error | Request quota increase |
| `CSS.0003` | Insufficient balance | Error | Recharge account |
| `CSS.0004` | Resource already exists | Error | Use different name |
| `CSS.0005` | Rate limit exceeded | Warning | Retry with backoff |
| `CSS.0006` | Internal server error | Error | Retry or escalate |
| `CSS.0010` | Invalid VPC/Subnet | Error | Check network config |
| `CSS.0011` | Invalid flavor specification | Error | Check available flavors |
| `CSS.0012` | Invalid engine version | Error | Use supported version |
| `CSS.0013` | KMS key not found | Error | Verify KMS key ID |
| `CSS.0014` | OBS bucket not found | Error | Create bucket first |
| `CSS.0015` | Insufficient permissions | Error | Check IAM policies |
| `CSS.0020` | Cluster not found | Error | Verify cluster ID |
| `CSS.0021` | Cluster not available | Warning | Wait for ready state |
| `CSS.0022` | Cluster operation in progress | Warning | Wait for completion |
| `CSS.0030` | Snapshot not found | Error | Verify snapshot ID |
| `CSS.0031` | Snapshot creation failed | Error | Check OBS permissions |
| `CSS.0032` | Snapshot restore failed | Error | Check compatibility |

## Diagnostic Decision Tree

```
Cluster Issue
├── Cannot Create Cluster
│   ├── Check quota: hcloud CSS ListQuotas
│   ├── Check balance: Account balance
│   ├── Check VPC/Subnet: hcloud VPC ShowVpc
│   └── Check flavor: hcloud CSS ListFlavors
├── Cluster Unavailable
│   ├── Check status: hcloud CSS ShowClusterDetail
│   ├── Check nodes: hcloud ECS ListServers (if ECS visible)
│   └── Check CES metrics: CPU/Memory/Disk
├── Query Performance Issues
│   ├── Check shard allocation: Cluster health API
│   ├── Check JVM heap: CES metrics
│   └── Check query patterns: Slow log analysis
└── Snapshot Issues
    ├── Check OBS permissions: IAM policy
    ├── Check bucket access: hcloud OBS HeadBucket
    └── Check cluster health: Must be green/yellow
```

## Common Issues

### Issue: Cluster Stuck in CREATING State

**Symptoms**: Cluster status remains `CREATING` for > 30 minutes

**Diagnosis**:
```bash
hcloud CSS ShowClusterDetail --cluster_id "{{user.cluster_id}}" -o json | jq '.status, .actions'
```

**Resolution**:
1. Check underlying ECS instance status
2. Verify VPC/Subnet connectivity
3. If stuck > 1 hour, contact support with cluster ID

### Issue: Shard Allocation Failed

**Symptoms**: Cluster health `red`, `unassigned_shards > 0`

**Diagnosis**:
```bash
# Via Elasticsearch API
curl -u admin:{{user.password}} https://{{cluster.endpoint}}:9200/_cluster/health
curl -u admin:{{user.password}} https://{{cluster.endpoint}}:9200/_cat/shards?v | grep UNASSIGNED
```

**Resolution**:
1. Check disk space: `disk.watermark` settings
2. Check node availability
3. Force reallocation if needed:
```bash
curl -X POST -u admin:{{user.password}} https://{{cluster.endpoint}}:9200/_cluster/reroute?retry_failed=true
```

### Issue: High Query Latency

**Symptoms**: Search response time > 1s consistently

**Diagnosis**:
```bash
# Check slow query log
curl -u admin:{{user.password}} https://{{cluster.endpoint}}:9200/_cluster/settings?include_defaults=true&filter_path=**.search

# Check hot threads
curl -u admin:{{user.password}} https://{{cluster.endpoint}}:9200/_nodes/hot_threads
```

**Resolution**:
1. Add client nodes for query load balancing
2. Optimize query structure (avoid deep pagination)
3. Increase node specs if CPU/heap saturated
4. Review index mapping and analyzer settings

### Issue: Snapshot Creation Failed

**Symptoms**: Snapshot status `FAILED`

**Diagnosis**:
```bash
hcloud CSS ListSnapshots --cluster_id "{{user.cluster_id}}" -o json | jq '.snapshots[] | select(.status == "FAILED")'
```

**Resolution**:
1. Verify OBS bucket exists and is accessible
2. Check IAM permissions for OBS write
3. Check cluster health (must not be red)
4. Verify available disk space

### Issue: Disk Full

**Symptoms**: `disk.usage > 90%`, writes rejected

**Diagnosis**:
```bash
# Check disk usage
curl -u admin:{{user.password}} https://{{cluster.endpoint}}:9200/_cat/allocation?v

# Check indices
curl -u admin:{{user.password}} https://{{cluster.endpoint}}:9200/_cat/indices?v&s=store.size:desc
```

**Resolution**:
1. Delete old indices if retention allows
2. Extend cluster storage: `hcloud CSS ExtendCluster`
3. Add cold nodes for archival data
4. Update index lifecycle policies

### Issue: Authentication Failed

**Symptoms**: HTTP 401 errors

**Diagnosis**:
```bash
# Test credentials
curl -u admin:{{user.password}} https://{{cluster.endpoint}}:9200/_cluster/health
```

**Resolution**:
1. Reset password: `hcloud CSS ResetPassword`
2. Verify HTTPS certificate if using custom CA
3. Check security group allows client IP

## Ordered Diagnostic Steps

### Step 1: Cluster Health Check

```bash
hcloud CSS ShowClusterDetail --cluster_id "{{user.cluster_id}}"
```

Verify:
- Status: `AVAILABLE`
- Endpoint accessible
- Nodes: Expected count

### Step 2: CES Metrics Check

```bash
hcloud CES ShowMetricData \
  --namespace SYS.CSS \
  --metric_name disk_usage \
  --dimensions cluster_id={{user.cluster_id}} \
  --from $(date -d '1 hour ago' +%s)000 \
  --to $(date +%s)000
```

Check metrics:
- `cpu_usage`
- `mem_usage`
- `disk_usage`
- `jvm_heap_usage`
- `search_latency`

### Step 3: Elasticsearch API Check

```bash
# Cluster health
curl -u admin:{{user.password}} https://{{cluster.endpoint}}:9200/_cluster/health?pretty

# Node stats
curl -u admin:{{user.password}} https://{{cluster.endpoint}}:9200/_nodes/stats

# Pending tasks
curl -u admin:{{user.password}} https://{{cluster.endpoint}}:9200/_cluster/pending_tasks
```

### Step 4: Network Connectivity

```bash
# Test endpoint connectivity
telnet {{cluster.endpoint}} 9200

# Test with SSL
curl -v https://{{cluster.endpoint}}:9200
```

### Step 5: IAM Permissions

Verify user has these permissions:
```json
{
  "Version": "1.1",
  "Statement": [
    {
      "Effect": "Allow",
      "Action": [
        "css:*:get",
        "css:*:list",
        "css:*:create",
        "css:*:delete",
        "css:*:modify"
      ]
    }
  ]
}
```

## Recovery Procedures

### Emergency: Cluster Recovery from Red State

1. **Identify unassigned shards**:
```bash
curl -u admin:{{user.password}} https://{{cluster.endpoint}}:9200/_cat/shards?v | grep UNASSIGNED
```

2. **Check allocation explanation**:
```bash
curl -u admin:{{user.password}} https://{{cluster.endpoint}}:9200/_cluster/allocation/explain?pretty
```

3. **Temporarily increase replica count to 0 for affected indices**:
```bash
curl -X PUT -u admin:{{user.password}} https://{{cluster.endpoint}}:9200/index-name/_settings \
  -H 'Content-Type: application/json' \
  -d '{"index": {"number_of_replicas": 0}}'
```

4. **Force reroute**:
```bash
curl -X POST -u admin:{{user.password}} https://{{cluster.endpoint}}:9200/_cluster/reroute?retry_failed=true
```

### Emergency: Snapshot Recovery

1. **List available snapshots**:
```bash
hcloud CSS ListSnapshots --cluster_id "{{user.cluster_id}}"
```

2. **Restore to new cluster** (recommended):
```bash
# Create new cluster first
hcloud CSS CreateCluster ...

# Then restore snapshot
hcloud CSS RestoreSnapshot \
  --cluster_id "{{new_cluster_id}}" \
  --snapshot_id "{{user.snapshot_id}}"
```

## Support Escalation

When to escalate to Huawei Cloud support:

1. Internal server errors (`CSS.0006`) persist after 3 retries
2. Cluster stuck in non-terminal state > 2 hours
3. Data loss suspected
4. Security incidents

**Required information**:
- Cluster ID
- Region
- Error codes/messages
- Time of occurrence
- Request ID (if available)
