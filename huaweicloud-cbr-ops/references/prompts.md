# Prompts — Huawei Cloud CBR

> **Purpose:** Structured prompts for CBR (Cloud Backup and Recovery) AIOps operations.
> **Version:** 1.0.0
> **Status:** Reference document

---

## 1. Diagnostic Prompts

### 1.1 Vault Health Check
```
Analyze CBR vault {{resource_id}} health status:
- Vault type: {{vault_type}} (server/file/disk)
- Storage used: {{storage_used}}GB / {{storage_quota}}GB ({{utilization}}%)
- Protected instances: {{protected_count}}
- Backup status: {{backup_status}}
- Last backup: {{last_backup_time}}
- Next scheduled: {{next_backup}}
Determine vault health and recommend actions.

Applicable CES metrics: SYS.CBR.backup_task_status, SYS.CBR.storage_usage, AGT.CBR.backup_progress
```

### 1.2 Backup Task Analysis
```
Analyze CBR backup task on {{resource_id}}:
- Task ID: {{task_id}}
- Task type: {{task_type}} (full/incremental/differential)
- Status: {{task_status}} (running/success/failed)
- Progress: {{progress}}% ({{completed_size}}GB / {{total_size}}GB)
- Duration: {{duration}} minutes (expected: {{expected_duration}}min)
- Speed: {{backup_speed}}MB/s
Diagnose task status.

CBR task states: queued, running, completing, success, failed, cancelled
```

### 1.3 Restore Progress Analysis
```
Analyze CBR restore progress on {{resource_id}}:
- Restore ID: {{restore_id}}
- Source backup: {{backup_id}}
- Target: {{target_resource}} ({{target_type}})
- Progress: {{progress}}%
- Data restored: {{data_restored}}GB / {{total_size}}GB
- Duration: {{duration}} minutes
- Estimated completion: {{eta}} minutes
Monitor restore operation.

CBR restore: volume restore, file restore, cross-region restore
```

### 1.4 Storage Utilization Analysis
```
Analyze CBR storage utilization on {{resource_id}}:
- Storage used: {{storage_used}}GB / {{storage_quota}}GB ({{utilization}}%)
- Storage trend: {{storage_trend}}GB/day
- Backup retention: {{retention_days}} days
- Backup count: {{backup_count}} (full: {{full_count}}, incr: {{incr_count}})
- Orphaned snapshots: {{orphaned_count}}
- Projected exhaustion: {{exhaustion_date}}
Assess storage capacity.

CBR storage: incremental backup chain, snapshot compression, retention policy
```

### 1.5 Replication Status Analysis
```
Analyze CBR replication on {{resource_id}}:
- Replication enabled: {{replication_enabled}}
- Source vault: {{source_vault}}
- Target region: {{target_region}}
- Replication lag: {{replication_lag}} minutes
- Last sync: {{last_sync_time}}
- Sync status: {{sync_status}}
- RPO achievement: {{rpo_achieved}}min (target: {{rpo_target}}min)
Monitor cross-region replication.

CBR replication: async replication, cross-region backup, RPO tracking
```

---

## 2. Root Cause Analysis Prompts

### 2.1 Backup Failure Analysis
```
Analyze CBR backup failure on {{resource_id}}:
- Task ID: {{task_id}}
- Error code: {{error_code}}
- Error message: {{error_message}}
- Failure time: {{failure_time}}
- Retry count: {{retry_count}} (max: {{max_retries}})
- Last successful backup: {{last_success}}
- Affected resources: {{affected_resources}}
Identify failure cause.

CBR failure causes: quota exceeded, agent issue, network timeout, resource locked, encryption mismatch
```

### 2.2 Restore Failure Analysis
```
Analyze CBR restore failure on {{resource_id}}:
- Restore ID: {{restore_id}}
- Error code: {{error_code}}
- Error message: {{error_message}}
- Target resource: {{target_resource}}
- Target status: {{target_status}}
- Backup age: {{backup_age}} days
Diagnose restore issue.

CBR restore failures: target disk full, resource conflict, encryption key missing, snapshot expired
```

### 2.3 Agent Issue Diagnosis
```
Diagnose CBR agent issue on {{resource_id}}:
- Instance: {{instance_id}}
- Agent status: {{agent_status}} (online/offline/error)
- Agent version: {{agent_version}}
- Last heartbeat: {{last_heartbeat}} minutes ago
- Backup capability: {{backup_capability}}
- Installed plugins: {{plugins}}
Identify agent problem.

CBR agent: hcloud-agent, installation, registration, plugin compatibility
```

### 2.4 Storage Quota Analysis
```
Analyze CBR storage quota on {{resource_id}}:
- Storage quota: {{storage_quota}}GB
- Storage used: {{storage_used}}GB ({{utilization}}%)
- Quota by vault: {{vault_usage}}
- Projected fill: {{projected_fill_date}} at {{growth_rate}}GB/day
- Available expansion: {{expansion_options}}
Assess quota risk.

CBR quota: per-vault limits, cross-vault aggregation, pay-per-use vs package
```

### 2.5 Backup Consistency Issue
```
Analyze CBR backup consistency on {{resource_id}}:
- Backup ID: {{backup_id}}
- Consistency status: {{consistency_status}}
- Checksum validation: {{checksum_status}}
- Application consistent: {{app_consistent}}
- Crash consistent: {{crash_consistent}}
- Excluded volumes: {{excluded_volumes}}
Identify consistency risk.

CBR consistency: VSS (Windows), quiesce (Linux), crash-consistent backup
```

---

## 3. Capacity Prompts

### 3.1 Capacity Planning Review
```
Review CBR capacity for {{resource_id}}:
- Current storage: {{storage_used}}GB / {{storage_quota}}GB
- Daily growth: {{daily_growth}}GB
- Backup count: {{backup_count}} (retain: {{retain_count}})
- Monthly growth: {{monthly_growth}}GB
- Projected exhaustion: {{exhaustion_date}}
Provide scaling recommendations.

Capacity dimensions: storage quota, backup count, retention period, replication bandwidth
```

### 3.2 Retention Policy Optimization
```
Optimize CBR retention for {{resource_id}}:
- Current retention: {{retention_days}} days
- Backup frequency: {{backup_frequency}} (daily/weekly/monthly)
- Backup chain: full {{full_count}}, incr {{incr_count}}
- Storage per backup: {{storage_per_backup}}GB
- Recommended retention: {{recommended_retention}} days
- Potential savings: {{savings_gb}}GB
Recommend policy tuning.

CBR retention: GFS (grandfather-father-son), legal hold, compliance requirements
```

### 3.3 Backup Schedule Optimization
```
Optimize CBR backup schedule for {{resource_id}}:
- Current schedule: {{current_schedule}} (start: {{start_time}})
- Backup duration: {{backup_duration}} minutes
- Business impact window: {{impact_window}}
- RPO target: {{rpo_target}} hours
- Current RPO: {{current_rpo}} hours
- Recommended schedule: {{recommended_schedule}}
Balance RPO and performance.

CBR scheduling: off-peak hours, staggered starts, frequency vs RPO
```

### 3.4 Storage Tier Analysis
```
Analyze CBR storage tiering for {{resource_id}}:
- Standard storage: {{standard_gb}}GB
- Archive storage: {{archive_gb}}GB (after {{archive_after}} days)
- Storage cost: {{standard_cost}} CNY + {{archive_cost}} CNY
- Access frequency: {{access_frequency}}
- Recommended transition: {{transition_plan}}
Optimize storage costs.

CBR tiering: standard→archive after N days, on-demand restore, early deletion fees
```

---

## 4. Availability Prompts

### 4.1 Disaster Recovery Readiness
```
Assess CBR DR readiness for {{resource_id}}:
- Replication: {{replication_enabled}}
- Target region: {{target_region}}
- RPO: {{rpo_achieved}}min (target: {{rpo_target}}min)
- RTO: {{rto_achieved}}min (target: {{rto_target}}min)
- Last DR test: {{last_dr_test}}
- DR test result: {{dr_test_result}}
Evaluate DR capabilities.

CBR DR: cross-region replication, cross-region restore, DR drill
```

### 4.2 Backup Coverage Analysis
```
Analyze backup coverage for {{resource_id}}:
- Total resources: {{total_resources}}
- Protected resources: {{protected_resources}} ({{protected_pct}}%)
- Unprotected resources: {{unprotected_resources}}
- Protection by type: {{protection_by_type}}
- Last backup age: {{last_backup_age}} days (max allowed: {{max_age}} days)
Identify coverage gaps.

CBR coverage: all resources protected, critical resources prioritized, RPO compliance
```

### 4.3 Restore Testing Status
```
Check CBR restore testing for {{resource_id}}:
- Last restore test: {{last_restore_test}}
- Test result: {{test_result}}
- Tested resources: {{tested_resources}} / {{total_resources}}
- Restore time: {{restore_time}} minutes (target: {{target_rto}}min)
- Data integrity: {{integrity_check}}%
Validate recoverability.

CBR restore test: file-level restore, volume restore, cross-region restore
```

### 4.4 SLA Compliance Report
```
Report CBR SLA compliance for {{resource_id}}:
- Backup success rate: {{backup_success_rate}}% (target: {{target_rate}}%)
- RPO compliance: {{rpo_compliance}}% (target: {{rpo_target}}min)
- RTO compliance: {{rto_compliance}}% (target: {{rto_target}}min)
- Restore success rate: {{restore_success_rate}}%
- Data loss incidents: {{data_loss_incidents}}
Report SLA violations.

CBR SLA: backup success, RPO achievement, restore success, data integrity
```

---

## 5. Inspection Prompts

### 5.1 Routine Health Inspection
```
Perform routine CBR inspection:
- List all CBR vaults in {{scope}}
- Check storage utilization > {{threshold_util}}%
- Identify failed backups > {{failed_threshold}}
- Flag backups older than {{age_threshold}} days
- Check replication lag > {{lag_threshold}}min
- Verify all critical resources protected
- Check for orphaned backups
Report findings.

Scope: region, project, tag-based resource group
```

### 5.2 Security Compliance Check
```
Audit CBR security compliance:
- Encryption at rest: {{encryption_at_rest}} (AES256/KMS)
- Encryption in transit: {{encryption_in_transit}}
- Key management: {{key_management}} (BYOK/HWSD)
- Access control: {{access_control}} (IAM policy)
- Audit logging: {{audit_logging}} (CTS enabled)
- Backup isolation: {{backup_isolation}}
Report compliance status.

Severity: High = no encryption, Medium = shared encryption key, Critical = no audit
```

### 5.3 Cost Optimization Scan
```
Scan CBR for cost optimization:
- Identify unused vaults (no backups > 90 days)
- Find oversized storage quotas
- Check premature archive transition
- Verify backup chaining efficiency
- Review retention vs compliance need
- Check reserved capacity vs pay-per-use
Provide action list with estimated savings.

CBR cost: storage consumption, archive storage, cross-region replication, API calls
```

### 5.4 Backup Chain Integrity
```
Inspect backup chain integrity on {{resource_id}}:
- Full backups: {{full_count}}
- Incremental chain: {{incr_chain_length}}
- Broken chain: {{broken_chain_count}}
- Orphaned snapshots: {{orphaned_count}}
- Checksum validation: {{checksum_status}}
- Consistency check: {{consistency_status}}
Recommend chain repair.

CBR chain: full-incremental chain, broken chain recovery, incremental merge
```

### 5.5 Configuration Drift Detection
```
Detect configuration drift for CBR {{resource_id}}:
- Expected retention: {{expected_retention}} days
- Actual retention: {{actual_retention}} days
- Expected schedule: {{expected_schedule}}
- Actual schedule: {{actual_schedule}}
- Expected replication: {{expected_replication}}
- Actual replication: {{actual_replication}}
Recommend reconciliation.

CBR drift: retention policy changes, schedule changes, replication toggle
```

---

## Appendix: CBR-Specific Placeholders

| Placeholder | Description | Example |
|-------------|-------------|---------|
| `{{resource_id}}` | CBR vault ID | `vault-abcd1234` |
| `{{vault_type}}` | Vault type | `server` |
| `{{task_id}}` | Backup task ID | `backup-12345` |
| `{{storage_used}}` | Storage used GB | `245.5` |
| `{{backup_id}}` | Backup ID | `backup-def456` |

---

*Prompts version 1.0.0 — for AIOps L4 compliance*
