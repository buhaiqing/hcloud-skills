# CSS Knowledge Base

## Fault Pattern Library

### Pattern: Cluster Health Red

**Symptoms**: Cluster status `red`, search/indexing failures

**Root Causes**:
1. Primary shard unassigned (node failure)
2. Disk full on data nodes
3. Network partition

**Diagnosis**:
```bash
# Check unassigned shards
curl -u admin:{{user.password}} https://{{cluster.endpoint}}:9200/_cat/shards?v | grep UNASSIGNED

# Check allocation explanation
curl -u admin:{{user.password}} https://{{cluster.endpoint}}:9200/_cluster/allocation/explain?pretty
```

**Resolution**:
1. If node failure: Replace failed node via CSS console or API
2. If disk full: Extend storage or delete old indices
3. If network issue: Check VPC connectivity

**Prevention**:
- Enable multi-AZ deployment
- Monitor disk usage alerts
- Regular snapshot backups

---

### Pattern: Yellow Health After Node Addition

**Symptoms**: Cluster health `yellow` after scaling out

**Root Cause**: Replica shards being allocated to new nodes

**Expected Behavior**: Temporary, should resolve within 10-30 minutes

**Resolution**:
- Wait for allocation to complete
- Monitor shard allocation progress

---

### Pattern: High Query Latency

**Symptoms**: Search response time > 500ms (p99)

**Root Causes**:
1. Hot shards (uneven data distribution)
2. Expensive aggregations
3. Insufficient client nodes
4. JVM GC pressure

**Diagnosis**:
```bash
# Check slow query log
curl -u admin:{{user.password}} https://{{cluster.endpoint}}:9200/_search/slowlog

# Check hot threads
curl -u admin:{{user.password}} https://{{cluster.endpoint}}:9200/_nodes/hot_threads
```

**Resolution**:
1. Add client nodes for query load balancing
2. Reindex with more shards if hot spotting
3. Optimize query structure

---

### Pattern: Snapshot Failure

**Symptoms**: Snapshot status `FAILED`

**Root Causes**:
1. OBS permissions insufficient
2. Cluster health red
3. OBS bucket not accessible

**Resolution**:
1. Verify OBS IAM permissions
2. Check cluster health
3. Verify bucket exists and is accessible

---

## Cascade Failure Scenarios

### Scenario: Node Failure Cascade

**Trigger**: Single node failure in small cluster

**Progression**:
1. Node fails → shards unassigned
2. Cluster health → yellow
3. If replicas insufficient → red
4. Recovery starts → new node provisioned
5. Shards reallocate → health recovers

**Mitigation**:
- Minimum 3 nodes for production
- Replica count >= 1
- Multi-AZ deployment

---

### Scenario: Disk Full Cascade

**Trigger**: Disk usage > 95% on data nodes

**Progression**:
1. Disk full → writes rejected
2. Indexing backlog → memory pressure
3. JVM heap full → OOM risk
4. Node may crash

**Mitigation**:
- Set disk watermark alerts at 75%, 85%, 95%
- Implement ILM for automatic cleanup
- Proactive scaling

---

## Historical Incident Analysis

| Incident ID | Date | Description | Root Cause | Resolution | Prevention |
|-------------|------|-------------|------------|------------|------------|
| INC-001 | 2026-01 | Cluster red for 2h | Master node failure | Auto-replacement | Multi-AZ master nodes |
| INC-002 | 2026-02 | Query latency spike | Hot shard | Reindex with more shards | Shard sizing guidelines |
| INC-003 | 2026-03 | Snapshot failures | OBS permission change | IAM policy update | Regular permission audit |

---

## Quick Reference: Symptom-to-Action

| Symptom | Immediate Action | Follow-up |
|---------|------------------|-----------|
| Cluster red | Check unassigned shards | Replace failed nodes |
| High latency | Check hot threads | Scale or optimize |
| Disk > 85% | Extend storage | Review retention |
| JVM > 90% | Scale up nodes | Tune GC settings |
| Snapshot fail | Check OBS permissions | Retry snapshot |
