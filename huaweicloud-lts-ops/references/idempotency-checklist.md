# Idempotency Checklist — Huawei Cloud LTS

## Overview

Idempotency ensures that repeated execution of the same operation produces the same result without side effects. This is critical for automation, retry logic, and CI/CD pipelines.

## Operation Idempotency

| Operation | Idempotent? | Behavior on Replay | Guidance |
|-----------|-------------|-------------------|----------|
| `CreateLogGroup` | ❌ No | Returns `LTS.0102` (name conflict) | Check existence via `ListLogGroups` before create |
| `ListLogGroups` | ✅ Yes | Same data (may change between calls) | Safe to retry |
| `UpdateLogGroup` (TTL) | ✅ Yes | Same TTL set; no side effect | Safe to retry |
| `DeleteLogGroup` | ✅ Yes | Second call returns `LTS.0401` (already deleted) | Accept not-found as success |
| `CreateLogStream` | ❌ No | Returns `LTS.0302` (name conflict) | Check existence via `ListLogStreams` before create |
| `ListLogStreams` | ✅ Yes | Same data | Safe to retry |
| `DeleteLogStream` | ✅ Yes | Second call returns not-found | Accept not-found as success |
| `ListLogs` (search) | ✅ Yes | Same results for same query params | Safe to retry |
| `CreateTransfer` | ❌ No | Returns `LTS.0602` (already exists) | Check via `ListTransfers` or append timestamp to name |
| `ListTransfers` | ✅ Yes | Same data | Safe to retry |
| `DeleteTransfer` | ✅ Yes | Second call returns not-found | Accept not-found as success |
| `CreateDashboard` | ❌ No | May create duplicate | Check via `ListDashboards` before create |
| `ListDashboards` | ✅ Yes | Same data | Safe to retry |

## Idempotency Patterns

### Pattern 1: Create-or-Skip

For log groups and streams, check existence before creation:

```bash
# Check if log group exists
EXISTING=$(hcloud LTS ListLogGroups --cli-region="cn-north-4" --cli-output=json | \
  jq -r '.log_groups[] | select(.log_group_name=="my-group") | .log_group_id')

if [ -z "$EXISTING" ]; then
    hcloud LTS CreateLogGroup --log_group_name="my-group" --ttl_in_days=30
else
    echo "Log group already exists: $EXISTING"
fi
```

### Pattern 2: Create-or-Update

For resources that can be updated:

```bash
hcloud LTS UpdateLogGroup \
  --log_group_id="$EXISTING" \
  --ttl_in_days=30
```

### Pattern 3: Delete-if-Exists (for cleanup)

```bash
if [ -n "$EXISTING" ]; then
    hcloud LTS DeleteLogGroup --log_group_id="$EXISTING"
fi
```

## Retry Configuration

| Scenario | Max Retries | Backoff | Idempotency Guard |
|----------|-------------|---------|-------------------|
| List operations | 3 | 1s linear | None needed (idempotent) |
| Create operations | 2 | 2s, 5s exponential | Check existence first |
| Delete operations | 2 | 2s, 5s exponential | Accept 404 as success |
| Transfer operations | 3 | 3s, 6s, 12s exponential | Check existence before create |
