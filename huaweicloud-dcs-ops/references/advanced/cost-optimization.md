# DCS FinOps — Cost Optimization Deep Dive

> Advanced FinOps patterns for Distributed Cache Service.
> Load when designing right-sizing, eviction policies, or capacity-planning
> recommendations.

## 1. Idle Detection

- CPU < 5% sustained 7 days → over-provisioned (downsize)
- Memory < 20% sustained 14 days → right-size candidate
- Zero connections for 3+ days → decommission or reuse

## 2. Right-Sizing Matrix

| Avg CPU (7d) | Avg Memory (7d) | Recommendation | Expected savings |
|--------------|-----------------|----------------|------------------|
| < 20%        | < 30%           | downgrade flavor | 30-60% |
| < 20%        | > 80%           | switch to memory-optimized | 10-20% |
| > 80%        | < 50%           | switch to compute-optimized | — |
| > 80%        | > 80%           | upgrade flavor or shard | — |
| Spiky (max > 3× avg) | — | burst plan + auto-scaling | 20-50% |

## 3. Big Key / Hot Key Detection

- `redis-cli --bigkeys` weekly; flag keys > 10 MB
- `INFO COMMANDSTATS` for hot key candidates
- Migrate hot keys to dedicated instance or shard

## 4. Cost Anomaly Alerts

| Pattern | Threshold | Severity |
|---------|-----------|----------|
| Daily cost > 7d_avg × 1.5 | sustained 2 days | Critical |
| Memory pressure > 90% | sustained 30 min | Warning |
| Connection usage > 80% of max | sustained 5 min | Warning |

> **Security-Sensitive**: `ResetPassword`, `DeleteInstance`, and `ResizeInstance`
> MUST require explicit operator confirmation. Document blast radius before
> resizing production clusters.