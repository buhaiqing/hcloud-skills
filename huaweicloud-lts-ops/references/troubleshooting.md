# Troubleshooting — Huawei Cloud LTS

## Error Code Reference

| Code | Meaning | Agent Action | Max Retries |
|------|---------|-------------|-------------|
| `LTS.0001` | Invalid parameter | Validate input (name format, TTL range, etc.) | 0 (HALT) |
| `LTS.0101` | Quota exceeded (log group) | List groups; suggest deleting unused or raise ticket | 0 (HALT) |
| `LTS.0102` | Name conflict | Suggest unique name (append timestamp) | 0 (HALT) |
| `LTS.0201` | Authentication failed | Verify AK/SK are valid and not expired | 0 (HALT) |
| `LTS.0202` | Insufficient IAM permissions | Check IAM policy: missing lts:* permission | 0 (HALT) |
| `LTS.0301` | Log group not found | Re-list groups with ListLogGroups; verify group ID | 1 (2s) |
| `LTS.0302` | Log stream not found | Re-list streams with ListLogStreams; verify stream ID | 1 (2s) |
| `LTS.0401` | Group not found on delete | Group may already be deleted; confirm with user | 0 (HALT) |
| `LTS.0402` | Active transfer exists | List transfers with ListTransfers; delete first | 0 (HALT) |
| `LTS.0501` | Invalid time range | Ensure start_time < end_time; format as epoch ms | 0 (HALT) |
| `LTS.0502` | Keywords too long | Truncate keywords to 2048 characters | 1 (1s) |
| `LTS.0503` | Index not configured | Guide user to enable structured indexing on stream | 0 (HALT) |
| `LTS.0601` | OBS bucket not found | Verify bucket exists and region matches | 1 (3s) |
| `LTS.0602` | Transfer already exists | List existing transfers; offer to update | 0 (HALT) |
| `LTS.0603` | Invalid OBS bucket policy | Grant LTS `PutObject` permission on bucket | 0 (HALT) |
| `LTS.0701` | TTL out of range | Must be 1–365 days | 0 (HALT) |
| `LTS.0801` | Internal server error | Retry with exponential backoff | 3 (5s, 10s, 20s) |
| `LTS.0802` | Service unavailable | Check region status page | 3 (10s, 30s, 60s) |

## Diagnostic Workflows

### Diagnosis 1: Logs Not Showing Up in Search

1. **Check log group exists** → `ListLogGroups` — if not found, create it.
2. **Check log stream exists** → `ListLogStreams` — if not found, create it.
3. **Verify ICAgent is running** → If using ECS, check ICAgent status via ECS console or SSH.
4. **Check ingestion** → Use `ListLogs` with wide time range (last 24h) and empty keywords. If empty, ingestion path is broken.
5. **Check index configuration** → If search returns no results but logs exist, indexing may not be configured.
6. **Check time zone** → Ensure log timestamps are in UTC+8 or correct timezone is configured.

### Diagnosis 2: Log Search Too Slow

1. **Narrow time range** → Limit to a few hours instead of days.
2. **Add keywords** → More specific query reduces scan volume.
3. **Check structured parsing** → Unstructured logs scan slower than indexed KV fields.
4. **Check index fields** → Ensure fields used in query are indexed.
5. **Consider cross-stream search limit** → Max 50 streams per cross-stream query.

### Diagnosis 3: Log Transfer Failed

1. **Verify OBS bucket exists** → Use `huaweicloud-obs-ops` or CLI `hcloud OBS ListBuckets`.
2. **Check bucket region** → Must be same region as LTS.
3. **Check bucket policy** → LTS service must have `PutObject` permission.
4. **List transfer rules** → `ListTransfers` to confirm rule exists.
5. **Check CES alarm** → LTS `lts_transfer_failed` metric should trigger alarm.
6. **Retry creation** → `DeleteTransfer` then `CreateTransfer`.

### Diagnosis 4: Cannot Create Log Group/Stream

1. **Check quota** → List existing groups; max 100 per project.
2. **Check name format** → 1–64 chars, letters/digits/underscores/hyphens only.
3. **Check for name conflict** → Log group names must be unique within project.
4. **Check IAM permissions** → `lts:logGroup:createLogGroup` or `lts:logStream:createLogStream`.

## Common User Scenarios

### Scenario: "I created a log group but can't see logs"
```
[INFO] Check if log streams exist in the group
[INFO] Check if ICAgent is deployed on the source host
[INFO] Check if log collection path is configured correctly
[INFO] Verify network connectivity: host → LTS endpoint
```

### Scenario: "Log search returns no results"
```
[INFO] Verify time range is correct
[INFO] Try empty keywords (match all logs)
[INFO] Check if structured indexing is enabled
[INFO] Check if logs have been cleaned by TTL
```

### Scenario: "OBS transfer not working"
```
[INFO] Verify OBS bucket exists and is in the same region
[INFO] Check bucket policy allows LTS to write
[INFO] Verify transfer rule is active via ListTransfers
[INFO] Check for any LTS.060x error codes
```
