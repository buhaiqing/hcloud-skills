# Monitoring — Huawei Cloud ELB

## CES Metrics (Cloud Eye Service)

Namespace: `SYS.ELB`

### Key Metrics

| Metric | Name | Unit | Recommended Threshold |
|--------|------|------|---------------------|
| `m1_cps` | New connections per second | count/s | Warning: > 80% of LB limit |
| `m2_act_conn` | Active connections | count | Warning: > 80% of max |
| `m3_inact_conn` | Inactive connections | count | — |
| `m4_ncps` | New inbound connections | count/s | Baseline-dependent |
| `m5_drop_rate` | Dropped packets rate | % | Warning: > 0% |
| `m6_max_conn` | Max concurrent connections | count | Warning: > 90% of limit |
| `m7_req_2xx` | 2xx responses | count | Baseline for throughput |
| `m7_req_3xx` | 3xx responses | count | Baseline |
| `m7_req_4xx` | 4xx responses | count | Warning: > 1% of total |
| `m7_req_5xx` | 5xx responses | count | Critical: > 0.1% of total |
| `m8_l4_upstream_rsp_time` | L4 upstream response time | ms | Warning: > 500ms |
| `m9_unhealthy_host` | Unhealthy hosts | count | Critical: > 0 |
| `m10_l7_upstream_rsp_time` | L7 upstream response time | ms | Warning: > 1000ms |
| `m11_l7_upstream_rsp_2xx` | L7 upstream 2xx | count | — |
| `m11_l7_upstream_rsp_5xx` | L7 upstream 5xx | count | Critical: > 0 |

## Alert Patterns

### Resource Pressure Alerts

| Alert | Metric | Condition | Severity |
|-------|--------|-----------|----------|
| Connection spike | `m1_cps` | > 80% of LB limit | Warning |
| High error rate | `m7_req_5xx` | > 1% of total requests | Critical |
| Backend unhealthy | `m9_unhealthy_host` | > 0 | Critical |
| Dropped connections | `m5_drop_rate` | > 0% | Critical |
| High latency (L7) | `m10_l7_upstream_rsp_time` | avg > 2s | Warning |

### Anomaly Patterns

| Pattern | Metrics | Detection Logic | Severity |
|---------|---------|----------------|----------|
| traffic_surge | `m1_cps`, `m2_act_conn` | CPS > 3× baseline | Warning |
| error_rate_spike | `m7_req_5xx`, `m7_req_4xx` | 5xx rate > 5% in 5min | Critical |
| backend_degradation | `m9_unhealthy_host`, `m8_l4_upstream_rsp_time` | Unhealthy > 0 AND response time > 2× baseline | Critical |
| connection_storm | `m2_act_conn`, `m5_drop_rate` | Drop rate > 1% AND active connections > 80% max | Critical |
| latency_degradation | `m10_l7_upstream_rsp_time` | P99 > 3s sustained 5min | Warning |

## Dashboards

- CES Console: `https://console.huaweicloud.com/ces/#/metricView/instances`
- Recommended dashboard: group by LB name, show key metrics:
  - Active connections (m2_act_conn)
  - Requests per second (m1_cps)
  - Response codes by class (m7_req_2xx/4xx/5xx)
  - Unhealthy hosts (m9_unhealthy_host)
  - Upstream response time P50/P95/P99

## SLA & Error Budget

| Metric | SLO Target | Error Budget |
|--------|-----------|-------------|
| LB availability | ≥ 99.95% | 21.6min/month |
| Backend pass rate (health) | ≥ 99% | — |
| P99 latency (L7) | ≤ 2s | — |

## Logs (LTS)

ELB can stream access logs to LTS (need to enable):
- Log format: `${time_iso8601} ${log_topic} ${request_id} ${client_ip}:${client_port} ${upstream_addr} ${request_method} ${request_uri} ${protocol} ${status} ${body_bytes_sent} ${request_time} ${upstream_status} ${upstream_response_time}`

## Cost Metrics

| Metric | Purpose | Action |
|--------|---------|--------|
| LB hourly cost | Base cost of LB instance | Choose dedicated vs shared based on traffic |
| Data processing fee | Per GB processed | Optimize data transfer, compress responses |
| EIP cost | Public IP | Release unused EIPs |
| Cross-AZ data transfer | Cross-AZ traffic | Co-locate members in same AZ when possible |
