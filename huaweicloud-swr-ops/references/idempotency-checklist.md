# Idempotency Checklist — Huawei Cloud SWR

## Idempotent Operations

| Operation | Idempotent | Strategy | Notes |
|-----------|-----------|----------|-------|
| ListOrganizations | ✅ Yes | Read-only | Always safe to retry |
| ListRepositories | ✅ Yes | Read-only | Always safe to retry |
| ListImages | ✅ Yes | Read-only | Always safe to retry |
| ListRetentionPolicies | ✅ Yes | Read-only | Always safe to retry |
| ListImageSync | ✅ Yes | Read-only | Always safe to retry |
| GenerateLoginToken | ✅ Yes | Read-only | New token each call, old still valid |
| CreateOrganization | ❌ No | Name uniqueness | Org names are globally unique |
| CreateRepository | ❌ No | Name uniqueness | Repo names are unique per org |
| CreateRetentionPolicy | ❌ No | One per repo | Delete existing before creating new |
| CreateImageSync | ❌ No | Rule uniqueness | Check existing rules before create |
| DeleteOrganization | ⚠️ Conditional | Idempotent after first delete | Returns 404 on retry |
| DeleteRepository | ⚠️ Conditional | Idempotent after first delete | Returns 404 on retry |
| DeleteImageTag | ⚠️ Conditional | Idempotent after first delete | Returns 404 on retry |

## Retry Strategy

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
