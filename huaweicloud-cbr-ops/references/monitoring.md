# Monitoring — Huawei Cloud CBR

## CES Metrics

CBR metrics are available in namespace `SYS.CBR`.

| Metric | Unit | Description | Recommended Alarm Threshold |
|--------|------|-------------|---------------------------|
| `vault_used_percent` | % | Vault storage usage percentage | >80% for 24h |
| `vault_used` | bytes | Used storage in vault | Monitor for capacity planning |
| `vault_size` | bytes | Total vault capacity | Compare with `vault_used` |
| `backup_count` | Count | Number of backups in vault | Monitor growth trend |
| `backup_success_count` | Count | Successful backups in period | Monitor for success rate |
| `backup_failure_count` | Count | Failed backups in period | >0 for any period |
| `backup_size` | bytes | Total backup data size | Monitor for cost analysis |
| `backup_avg_duration` | seconds | Average backup duration | Spike indicates issues |
| `restore_success_count` | Count | Successful restores | Track for SLA reporting |
| `restore_failure_count` | Count | Failed restores | >0 for any period |
| `replication_bandwidth` | bytes/s | Cross-region replication throughput | Monitor for replication delays |

## Recommended Alarm Rules

```bash
# Alarm: Vault capacity high
hcloud CES CreateAlarm \
  --name="cbr-vault-capacity-high" \
  --namespace="SYS.CBR" \
  --metric_name="vault_used_percent" \
  --threshold=80 \
  --comparison_operator="gt" \
  --period=86400 \
  --evaluation_periods=1

# Alarm: Backup failure detected
hcloud CES CreateAlarm \
  --name="cbr-backup-failure" \
  --namespace="SYS.CBR" \
  --metric_name="backup_failure_count" \
  --threshold=0 \
  --comparison_operator="gt" \
  --period=3600 \
  --evaluation_periods=1
```

## Dashboard Suggestion

| Panel | Metrics | Period |
|-------|---------|--------|
| Vault Capacity | `vault_used_percent` by vault | 1 day |
| Backup Success Rate | `backup_success_count`, `backup_failure_count` | 1 day |
| Backup Duration | `backup_avg_duration` by vault | 1 day |
| Replication | `replication_bandwidth` | 1 hour |
| Storage Cost | `backup_size` trend | 30 days |
