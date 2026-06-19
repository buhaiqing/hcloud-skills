# LTS AIOps Patterns — Huawei Cloud Log Tank Service

> Advanced AIOps patterns for Log Tank Service.
> Load when designing log-based anomaly detection, alarm routing, or
> log retention optimization.

## 1. Multi-Source Log Correlation

| Source | Pattern | Detection |
|--------|---------|-----------|
| Application + Infrastructure | error_rate spike + cpu spike same 5m window | service degradation |
| Audit + Performance | failed_login > 10 AND api_latency > p99 | credential attack |
| Network + Application | bandwidth > 80% AND 5xx > baseline | traffic spike |

## 2. Alarm Storm Suppression

- Group identical `resource_id + metric` alarms in a 5-minute window
- Emit one notification per group with severity worst-of
- Suppress downstream notifications if upstream P0 is firing

## 3. Log Retention Tiers

| Tier | Storage | Retention | Use |
|------|---------|-----------|-----|
| Hot | LTS standard | 7 days | real-time queries |
| Warm | LTS cold | 30 days | weekly review |
| Archive | OBS + lifecycle | 365 days | compliance / forensic |

## 4. Knowledge-Base Patterns (≥5)

| Pattern | Symptoms | Diagnostic |
|---------|----------|------------|
| Out of disk | `lts_disk_usage > 95%` | expand storage, archive |
| Query latency | `query_p95 > 30s` | partition by date |
| Log loss | `ingest_lag > 60s` | check agent health |
| Index corruption | `index_error_count > 0` | rebuild index |
| Permission denied | 403 spike | verify agency / policy |

> **Security-Sensitive**: log stream deletion, log group purge, and
> cross-region transfer MUST require explicit operator confirmation and
> preserve evidence under the documented retention policy.