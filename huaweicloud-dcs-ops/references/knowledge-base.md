# Knowledge Base — Huawei Cloud DCS Fault Patterns

## Product Fault Pattern Library

### FP-001: OOM (Out of Memory)

| Field | Value |
|-------|-------|
| **Pattern ID** | FP-001 |
| **Symptom** | Instance becomes unresponsive, `redis-cli PING` fails or returns very slowly, `memory_usage` at 99%+ |
| **Trigger Conditions** | `memory_usage > 95%` AND `evicted_keys > 0` AND `hit_rate < 50%` |
| **Root Cause** | Dataset exceeds maxmemory capacity; eviction policy cannot free enough memory |
| **Diagnosis Flow** | 1. `hcloud dcs show-instance` → check max_memory_mb<br>2. `redis-cli INFO memory` → check used_memory vs maxmemory<br>3. `redis-cli --bigkeys` → identify large consumers<br>4. `redis-cli INFO keyspace` → check key count per DB |
| **Resolution Steps** | 1. If running: create backup first<br>2. Resize instance with larger capacity<br>3. If resize impossible: delete unnecessary keys (`FLUSHDB`, `DEL`)<br>4. Set maxmemory-policy to appropriate eviction (volatile-lru, allkeys-lru) |
| **Prevention** | Monitor memory_usage > 80% threshold, set maxmemory-policy, use Redis 6.0 with proper eviction |
| **CES Metrics** | memory_usage, evicted_keys, hit_rate |
| **Cross-Reference** | Error codes: DCS.0003 (InvalidInstanceStatus when OOM causes state change) |

### FP-002: Connection Limit Exceeded

| Field | Value |
|-------|-------|
| **Pattern ID** | FP-002 |
| **Symptom** | New Redis connections refused, `redis-cli` shows "ERR max number of clients reached" |
| **Trigger Conditions** | `connected_clients > max_clients * 0.95` |
| **Root Cause** | Application connection pool leak or insufficient max-connections in pool config |
| **Diagnosis Flow** | 1. `redis-cli CLIENT LIST` → check all connected clients and their addresses<br>2. `redis-cli CLIENT GETNAME` → check if connection names identify apps<br>3. Review application connection pool settings (max-active, max-idle, test-on-borrow) |
| **Resolution Steps** | 1. Restart affected application pods (forces connection cleanup)<br>2. Fix application connection pool configuration<br>3. If legitimate growth: resize DCS to higher spec with more max clients |
| **Prevention** | Set connection pool `max-active` to ≤ 80% of DCS max clients, enable pool idle timeout |
| **CES Metrics** | connected_clients, latency |
| **Cross-Reference** | Error codes: DCS.0003 (instance operational but connections limited) |

### FP-003: Cache Avalanche (缓存雪崩)

| Field | Value |
|-------|-------|
| **Pattern ID** | FP-003 |
| **Symptom** | Backend database load suddenly spikes, application timeout errors increase dramatically |
| **Trigger Conditions** | `hit_rate < 50%` within 5 min window AND `expired_keys > 3x normal rate` AND `commands/sec` spike |
| **Root Cause** | Large number of keys with same TTL expire simultaneously; all subsequent requests miss cache and hit backend |
| **Diagnosis Flow** | 1. Check `INFO stats` for expired_keys rate<br>2. Check `INFO keyspace` for key counts per DB<br>3. Correlate with backend database QPS spike |
| **Resolution Steps** | 1. Short-term: throttle application requests to backend DB<br>2. Re-populate cache for critical keys with staggered TTLs<br>3. Add random jitter (±30%) to all new key TTLs |
| **Prevention** | Never set identical TTLs for bulk keys; use `TTL = base + random(0, max_jitter)`;<br>warm up cache before peak traffic |
| **CES Metrics** | hit_rate, expired_keys, commands |
| **Cross-Reference** | Related to FP-001 (OOM) if cache rebuild floods memory |

### FP-004: Cache Penetration (缓存穿透)

| Field | Value |
|-------|-------|
| **Pattern ID** | FP-004 |
| **Symptom** | High request rate with extremely low hit rate, backend DB queries for non-existent keys |
| **Trigger Conditions** | `hit_rate < 30%` sustained AND `expired_keys` normal (not avalanche) AND specific key patterns queried repeatedly |
| **Root Cause** | Attack or bug generating requests for keys that don't exist and never will; cache miss every time |
| **Diagnosis Flow** | 1. `redis-cli --stat` → check hit rate in real-time<br>2. Check application logs for query patterns<br>3. Identify key patterns being queried (e.g., sequential user IDs that don't exist) |
| **Resolution Steps** | 1. Deploy Bloom Filter in application to reject known non-existent keys<br>2. Cache null values with short TTL (e.g., 60s) for frequently-missed keys<br>3. Implement rate limiting at application gateway |
| **Prevention** | Bloom Filter for key existence check, cache negative results, input validation, rate limiting |
| **CES Metrics** | hit_rate, commands |
| **Cross-Reference** | Different from FP-003 (avalanche): here expired_keys is normal, specific key patterns are queried |

### FP-005: Master-Standby Split Brain (双主脑裂)

| Field | Value |
|-------|-------|
| **Pattern ID** | FP-005 |
| **Symptom** | Data inconsistency between master and standby, potential data loss after failover |
| **Trigger Conditions** | `instance status = ERROR` AND replication link broken AND both nodes accepting writes |
| **Root Cause** | Network partition between master and standby nodes; both become primary independently |
| **Diagnosis Flow** | 1. `hcloud dcs show-instance` → check status<br>2. `redis-cli INFO replication` → check replication role and connected_slaves<br>3. Check VPC network connectivity between AZs<br>4. Compare key counts on master vs standby |
| **Resolution Steps** | 1. If DCS managed: trust auto-recovery (re-sync from surviving master)<br>2. If split brain persists: manual restart instance<br>3. After recovery: compare data checksums between replicas<br>4. Accept possible data loss from partitioned period |
| **Prevention** | Use HA mode in multi-AZ deployment, monitor replication lag, alert on network partition |
| **CES Metrics** | instance status, memory_usage divergence between nodes |
| **Cross-Reference** | Error codes: DCS.0003 (InvalidInstanceStatus → ERROR) |

### FP-006: Big Key Blocking (大Key阻塞)

| Field | Value |
|-------|-------|
| **Pattern ID** | FP-006 |
| **Symptom** | Latency spikes (>100ms) for simple GET/SET commands, periodic slowdowns every few seconds |
| **Trigger Conditions** | `latency > 100ms` AND `memory_usage` normal AND `connected_clients` normal |
| **Root Cause** | Single key with large value (>10MB) or collection with millions of elements blocks Redis during serialization |
| **Diagnosis Flow** | 1. Run `redis-cli --bigkeys` (non-destructive scan)<br>2. `redis-cli --latency` to measure real-time latency<br>3. SLOWLOG: `redis-cli SLOWLOG GET 20` to find blocking commands |
| **Resolution Steps** | 1. Split big hash into smaller hashes with hash tags<br>2. For large lists: use LRANGE with pagination, not GET all<br>3. Delete big keys during off-peak using SCAN + DEL in batches<br>4. Set a per-key size limit in application logic |
| **Prevention** | Application-side key size validation, periodic big key scans, use Redis 6.0 thread I/O |
| **CES Metrics** | latency, memory_usage |
| **Cross-Reference** | May trigger FP-001 (OOM) if big keys accumulate over time |

### FP-007: Hot Key Contention (热Key竞争)

| Field | Value |
|-------|-------|
| **Pattern ID** | FP-007 |
| **Symptom** | Single key accounts for >50% of total operations, single-core CPU bottleneck on one shard |
| **Trigger Conditions** | Per-key QPS > 10,000 (or > 50% of total QPS) AND per-shard CPU skew |
| **Root Cause** | One key (e.g., trending item, session token) receives disproportionate traffic |
| **Diagnosis Flow** | 1. `INFO COMMANDSTATS` → check per-command distribution<br>2. Application-side: track per-key access counts<br>3. Check for single-shard CPU spike in cluster mode |
| **Resolution Steps** | 1. Local cache: cache hot key in application memory (e.g., Guava, Caffeine)<br>2. Read replicas: use RW-split mode to distribute reads<br>3. Key sharding: split hot key into `hot_key:1`, `hot_key:2`, ..., `hot_key:N`<br>4. CDN/offload: serve from CDN if data is static-ish |
| **Prevention** | Monitor per-key QPS in application, implement local cache for known hot patterns |
| **CES Metrics** | commands, cpu_usage (per-shard), latency |
| **Cross-Reference** | Related to FP-006 (big key) in that a single key causes disproportionate load |

## Cross-Product Cascade Faults

### CF-001: VPC Disruption → DCS Unreachable

| Field | Value |
|-------|-------|
| **Cascade ID** | CF-001 |
| **Root Product** | VPC (`huaweicloud-vpc-ops`) |
| **Affected Product** | DCS |
| **Symptom** | All DCS instances in a subnet become unreachable from application ECS |
| **Trigger** | VPC route table change, subnet deletion, VPC peering disruption |
| **Diagnosis** | 1. Check VPC route table for DCS subnet<br>2. Verify subnet exists and is not deleted<br>3. Check VPC peering status (if applicable)<br>4. Check if VPC endpoint is still active |
| **Resolution** | 1. Fix route table to include correct VPC routes<br>2. If subnet deleted: recreate subnet, migrate DCS (if supported) or recreate<br>3. If VPC peering: re-establish peering connection |
| **Prevention** | Monitor VPC route table changes, use CTS to audit network changes |

### CF-002: ECS Crash → Redis Client Pool Exhaustion

| Field | Value |
|-------|-------|
| **Cascade ID** | CF-002 |
| **Root Product** | ECS (`huaweicloud-ecs-ops`) |
| **Affected Product** | DCS |
| **Symptom** | DCS max clients reached, new connections refused for ALL applications |
| **Trigger** | Application ECS instance crashes without closing Redis connections |
| **Diagnosis** | 1. Check DCS `connected_clients` count vs max_clients<br>2. `redis-cli CLIENT LIST` → identify stale connections (idle > 30s)<br>3. Check ECS instance status → crashed/terminated |
| **Resolution** | 1. Restart affected ECS instances (forces connection cleanup)<br>2. If ECS unavailable: kill idle connections via `CLIENT KILL`<br>3. Monitor `connected_clients` returning to normal |
| **Prevention** | Configure connection pool idle timeout, enable TCP keepalive, use ECS Auto Healing |

### CF-003: Security Group Change → DCS Port Blocked

| Field | Value |
|-------|-------|
| **Cascade ID** | CF-003 |
| **Root Product** | VPC Security Group (`huaweicloud-vpc-ops`) |
| **Affected Product** | DCS |
| **Symptom** | All application connections to DCS timeout simultaneously |
| **Trigger** | Network team modifies security group, removing inbound 6379 rule |
| **Diagnosis** | 1. Check SG inbound rules → port 6379 allowed?<br>2. Check if SG change correlates with connection failures (CTS audit log)<br>3. Test telnet from ECS to DCS port |
| **Resolution** | 1. Add inbound rule: TCP 6379 from ECS security group or subnet CIDR<br>2. Verify connection restoration |
| **Prevention** | CTS monitoring for security group changes, change management process, pre-change testing |
