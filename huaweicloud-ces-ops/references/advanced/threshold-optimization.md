# Threshold Optimization — L5 Self-Learning

> **Purpose**: Automatic adjustment of alarm thresholds based on historical patterns.
> **Extends**: `self-learning-framework.md`
> **Version**: 1.0.0
> **Last Updated**: 2026-07-18

---

## 1. Threshold Optimization Model

```
New_Threshold = α × Historical_P95 + (1-α) × Current_Threshold

Where:
  α = learning_rate (configurable, default 0.2)
  P95 = 95th percentile of historical values during normal operations
```

### 1.1 Constraints

| Constraint | Value | Rationale |
|------------|-------|-----------|
| Max change per update | ±20% | Prevent drastic changes |
| Minimum stability period | 7 days | Avoid reacting to temporary changes |
| Learning rate (α) | 0.2 | Gradual adjustment |
| Update frequency | Weekly | Allow pattern stabilization |

---

## 2. Threshold Types

### 2.1 Resource Thresholds

| Metric | Default | Min | Max | Unit |
|--------|---------|-----|-----|------|
| CPU usage | 80 | 50 | 95 | % |
| Memory usage | 85 | 60 | 95 | % |
| Disk usage | 90 | 70 | 98 | % |
| Disk I/O | 80 | 50 | 95 | % |
| Network in | 80 | 50 | 95 | % |

### 2.2 Service Thresholds

| Metric | Default | Min | Max | Unit |
|--------|---------|-----|-----|------|
| Response time P99 | 500 | 100 | 2000 | ms |
| Error rate | 1 | 0.1 | 10 | % |
| Availability | 99.9 | 99 | 99.99 | % |
| Slow query time | 5 | 1 | 30 | s |

---

## 3. Optimization Algorithm

### 3.1 Weekly Threshold Update

```python
def compute_optimal_thresholds(skill, metric, historical_data):
    """
    Compute optimal threshold based on historical data.
    """
    # Filter to normal operations only (no incidents)
    normal_data = filter_normal_operations(historical_data, incident_log)

    if len(normal_data) < 100:
        return Current_Threshold  # Not enough data

    # Calculate P95 of normal operations
    p95 = calculate_percentile(normal_data, 95)

    # Calculate business-adjusted threshold
    # (some metrics need buffer for business peaks)
    business_factor = get_business_factor(skill, metric)
    adjusted_p95 = p95 * business_factor

    # Apply learning rate
    current = get_current_threshold(skill, metric)
    alpha = 0.2

    new_threshold = alpha * adjusted_p95 + (1 - alpha) * current

    # Apply constraints
    min_threshold = current * 0.8  # Max 20% decrease
    max_threshold = current * 1.2  # Max 20% increase

    new_threshold = max(min_threshold, min(max_threshold, new_threshold))

    return round(new_threshold, 1)
```

### 3.2 Business Factor

Different services have different tolerance levels:

```python
def get_business_factor(skill, metric):
    """
    Get business adjustment factor for threshold.
    """
    factors = {
        "ecs:cpu_usage": 1.0,
        "ecs:memory_usage": 1.0,
        "rds:cpu_usage": 0.9,         # RDS needs more headroom
        "rds:disk_usage": 0.85,       # Database storage critical
        "rds:connections": 0.8,       # Connection exhaustion severe
        "elb:latency": 1.1,           # Can tolerate slightly higher latency
        "elb:error_rate": 0.9,        # Error rate more sensitive
    }

    return factors.get(f"{skill}:{metric}", 1.0)
```

---

## 4. Anomaly Exclusion

### 4.1 Known Patterns to Exclude

When computing thresholds, exclude periods with:

| Pattern | Detection | Reason |
|---------|-----------|--------|
| Planned maintenance | Incident tagged "planned" | Not abnormal |
| Business peak hours | Time-based pattern | Expected high utilization |
| Known issues | Tagged incident | Already being addressed |
| External attack | Security incident flag | Abnormal but not operational |

### 4.2 Exclusion Algorithm

```python
def filter_normal_operations(data, incident_log):
    """
    Filter out anomalous periods from threshold calculation.
    """
    normal_data = []

    for point in data:
        timestamp = point.timestamp
        value = point.value

        # Check if timestamp overlaps with any incident
        overlapping_incidents = [
            inc for inc in incident_log
            if inc.start <= timestamp <= inc.end
        ]

        if not overlapping_incidents:
            # Check business hours pattern
            if is_business_hours(timestamp) or not is_high_traffic_period(timestamp):
                normal_data.append(point)

    return normal_data
```

---

## 5. Threshold Update Workflow

```
Weekly Trigger
      │
      ├── Collect historical data (past 30 days)
      │       │
      │       ├── CES metric query
      │       ├── Incident log correlation
      │       └── Business calendar data
      │
      ├── Compute optimal thresholds
      │       │
      │       ├── Calculate P95 for each metric
      │       ├── Apply business factor
      │       ├── Apply learning rate
      │       └── Apply constraints
      │
      ├── Validate proposed changes
      │       │
      │       ├── Compare with SLA requirements
      │       ├── Check with on-call team (if > 15% change)
      │       └── Dry-run threshold update
      │
      └── Apply approved thresholds
              │
              ├── Update CES alarm rules
              ├── Log threshold change
              └── Notify stakeholders
```

---

## 6. Auto-Apply vs. Manual Approval

| Change Type | Auto-Apply | Manual Approval Required |
|-------------|------------|--------------------------|
| ±5% change | ✅ | |
| ±5-15% change | ✅ (after 2 consecutive weeks) | |
| ±15-20% change | | ✅ Required |
| > 20% change | | ❌ Rejected, investigate |

---

## 7. Monitoring & Alerts

### 7.1 Threshold Drift Alert

```yaml
alert:
  name: threshold_drift_detected
  condition: |
    abs(current_threshold - historical_optimal) > 30%
  severity: warning
  message: "Threshold {metric} has drifted significantly from optimal"
  action: Review and potentially reset threshold
```

### 7.2 Threshold Update Alert

```yaml
alert:
  name: threshold_updated
  condition: threshold_value_changed
  severity: info
  message: "Threshold {metric} updated from {old} to {new}"
  action: Log for review
```

---

## 8. Compliance Checklist

- [ ] Threshold optimization formula documented
- [ ] Learning rate (α = 0.2) and constraints defined
- [ ] Business factor per skill/metric defined
- [ ] Anomaly exclusion logic implemented
- [ ] Auto-apply vs. manual approval thresholds defined
- [ ] Weekly update workflow documented
- [ ] Threshold drift monitoring
