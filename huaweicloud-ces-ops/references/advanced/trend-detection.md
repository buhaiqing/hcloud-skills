# Trend Detection — CES

> **Purpose**: Detect trends, accelerations, and sudden changes in Cloud Eye Service metrics.
> **Version**: 1.0.0
> **Last Updated**: 2026-07-18

---

## 1. Trend Detection Algorithms

### 1.1 Slope Detection

```
slope = (y2 - y1) / (x2 - x1)

Where:
- y = metric value
- x = time
- slope = rate of change per unit time

Alert if: |slope| > threshold AND持续 > duration
```

### 1.2 Acceleration Detection

```
acceleration = (slope2 - slope1) / (t2 - t1)

Where:
- slope1 = previous slope
- slope2 = current slope
- acceleration = change in rate of change

Alert if: acceleration > threshold (趋势加速)
Alert if: acceleration < -threshold (趋势减缓)
```

### 1.3 Sudden Change Detection

```
z = (x - μ) / σ

Where:
- μ = historical mean
- σ = historical standard deviation
- z = standard score

Alert if: |z| > 3 (3-sigma rule)
```

### 1.4 Seasonal Decomposition

```
y(t) = Trend(t) + Seasonal(t) + Residual(t)

Alert if: Residual(t) > 3σ (异常值)
```

---

## 2. Alert Thresholds

| Metric | Slope Threshold | Duration | Severity |
|--------|----------------|----------|----------|
| CPU usage | > 5%/min | 5min | Warning |
| Memory usage | > 2%/min | 10min | Warning |
| Disk usage | > 1%/hour | 30min | Warning |
| Request count | > 50%/min | 2min | Warning |
| Latency | > 10%/min | 3min | Warning |
| Error rate | > 5%/min | 2min | Warning |
| Availability | < -1%/min | 5min | Warning |

---

## 3. Detection Examples

### 3.1 Alarm Storm Detection

```
Pattern: Sudden spike in alarm volume
Detection:
  - Baseline: 7-day average alarms per 5min at same hour
  - Current: > 5x baseline
  - Duration: > 2min
  - Severity: Critical (alarm storm)
```

### 3.2 Metric Anomaly Trend

```
Pattern: Metric showing consistent deviation from baseline
Detection:
  - Check: Residual(t) > 2σ for last 30min
  - Trend: Check if residual is increasing
  - Severity: Warning if consistent deviation > 15min
  - Severity: Critical if consistent deviation > 30min
```

### 3.3 Service Health Degradation

```
Pattern: Multiple metrics degrading simultaneously
Detection:
  - Check: 3+ metrics showing negative slope simultaneously
  - Correlation: Metrics belong to same service
  - Severity: Warning if duration > 5min
  - Severity: Critical if duration > 10min
```

### 3.4 Recovery Detection

```
Pattern: Metrics returning to normal after anomaly
Detection:
  - Check: Slope reversal from negative to positive
  - Check: Values approaching baseline (|z| < 2)
  - Severity: Info (recovery notification)
```

---

## 4. Compliance Checklist

- [ ] All 4 detection algorithms documented
- [ ] Alert thresholds defined for key metrics
- [ ] Detection examples provided
- [ ] Severity levels specified
