# Trend Detection — ECS

> **Purpose**: Detect trends, accelerations, and sudden changes in ECS metrics.
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
| Disk I/O | > 10%/min | 5min | Warning |
| Network in | > 50%/min | 2min | Warning |
| Network out | > 50%/min | 2min | Warning |
| Request count | > 50%/min | 2min | Warning |
| Latency | > 10%/min | 3min | Warning |

---

## 3. Detection Examples

### 3.1 Memory Leak Detection

```
Pattern: Memory monotonically increasing
Detection:
  - Check: value[n] > value[n-1] for all n in last 30min
  - Rate: slope > 0.5%/min
  - Severity: Warning if duration > 15min
  - Severity: Critical if duration > 30min
```

### 3.2 Traffic Spike Detection

```
Pattern: Sudden increase in traffic
Detection:
  - Baseline: 7-day average at same hour
  - Current: > 3x baseline
  - Duration: > 2min
  - Severity: Warning
```

### 3.3 Performance Degradation Detection

```
Pattern: Latency gradually increasing
Detection:
  - Slope: > 10%/min
  - Duration: > 5min
  - Severity: Warning
  - Severity: Critical if P99 > 500ms
```

### 3.4 CPU Throttling Detection

```
Pattern: CPU usage suddenly drops then stabilizes (throttling)
Detection:
  - Check: CPU_utilization drops > 20% within 1min
  - Follow-up: Check CPU_credit_usage for credit exhaustion
  - Severity: Warning if throttling duration > 5min
  - Severity: Critical if credits exhausted
```

---

## 4. Compliance Checklist

- [ ] All 4 detection algorithms documented
- [ ] Alert thresholds defined for key metrics
- [ ] Detection examples provided
- [ ] Severity levels specified
