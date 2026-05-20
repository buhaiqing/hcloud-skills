# Troubleshooting Guide — Huawei Cloud DCS (Redis)

## Error Code Taxonomy

| Error Code | HTTP Status | Description | Retryable | Max Retries | Agent Action | User Guidance |
|-----------|------------|-------------|-----------|------------|--------------|---------------|
| DCS.0001 | 400 | InvalidParameter — malformed request params | Yes (1) | 1 | Fix args per OpenAPI spec | `[ERROR] DCS.0001: Invalid parameters. Check engine version, capacity, VPC ID against API docs.` |
| DCS.0002 | 404 | InstanceNotFound — instance ID does not exist | No | 0 | Verify instance_id | `[ERROR] DCS.0002: Instance not found. Verify the instance_id: {{user.instance_id}}` |
| DCS.0003 | 400 | InvalidInstanceStatus — instance not in expected state | Yes | 3 | Wait then poll status | `[ERROR] DCS.0003: Instance status invalid. Current state: {{output.status}}. Wait for RUNNING and retry.` |
| DCS.0007 | 403 | QuotaExceeded — DCS instance limit reached | No | 0 | HALT, suggest quota increase | `[ERROR] DCS.0007: Quota exceeded. Current instances: N/30. Request quota increase or delete unused instances.` |
| DCS.0011 | 403 | InsufficientBalance — account balance insufficient | No | 0 | HALT | `[ERROR] DCS.0011: Insufficient balance. Please recharge your Huawei Cloud account.` |
| DCS.0014 | 409 | InstanceAlreadyExists — name conflict | No | 0 | Suggest new name | `[ERROR] DCS.0014: Instance name already exists. Use unique name or reuse existing instance.` |
| DCS.0019 | 400 | SecurityGroupNotFound — SG does not exist | No | 0 | HALT, ask to create SG | `[ERROR] DCS.0019: Security group not found. Create SG with port 6379 allowed.` |
| DCS.0021 | 400 | VPCNotExists — VPC does not exist in region | No | 0 | HALT, verify VPC | `[ERROR] DCS.0021: VPC not found. Check VPC ID exists in region {{env.HW_REGION_ID}}.` |
| DCS.0025 | 400 | PasswordInvalid — password does not meet requirements | No | 0 | Fix password | `[ERROR] DCS.0025: Password must be 8-64 chars with letters, digits, and special characters.` |
| DCS.0038 | 400 | EngineNotSupported — invalid engine or version | No | 0 | Use valid engine | `[ERROR] DCS.0038: Unsupported engine. Use Redis 4.0/5.0/6.0 or Memcached.` |
| Throttling/429 | 429 | TooManyRequests — API rate limit exceeded | Yes | 3 | Exponential backoff | `⚠️ Rate limited. Retrying in {backoff}s...` |
| InternalError/5xx | 500/502/503 | Server-side error | Yes | 3 | Retry; then HALT with RequestId | `[ERROR] Internal error. Retrying... If persistent, escalate with RequestId: {{output.request_id}}` |

## Ordered Diagnostic Steps

### 1. Connection Refused to Redis

```
Step 1: Check instance status
  hcloud dcs show-instance --instance-id "dcs-xxx"
  → Must be "RUNNING"

Step 2: Check security group
  hcloud vpc show-security-group --sg-id "sg-xxx"
  → Must allow inbound TCP 6379 from app ECS IP

Step 3: Check whitelist
  hcloud dcs show-whitelist --instance-id "dcs-xxx"
  → Whitelist must include app ECS IP

Step 4: Test TCP connectivity from ECS
  telnet {dcs_ip} 6379
  → Must connect

Step 5: Test Redis AUTH
  redis-cli -h {dcs_ip} -p 6379 -a {password} PING
  → Must return "PONG"
```

### 2. Slow Query Performance / High Latency

```
Step 1: Check CPU usage via CES
  → Look at cpu_usage metric for instance

Step 2: Check memory usage
  → memory_usage > 90% indicates eviction overhead

Step 3: Run Redis SLOWLOG on instance
  → redis-cli -h {ip} -p {port} -a {pwd} SLOWLOG GET 10

Step 4: Check for big keys
  → redis-cli --bigkeys

Step 5: Check for hot keys
  → Monitor INFO COMMANDSTATS for per-command counts
```

### 3. OOM (Out of Memory)

```
Step 1: Check memory usage
  → hcloud dcs show-instance → max_memory_mb

Step 2: Check eviction policy
  → redis-cli CONFIG GET maxmemory-policy

Step 3: Check key count and memory breakdown
  → redis-cli INFO memory | grep used_memory

Step 4: Identify top consuming keys
  → redis-cli --bigkeys

Step 5: Resolution
  → Increase capacity (resize) OR clean up unnecessary keys
```

### 4. Instance Stuck in Creating State

```
Step 1: Check instance status
  → hcloud dcs show-instance → status

Step 2: Wait up to 300 seconds
  → Retry ShowInstance every 5 seconds

Step 3: If still CREATING after 300s → HALT
  → Report as ticket to Huawei Cloud with request_id

Step 4: If status = ERROR
  → Delete and recreate with same params
```

### 5. Backup Failed

```
Step 1: Check instance status during backup window
  → Instance must be RUNNING for backup

Step 2: Check storage availability
  → Backup data stored in OBS; verify OBS accessible

Step 3: Retry manual backup
  → hcloud dcs create-backup --instance-id "dcs-xxx" --name "test"

Step 4: Check backup list for error status
  → hcloud dcs list-backups --instance-id "dcs-xxx"
```

### 6. Password Reset Failed

```
Step 1: Verify instance status = RUNNING
Step 2: Verify new password meets complexity requirements
Step 3: Check that no other password reset is in progress
Step 4: Retry with valid password
Step 5: After success, notify all app teams to update connection strings
```

### 7. Network Timeout / Security Group Blocking

```
Step 1: Verify ECS and DCS are in same VPC (or have VPC peering)
Step 2: Verify security group inbound rule allows port 6379
Step 3: Verify whitelist includes ECS IP
Step 4: Check if network route table has correct VPC routes
Step 5: If VPC peering, verify peering status = "ACTIVE"
```

## Multi-Round Diagnosis Flow: DCS Instance Unreachable

```
Round 1: Basic Connectivity
  1. hcloud dcs show-instance → status = RUNNING?
  2. hcloud vpc show-security-group → port 6379 allowed?
  3. hcloud dcs show-whitelist → IP included?
  → If all pass, proceed to Round 2. If any fail, fix and retest.

Round 2: Application Level
  4. redis-cli PING from ECS
  5. redis-cli AUTH with correct password
  6. Check if app uses connection pooling → pool exhausted?
  → If still failing, proceed to Round 3.

Round 3: Network Infrastructure
  7. traceroute from ECS to DCS IP
  8. Check VPC route table for correct routes
  9. Check if VPC Endpoint (if used) is healthy
  → If still failing, proceed to Round 4.

Round 4: Escalation
  10. Collect: RequestId, timestamp, instance_id, operation attempted
  11. Submit ticket to Huawei Cloud support
  12. Do NOT include credentials in ticket
```

## Common Redis Operational Issues

### Cache Penetration (缓存穿透)
- **Symptom**: Massive requests for non-existent keys bypass cache → backend DB overload
- **Detection**: hit_rate suddenly drops below 30%, expired_keys spike
- **Mitigation**: Use Bloom Filter, cache null values with short TTL

### Cache Avalanche (缓存雪崩)
- **Symptom**: Many keys expire simultaneously → all requests hit backend
- **Detection**: hit_rate drops + commands count spikes in short time
- **Mitigation**: Add random jitter to TTL values, stagger expiration times

### Cache Breakdown (缓存击穿)
- **Symptom**: Hot key expires → concurrent requests flood backend
- **Detection**: Sudden latency spike on specific keys, single-key throughput spike
- **Mitigation**: Use mutex lock for hot key rebuild, set eternal TTL for critical keys

### Hot Key (热Key)
- **Symptom**: One key receives >50% of traffic → single-core bottleneck
- **Detection**: INFO COMMANDSTATS shows one key dominating, per-shard CPU skew
- **Mitigation**: Local cache on app side, split hot key into shards, use MULTI-GET

### Big Key (大Key)
- **Symptom**: Single large key (>10MB) blocks Redis thread during operations
- **Detection**: Latency spikes during GET/SET of specific keys, memory usage spike
- **Mitigation**: Split big hash/list into smaller keys, use HSCAN for iteration, delete in batches

## Recovery Procedures

| Scenario | Backup First? | Recovery Action | Expected RTO | Post-Recovery Verify |
|----------|--------------|----------------|-------------|---------------------|
| OOM | YES — create backup before resizing | Resize instance with larger capacity | 2–5 min | Check new max_memory_mb, run INFO memory |
| Instance crash | N/A (HA auto-failover) | Wait for auto-failover; if not, restart | <30s (HA) | PING test, verify data with DBSIZE |
| Data corruption | YES (if backup exists) | Restore from latest successful backup | 5–15 min | PING, INFO keyspace, application smoke test |
| Network disruption | N/A | Fix VPC/SG/whitelist, restart if needed | <2 min | telnet port 6379, redis-cli PING |
| Password compromise | N/A | Reset password, update all clients | <1 min | AUTH test, monitor connection logs |

## Escalation Matrix

| Scenario | Action | When to Escalate |
|----------|--------|-----------------|
| Instance stuck CREATING > 10 min | HALT, collect RequestId | Immediately to Huawei Cloud |
| Quota increase needed | HALT, inform user | After user confirms |
| InternalError persists after 3 retries | HALT, log full error | Immediately with RequestId |
| Data loss (no backup) | HALT, inform user | Immediately — this is critical |
| Cross-product issue (VPC, SG) | Delegate to vpc-ops | If vpc-ops cannot resolve |
