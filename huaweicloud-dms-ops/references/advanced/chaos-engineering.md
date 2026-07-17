# Chaos Engineering — DMS

> **Purpose**: Document fault injection experiments for DMS resilience verification.
> **Extends**: `huaweicloud-skill-generator/references/chaos-engineering-template.md`
> **Version**: 1.0.0
> **Last Updated**: 2026-07-18

---

## 1. Fault Injection Experiment Design

| Experiment Type | Injection Method | Observed Metrics | Expected Behavior | Termination Condition |
|----------------|-----------------|-----------------|------------------|---------------------|
| Queue full | Fill queue to max capacity | Producer block time, queue depth | Producer backpressure, DLQ | Queue full >5min |
| Consumer group failure | Stop consumer instances | Consumer lag, message processing rate | Rebalance + redelivery | Lag >10000 for >10min |
| Partition rebalance | Remove broker from cluster | Partition reassignment time | Auto-rebalance | Rebalance >5min |
| Network partition | Block port via security group | Producer/consumer connectivity | Retry with backoff | Failure rate >20% for >3min |
| DLQ overflow | Flood DLQ with failed messages | DLQ depth, message loss | Alert triggered | DLQ full >2min |
| Producer timeout | Inject network delay 5s | Producer success rate, latency | Retry + circuit breaker | Failure rate >30% for >2min |

## 2. Resilience Score Model

### Scoring Dimensions (0-10 each)

| Dimension | Scoring Criteria | Weight |
|-----------|----------------|--------|
| Fault detection speed | Time from fault to CES alert | 20% |
| Fault isolation ability | Explosion radius (affected messages) | 20% |
| Recovery automation | Auto-healing success rate, MTTR | 25% |
| Degradation quality | Message delivery during degradation | 15% |
| Data consistency | Message integrity after recovery | 20% |

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
  name: "dms-queue-full"
  objective: "Verify DMS handles queue full scenario gracefully"

  preconditions:
    - "DMS queue with DLQ configured"
    - "CES alarm configured for queue depth"
    - "Consumer group with multiple instances"

  steps:
    - inject_fault: "Fill queue to max capacity via test producer"
    - observe_metrics: "Monitor queue depth, DLQ, producer block time"
    - verify_behavior: "Confirm producer backpressure + DLQ flow"
    - rollback_fault: "Scale consumers, drain queue"

  success_criteria:
    - "Producer backpressure triggered"
    - "DLQ captures undeliverable messages"
    - "No message loss after recovery"

  emergency_rollback:
    - "Scale consumer instances"
    - "Enable DLQ drain"
    - "Clear queue if needed"
```

## 4. DMS-Specific Experiment Details

### 4.1 Queue Full (Primary Scenario)

**Objective**: Verify producer backpressure and DLQ flow when queue reaches capacity.

**Injection**:
```bash
# Rapidly produce messages until queue is full
hcloud DMS PublishMessage --queue_id <queue-id> --message "test" --count 10000
```

**Metrics to Monitor**:
- `DMS.QueueDepth` via CES
- `DMS.ProducerBlockTime`
- `DMS.DLQDepth`

**Expected**: Producer blocks, DLQ captures overflow messages.

### 4.2 Consumer Group Failure

**Objective**: Verify consumer group rebalance and message redelivery.

**Injection**:
```bash
# Stop consumer instances
hcloud ECS StopServers --instance_ids <consumer-instance-ids> --force
```

**Metrics**: Consumer lag, rebalance time, message redelivery count.

### 4.3 Network Partition

**Objective**: Verify producer/consumer retry behavior during network issues.

**Injection**:
```bash
# Block DMS port via security group (simulate network partition)
hcloud VPC CreateSecurityGroupRule --security_group_id <sg-id> \
  --direction egress --remote_ip_prefix 0.0.0.0/0 --protocol tcp \
  --port 5672 --description "CHAOS: Block DMS port"
```

**Metrics**: Connection failure rate, retry count, message latency.

## 5. Emergency Rollback Procedures

| Scenario | Rollback Action |
|----------|-----------------|
| Queue full persists | Scale consumers, enable DLQ drain |
| Consumer rebalance timeout | Manual rebalance trigger |
| DLQ overflow | Expand DLQ capacity, drain DLQ |
| Network partition persists | Remove blocking SG rule |

## 6. Compliance Checklist

- [x] ≥5 fault injection experiments designed (6 experiments)
- [x] Resilience scoring model defined (5 dimensions, 20% weights)
- [x] Experiment workflow documented (YAML format)
- [x] Emergency rollback procedures defined (4 scenarios)
