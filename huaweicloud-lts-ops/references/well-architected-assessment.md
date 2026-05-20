# Well-Architected Assessment — Huawei Cloud LTS

## Five Pillars + FinOps + SecOps + AIOps

### Security (安全性)

| Aspect | Assessment | Recommendation |
|--------|-----------|---------------|
| IAM | `lts:logGroup:createLogGroup` etc. per operation | Apply least-privilege policies per user/role |
| Encryption at rest | AES-256 default | Accept default unless compliance requires KMS |
| Encryption in transit | TLS 1.2+ | Ensure endpoints use HTTPS only |
| Network isolation | VPC Endpoint available | Use VPCEP for private access in production |
| Audit | CTS integration | Enable CTS to log all LTS API calls |
| Data isolation | Per project | Use separate projects for different environments |
| Credential management | AK/SK via env vars | Rotate every 90 days; never hardcode |

### Reliability (可靠性)

| Aspect | Assessment | Recommendation |
|--------|-----------|---------------|
| Multi-AZ | LTS is regional (multi-AZ by default) | No additional config needed |
| Log durability | Triple replication | Default for all ingested logs |
| Transfer retry | Built-in retry mechanism | Monitor with CES alarm on `lts_transfer_failed_count` |
| ICAgent resilience | Auto-restart on crash | Monitor ICAgent via CES heartbeat |
| Quota management | 100 groups, 200 streams/group | Monitor via CES; raise tickets before hitting limits |
| Disaster recovery | Cross-region not supported for LTS | Use OBS transfer to export logs cross-region |

### Cost Optimization (成本优化)

| Aspect | Guidance |
|--------|----------|
| **Billing model** | LTS is pay-per-use (no reserved capacity). Costs = ingestion + storage + index + transfer. |
| **TTL management** | Production: 30 days hot, then OBS cold storage. Dev: 7 days. |
| **Index cost** | Only index fields needed for search. Avoid indexing high-cardinality fields unnecessarily. |
| **Transfer cost** | Transfer within region is free; cross-region OBS transfer incurs network cost. |
| **Log volume control** | Set appropriate log levels (WARN+ in production). Avoid DEBUG in production. |
| **Idle detection** | Monitor `lts_log_volume`; if a stream has < 1MB/day for 7 days, suggest consolidation. |

**Cost comparison table:**

| Strategy | Ingestion Cost | Storage Cost | Search Cost | Use Case |
|----------|---------------|-------------|-------------|----------|
| Short TTL (7d), no transfer | Medium | Low | Medium | Dev/test |
| Medium TTL (30d), OBS transfer | Medium | Medium | Medium | Production standard |
| Long TTL (90d), indexed | Medium | High | High | Compliance/audit |
| No index, raw only | Medium | Medium | Low (full scan) | High-volume low-value logs |

### Performance (性能效率)

| Aspect | Guidance |
|--------|----------|
| **Search optimization** | Enable structured parsing and indexing for frequently-queried fields |
| **Query patterns** | Narrow time ranges improve speed dramatically |
| **Cross-stream search** | Limit to ≤ 10 streams per cross-stream query |
| **Dashboard refresh** | Set appropriate refresh intervals (≥ 5 min for production) |
| **Ingestion throughput** | Use the dedicated LTS Go SDK for high-volume production ingestion |

### Operational Excellence (卓越运营)

| Aspect | Guidance |
|--------|----------|
| **Naming conventions** | `{env}-{app}-{type}` (e.g., `prod-api-error`) |
| **Tagging** | Tag log groups with `env`, `app`, `team` for filtering |
| **Alerts** | CES alarms on log volume spike/drop, transfer failures |
| **Runbooks** | Use this skill for standard LTS operations |
| **Incident response** | LTS search is primary tool for incident investigation |

---

## FinOps Deep Dive

### Cost Visibility
- Use CES billing metrics to track LTS spend per project
- Tag log groups with cost center (`cost_center: team-alpha`)
- Monitor `lts_storage_usage` vs billing tier thresholds

### Cost Optimization
- **Right-sizing TTL**: 7 days for debug/dev, 30 days for production, 90+ days only for compliance
- **Index reduction**: Review indexed fields monthly; remove unused indices
- **Transfer tiering**: Use OBS Standard for hot logs (queried within 30d), OBS Infrequent Access for cool logs, OBS Archive for audit logs

### Idle Resource Detection
- Pattern: `lts_log_volume` < 1 MB/day for 7 consecutive days → suggest deleting or merging the stream
- Pattern: Quick searches not used in 90 days → suggest archiving

---

## SecOps Deep Dive

### IAM Minimum Permissions

| Role | Permissions | Use Case |
|------|-------------|----------|
| LTS Viewer | `lts:logGroup:listLogGroup`, `lts:logStream:listLogStream`, `lts:logs:listLogs` | Read-only operators |
| LTS Operator | Viewer + `lts:logGroup:createLogGroup`, `lts:logStream:createLogStream`, `lts:transfer:createTransfer` | Daily Ops |
| LTS Admin | `lts:*` | Full management |
| LTS Transfer | `lts:transfer:*` + `obs:object:PutObject` | Transfer automation |

### Network Security
- Use VPC Endpoint for LTS to avoid internet exposure
- Security group: allow ICAgent outbound to LTS endpoint over HTTPS (443)
- Restrict ICAgent source IPs to known host ranges

### Data Security
- Enable CTS for all LTS management events
- Set log retention based on compliance requirements (e.g., 180 days for finance)
- Use OBS bucket policies with encryption for transferred logs

---

## AIOps Deep Dive

### Anomaly Patterns (≥4)

| Pattern | Detection Logic | Severity | Response |
|---------|----------------|----------|----------|
| **Log volume spike** | `lts_log_volume` > 2x baseline for 5min | Warning | Query logs for error patterns; notify app team |
| **Zero ingestion** | `lts_log_volume` ≈ 0 for 10min | Critical | Check ICAgent; verify network; restart agent |
| **Transfer failure** | `lts_transfer_failed_count` > 0 for 5min | Critical | Verify OBS bucket; check policy; retry |
| **Search degradation** | `lts_log_search_latency` P95 > 5s | Warning | Review index; narrow queries; reduce scan |
| **Storage near full** | `lts_storage_usage` > 80% quota | Warning | Increase TTL or expand quota |
| **Rapid index growth** | `lts_index_volume` > 1.5x baseline for 24h | Info | Review index configuration; remove unused fields |

### Cross-Skill Diagnosis
Refer to `references/integration.md` Delegation Matrix for cross-skill routing.

### Fault Pattern Knowledge Base

| Fault Pattern | Symptoms | Root Cause | Resolution |
|--------------|----------|------------|------------|
| No logs in stream | Search returns empty | ICAgent not running or network down | Check ICAgent status; restart agent |
| Transfer missing files | OBS bucket has gaps | Bucket policy change | Re-create transfer rule |
| Search timeout | Query takes > 30s | Missing index or full scan | Enable structured parsing |
| Log group undeletable | `LTS.0402` | Active transfer rule | List & delete transfers first |
| Stream creation fails | `LTS.0101` | Stream quota exceeded (200/group) | Delete unused streams |
