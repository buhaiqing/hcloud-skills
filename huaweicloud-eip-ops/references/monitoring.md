# EIP Monitoring & Alerts — Huawei Cloud Elastic IP

## Key Metrics (CES)

| Metric | Dimension | Unit | Default Window |
|---|---|---|---|
| `outgoing_bandwidth` | `publicip_id` | bit/s | 1 min |
| `incoming_bandwidth` | `publicip_id` | bit/s | 1 min |
| `outgoing_bytes` | `publicip_id` | byte | 1 min (按流量 billing input) |
| `incoming_bytes` | `publicip_id` | byte | 1 min |
| `eip_status` | `publicip_id` | enum | 30 s |
| `eip_association_status` | `publicip_id` | enum | 30 s |

> Configure CES alarms via `huaweicloud-ces-ops` (delegate); this skill only
> documents the **metric contract**.

## Recommended Thresholds

| Metric | Warning | Critical | Action |
|---|---:|---:|---|
| `outgoing_bandwidth / bandwidth_size` | 0.8 | 0.95 | Resize bandwidth |
| `eip_status` ≠ `ACTIVE` for bound EIP | 5 min | 15 min | Diagnose bind |
| `eip_association_status` flips | 1 event | — | Audit who unbound |
| Daily `outgoing_bytes` > 3× 7-day median | — | 1 day | FinOps: cost shock |

## Idle EIP Detector (cron / event-driven)

```bash
# Find EIPs unbound for 7+ days
hcloud eip list --region {{env.HW_REGION_ID}} --output json \
  | jq -r '.publicips[] | select(.port_id == null) | .id' \
  | while read EIP_ID; do
      echo "[idle] $EIP_ID"
    done
```

Schedule via SMN topic or CES event rule; see
`huaweicloud-ces-ops/references/monitoring.md` for wiring.

## 95计费 Monitoring

For EIPs in 95th-percentile subscriptions, expose:
- `5min_sample` (in-bandwidth bytes / 300s)
- `monthly_95th` (rolling calculation in custom metric)

`huaweicloud-ces-ops` provides the calculation scaffolding; this skill
**owns the metric names and the export frequency contract**.
