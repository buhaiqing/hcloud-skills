# Chaos Engineering — CTS

> **Purpose**: Document fault injection experiments for CTS resilience verification.
> **Extends**: `huaweicloud-skill-generator/references/chaos-engineering-template.md`
> **Version**: 1.0.0
> **Last Updated**: 2026-07-18

---

## 1. Fault Injection Experiment Design

| Experiment Type | Injection Method | Observed Metrics | Expected Behavior | Termination Condition |
|----------------|-----------------|-----------------|------------------|---------------------|
| Audit log interruption | Disable CTS tracker | Event tracking rate, alert trigger | Events queued or dropped | Event gap >5min |
| Event loss simulation | Delete recent events via API | Event query result, trace integrity | Event gap detected | Trace gap >10 events |
| Tracker configuration error | Set invalid OBS bucket | Tracker status, event delivery rate | Event delivery failure | Delivery failure >5min |
| Sampling rate change | Modify trace sampling | Event volume, query completeness | Reduced visibility | Event volume <50% for >10min |
| OBS bucket unavailable | Delete CTS target bucket | Event delivery success rate | Events buffered | Buffer full >2min |
| Permission revocation | Remove CTS service account | API call success rate | Permission denied | Failure rate >50% for >3min |

## 2. Resilience Score Model

### Scoring Dimensions (0-10 each)

| Dimension | Scoring Criteria | Weight |
|-----------|----------------|--------|
| Fault detection speed | Time from fault to CES alert | 20% |
| Fault isolation ability | Explosion radius (affected events) | 20% |
| Recovery automation | Auto-healing success rate, MTTR | 25% |
| Degradation quality | Event tracking during degradation | 15% |
| Data consistency | Event integrity after recovery | 20% |

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
  name: "cts-audit-log-interruption"
  objective: "Verify CTS handles audit log interruption gracefully"

  preconditions:
    - "CTS tracker enabled with OBS target"
    - "CES alarm configured for tracker status"
    - "Recent events stored for verification"

  steps:
    - inject_fault: "Disable CTS tracker via API"
    - observe_metrics: "Monitor event tracking rate, alert trigger"
    - verify_behavior: "Confirm alert fires, events buffered or queued"
    - rollback_fault: "Re-enable tracker, verify event continuity"

  success_criteria:
    - "Alert triggered within 5min"
    - "No permanent event loss after recovery"
    - "Event sequence integrity verified"

  emergency_rollback:
    - "Re-enable tracker immediately"
    - "Verify OBS bucket accessibility"
    - "Manual event re-trace if needed"
```

## 4. CTS-Specific Experiment Details

### 4.1 Audit Log Interruption (Primary Scenario)

**Objective**: Verify audit log interruption detection and event recovery.

**Injection**:
```bash
# Disable CTS tracker
hcloud CTS DeleteTracker --tracker_name <tracker-name>
```

**Metrics to Monitor**:
- `CTS.EventTrackingRate` via CES
- Tracker status
- OBS event delivery rate

**Expected**: Alert fires, events buffered during outage.

### 4.2 Event Loss Simulation

**Objective**: Verify event gap detection.

**Injection**:
```bash
# Delete recent events
hcloud CTS DeleteEvents --event_ids <event-ids>
```

**Metrics**: Event query result, sequence gap detection.

### 4.3 Tracker Configuration Error

**Objective**: Verify event delivery failure detection.

**Injection**:
```bash
# Modify tracker to point to non-existent bucket
hcloud CTS UpdateTracker --tracker_name <tracker-name> \
  --OBS_PUBLISH_BUCKET <non-existent-bucket>
```

**Metrics**: Event delivery success rate, tracker status.

## 5. Emergency Rollback Procedures

| Scenario | Rollback Action |
|----------|-----------------|
| Tracker disabled | Re-enable tracker immediately |
| Event loss | Verify OBS backup, manual re-trace |
| OBS bucket unavailable | Create new bucket, update tracker |
| Permission revocation | Re-grant CTS service account permissions |

## 6. Compliance Checklist

- [x] ≥5 fault injection experiments designed (6 experiments)
- [x] Resilience scoring model defined (5 dimensions, 20% weights)
- [x] Experiment workflow documented (YAML format)
- [x] Emergency rollback procedures defined (4 scenarios)
