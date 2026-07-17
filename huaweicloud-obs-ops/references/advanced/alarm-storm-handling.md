# Alarm Storm Handling — OBS

> **Purpose**: Handle alarm storms for OBS buckets during traffic spikes or incidents.
> **Version**: 1.0.0
> **Last Updated**: 2026-07-18

---

## 1. Alarm Storm Detection

| Threshold | Time Window | Severity |
|-----------|-------------|----------|
| > 10 alarms | 5 min | Warning |
| > 50 alarms | 5 min | Critical |
| > 50% buckets affected | Any | Critical |

## 2. Aggregation Rules

| Original Alarm | Aggregated To |
|---------------|---------------|
| Multiple bucket throttling | "OBS Throttling Storm" |
| Multiple bucket latency | "OBS Latency Storm" |
| Bucket quota + bandwidth | "OBS Capacity Storm" |

## 3. Suppression Rules

| Condition | Action |
|-----------|--------|
| Alarm rate > 10/min | Enable aggregation |
| Same bucket 3+ alarms | Suppress, count only |
| Cross-bucket correlation | Send summary only |
