# Troubleshooting — Huawei Cloud FunctionGraph

## Common API Error Codes

| Code | HTTP | Meaning | Agent Action |
|------|------|---------|--------------|
| `FSS.0101` | 400 | Invalid parameter | Verify function name, runtime, handler, timeout against API docs |
| `FSS.0102` | 409 | Function name already exists | Choose unique name |
| `FSS.0103` | 400 | Code package invalid | Verify OBS URL accessibility, check ZIP format |
| `FSS.0104` | 400 | Runtime not supported | List supported runtimes via SDK `ListQuotas` |
| `FSS.0201` | 403 | Quota exceeded | Delete unused functions or request quota increase |
| `FSS.0202` | 403 | Insufficient balance | Recharge Huawei Cloud account |
| `FSS.0301` | 400 | VPC configuration invalid | Verify VPC ID, subnet ID, check VPC exists |
| `FSS.0401` | 404 | Resource not found | Verify function URN or trigger ID |
| `FSS.0501` | 409 | Trigger conflict | Trigger already exists for this event source |
| `FSS.0502` | 400 | Trigger not supported for runtime | Check trigger type compatibility with runtime |
| `FSS.0601` | 400 | Function execution error | Check function code, logs in LTS |
| `FSS.0602` | 400 | Function timeout | Increase timeout value, optimize code |
| `FSS.0603` | 400 | Memory exhausted | Increase memory allocation, optimize memory usage |
| `FSS.0701` | 403 | No permission to trigger event source | Check IAM agency configuration |
| `FSS.0702` | 403 | No permission to access VPC | Check VPC permissions and endpoint configuration |
| Throttling 429 | 429 | Rate limit exceeded | Exponential backoff, respect retry-after |
| InternalError 500 | 500 | Server error | Retry with backoff 2s→4s→8s |

## Diagnostic Order

1. **Verify function exists**: `ShowFunctionConfig(function_urn)` — check status
2. **Check function state**: Should be `Active` — if `Failed`, check last error
3. **Test sync invocation**: `InvokeFunction` with minimal payload — if fails, check error
4. **Check execution logs**: Query LTS for function logs (delegate to LTS skill when available)
5. **Check CES metrics**: `count`, `fail_count`, `duration`, `reject_count`
6. **Verify triggers**: `ListFunctionTriggers` — confirm trigger status `ACTIVE`
7. **Check trigger event source**: Verify event source (OBS bucket, SMN topic, etc.) is active
8. **Verify IAM agency**: Function → agency must have permissions to access trigger service

## Function Invocation Failure

### Common Causes

| Symptom | Cause | Fix |
|---------|-------|-----|
| Timeout | Code too slow, timeout too low | Increase timeout (up to 300s), optimize code |
| Out of memory | Memory too low for workload | Increase memory (up to 4096 MB) |
| Runtime error | Code exception | Check LTS logs, fix code |
| `Cannot find handler` | Handler path incorrect | Verify handler format: `file.method` or `package.Class::method` |
| Dependency missing | Incomplete code package | Include all dependencies in ZIP |
| VPC connection failed | VPC config wrong or unreachable | Verify VPC/subnet, check security group rules |

## Trigger Issues

| Symptom | Cause | Fix |
|---------|-------|-----|
| Trigger not firing | Trigger status not `ACTIVE` | Enable or recreate trigger |
| APIG trigger 502 | Function returns non-200 | Check function code, response format |
| OBS trigger not firing | Event type mismatch | Verify OBS event type (ObjectCreated, ObjectRemoved) |
| Timer trigger missed | Max retries exceeded | Check function execution, increase timeout |
| Duplicate invocations | At-least-once delivery | Design idempotent functions |

## Cold Start Issues

| Mitigation | Cost Impact | Effectiveness |
|------------|-------------|---------------|
| Reserved instances | Pay for idle capacity | Eliminates cold start |
| Keep-warm pattern | Low cost | ~70% reduction |
| Provisioned concurrency | Medium cost | ~95% reduction |
| Choose Java/Python | Lower cold start vs Node.js | Runtime-dependent |
| Minimize dependencies | No extra cost | Package size-dependent |

## Quota Issues

| Error | Action |
|-------|--------|
| `FSS.0201` Quota exceeded | `ListFunctions` → identify unused → delete; or submit quota increase |
| Concurrent limit | Check `ListAsyncInvocations` for backlog; increase function efficiency |
| Code size > 10MB inline | Use OBS URL upload for larger packages |
| Trigger count limit | Check existing triggers, remove unused ones |
