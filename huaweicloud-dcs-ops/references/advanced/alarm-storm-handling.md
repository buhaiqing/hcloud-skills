# Alarm Storm Handling — DCS

> **Purpose**: Guidance for detecting and mitigating alarm storms involving Distributed Cache Service (Redis).
> **Version**: 1.0.0
> **Last Updated**: 2026-07-18

---

## 1. Alarm Storm Detection

DCS alarm storms typically arise from resource exhaustion cascades. Signals come from the CES namespace `SYS.DCS`.

| Pattern | Indicators | Severity |
|---------|-----------|----------|
| OOM risk | `memory_usage` > 90% AND `evicted_keys` > 0, OR `hit_rate` < 70% | Critical |
| Connection exhaustion | `connected_clients` > 80% of max AND `latency` > 2x baseline | Critical |
| Cache breakdown / avalanche | `hit_rate` < 70% AND `expired_keys` surge > 3x baseline | Warning |
| Network saturation | `bytes_out` > 80% of bandwidth | Warning |
| Resource cascade | `cpu` / `memory` / `clients` all > 80% simultaneously | Critical |

Query these signals via `hcloud ces metric-data-query --namespace=SYS.DCS` (one call per metric, e.g. `memory_usage`, `evicted_keys`).

---

## 2. Aggregation Rules

- **Same-window aggregation**: Multiple indicators firing within a 5-minute window AND deviating > 2σ from baseline are merged into a single incident.
- **OOM merge**: `memory_usage` > 90% + `evicted_keys` > 0 are collapsed into a single "OOM warning" to suppress duplicate keyspace eviction noise.
- **Cascade collapse**: When cpu/memory/clients all exceed 80%, attribute to one root cause instead of 3 separate alarms.
- **Baseline reference**: Baselines are derived from the prior 7-day same-time-window via CES; avoid static thresholds that misfire under normal load peaks.

---

## 3. Suppression Rules

| Scenario | Suppression |
|----------|-------------|
| Planned cache flush / migration | Suppress eviction + hit-rate alarms for 2x operation duration |
| Known load test | Suppress connection/latency alarms for test window |
| Confirmed OOM cascade | Suppress `evicted_keys` duplicates once OOM warning fired (15 min, re-evaluate) |

Suppress a CES alarm during maintenance:

```bash
hcloud ces alarm-action modify --alarm_id <alarm-id> --suppress_duration 3600
```

---

## 4. Response Procedures

### Phase 1: Triage (0-5 min)
1. Identify dominant pattern via detection commands above.
2. Confirm whether load is legitimate (peak traffic) or anomalous (attack/leak).

### Phase 2: OOM / Eviction
- Inspect instance state with `hcloud dcs show-instance --instance {{user.instance_id}}`; add capacity or enable `maxmemory-policy` via instance config (delegate to DCS).

### Phase 3: Connection Exhaustion
- Scale max_clients or kill idle connections; verify `connected_clients` trend returns below 80%.

### Phase 4: Post-Incident
- Review CES metrics to confirm storm subsided; add detection rule if pattern was novel.

---

## 5. Delegation Matrix

| Trigger | Delegate To |
|---------|-------------|
| Network saturation / bandwidth limit | `huaweicloud-vpc-ops` |
| Metric gaps / alarm config | `huaweicloud-ces-ops` |
| Cost of scale-up / capacity | `huaweicloud-billing-ops` |
| Permission / key access issues | `huaweicloud-iam-ops` |
| Backend compute pressure | `huaweicloud-ecs-ops` |
| Host-level intrusion on cache nodes | `huaweicloud-hss-ops` |
