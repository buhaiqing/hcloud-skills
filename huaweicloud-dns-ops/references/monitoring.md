# DNS Monitoring & Alerts — Huawei Cloud DNS

## Key Metrics (via CES)

| Metric | Description | Unit |
|---|---|---|
| `dns_request_count` | Total DNS queries | count |
| `dns_request_success_rate` | Successful resolution rate | % |
| `dns_request_latency_p99` | 99th percentile resolution time | ms |
| `nxdomain_count` | NXDOMAIN (domain not found) rate | count |

## Alarm Templates

```yaml
# Resolution success rate
- name: "dns-resolution-rate-low"
  metric: "dns_request_success_rate"
  threshold: 95   # %
  period_minutes: 5
  alarm_level: 1
  notification: "urn:smn:{{region}}:{{project}}:dns-critical"

# NXDOMAIN spike
- name: "dns-nxdomain-spike"
  metric: "nxdomain_count"
  threshold: 1000  # per minute
  alarm_level: 2
  notification: "urn:smn:{{region}}:{{project}}:dns-warning"
```

## Propagation Check

```bash
# After modifying a record, check global propagation
for ns in ns1.hwclouds-dns.com ns2.hwclouds-dns.com; do
  echo "=== $ns ==="
  dig @"$ns" www.example.com. A +short
done
```
