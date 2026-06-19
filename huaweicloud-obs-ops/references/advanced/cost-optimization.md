# OBS FinOps — Storage Cost Optimization Deep Dive

> Advanced FinOps patterns for Object Storage Service.
> Load when designing lifecycle policies, multi-region replication cost,
> or storage-class selection.

## 1. Storage Class Selection

| Class | Best for | Retrieval cost | Min duration |
|-------|----------|----------------|--------------|
| 标准 | frequent access | none | none |
| 低频 | 30+ day archive | per-GB | 30 days |
| 归档 | 90+ day archive | per-GB + restore | 90 days |
| 深度归档 | 180+ day archive | per-GB + restore | 180 days |

## 2. Lifecycle Policy

```text
Standard → LowFreq  : 30 days idle
LowFreq → Archive   : 90 days idle
Archive → Delete    : 7 years (compliance horizon)
```

- Apply per bucket prefix; verify transition costs before rollout
- Document compliance retention before enabling auto-delete

## 3. Cost Anomaly Alerts

| Pattern | Threshold | Action |
|---------|-----------|--------|
| Daily egress > 7d_avg × 2 | sustained 2 days | investigate CDN / download spikes |
| Storage growth > 30% WoW | sustained 1 week | lifecycle review |
| Request count > 7d_avg × 5 | sustained 1 hour | check hot-loop bug |

## 4. Multi-Region Replication Cost

- Cross-region replication billed per GB transferred
- Use bucket policies + lifecycle to avoid replicating cold data
- Document `RTO ≤ 1 hour` for cross-region DR

> **Security-Sensitive**: bucket deletion, lifecycle policy changes, and
> cross-region replication MUST require explicit operator confirmation.
> Public-read buckets MUST never be created without the Security-Sensitive gate.