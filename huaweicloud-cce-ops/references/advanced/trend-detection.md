# Trend Detection — CCE

> **Purpose**: Detect trends, accelerations, and sudden changes in Cloud Container Engine metrics.
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
| Pod count | > 10%/min | 3min | Warning |
| Network I/O | > 50%/min | 2min | Warning |
| Node count | > 5%/min | 5min | Warning |
| Pending pods | > 5/min | 5min | Warning |
| Restart count | > 3/min | 5min | Warning |

---

## 3. Detection Examples

### 3.1 Pod Eviction Trend

```
Pattern: Increasing pod restarts or evictions
Detection:
  - Check: restart_count[n] > restart_count[n-1] for last 30min
  - Rate: slope > 0.5/min
  - Severity: Warning if restarts > 10 in 30min
  - Severity: Critical if restarts > 30 in 30min OR node unavailable
```

### 3.2 Scaling Event Detection

```
Pattern: Sudden change in pod count (scaling event)
Detection:
  - Baseline: 7-day average pod count at same hour
  - Current: > 2x or < 0.5x baseline
  - Duration: > 2min
  - Severity: Info (scaling event detected)
```

### 3.3 Resource Saturation Trend

```
Pattern: CPU/Memory usage approaching limits
Detection:
  - Check: usage > 80% AND slope > 1%/min
  - Duration: > 10min
  - Severity: Warning if projected to reach 95% within 1h
  - Severity: Critical if projected to reach 95% within 15min
```

### 3.4 Node Pool Imbalance

```
Pattern: Pod distribution becoming uneven across nodes
Detection:
  - Check: std(pod_count_per_node) > threshold
  - Trend: std increasing over time
  - Severity: Warning if max_nodes > 2x min_nodes
  - Severity: Critical if any node > 90% utilization
```

---

## 4. Compliance Checklist

- [ ] All 4 detection algorithms documented
- [ ] Alert thresholds defined for key metrics
- [ ] Detection examples provided
- [ ] Severity levels specified
