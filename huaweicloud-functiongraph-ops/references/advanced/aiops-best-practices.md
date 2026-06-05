# AIOps Best Practices — Huawei Cloud FunctionGraph

> Intelligent operations integration patterns for FunctionGraph:
> cold-start anomaly, concurrency saturation, error-rate correlation,
> and trigger-source health.
> **Version:** 1.0.0

## AIOps Goals for FunctionGraph

FunctionGraph is event-driven serverless. AIOps workflows should:

- Detect cold-start latency spikes correlated with concurrency bursts
- Correlate function error rates with trigger source (OBS / SMN / API
  Gateway / Timer) anomalies
- Surface quota exhaustion before throttling starts
- Auto-tune concurrency / memory based on traffic patterns

## Recommended AIOps Patterns

### 1. Cold-Start / Concurrency Anomaly

| Pattern | Metrics Correlated | Detection Logic | Remediation |
|---------|-------------------|-----------------|-------------|
| `cold_start_storm` | `cold_start_count` + `invocations` | cold/invocation > 0.3 for 10 min | Pre-warm (publish warmup events) |
| `concurrency_saturation` | `concurrency / max_concurrency` | ratio > 0.85 sustained 5 min | Apply for quota increase |
| `duration_p99_spike` | `duration_p99` | 3× baseline for 5 min | Page on-call; check downstream latency |
| `error_rate_rise` | `failed_invocations / total` | ratio > 0.05 sustained 5 min | Open Sev3 ticket |
| `timeout_cascade` | `timeouts` | rate > 10/min for 5 min | Lower concurrency, increase timeout |
| `quota_throttling` | `throttled_invocations` | rate > 0 for 1 min | Open Sev2 quota ticket |

### 2. Trigger-Source Health Correlation

For each function, group invocation outcomes by `trigger_type` and
cross-reference with the trigger source's health:

| Trigger Type | Source Skill | Correlation Method |
|--------------|--------------|--------------------|
| OBS event | `huaweicloud-obs-ops` | Bucket notification failures → function invocation drop |
| API Gateway | `huaweicloud-elb-ops` (or APIG skill) | 5xx upstream → function 5xx |
| SMN topic | `huaweicloud-dms-ops` | Topic publish failures → no invocations |
| Timer | (internal) | Skip invocations → check cron rule |
| CTS event | (internal) | CTS throttle → invocation backlog |

### 3. Anomaly Storm Handling

When ≥ 3 functions trigger Critical alarms within 5 min (e.g., shared
dependency, regional degradation):

1. Pause non-essential remediation
2. Snapshot function config + last 100 failed invocations
3. Emit a single consolidated page
4. Auto-create CES event tagged `aiops-cluster:functiongraph`

## ML Integration Hooks

FunctionGraph AIOps can leverage:

| Metric (CES namespace `SYS.FunctionGraph`) | Aggregation | Use Case |
|-------------------------------------------|-------------|----------|
| `invocations` | 1-min | Traffic baseline |
| `duration_p99` | 1-min | Latency anomaly |
| `errors` | 1-min | Error rate |
| `throttles` | 1-min | Quota pressure |
| `concurrent_executions` | 1-min | Concurrency tuning |
| `cold_starts` | 1-min | Warmup need |

## Cross-Skill Delegation Matrix

| Symptom | Delegate To |
|---------|-------------|
| Function times out, downstream is RDS | `huaweicloud-rds-ops` (RDS latency) + this skill (timeout config) |
| OBS trigger stops firing | `huaweicloud-obs-ops` (bucket notification) + this skill (trigger config) |
| SMN trigger fails | `huaweicloud-dms-ops` (topic) + this skill (subscription) |
| Concurrent execution quota | This skill (apply for increase) + `huaweicloud-iam-ops` (permission) |
| Cost spike (over-provisioned memory) | `huaweicloud-billing-ops` + this skill (right-size) |
| Function log analysis | `huaweicloud-lts-ops` (log group) + this skill (invocation log) |

## Self-Healing Playbook

| Trigger | Auto Action | Manual Step |
|---------|------------|-------------|
| Cold start ratio > 0.3 for 10 min | Emit warmup warning | Configure provisioned concurrency |
| Duration p99 > 3× baseline 5 min | Page on-call | Profile, lower concurrency |
| Error rate > 5% for 5 min | Snapshot last 100 failed | Investigate code or trigger |
| Quota throttling observed | Open Sev2 ticket | Apply for concurrency increase |
| Function version rollback needed | Stop the alias traffic | Manual `update-function-alias` |

## Reference: jq paths for FunctionGraph AIOps

```bash
# Functions with elevated error rate
hcloud functiongraph list-functions -o json \
  | jq '.functions[] | {name, error_rate: .error_rate, invocations: .invocations_24h}'

# Concurrent execution utilization
hcloud functiongraph show-function-metrics --function $NAME --metric concurrent_executions -o json \
  | jq '.datapoints[-1].value'

# Last 100 failed invocations (requires LTS export)
hcloud lts list-logs --log-group $LG --query "error" --limit 100 -o json \
  | jq '.events[] | {time, message}'
```

## Knowledge Base Anchors

- Function ↔ trigger source: `references/integration.md` §3
- Function failure patterns: `references/troubleshooting.md`
- Cost anomaly: `references/well-architected-assessment.md` §3 (FinOps)
