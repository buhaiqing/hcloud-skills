# Knowledge Base — LTS

> **Purpose**: Operational knowledge base for Huawei Cloud LTS troubleshooting.
> **Version**: 1.0.0
> **Last Updated**: 2026-07-18

---

## 1. Common Issues and Resolutions

### 1.1 Ingestion Issues

| Issue | Symptoms | Root Cause | Resolution |
|-------|----------|------------|------------|
| Log gap | Missing logs in time range | ICAgent down or network issue | Restart ICAgent, check network |
| Rate limit | `LTS.0203` error code | Quota exceeded | Increase quota or optimize ingestion |
| Index delay | Search results stale | Indexing backlog | Check shard capacity, split if needed |

### 1.2 Storage Issues

| Issue | Symptoms | Root Cause | Resolution |
|-------|----------|------------|------------|
| Storage full | `LTS.0301` error code | Quota exceeded | Delete old logs, expand storage |
| Slow query | Query timeout | Too large log group | Partition by time range |
| Data loss | Logs missing after transfer | OBS permission issue | Verify OBS bucket policy |

### 1.3 Access Issues

| Issue | Symptoms | Root Cause | Resolution |
|-------|----------|------------|------------|
| Permission denied | `LTS.0202` error code | IAM policy missing | Grant `lts:*` permission |
| AK/SK invalid | `LTS.0101` error code | Credential expired | Update AK/SK |
| Region mismatch | `LTS.0103` error code | Wrong region | Specify correct `HW_REGION_ID` |

---

## 2. Error Code Reference

| Error Code | Description | Severity | Resolution |
|------------|-------------|----------|------------|
| LTS.0101 | Invalid credentials | Critical | Verify AK/SK |
| LTS.0103 | Region not available | Warning | Check region support |
| LTS.0201 | Quota exceeded | Warning | Increase quota or cleanup |
| LTS.0202 | Permission denied | Critical | Check IAM policy |
| LTS.0203 | Rate limit exceeded | Warning | Retry with backoff |
| LTS.0301 | Storage quota exceeded | Warning | Delete old data or expand |
| LTS.0401 | Transfer failed | Warning | Check target service |
| LTS.0402 | OBS bucket unreachable | Warning | Verify OBS connectivity |

---

## 3. Operational Runbooks

### 3.1 Log Gap Detection

1. Check ICAgent status on source hosts
2. Verify LTS ingestion quota
3. Query CTS for potential deletion events
4. Check network connectivity between agent and LTS

### 3.2 Storage Expansion

1. Assess current storage usage and growth trend
2. Calculate required additional storage
3. Create new log group with extended TTL if needed
4. Archive or delete old logs from existing groups

### 3.3 Query Performance Optimization

1. Narrow search time range
2. Use key-value indexing for specific fields
3. Partition large log groups by time
4. Consider creating dedicated log groups per service

---

## 4. Security Considerations

> **Security-Sensitive**: Log stream deletion, log group purge, and
> cross-region transfer MUST require explicit operator confirmation and
> preserve evidence under the documented retention policy.

| Operation | Risk Level | Required Confirmation |
|-----------|------------|----------------------|
| Delete log group | Critical | Explicit confirmation + CTS evidence |
| Purge log stream | Critical | Explicit confirmation + backup verification |
| Cross-region transfer | High | Confirm target region compliance |
| Modify retention | Medium | Document business justification |
