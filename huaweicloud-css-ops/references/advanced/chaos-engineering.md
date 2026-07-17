# Chaos Engineering — CSS

> **Purpose**: Document fault injection experiments for CSS resilience verification.
> **Extends**: `huaweicloud-skill-generator/references/chaos-engineering-template.md`
> **Version**: 1.0.0
> **Last Updated**: 2026-07-18

---

## 1. Fault Injection Experiment Design

| Experiment Type | Injection Method | Observed Metrics | Expected Behavior | Termination Condition |
|----------------|-----------------|-----------------|------------------|---------------------|
| Shard loss | Stop data node instance | Cluster health, search success rate | Auto-shard reallocation | Search failure >5% for >3min |
| JVM OOM | Exhaust JVM heap via stress-ng | JVM heap usage, node status | JVM restart, node recovery | Node unavailable >2min |
| Index corruption | Delete index files manually | Index health, search results | Snapshot restore, index rebuild | Data loss detected |
| Search timeout | Inject query delay 30s | Search latency, timeout rate | Timeout retry, circuit breaker | Timeout rate >20% for >5min |
| Disk pressure | Fill disk to 90% | Disk usage, write latency | Alert at 80%, write throttling | Write latency >1s for >2min |
| Network latency | Inject 500ms delay via TC | Node-to-node latency, cluster health | Slow operation timeout | Cluster unstable >3min |

## 2. Resilience Score Model

### Scoring Dimensions (0-10 each)

| Dimension | Scoring Criteria | Weight |
|-----------|----------------|--------|
| Fault detection speed | Time from fault to CES alert | 20% |
| Fault isolation ability | Explosion radius (affected shards) | 20% |
| Recovery automation | Auto-healing success rate, MTTR | 25% |
| Degradation quality | Search availability during degradation | 15% |
| Data consistency | Data integrity after recovery (translog) | 20% |

### Resilience Grades

| Score | Grade | Recommendation |
|-------|-------|---------------|
| 8-10 | A (Excellent) | Regular chaos validation, maintain |
| 6-8 | B (Good) | Supplement missing fault scenarios |
| 4-6 | C (Fair) | Increase self-healing, improve degradation |
| 0-4 | D (Weak) | Prioritize critical resilience gaps |

## 3. Chaos Experiment Workflow

```yaml
chaos_experiment:
  name: "css-shard-loss"
  objective: "Verify CSS handles shard loss with auto-recovery"

  preconditions:
    - "CSS cluster with ≥3 data nodes"
    - "CES alarm configured for cluster health"
    - "Index with replica ≥1"

  steps:
    - inject_fault: "Stop data node instance via API"
    - observe_metrics: "Monitor shard allocation via CES"
    - verify_behavior: "Confirm auto-shard reallocation within 5min"
    - rollback_fault: "Restart original data node"

  success_criteria:
    - "Cluster health restored within 5min"
    - "No search failures during recovery"
    - "All shards allocated"

  emergency_rollback:
    - "Restart affected data node"
    - "Manual shard allocation if auto fails"
    - "Restore from snapshot if data loss"
```

## 4. CSS-Specific Experiment Details

### 4.1 Shard Loss (Primary Scenario)

**Objective**: Verify CSS handles data node failure with automatic shard reallocation.

**Injection**:
```bash
# Stop data node instance
hcloud CSS StopClusters --cluster_id <cluster-id> --node_id <node-id>
```

**Metrics to Monitor**:
- `CSS.ClusterHealthStatus` via CES
- `CSS.ShardAllocationStatus`
- Search request success rate

**Expected**: Cluster reallocates shards to remaining nodes within 5min.

### 4.2 JVM OOM

**Objective**: Verify JVM restart and index recovery.

**Injection**:
```bash
# Exhaust JVM heap on data node
ssh <node> "stress-ng --vm 1 --vm-bytes 90% --timeout 60s"
```

**Metrics**: JVM heap used%, node status, restart time.

### 4.3 Index Corruption

**Objective**: Verify snapshot restore and index rebuild.

**Injection**:
```bash
# Delete index files (simulate corruption)
ssh <node> "rm -rf /data/css/nodes/0/indices/<index-uid>"
```

**Metrics**: Index health status, search results, restore time.

## 5. Emergency Rollback Procedures

| Scenario | Rollback Action |
|----------|-----------------|
| Shard reallocation timeout | Manual shard reroute, restart cluster |
| JVM restart failure | Force restart data node, increase heap |
| Index corruption | Restore from latest snapshot |
| Disk full | Emergency snapshot, expand disk |

## 6. Compliance Checklist

- [x] ≥5 fault injection experiments designed (6 experiments)
- [x] Resilience scoring model defined (5 dimensions, 20% weights)
- [x] Experiment workflow documented (YAML format)
- [x] Emergency rollback procedures defined (4 scenarios)
