# CES Self-Healing Operations — Huawei Cloud Cloud Eye Service

> Advanced self-healing patterns for CES alarm management. Load when the agent
> needs post-deployment alarm re-enable or threshold auto-tuning.

## 1. Auto Re-enable Alarms After Deployment

**Context**: During deployments, alarms are often disabled to prevent false alerts from resource restarts. This self-healing flow ensures alarms are automatically re-enabled after deployment completes.

### Pre-flight Checks

| Check | Method | Expected | On Failure |
|-------|--------|----------|------------|
| Deployment status | Check deployment workflow completion signal | `deployment_complete=true` | HALT; deployment still in progress |
| Disabled alarms list | Query alarms with `alarm_enabled=false` | Non-empty list | No action needed; all alarms enabled |
| Deployment window elapsed | Check timestamp since disable | Within grace period (≤ 30 min) | Manual intervention; alarm stuck disabled |

### Execution — CLI

```bash
REGION="{{env.HW_REGION_ID}}"
DEPLOYMENT_ID="{{output.deployment_id}}"
GRACE_PERIOD_MINUTES=30

# List disabled alarms
DISABLED_ALARMS=$(hcloud ces list-alarms --region "$REGION" --alarm-enabled false --output json)

# Re-enable each alarm
for ALARM_ID in $(echo "$DISABLED_ALARMS" | jq -r '.alarms[].alarm_id'); do
  hcloud ces enable-alarm --region "$REGION" --alarm-id "$ALARM_ID"
  # Validate
  ALARM_STATE=$(hcloud ces describe-alarm --region "$REGION" --alarm-id "$ALARM_ID" --query "alarm_enabled")
  [ "$ALARM_STATE" = "true" ] && echo "✅ Re-enabled: $ALARM_ID" || echo "❌ Failed: $ALARM_ID"
done
```

### Execution — Go SDK

```go
// List disabled alarms, re-enable each, validate state
listReq := &model.ListAlarmsRequest{Region: cfg.Region}
listResp, _ := client.ListAlarms(listReq)
for _, alarm := range listResp.Alarms {
    if !alarm.AlarmEnabled {
        client.EnableAlarm(&model.EnableAlarmRequest{
            Region: cfg.Region, AlarmId: alarm.AlarmId,
            Body: &model.EnableAlarmRequestBody{AlarmEnabled: true},
        })
    }
}
```

### Post-execution Validation

| Validation | Method | Expected | Action on Failure |
|------------|--------|----------|-------------------|
| All alarms enabled | List alarms with `alarm_enabled=false` | Empty list | Re-run or manual intervention |
| Alarm evaluation active | Query alarm state | `alarm_enabled=true` | Escalate to incident system |
| Metrics flowing | Query metric data | Non-empty datapoints within 5 min | Check agent connectivity |

### Failure Recovery

| Error | Max retries | Agent Action |
|-------|-------------|--------------|
| `CES.0011` AlarmNotFound | 0 | Alarm deleted during deployment; skip |
| `CES.0016` Unauthorized | 0 | Check IAM permissions |
| API timeout | 3 | Retry with exponential backoff; then escalate |

### Idempotency

Enable alarm is idempotent: enabling an already-enabled alarm returns success. Multiple executions safe.

---

## 2. Auto-adjust Alarm Thresholds

**Context**: Alarms may trigger false positives due to temporary workload spikes. This flow analyzes historical baselines and adjusts thresholds to reduce noise while maintaining sensitivity.

### Pre-flight Checks

| Check | Method | Expected | On Failure |
|-------|--------|----------|------------|
| Historical data available | Query 30-day metric data | ≥ 100 datapoints | HALT; insufficient baseline |
| Current threshold | Describe alarm | Threshold value recorded | Proceed with adjustment |
| Adjustment approved | Policy check or user consent | `auto_adjust_enabled=true` | HALT; manual adjustment only |

### Execution — CLI

```bash
REGION="{{env.HW_REGION_ID}}"
ALARM_ID="{{user.alarm_id}}"
METRIC_NAMESPACE="{{user.metric_namespace}}"
METRIC_NAME="{{user.metric_name}}"
RESOURCE_ID="{{user.resource_id}}"

# Query 30-day historical data
METRIC_DATA=$(hcloud ces query-metric-data \
  --region "$REGION" --metric-namespace "$METRIC_NAMESPACE" --metric-name "$METRIC_NAME" \
  --metric-dimension.0.name "instance_id" --metric-dimension.0.value "$RESOURCE_ID" \
  --from "$(date -d '-30 days' +%s)000" --to "$(date +%s)000" \
  --filter "average,max" --period "3600" --output json)

# Calculate P95 baseline
P95_VALUE=$(echo "$METRIC_DATA" | jq '[.datapoints[].average] | sort | .[int(length * 0.95)]')

# New threshold = P95 + 10% buffer (must not decrease sensitivity)
CURRENT_THRESHOLD=$(hcloud ces describe-alarm --region "$REGION" --alarm-id "$ALARM_ID" --query "threshold")
NEW_THRESHOLD=$(echo "$P95_VALUE * 1.10" | bc | cut -c1-5)

if [ "$NEW_THRESHOLD" -ge "$CURRENT_THRESHOLD" ]; then
  hcloud ces update-alarm --region "$REGION" --alarm-id "$ALARM_ID" --threshold "$NEW_THRESHOLD"
fi
```

### Post-execution Validation

- Verify threshold changed as expected
- Monitor alarm trigger rate over next 24 hours
- Compare false positive rate before/after adjustment
