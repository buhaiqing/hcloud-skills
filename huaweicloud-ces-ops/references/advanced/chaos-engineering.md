# Chaos Engineering — CES

> **Purpose**: Document fault injection experiments for CES (Cloud Eye Service) resilience verification.
> **Extends**: `huaweicloud-skill-generator/references/chaos-engineering-template.md`
> **Version**: 1.0.0
> **Last Updated**: 2026-07-18

---

## 1. Fault Injection Experiment Design

| Experiment Type | Injection Method | Observed Metrics | Expected Behavior | Termination Condition |
|----------------|-----------------|-----------------|------------------|---------------------|
| Metrics ingestion failure | Block CES agent on target | Missing data points, gap detection | Gap alert within 5min | Gap >15min |
| Alarm storm | Trigger bulk alarm threshold | Alarm delivery latency, throttle | Throttle triggers, priority queuing | Delivery latency >60s |
| Alarm channel failure | Disable notification channel | Alarm delivery failure rate | Fallback channel activated | Failure rate 100% for >5min |
| Metric query timeout | Query large time range | API response time, timeout rate | Timeout with retry, cached results | Timeout rate >10% for >10min |
| Storage quota | Exhaust CES metric storage | Data retention, overwrite policy | Oldest data dropped, alert triggered | New data dropped >1min |
| Regional outage | Simulate regional CES endpoint | Cross-region alarm routing | Backup region serves reads | Read unavailability >3min |

## 2. Resilience Score Model

### Scoring Dimensions (0-10 each)

| Dimension | Scoring Criteria | Weight |
|-----------|----------------|--------|
| Fault detection speed | Time from data gap to alarm | 20% |
| Fault isolation ability | Alarm channel explosion radius | 20% |
| Recovery automation | Auto-routing, retry, fallback success | 25% |
| Degradation quality | Alarm delivery during channel failure | 15% |
| Data consistency | Metric data integrity after recovery | 20% |

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
  name: "ces-metrics-ingestion-failure"
  objective: "Verify metrics gap detection within 5 minutes"

  preconditions:
    - "CES monitoring agent installed on target ECS"
    - "Alarm rule configured for metric gap"
    - "Alternative notification channel available"

  steps:
    - inject_fault: "Block CES agent: systemctl stop ces-agent"
    - observe_metrics: "Monitor metric stream via CES console"
    - verify_behavior: "Confirm gap alert fires ≤ 5min"
    - rollback_fault: "Restart ces-agent, verify data stream resumes"

  success_criteria:
    - "Gap alert fires within 5 minutes"
    - "Alarm delivery ≤ 60s after detection"
    - "No data gap on agent restart (backfill)"

  emergency_rollback:
    - "Force restart ces-agent"
    - "Manually push buffered metrics if available"
    - "Validate data continuity post-recovery"
```

## 4. CES-Specific Experiment Details

### 4.1 Metrics Ingestion Failure (Primary Scenario)

**Objective**: Verify metric gap detection and alarm latency.

**Injection**:
```bash
# Stop CES monitoring agent on target
systemctl stop ces-agent
# Block port 19999 outbound (CES agent port)
iptables -A OUTPUT -p tcp --dport 19999 -j REJECT
```

**Metrics to Monitor**:
- `ces_agent_status` custom metric
- Metric data gap in CES console
- Alarm rule evaluation status

**Expected**: Gap alert within 5 minutes, delivery within 60 seconds.

### 4.2 Alarm Storm & Throttling

**Objective**: Verify CES throttling and priority queuing under alarm storm.

**Injection**:
```bash
# Trigger multiple alarm rules simultaneously
# Use CES API to create high-frequency alarm actions
hcloud CES AlarmRule CreateAlarm --alarm-type multi
```

**Metrics**: Alarm delivery latency, throttle count, queue depth.

### 4.3 Alarm Channel Failure & Fallback

**Objective**: Verify fallback to backup notification channel.

**Injection**:
```bash
# Disable primary notification channel (SMN topic)
hcloud SMN DeleteTopic --topic-urn <primary-topic-urn>
```

**Metrics**: Alarm delivery failure rate, fallback activation time.

### 4.4 Metric Query Timeout

**Objective**: Verify query timeout handling and cache fallback.

**Injection**:
```bash
# Query large time range that exceeds timeout
hcloud CES MetricData Query --namespace <namespace> \
  --start_time 0 --end_time $(date +%s) --period 60
```

**Metrics**: API response time, timeout error rate, cache hit rate.

## 5. Emergency Rollback Procedures

| Scenario | Rollback Action |
|----------|----------------|
| Agent restart fails | Reinstall CES agent, re-register instance |
| Alarm storm persists | Enable throttling rule, acknowledge stale alarms |
| Notification channel down | Manually activate backup SMN topic |
| Data gap persists after agent restart | Contact CES support for backfill |
| Query timeout causes downstream failure | Implement circuit breaker, use cached metrics |

## 6. Compliance Checklist

- [x] ≥5 fault injection experiments designed (6 experiments)
- [x] Resilience scoring model defined (5 dimensions, 20% weights)
- [x] Experiment workflow documented (YAML format)
- [x] Emergency rollback procedures defined (5 scenarios)
