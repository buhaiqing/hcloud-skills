# Chaos Engineering — ECS

> **Purpose**: Document fault injection experiments for ECS resilience verification.
> **Extends**: `huaweicloud-skill-generator/references/chaos-engineering-template.md`
> **Version**: 1.0.0
> **Last Updated**: 2026-07-18

---

## 1. Fault Injection Experiment Design

| Experiment Type | Injection Method | Observed Metrics | Expected Behavior | Termination Condition |
|----------------|-----------------|-----------------|------------------|---------------------|
| Instance failure | Stop primary ECS instance | HA failover time, availability | ≤ 60s failover, ≥99.9% availability | Availability <99% for >5min |
| AZ failure | Block cross-AZ traffic via SG rule | Cross-AZ request success rate | Degraded but not failed | Success rate <50% for >2min |
| Disk failure | Fill system disk to 95% | Write success rate, alert trigger | Alert triggered at 80% threshold | Write failure >1min |
| CPU spike | Stress test CPU to 100% | AS scaling time, request latency | 5min scaling, latency <500ms | Latency >5s for >10min |
| Network latency | Inject 500ms delay via TC qdisc | Request round-trip time, timeout rate | Timeout retry, circuit breaker | Failure rate >10% for >3min |
| Memory pressure | Exhaust memory via stress-ng | OOM events, process restart | Auto-healing restart within 60s | OOM persists >2min |

## 2. Resilience Score Model

### Scoring Dimensions (0-10 each)

| Dimension | Scoring Criteria | Weight |
|-----------|----------------|--------|
| Fault detection speed | Time from fault to CES alert | 20% |
| Fault isolation ability | Explosion radius (affected instances) | 20% |
| Recovery automation | Auto-healing success rate, MTTR | 25% |
| Degradation quality | Availability during degradation | 15% |
| Data consistency | Data integrity after recovery (EVS快照) | 20% |

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
  name: "ecs-instance-failure"
  objective: "Verify ECS HA failover within 60s"

  preconditions:
    - "ECS instance in HA-enabled AS group"
    - "CES alarm configured for instance health"
    - "Backup snapshot available"

  steps:
    - inject_fault: "Stop primary ECS instance via API"
    - observe_metrics: "Monitor failover time via CES"
    - verify_behavior: "Confirm secondary takes over ≤ 60s"
    - rollback_fault: "Restart original instance"

  success_criteria:
    - "Failover completes ≤ 60s"
    - "Availability ≥ 99.9% during failover"
    - "No data loss (verified via EVS snapshot)"

  emergency_rollback:
    - "Force restart original instance"
    - "Detach secondary from AS group if needed"
    - "Restore from EVS snapshot if data corruption"
```

## 4. ECS-Specific Experiment Details

### 4.1 Instance Failure (Primary Scenario)

**Objective**: Verify AS group handles primary instance failure gracefully.

**Injection**:
```bash
# Stop primary instance
hcloud ECS StopServers --instance-ids <primary-id> --force
```

**Metrics to Monitor**:
- `ECS.InstanceHaFailoverTime` via CES
- `ECS.InstanceStatus` state transitions
- AS group scaling events

**Expected**: AS automatically provisions replacement, failover ≤ 60s.

### 4.2 AZ Failure

**Objective**: Verify cross-AZ traffic isolation.

**Injection**:
```bash
# Block cross-AZ security group rule (simulate AZ network partition)
hcloud VPC CreateSecurityGroupRule --security-group-id <sg-id> \
  --direction ingress --remote-ip-prefix 0.0.0.0/0 --protocol tcp \
  --port 3306 --description "CHAOS: Block cross-AZ"
```

**Metrics**: Cross-AZ request success rate, latency distribution.

### 4.3 Disk Pressure

**Objective**: Verify disk alert triggers and write degradation.

**Injection**:
```bash
# Fill disk to 95% on target instance
ssh <instance> "dd if=/dev/zero of=/tmp/fill bs=1 count=$(( $(df / | tail -1 | awk '{print $2}') * 95 / 100 ))"
```

**Metrics**: `sys_disk_usage`, write latency, alert firing time.

## 5. Emergency Rollback Procedures

| Scenario | Rollback Action |
|----------|----------------|
| Failover timeout | Force restart original instance, detach replacement |
| Data corruption | Restore from latest EVS snapshot |
| AS scaling storm | Set AS group to manual mode, drain scaling requests |
| Network partition persists | Remove blocking SG rule, verify connectivity |

## 6. Compliance Checklist

- [x] ≥5 fault injection experiments designed (6 experiments)
- [x] Resilience scoring model defined (5 dimensions, 20% weights)
- [x] Experiment workflow documented (YAML format)
- [x] Emergency rollback procedures defined (4 scenarios)
