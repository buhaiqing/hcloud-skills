# Monitoring — Huawei Cloud SWR

## CES Metrics

SWR metrics are available in namespace `SYS.SWR`.

| Metric | Unit | Description | Recommended Alarm Threshold |
|--------|------|-------------|---------------------------|
| `repo_storage_usage` | bytes | Storage used by repository | >80% of quota |
| `repo_image_count` | Count | Number of images/tags in repo | Monitor for growth |
| `repo_pull_count` | Count | Number of image pulls (daily) | Monitor for popularity |
| `repo_push_count` | Count | Number of image pushes (daily) | Monitor for activity |
| `total_storage_usage` | bytes | Total storage across all repos | >80% of account quota |
| `sync_lag` | seconds | Cross-region sync delay | >3600s (1 hour) |
| `auth_failure_count` | Count | Failed authentication attempts | >10 per hour |

## Recommended Alarm Rules

```bash
# Alarm: Repository storage high
hcloud CES CreateAlarm \
  --name="swr-storage-high" \
  --namespace="SYS.SWR" \
  --metric_name="repo_storage_usage" \
  --threshold=8589934592 \
  --comparison_operator="gt" \
  --period=86400 \
  --evaluation_periods=1 \
  --unit="bytes"

# Alarm: Sync delay
hcloud CES CreateAlarm \
  --name="swr-sync-delay" \
  --namespace="SYS.SWR" \
  --metric_name="sync_lag" \
  --threshold=3600 \
  --comparison_operator="gt" \
  --period=3600 \
  --evaluation_periods=2
```

## Dashboard Suggestion

| Panel | Metrics | Period |
|-------|---------|--------|
| Storage Usage | `repo_storage_usage` by repo | 1 day |
| Image Count | `repo_image_count` by repo | 1 day |
| Pull Activity | `repo_pull_count` by repo | 1 day |
| Sync Health | `sync_lag` by sync rule | 1 hour |
| Auth Failures | `auth_failure_count` | 1 hour |
