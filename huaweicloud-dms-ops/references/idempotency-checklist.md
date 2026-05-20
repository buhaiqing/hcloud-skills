# Idempotency Checklist — Huawei Cloud DMS

## Idempotent Operations

| Operation | Idempotent | Strategy | Notes |
|-----------|-----------|----------|-------|
| ListInstances | ✅ Yes | Read-only | Always safe to retry |
| ShowInstance | ✅ Yes | Read-only | Always safe to retry |
| ListTopics | ✅ Yes | Read-only | Always safe to retry |
| ListQueues | ✅ Yes | Read-only | Always safe to retry |
| ListConsumerGroups | ✅ Yes | Read-only | Always safe to retry |
| ListBackups | ✅ Yes | Read-only | Always safe to retry |
| CreateInstance | ❌ No | Name uniqueness | Check name collision before retry |
| CreateTopic | ❌ No | Topic uniqueness | Check topic existence before retry |
| CreateQueue | ❌ No | Queue uniqueness | Check queue existence before retry |
| DeleteInstance | ⚠️ Conditional | Idempotent after first delete (404) | Safe to retry (returns 404) |
| DeleteTopic | ⚠️ Conditional | Idempotent after first delete (404) | Safe to retry (returns 404) |
| UpdateInstance | ✅ Yes | Last-write-wins | Repeat sets same final state |
| CreateBackup | ❌ No | Creates new backup each call | Check for duplicate backups before retry |
| RestoreInstance | ❌ No | State-changing | Only retry if restore failed |

## Retry Strategy

```yaml
idempotent_operations:
  read_operations:
    max_retries: 3
    backoff: "exponential"
    initial_delay_ms: 1000
    max_delay_ms: 10000

non_idempotent_operations:
  create_operations:
    max_retries: 2
    pre_check: true  # Check existence before create
    duplicate_handling: "return_existing"
  delete_operations:
    max_retries: 2
    ignore_404: true
```
