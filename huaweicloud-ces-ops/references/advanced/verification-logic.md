# Verification Logic — L5 Autonomous Operations

> **Purpose**: Post-execution validation logic for L5 autonomous operations.
> **Extends**: `actor-framework.md` §4 (Verification Logic)
> **Version**: 1.0.0
> **Last Updated**: 2026-07-18

---

## 1. Verification Overview

Verification confirms that an executed action achieved its intended outcome. Failed verification triggers automatic rollback.

```
Execution Result ──▶ Verification Logic ──▶ Verification Result
                              │
                              ├── PASS ──▶ Continue (Complete)
                              │
                              └── FAIL ──▶ Rollback + Escalate
```

---

## 2. Verification Methods

### 2.1 Health Check Verification

**Used For**: Instance start/stop, service restart, scaling operations

```python
def verify_health_check(action, context):
    """
    Verify resource health via health check API.
    """
    health_endpoint = f"{action.resource_endpoint}/health"
    max_attempts = 12  # 60 seconds
    interval = 5

    for attempt in range(max_attempts):
        response = http.get(health_endpoint, timeout=10)

        if response.status == "healthy":
            return VerificationResult(
                passed=True,
                method="health_check",
                details=f"Health check passed after {attempt * interval}s"
            )

        time.sleep(interval)

    return VerificationResult(
        passed=False,
        method="health_check",
        details=f"Health check failed after {max_attempts * interval}s",
        error="Resource did not become healthy within timeout"
    )
```

### 2.2 Metric Threshold Verification

**Used For**: Resource pressure issues (CPU, memory, disk), scaling verification

```python
def verify_metric_threshold(metric_name, threshold, operator, duration_seconds):
    """
    Verify metric is within threshold for a duration.
    operator: "gt" | "lt" | "gte" | "lte"
    """
    ces_client = get_ces_client()
    end_time = current_timestamp()
    start_time = end_time - duration_seconds

    metric_data = ces_client.getMetricData(
        namespace=get_namespace_for_metric(metric_name),
        metric_name=metric_name,
        start_time=start_time,
        end_time=end_time,
        period=60,
        filter="avg"
    )

    values = metric_data["datapoints"]
    if not values:
        return VerificationResult(passed=False, error="No metric data available")

    # Check if all values meet the threshold condition
    passed = all(compare(v, operator, threshold) for v in values)

    return VerificationResult(
        passed=passed,
        method="metric_threshold",
        details={
            "metric": metric_name,
            "threshold": threshold,
            "operator": operator,
            "actual_values": values,
            "duration_checked": f"{duration_seconds}s"
        }
    )
```

### 2.3 State Verification

**Used For**: Configuration changes, security group rules, network changes

```python
def verify_state_change(expected_state, resource_id, state_field="status"):
    """
    Verify resource is in expected state.
    """
    resource = get_resource_details(resource_id)
    actual_state = resource.get(state_field)

    if actual_state == expected_state:
        return VerificationResult(
            passed=True,
            method="state_check",
            details=f"Resource in expected state: {expected_state}"
        )

    return VerificationResult(
        passed=False,
        method="state_check",
        details=f"Expected {expected_state}, got {actual_state}",
        error="Resource state mismatch"
    )
```

### 2.4 Log Pattern Verification

**Used For**: Process restart, service issues, application errors

```python
def verify_log_pattern(expected_pattern, log_group, log_stream, time_window_minutes=5):
    """
    Verify specific log pattern exists (or does not exist) in logs.
    """
    lts_client = get_lts_client()
    end_time = current_timestamp()
    start_time = end_time - timedelta(minutes=time_window_minutes)

    logs = lts_client.queryLogs(
        log_group=log_group,
        log_stream=log_stream,
        start_time=start_time,
        end_time=end_time,
        filter_pattern=expected_pattern
    )

    if logs["count"] > 0:
        return VerificationResult(
            passed=True,
            method="log_check",
            details=f"Found {logs['count']} matching log entries"
        )

    return VerificationResult(
        passed=False,
        method="log_check",
        details="No matching log entries found",
        error="Expected log pattern not found"
    )
```

### 2.5 Combination Verification

**Used For**: Complex scenarios requiring multiple checks

```python
def verify_combination(verification_list):
    """
    Run multiple verifications and require ALL to pass.
    """
    results = []
    for verification in verification_list:
        result = verification.run()
        results.append(result)

    all_passed = all(r.passed for r in results)

    return VerificationResult(
        passed=all_passed,
        method="combination",
        details=[r.details for r in results],
        error="; ".join([r.error for r in results if not r.passed]) if not all_passed else None
    )
```

---

## 3. Verification Time Windows

| Action Category | Default Window | Max Window | Polling Interval |
|-----------------|---------------|------------|------------------|
| Instance start | 60s | 120s | 5s |
| Instance stop | 30s | 60s | 5s |
| Scale up/down | 120s | 300s | 10s |
| Config change | 30s | 60s | 5s |
| Network change | 60s | 120s | 5s |
| Storage expand | 120s | 300s | 10s |
| Service restart | 60s | 120s | 5s |
| Backup/restore | 180s | 600s | 15s |

---

## 4. Verification Failure Handling

### 4.1 Failure Classification

| Failure Type | Action |
|--------------|--------|
| Transient (timeout, network blip) | Retry verification 3x, then rollback |
| Permanent (wrong state, config error) | Immediate rollback |
| Partial (some checks pass) | Review and decide based on severity |

### 4.2 Rollback Trigger Flow

```
Verification Failed
       │
       ├── Classify failure
       │       │
       │       ├── Transient → Retry (max 3)
       │       │       │
       │       │       └── Retry success → PASS
       │       │       └── Retry fail → Rollback
       │       │
       │       └── Permanent → Rollback
       │
       └── Execute Rollback
               │
               ├── Rollback success → Log + Escalate
               └── Rollback fail → Immediate escalation
```

---

## 5. Verification Result Schema

```yaml
verification_result:
  plan_id: string
  action_id: string
  verification_id: string          # UUID
  method: string                   # health_check / metric_threshold / state_check / log_check / combination
  status: string                   # passed / failed / timeout / error
  start_time: timestamp
  end_time: timestamp
  duration_ms: int
  details: dict                    # Method-specific details
  error: string | null             # Error message if failed
  attempts: int                    # Number of verification attempts
  rollback_triggered: bool
  rollback_result: dict | null
```

---

## 6. SLO Impact Verification

### 6.1 SLO Check Before Verification

Before marking verification as passed, check if action affected SLO:

```python
def check_slo_impact(action, context):
    """
    Check if execution impacted SLOs.
    """
    slo_violation = False
    slo_details = []

    # Check availability
    if action.impact == "availability":
        # Query availability metric during action window
        availability = query_availability_metric(
            resource_id=context.resource_id,
            start_time=context.action_start,
            end_time=context.action_end
        )
        if availability < SLO_AVAILABILITY_TARGET:
            slo_violation = True
            slo_details.append(f"Availability: {availability} < {SLO_AVAILABILITY_TARGET}")

    # Check latency
    if action.impact == "latency":
        latency_p99 = query_latency_metric(
            resource_id=context.resource_id,
            start_time=context.action_start,
            end_time=context.action_end
        )
        if latency_p99 > SLO_LATENCY_P99_TARGET:
            slo_violation = True
            slo_details.append(f"Latency P99: {latency_p99} > {SLO_LATENCY_P99_TARGET}")

    return {
        "slo_violation": slo_violation,
        "details": slo_details,
        "should_escalate": slo_violation and action.risk_level in ("High", "Critical")
    }
```

---

## 7. Compliance Checklist

- [ ] All verification methods implemented (health, metric, state, log, combination)
- [ ] Verification time windows defined by action type
- [ ] Failure classification and handling defined
- [ ] Rollback trigger logic implemented
- [ ] Verification result schema documented
- [ ] SLO impact verification included
- [ ] Retry logic for transient failures
