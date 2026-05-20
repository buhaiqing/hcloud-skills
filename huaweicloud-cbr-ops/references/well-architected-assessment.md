# Well-Architected Assessment — Huawei Cloud CBR

## 1. Security (安全)

### IAM Minimum Permissions

| Operation | Required Policy | Action |
|-----------|----------------|--------|
| List vaults | `CBR ReadOnlyAccess` | `cbr:vaults:list` |
| Create vault | `CBR FullAccess` | `cbr:vaults:create` |
| Delete vault | `CBR FullAccess` | `cbr:vaults:delete` |
| Create backup | `CBR FullAccess` | `cbr:backups:create` |
| Restore backup | `CBR FullAccess` | `cbr:backups:restore` |
| Delete backup | `CBR FullAccess` | `cbr:backups:delete` |

### Network Security

- Backup traffic stays within Huawei Cloud network
- No public internet exposure for backup data
- Vault-level access policies for additional isolation
- KMS encryption for backup data at rest

### Data Protection

- KMS SSE-KMS encryption for backup data
- Encrypted in transit (HTTPS for API, internal network for data)
- Immutable backups (write-once, retention-enforced)
- Cross-region replication with encryption

## 2. Stability (稳定)

### High Availability

- CBR service is region-resilient (multi-AZ by default)
- Backup data replicated within region across AZs
- Cross-region replication for DR

### Disaster Recovery

| Phase | Action | RTO | RPO |
|-------|--------|-----|-----|
| Phase 1 | Failover to DR region | <30 min | Based on replication schedule |
| Phase 2 | Restore critical resources from backup | <2 hours | <24 hours (daily backup) |
| Phase 3 | Restore remaining resources | <8 hours | <24 hours |
| Phase 4 | Validate data integrity | <1 hour | N/A |

## 3. Cost (成本)

### Billing Model Comparison

| Model | Storage Cost | Replication Cost | Best For |
|-------|-------------|-----------------|----------|
| 按需 (On-demand) | Per GB/hour | Per GB transferred | Variable backup needs |
| 包年包月 | Up to 50% discount | Included | Predictable backup capacity |

### Cost Optimization

| Pattern | Detection | Recommendation |
|---------|-----------|---------------|
| Over-retained backups | Vault > 80% full | Reduce retention period |
| Orphaned vaults | Vault with 0 backups for 30 days | Delete vault |
| Inefficient backup schedule | Daily full backup | Switch to incremental with weekly full |

## 4. Efficiency (效率)

- Use backup policies for automated scheduling
- Tag vaults with `env`, `app`, `team` for management
- Use `ListBackups` filters to find specific restore points
- JIT Go SDK for complex automation scripts

## 5. Performance (性能)

| Operation | Typical Duration | Notes |
|-----------|-----------------|-------|
| ECS full backup (50GB) | 15-30 min | Depends on disk I/O |
| Incremental backup (1GB changed) | 2-5 min | Fast for regular intervals |
| Cross-region replication (50GB) | 1-4 hours | Depends on bandwidth |
| Restore (50GB) | 10-30 min | Depends on disk speed |

## 6. FinOps (财务运营)

| Tool | Purpose |
|------|---------|
| CES `vault_used_percent` | Track storage utilization |
| BSS cost analysis | Monthly backup spending by vault |
| Tag-based cost allocation | Chargeback to teams |

## 7. SecOps (安全运营)

| Control | Implementation |
|---------|---------------|
| IAM least privilege | Use `CBR ReadOnlyAccess` for auditors |
| Encrypted backups | KMS keys with rotation |
| Vault access policy | Restrict which IAM users can restore |
| Audit logging | CTS for all backup/restore events |
| Backup immutability | Retention policy prevents early deletion |

## 8. AIOps (智能运营)

### Anomaly Patterns

| Pattern | Detection Logic | Severity |
|---------|----------------|----------|
| Backup failure spike | > 2 failures per vault in 24h | Critical |
| Vault capacity trend | Usage > 80% with >5% weekly growth | Warning |
| Slow backup | Duration > 2x historical average | Warning |
| Restore failure | Any restore failure | Critical |
| Backup chain broken | Missing incremental backup | Warning |
| Replication delay | Backup not replicated within 24h | Warning |

### Cross-Skill Diagnosis

| Symptom | Primary Skill | Supporting Skills |
|---------|--------------|-------------------|
| Backup failure (resource) | `huaweicloud-cbr-ops` | `huaweicloud-ecs-ops` / `huaweicloud-rds-ops` |
| Permission denied | `huaweicloud-cbr-ops` | `huaweicloud-iam-ops` |
| Vault capacity full | `huaweicloud-cbr-ops` | `huaweicloud-ces-ops` (metrics) |
| Cross-region replication slow | `huaweicloud-cbr-ops` | `huaweicloud-vpc-ops` (network) |
