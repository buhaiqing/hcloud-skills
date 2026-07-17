# Prediction Alerts — L5 Predictive Maintenance

> **Purpose**: Alerting on predicted anomalies before they cause incidents.
> **Extends**: `prediction-service.md`
> **Version**: 1.0.0
> **Last Updated**: 2026-07-18

---

## 1. Alert Types

| Alert Type | Trigger | Severity |
|------------|---------|----------|
| `PREDICTION_THRESHOLD_EXCEEDED` | Forecast > threshold | warning / critical |
| `PREDICTION_ANOMALY` | Anomaly detected in forecast | warning / critical |
| `PREDICTION_ACCURACY_LOW` | Model accuracy < 70% | warning |

---

## 2. Alert Rules

### 2.1 Threshold Exceeded Alert

```yaml
alert_rule:
  name: prediction_threshold_exceeded
  condition: |
    any(forecast_values > threshold)
  forecast_window: 7 days
  evaluation: |
    For each resource, check if any forecasted value
    exceeds the defined threshold for that metric.
  severity_mapping: |
    forecast > threshold * 1.1 → critical
    forecast > threshold → warning
```

### 2.2 Anomaly Prediction Alert

```yaml
alert_rule:
  name: prediction_anomaly_detected
  condition: |
    anomaly_detection(forecast_values, threshold=3) returns anomalies
  severity: warning
  message: |
    Predicted anomaly detected for {resource_id} {metric}:
    Value {value} expected at {time}, which deviates {deviation}σ from normal
```

---

## 3. Alert Payload

```json
{
  "alert_type": "PREDICTION_THRESHOLD_EXCEEDED",
  "alert_id": "alert-xxxxx",
  "severity": "warning",
  "resource_id": "ecs-xxxxx",
  "metric": "cpu_util",
  "predicted_value": 92.5,
  "threshold": 85.0,
  "predicted_time": "2026-07-20T14:00:00Z",
  "confidence": 0.85,
  "forecast_values": [75.2, 78.1, 82.3, 88.5, 92.5, 89.2, 85.1],
  "recommended_action": "Scale up ECS instance before threshold exceeded",
  "auto_action_candidates": ["ECS-A02", "ECS-A03"]
}
```

---

## 4. Integration with CES

Predicted alerts are sent to CES just like normal alarms:

```python
def create_prediction_alert(alert_payload):
    """
    Create CES alarm from prediction alert.
    """
    ces_client.createAlarm({
        "alarm_name": f"pred-{alert_payload.resource_id}-{alert_payload.metric}",
        "alarm_type": "prediction",
        "resource_id": alert_payload.resource_id,
        "metric": alert_payload.metric,
        "condition": f"{alert_payload.metric} > {alert_payload.threshold}",
        "severity": alert_payload.severity,
        "alarm_desc": alert_payload.message,
        "custom_labels": {
            "prediction_id": alert_payload.prediction_id,
            "auto_action_candidates": ",".join(alert_payload.auto_action_candidates)
        }
    })
```

---

## 5. Alert Accuracy Tracking

```python
def track_prediction_accuracy(prediction_id):
    """
    Track whether predicted alerts actually materialized.
    """
    prediction = get_prediction(prediction_id)

    # Wait for forecast window to pass
    wait(forecast_days)

    # Check if actual values exceeded threshold
    actual_values = get_actual_metrics(
        resource_id=prediction.resource_id,
        metric=prediction.metric,
        start_time=prediction.created_at,
        end_time=prediction.created_at + forecast_days
    )

    exceeded = any(actual_values > prediction.threshold)

    # Update prediction record
    update_prediction(prediction_id, {
        "alert_materialized": exceeded,
        "actual_max": max(actual_values),
        "prediction_accuracy_checked": True
    })

    # Update model accuracy
    if exceeded == was_predicted:
        increment_model_accuracy_score(prediction.model)
    else:
        decrement_model_accuracy_score(prediction.model)
```

---

## 6. Compliance Checklist

- [ ] Threshold exceeded alert rule defined
- [ ] Anomaly prediction alert rule defined
- [ ] Alert payload schema documented
- [ ] CES integration for predicted alerts
- [ ] Prediction accuracy tracking
