# Capacity Forecasting — Huawei Cloud ECS

> **Purpose**: Predict resource exhaustion and growth trends for proactive capacity management.
> **Extends**: `huaweicloud-skill-generator/references/aiops-best-practices.md` §14
> **Version**: 1.0.0
> **Last Updated**: 2026-07-18

---

## 1. Prediction Models

### 1.1 Linear Regression (Stable Growth)

```
y = mx + b

Where:
- y = predicted metric value
- m = slope (growth rate per day)
- b = current value

Exhaustion_Date = (Quota_Limit - Current_Value) / m
```

**Use case**: Stable, linear growth patterns (disk usage, connection count)
**Accuracy**: Medium
**Complexity**: Low

### 1.2 Seasonal Decomposition (Periodic)

```
y(t) = Trend(t) + Seasonal(t) + Residual(t)

Where:
- Trend = long-term growth direction
- Seasonal = periodic pattern (weekly/monthly)
- Residual = noise/anomalies
```

**Use case**: Load patterns with clear seasonality
**Accuracy**: High
**Complexity**: Medium

### 1.3 Exponential Smoothing

```
Forecast = α × Last_Value + (1-α) × Previous_Forecast

Where α = smoothing factor (0.1 ~ 0.3)
```

**Use case**: Short-term prediction, trending data
**Accuracy**: High
**Complexity**: Low

---

## 2. ECS-Specific Metrics

| Metric | Namespace | Unit | Quota Reference |
|--------|-----------|------|-----------------|
| `cpu_util` | SYS.ECS | % | ECS CPU utilization quota |
| `mem_usedPercent` | SYS.ECS | % | ECS memory quota |
| `diskUsage_percent` | AGT.ECS | % | EVS disk quota |
| `net_bits_in` | SYS.ECS | bit/s | VPC bandwidth quota |
| `net_bits_out` | SYS.ECS | bit/s | VPC bandwidth quota |
| `load1` | AGT.ECS | - | OS load average |
| `read_iops` | SYS.ECS | count/s | EVS IOPS quota |
| `write_iops` | SYS.ECS | count/s | EVS IOPS quota |

---

## 3. Capacity Planning Workflow

### Step 1: Collect Historical Data

```bash
# Query historical metrics from CES
REGION="{{env.HW_REGION_ID}}"
INSTANCE_ID="[resource_id]"
PERIOD="30d"

# CPU utilization
hcloud ces query-metric-data \
  --region "$REGION" \
  --namespace "SYS.ECS" \
  --metric-name "cpu_util" \
  --dim.0 "instance_id=$INSTANCE_ID" \
  --period 3600 \
  --from "$(( $(date +%s) - 86400 * 30 ))" \
  --to "$(date +%s)" \
  --output json

# Memory usage
hcloud ces query-metric-data \
  --region "$REGION" \
  --namespace "SYS.ECS" \
  --metric-name "mem_usedPercent" \
  --dim.0 "instance_id=$INSTANCE_ID" \
  --period 3600 \
  --from "$(( $(date +%s) - 86400 * 30 ))" \
  --to "$(date +%s)" \
  --output json

# Disk usage
hcloud ces query-metric-data \
  --region "$REGION" \
  --namespace "AGT.ECS" \
  --metric-name "diskUsage_percent" \
  --dim.0 "instance_id=$INSTANCE_ID" \
  --period 3600 \
  --from "$(( $(date +%s) - 86400 * 30 ))" \
  --to "$(date +%s)" \
  --output json
```

### Step 2: Calculate Growth Rate

```python
import numpy as np

def calculate_growth_rate(values, dates):
    """
    Calculate daily growth rate using linear regression.
    Returns slope (growth per day) and R² (confidence).
    """
    x = np.arange(len(values))
    coefficients = np.polyfit(x, values, 1)
    slope = coefficients[0]

    # Calculate R² for confidence
    y_pred = np.polyval(coefficients, x)
    ss_res = np.sum((values - y_pred) ** 2)
    ss_tot = np.sum((values - np.mean(values)) ** 2)
    r_squared = 1 - (ss_res / ss_tot) if ss_tot != 0 else 0

    return slope, r_squared

def predict_exhaustion(current_value, quota_limit, daily_growth_rate):
    """Predict days until resource exhaustion."""
    if daily_growth_rate <= 0:
        return None  # No exhaustion risk

    days_to_exhaustion = (quota_limit - current_value) / daily_growth_rate
    return days_to_exhaustion
```

### Step 3: Detect Seasonality

```python
import numpy as np

def detect_seasonality(values, period=7):
    """
    Detect weekly seasonality using FFT.
    Returns seasonal amplitude and phase.
    """
    if len(values) < period * 2:
        return None  # Insufficient data

    fft = np.fft.fft(values)
    frequencies = np.fft.fftfreq(len(values))

    # Find dominant frequency (exclude DC component)
    half = len(fft) // 2
    dominant_idx = np.argmax(np.abs(fft[1:half])) + 1
    dominant_freq = frequencies[dominant_idx]

    period_detected = int(round(1 / dominant_freq)) if dominant_freq != 0 else None
    return period_detected
```

### Step 4: Generate Capacity Report

```yaml
capacity_report:
  resource_id: "[ECS instance_id]"
  product: "ECS"
  generated_at: "[timestamp]"

  current_usage:
    cpu_util: [current_value]
    mem_usedPercent: [current_value]
    diskUsage_percent: [current_value]
    collected_at: "[timestamp]"

  growth_analysis:
    model: "[linear|seasonal|exponential]"
    daily_growth_rate:
      cpu: [rate]
      memory: [rate]
      disk: [rate]
    r_squared: [confidence]
    trend_direction: "[increasing|decreasing|stable]"

  predictions:
    - metric: "cpu_util"
      predicted_value_at: "[future_date]"
      predicted_value: [value]
      confidence: [0-1]
    - metric: "mem_usedPercent"
      predicted_value_at: "[future_date]"
      predicted_value: [value]
      confidence: [0-1]
    - metric: "diskUsage_percent"
      predicted_value_at: "[future_date]"
      predicted_value: [value]
      confidence: [0-1]

  exhaustion_analysis:
    cpu:
      quota_limit: 100
      days_to_exhaustion: [days]
      exhaustion_date: "[date]"
      risk_level: "[critical|high|medium|low]"
    memory:
      quota_limit: 100
      days_to_exhaustion: [days]
      exhaustion_date: "[date]"
      risk_level: "[critical|high|medium|low]"
    disk:
      quota_limit: 100
      days_to_exhaustion: [days]
      exhaustion_date: "[date]"
      risk_level: "[critical|high|medium|low]"

  recommendations:
    - action: "[scale_up|optimize|cleanup|request_quota]"
      target: "[cpu|memory|disk]"
      estimated_cost: "[cost_impact]"
      priority: "[P0|P1|P2]"
```

---

## 4. Capacity Alert Rules

| Metric | Warning Threshold | Critical Threshold | Recommended Action |
|--------|-----------------|-------------------|-------------------|
| CPU utilization trend | 30 days to >80% | 14 days to >90% | Scale up instance type / optimize workload |
| Memory utilization trend | 30 days to >85% | 14 days to >95% | Optimize memory usage / scale up |
| Disk usage growth | 60 days to >85% | 30 days to >95% | Expand disk / cleanup logs |
| Load average trend | 30 days to >load_15 threshold | 14 days to >2x threshold | Scale horizontally / investigate processes |
| EVS IOPS trend | 60 days to >80% quota | 30 days to >95% quota | Choose higher IOPS disk type |

---

## 5. Model Selection Guide

| Scenario | Recommended Model | Why |
|----------|-----------------|-----|
| Stable, linear growth (disk fill) | Linear Regression | Simple, reliable for steady trends |
| Periodic workload (batch processing) | Seasonal Decomposition | Captures weekly/monthly cycles |
| Bursty traffic (web servers) | Exponential Smoothing | Reacts quickly to changes |
| Complex multi-pattern | Ensemble (weighted average) | Combines multiple models |
| New instance (< 7 days data) | Rule-based | Insufficient data for ML |

---

## 6. ECS-Specific Considerations

### 6.1 Instance Type Limits

| Instance Type | Max CPU Cores | Max Memory (GB) | Max Data Disks |
|---------------|---------------|-----------------|----------------|
| s6 | 8 | 32 | 16 |
| c6 | 64 | 128 | 16 |
| m6 | 64 | 256 | 16 |
| hwc6 | 64 | 192 | 16 |
| hwm6 | 64 | 384 | 16 |

> Query current quotas: `hcloud ecs list-quotas --region {{env.HW_REGION_ID}}`

### 6.2 Scaling Operations

| Operation | Command | Cooldown |
|-----------|---------|----------|
| Vertical scale | `hcloud ecs resize-instance --instance-id X --flavor-id Y` | 5 min |
| Horizontal scale | `hcloud as scaling-group --scaling-group-id X --action ADD` | 3 min |
| Disk expand | `hcloud evs resize-disk --disk-id X --size Y` | 1 min |

---

## 7. Compliance Checklist

- [ ] All 3 prediction models documented
- [ ] Capacity planning workflow implemented
- [ ] Alert rules defined for all key ECS metrics
- [ ] Model selection guide provided
- [ ] Minimum data requirements specified
- [ ] Confidence thresholds defined
- [ ] ECS-specific instance type limits documented
