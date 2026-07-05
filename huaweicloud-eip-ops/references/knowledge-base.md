# EIP Knowledge Base — Fault Patterns

## Pattern K1: Idle EIP Cost Leak

| Symptom | EIP allocated weeks ago, never bound, bill keeps growing |
|---|---|
| Root Cause | `release-eip` was never called; 按带宽 / 按流量 both bill even when unbound |
| Diagnosis | `hcloud eip list` → filter `port_id == null` → compare to create date |
| Resolution | (a) If truly unused: `release-eip` after user confirmation. (b) If reserved: tag with `purpose: warm-pool` and document in `example-config.yaml` |
| Cross-skill | `huaweicloud-billing-ops` — cost attribution per tag |

## Pattern K2: Bandwidth Saturation Without Auto-Scale

| Symptom | Latency spikes coincide with bandwidth %; users report slowness |
|---|---|
| Root Cause | EIP is `bandwidth` mode and `bandwidth.size` is fixed; no alarm → no resize |
| Diagnosis | CES metric `outgoing_bandwidth / bandwidth.size` for last 24h |
| Resolution | (a) Add CES alarm at 80% (warning) / 95% (critical). (b) Manual resize during off-peak. (c) Consider switching to `traffic` mode for spiky workloads |
| Cross-skill | `huaweicloud-ces-ops` (alarm wiring), `huaweicloud-billing-ops` (overage cost) |

## Pattern K3: EIP Still Billed After ECS Deleted

| Symptom | ECS gone but `outgoing_bytes` continues to register on a now-orphaned EIP |
|---|---|
| Root Cause | ECS delete did not unbind / release EIP first |
| Diagnosis | `hcloud eip list` → EIP `port_id` set but ECS no longer exists |
| Resolution | `unbind` then `release` (both with safety gate). Audit: ensure ECS delete runbook unbind-first. |
| Cross-skill | `huaweicloud-ecs-ops` (lifecycle order) |

## Pattern K4: Shared Bandwidth Pool Empty After Cleanup

| Symptom | A WHOLE shared bandwidth is "empty" — still billed but no EIPs |
|---|---|
| Root Cause | EIPs were released without removing them from the shared bandwidth first |
| Diagnosis | `hcloud bandwidth list` → `share_type=WHOLE` → `eip_count == 0` |
| Resolution | `hcloud bandwidth delete --bandwidth-id <id>` after confirmation |
| Cross-skill | `huaweicloud-billing-ops` (cost leak) |

## Pattern K5: Cross-Region Bind Impossible

| Symptom | User: "Bind this EIP to an ECS in another region" |
|---|---|
| Root Cause | EIP is region-scoped; bind is intra-region only |
| Resolution | Allocate a new EIP in the target region, then bind. If the original EIP is no longer needed, release it. |

## Pattern K6: 95计费 Bill Shock

| Symptom | 24h EIP cost is 3× the 7-day median |
|---|---|
| Root Cause | 95计费 bills the top 5% of 5-min samples; a single burst month-end causes a step-up |
| Diagnosis | `huaweicloud-billing-ops` invoice detail; check sample distribution in custom metric |
| Resolution | (a) Pre-allocate 95th ceiling via contract renegotiation. (b) Add CES alarm on `5min_sample` approaching historical 95th. |
| Cross-skill | `huaweicloud-billing-ops` (invoice), `huaweicloud-ces-ops` (alarm) |

## Pattern K7: DDoS-Induced EIP Blackhole

| Symptom | `incoming_bandwidth` spikes 10× p50, all incoming traffic fails |
|---|---|
| Root Cause | EIP is being attacked; upstream carrier null-routes |
| Diagnosis | CES traffic shape; CES 5xx on the EIP |
| Resolution | Delegate to `huaweicloud-ddos-ops` (when present) / `huaweicloud-hss-ops` for mitigation |
| Cross-skill | `huaweicloud-ddos-ops`, `huaweicloud-hss-ops` |

## Pattern K8: 95计费 Cooldown Missed Resize

| Symptom | Resize "succeeded" but `bandwidth.size` reverted within 24h |
|---|---|
| Root Cause | 95计费 shared bandwidth has a cooldown window after each change |
| Diagnosis | `hcloud bandwidth describe` — check `cooldown_at` |
| Resolution | Wait for cooldown end OR design resize schedule around it |
