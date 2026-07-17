# CDN Monitoring & Alerts — Huawei Cloud CDN

## Key Metrics

| Metric | Source | Description | Unit |
|---|---|---|---|
| `outgoing_bandwidth` | CES | CDN egress bandwidth (user → edge) | bps |
| `outgoing_traffic` | CES | CDN egress total traffic (user → edge) | byte |
| `bandwidth_flux` | CDN API | Bandwidth (5-min peak samples) | bps |
| `flux_hit_rate` | CDN API | Cache hit rate | % |
| `origin_bandwidth` | CDN API | Origin pull bandwidth (edge → origin) | bps |
| `origin_flux` | CDN API | Origin pull traffic | byte |
| `http_code_2xx` / `http_code_4xx` / `http_code_5xx` | CDN API | HTTP status distribution | count |

## CES Alarm Templates

```yaml
# Threshold alarms (wire via hcloud ces alarm-plan/apply)
- name: "cdn-bandwidth-warning"
  metric: "cdn_bandwidth"
  threshold: 100000000000   # 100 Gbps
  period: 300
  evaluation_periods: 2
  alarm_level: 2
  notification: "urn:smn:{{region}}:{{project}}:cdn-critical"
```

## Idle Domain Detection

```bash
# Domains online but with zero bandwidth for 7 days
hcloud cdn list-domain --region {{env.HW_REGION_ID}} --output json \
  | jq -r '.result[] | select(.status=="online") | .domain_name' \
  | while read domain; do
    bw=$(hcloud cdn list-stats --domain-id "$domain" \
      --start-time "$(date -d '7 days ago' +%Y%m%d%H%M%S)" \
      --end-time "$(date +%Y%m%d%H%M%S)" \
      --stat-type bandwidth --output json | jq '.result[0].bandwidth // 0')
    [ "$bw" -eq 0 ] && echo "IDLE: $domain"
  done
```

## Hit Rate Optimization Triggers

| Condition | Action |
|---|---|
| Hit rate < 70% for 24h | Review cache rules; increase TTL for static assets |
| Hit rate < 50% for 7d | Investigate origin-side dynamic content; consider edge caching strategy |
| Origin flux > 2× user flux | Cache inefficiency; add cache rules for high-traffic paths |
