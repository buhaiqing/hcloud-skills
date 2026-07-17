# Trend Detection — RDS

> **Purpose**: Detect trends, accelerations, and sudden changes in Relational Database Service metrics.
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
| Connection count | > 20%/min | 5min | Warning |
| QPS | > 30%/min | 3min | Warning |
| Latency | > 10%/min | 3min | Warning |
| Slow queries | > 5/min | 5min | Warning |
| Replication lag | > 1%/min | 2min | Warning |

---

## 3. Detection Examples

### 3.1 Database Connection Leak

```
Pattern: Connection count monotonically increasing
Detection:
  - Check: connections[n] > connections[n-1] for all n in last 30min
  - Rate: slope > 1%/min
  - Severity: Warning if duration > 15min
  - Severity: Critical if duration > 30min OR connections > 80% max
```

### 3.2 Query Performance Degradation

```
Pattern: Latency gradually increasing
Detection:
  - Slope: > 10%/min
  - Duration: > 5min
  - Severity: Warning
  - Severity: Critical if P99 > 1000ms
```

### 3.3 Storage Growth Anomaly

```
Pattern: Disk usage growing faster than normal
Detection:
  - Baseline: 7-day average growth rate at same hour
  - Current: > 3x baseline growth rate
  - Severity: Warning if projected to fill within 24h
  - Severity: Critical if projected to fill within 4h
```

### 3.4 Replication Lag Growth

```
Pattern: Replication lag increasing over time
Detection:
  - Check: lag[n] > lag[n-1] for all n in last 15min
  - Rate: slope > 0.5%/min
  - Severity: Warning if lag > 10s
  - Severity: Critical if lag > 60s
```

---

## 4. Compliance Checklist

- [ ] All 4 detection algorithms documented
- [ ] Alert thresholds defined for key metrics
- [ ] Detection examples provided
- [ ] Severity levels specified
