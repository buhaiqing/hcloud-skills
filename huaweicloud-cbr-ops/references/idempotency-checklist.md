# Idempotency Checklist — Huawei Cloud CBR

## Idempotent Operations

| Operation | Idempotent | Strategy | Notes |
|-----------|-----------|----------|-------|
| ListVaults | ✅ Yes | Read-only | Always safe to retry |
| ShowVault | ✅ Yes | Read-only | Always safe to retry |
| ListPolicies | ✅ Yes | Read-only | Always safe to retry |
| ListBackups | ✅ Yes | Read-only | Always safe to retry |
| ShowBackup | ✅ Yes | Read-only | Always safe to retry |
| CreateVault | ❌ No | Name uniqueness | Check name collision before retry |
| CreatePolicy | ❌ No | Name uniqueness | Check name collision before retry |
| CreateBackup | ❌ No | Creates new backup | Cannot retry without duplication |
| DeleteVault | ⚠️ Conditional | Idempotent after first delete | Returns 404 on retry |
| DeletePolicy | ⚠️ Conditional | Idempotent after first delete | Returns 404 on retry |
| DeleteBackup | ⚠️ Conditional | Idempotent after first delete | Returns 404 on retry |
| UpdateVault | ✅ Yes | Last-write-wins | Name, size updates |
| UpdatePolicy | ✅ Yes | Last-write-wins | Schedule, retention updates |
| RestoreBackup | ❌ No | State-changing | Only retry if restore explicitly failed |
| ReplicateBackup | ❌ No | State-changing | Only retry if replication failed |

## Automation Retry Policy

```yaml
automation_retry:
  read_operations:
    max_retries: 3
    backoff: "exponential"
  
  create_operations:
    max_retries: 2
    pre_check: true
    
  delete_operations:
    max_retries: 2
    ignore_404: true
```
