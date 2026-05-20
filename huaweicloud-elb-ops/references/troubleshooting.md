# Troubleshooting — Huawei Cloud ELB

## Common API Error Codes

| Code | HTTP | Meaning | Agent Action |
|------|------|---------|--------------|
| `ELB.1001` | 400 | Invalid parameter | Verify all parameters against API docs |
| `ELB.1002` | 404 | VPC not found | Create VPC via VPC skill first |
| `ELB.1003` | 404 | Subnet not found | Verify subnet ID, ensure LB subnet supports ELB |
| `ELB.1004` | 403 | Quota exceeded | Delete unused LBs or request quota increase |
| `ELB.1005` | 403 | Insufficient balance | Recharge Huawei Cloud account |
| `ELB.1006` | 400 | AZ not supported | List available AZs and choose supported one |
| `ELB.1007` | 400 | EIP not found | Check EIP association, create via VPC skill |
| `ELB.1008` | 409 | LB in provisioning state | Wait for current operation to complete |
| `ELB.2001` | 409 | Listener port conflict | Choose different port |
| `ELB.2002` | 400 | Invalid listener protocol | Check protocol support for LB type |
| `ELB.2003` | 400 | Certificate required for HTTPS | Upload or reference certificate |
| `ELB.3001` | 404 | Pool not found | Verify pool ID, create pool first |
| `ELB.3002` | 409 | Member already exists | Member already in pool — verify or update |
| `ELB.3003` | 400 | Health check invalid | Verify delay/timeout/max_retries values |
| `ELB.3004` | 400 | Member subnet mismatch | Member must be in LB's VPC subnet |
| `ELB.4001` | 404 | Certificate not found | Verify certificate ID |
| `ELB.4002` | 400 | Certificate expired | Upload new certificate |
| `ELB.4003` | 400 | Invalid certificate format | Check PEM format with correct chain |
| Throttling 429 | 429 | Rate limited | Exponential backoff |
| InternalError 500 | 500 | Server error | Retry with backoff 2s→4s→8s |

## Diagnostic Order

1. **Verify LB exists**: `ShowLoadBalancer(lb_id)` — check provisioning_status
2. **Check LB status**: `provisioning_status` = ACTIVE, `operating_status` = ONLINE
3. **Check listener config**: `ListListeners(loadbalancer_id)` — verify port/protocol
4. **Check backend pool**: `ListPools(loadbalancer_id)` — verify pool exists
5. **Check member health**: `ListMembers(pool_id)` — check operating_status of each member
6. **Check health monitor**: `ShowHealthMonitor(monitor_id)` — verify config
7. **Check backend server**: via ECS skill — verify server is running and service listening
8. **Check security group**: verify member's SG allows traffic from LB subnet
9. **Check CES metrics**: `m1_cps`, `m2_act_conn`, `m3_inact_conn`, `m7_req_2xx/3xx/4xx/5xx`

## Backend Member Unhealthy

| Symptom | Cause | Fix |
|---------|-------|-----|
| All members unhealthy | Health check path wrong | Verify `url_path` and `expected_codes` |
| Single member unhealthy | Member service down | Check service status on member ECS |
| Intermittent unhealthy | Timeout too low | Increase `timeout` to 5s+ |
| Members flapping | Check interval too short | Increase `delay` to 10s+ |
| New member unhealthy | Slow start period | Wait for slow start to complete |
| Cross-AZ unhealthy | AZ-level network issue | Check AZ route and security group |
| HTTP 404 from health check | Path not found | Correct `url_path` to actual endpoint |

## Connection Issues

| Symptom | Cause | Fix |
|---------|-------|-----|
| Timeout connecting to LB | Security group blocks traffic | Verify listener SG allows inbound |
| Connection resets | Idle timeout too low | Increase listener `keepalive_timeout` |
| Slow connection | Backend overloaded | Scale out backend members |
| 502 Bad Gateway | Backend returns error | Check backend app health |
| 504 Gateway Timeout | Backend response too slow | Increase backend timeout, check app performance |
| Unbalanced traffic | Session persistence issue | Check persistence type and cookie config |
| SSL handshake failure | Certificate mismatch or expired | Verify certificate validity and domain match |

## Listener Issues

| Symptom | Cause | Fix |
|---------|-------|-----|
| HTTPS listener not working | Certificate missing/expired | Upload new cert, associate with listener |
| Port already in use | Conflicting listener | Choose different port or remove conflicting |
| Health check never passes | Incorrect protocol | Match health check type to backend protocol |
| Listener can't be deleted | Pool still attached | Remove pool association first |

## LB Performance Issues

| Metric | Threshold | Action |
|--------|-----------|--------|
| `m1_cps` (connections/sec) | > 80% of LB limit | Upgrade to higher spec LB |
| `m2_act_conn` (active connections) | Near LB max | Scale out or upgrade LB |
| `m7_req_2xx` (throughput) | Baseline-dependent | Monitor trend |
| `m7_req_5xx` (error rate) | > 1% | Check backend health |
| `m9_unhealthy_host` | > 0 | Check individual member health |
