# EIP AIOps Patterns

> Tier-2 advanced content per **TE-7**: AIOps depth lives here, not in SKILL.md.
> All patterns below emit CES metric + SMN topic + cross-skill delegation.

## Pattern 1: Bandwidth Saturation (Pressure)

| Field | Value |
|---|---|
| Metric | `outgoing_bandwidth / bandwidth.size` |
| Window | 5 min |
| Threshold | warn ≥ 0.8, critical ≥ 0.95 |
| Action | (a) Resize via `huaweicloud-eip-ops` (Operation 6). (b) If traffic is bursty, consider switching to `traffic` mode. |
| Cross-skill | `huaweicloud-ces-ops` (alarm wiring), `huaweicloud-billing-ops` (overage cost) |

## Pattern 2: Burst / DDoS Shape (Spike + Correlation)

| Field | Value |
|---|---|
| Metric | `incoming_bandwidth` p99 vs p50 |
| Window | 10 min |
| Threshold | p99 > 10× p50 |
| Action | (a) Pull `incoming_bandwidth` and `eip_status` series. (b) Delegate to `huaweicloud-ddos-ops` (when present) or `huaweicloud-hss-ops` for mitigation. (c) If EIP is on a 95计费 subscription, check sample distribution. |
| Cross-skill | `huaweicloud-ddos-ops`, `huaweicloud-hss-ops`, `huaweicloud-billing-ops` |

## Pattern 3: Idle EIP (Trend)

| Field | Value |
|---|---|
| Metric | `port_id == null` for ≥7 d AND `bandwidth.size > 0` |
| Window | daily |
| Threshold | 7 days |
| Action | (a) Tag with `purpose: warm-pool` if intentional. (b) Otherwise: candidate for `release-eip` after user confirmation. (c) Cross-skill: cost attribution via `huaweicloud-billing-ops`. |
| Cross-skill | `huaweicloud-billing-ops` |

## Pattern 4: Billing Shock (Trend Anomaly)

| Field | Value |
|---|---|
| Metric | daily EIP cost |
| Window | 7-day rolling median |
| Threshold | 24h cost > 3× 7-day median |
| Action | (a) Cross-check CES `outgoing_bytes` series for that day. (b) If 按流量: confirm burst. (c) If 95计费: check 5-min sample distribution. (d) Delegate to `huaweicloud-billing-ops` for invoice audit. |
| Cross-skill | `huaweicloud-billing-ops`, `huaweicloud-ces-ops` |

## Pattern 5: Bandwidth Pool Imbalance (Cross-EIP Correlation)

| Field | Value |
|---|---|
| Metric | per-EIP `outgoing_bandwidth` within a `WHOLE` shared bandwidth |
| Window | 1h |
| Threshold | one EIP > 80% of pool capacity while siblings < 20% |
| Action | (a) Consider migrating the dominant EIP back to `PER` (独占) sized to its actual load. (b) Or accept and resize the pool. |
| Cross-skill | `huaweicloud-billing-ops` (cost shape), `huaweicloud-ces-ops` (visualize) |

## Cross-Skill Delegation Matrix (AIOps)

| Pattern | EIP | CES | Billing | DDoS | HSS | VPC | NAT | ECS |
|---|:-:|:-:|:-:|:-:|:-:|:-:|:-:|:-:|
| 1 Saturation | ✅ | ✅ | ✅ | | | | | |
| 2 Burst/DDoS | ✅ | ✅ | | ✅ | ✅ | | | |
| 3 Idle EIP | ✅ | | ✅ | | | | | |
| 4 Billing shock | ✅ | ✅ | ✅ | | | | | |
| 5 Pool imbalance | ✅ | ✅ | ✅ | | | | | |

## Knowledge Base Cross-Reference

See `references/knowledge-base.md` (K1–K8) for full root-cause chains and resolutions.
