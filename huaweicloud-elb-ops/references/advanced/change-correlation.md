# Change Correlation — ELB

> **Purpose**: Correlate CTS change events with ELB anomalies for root cause analysis.
> **Extends**: `huaweicloud-skill-generator/references/aiops-best-practices.md` §Change Correlation
> **Version**: 1.0.0
> **Last Updated**: 2026-07-18

---

## 1. CTS Event Types → Fault Type Mapping

| CTS Event Type | Resource Type | Change Impact | Associated Fault Type |
|----------------|---------------|---------------|----------------------|
| LoadBalancer.update | Load Balancer | LB configuration changed | Traffic anomaly |
| LoadBalancer.delete | Load Balancer | LB removed | Service disruption |
| TargetGroup.update | Target Group | Backend target group changed | Backend instance unhealthy |
| TargetGroup.create | Target Group | New target group created | Route misconfiguration |
| TargetGroup.delete | Target Group | Target group removed | Backend unreachable |
| Certificate.update | SSL Certificate | SSL certificate changed | HTTPS connection failure |
| Certificate.create | SSL Certificate | New certificate deployed | TLS handshake failure |
| HealthCheck.update | Health Check | Health check config changed | False positive unhealthy |
| Rule.update | Forwarding Rule | Forwarding rule changed | Routing error |
| Rule.create | Forwarding Rule | New rule added | Traffic misrouted |
| Rule.delete | Forwarding Rule | Rule removed | Service not reachable |
| Listener.update | Listener | Listener config changed | Connection failure |
| Member.add | Backend Member | Member added to pool | Load imbalance |
| Member.remove | Backend Member | Member removed | Traffic spike to remaining |

### 1.1 Time Window Correlation

| Correlation Window | Use Case |
|--------------------|----------|
| 0-15 min before alarm | Immediate cause |
| 15-30 min before alarm | Contributing factor |
| 30-60 min before alarm | Indirect cause |
| 60-120 min before alarm | Background context |

---

## 2. Common Change-Triggered Alarm Patterns

### 2.1 LoadBalancer Update Triggers Traffic Anomaly

```
Pattern: LoadBalancer.update → Traffic distribution changed → Alarm
Detection:
  - CTS event: LoadBalancer.update
  - Time window: 0-15 min after change
  - Expected impact: Redistribution of traffic across backends
  - Alert if: Traffic drops > 30% or error rate > 5% after change
```

### 2.2 TargetGroup Update Triggers Backend Unhealthy

```
Pattern: TargetGroup.update → Backend instance marked unhealthy → Alarm
Detection:
  - CTS event: TargetGroup.update
  - Time window: 0-10 min after change
  - Expected impact: Health check reconfiguration may mark instances unhealthy
  - Alert if: > 50% backend instances unhealthy for > 5 min
```

### 2.3 Certificate Update Triggers HTTPS Connection Failure

```
Pattern: Certificate.update → SSL/TLS handshake failure → Alarm
Detection:
  - CTS event: Certificate.update
  - Time window: 0-5 min after change
  - Expected impact: New certificate propagation delay
  - Alert if: HTTPS error rate > 10% within 5 min of cert change
```

### 2.4 HealthCheck Update Triggers False Positive Unhealthy

```
Pattern: HealthCheck.update → Instances incorrectly marked unhealthy → Alarm
Detection:
  - CTS event: HealthCheck.update
  - Time window: 0-15 min after change
  - Expected impact: Stricter or different health check parameters
  - Alert if: Multiple backends marked unhealthy simultaneously after HC change
```

### 2.5 Rule Update Triggers Routing Error

```
Pattern: Rule.update → Traffic routed to wrong backend → Alarm
Detection:
  - CTS event: Rule.update
  - Time window: 0-10 min after change
  - Expected impact: Forwarding rule path or backend changed
  - Alert if: 5xx error rate > 10% or latency spike > 500ms after rule change
```

---

## 3. Correlation Query Examples

### 3.1 Query CTS Events Before Alarm

```bash
REGION="{{env.HW_REGION_ID}}"
RESOURCE_ID="{{output.loadbalancer_id}}"
ALARM_TIME="{{output.alarm_time}}"
WINDOW_START=$(date -d "$ALARM_TIME - 60 minutes" +%Y-%m-%dT%H:%M:%SZ)
WINDOW_END=$(date -d "$ALARM_TIME + 5 minutes" +%Y-%m-%dT%H:%M:%SZ)

# Query CTS for changes on this ELB
hcloud cts list-traces \
  --region "$REGION" \
  --resource_id "$RESOURCE_ID" \
  --start_time "$WINDOW_START" \
  --end_time "$WINDOW_END" \
  --output json
```

### 3.2 Correlate Events with Alarm

```python
def correlate_change_with_alarm(alarm, cts_events):
    """
    Correlate CTS change events with alarm occurrence.
    Returns list of (event, correlation_score) tuples.
    """
    results = []
    for event in cts_events:
        time_delta = alarm.timestamp - event.timestamp

        # Calculate correlation score
        score = 0.0
        if 0 <= time_delta.minutes <= 5:
            score = 1.0  # High confidence: immediate cause
        elif 5 < time_delta.minutes <= 15:
            score = 0.7  # Medium confidence
        elif 15 < time_delta.minutes <= 30:
            score = 0.4  # Lower confidence
        elif 30 < time_delta.minutes <= 60:
            score = 0.2  # Background context

        results.append((event, score))

    return sorted(results, key=lambda x: x[1], reverse=True)
```

---

## 4. Change Correlation Workflow

```yaml
change_correlation:
  name: "CTS-Based ELB Root Cause Correlation"

  steps:
    - name: collect_alarm_context
      input: alarm_id
      output:
        - alarm_time
        - loadbalancer_id
        - metric_name
        - threshold

    - name: query_cts_events
      input: loadbalancer_id + alarm_time
      output: cts_events[]
      params:
        window_before: 60m
        window_after: 5m

    - name: score_correlation
      input: cts_events + alarm
      output: correlated_events[]
      algorithm: time_distance_based_scoring

    - name: identify_root_cause
      input: correlated_events
      output: root_cause_event
      criteria: highest_score_event

    - name: generate_report
      input: root_cause_event + alarm
      output: correlation_report
```

---

## 5. Compliance Checklist

- [x] At least 5 change-triggered alarm patterns documented
- [x] Time window correlation rules defined
- [x] CTS query examples provided
- [x] Correlation workflow implemented
- [x] Product-specific event types mapped
