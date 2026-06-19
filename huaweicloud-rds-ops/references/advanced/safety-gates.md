# RDS Safety Gates — High-Risk Operation Controls

> Advanced safety controls for Relational Database Service.
> Load when deleting instances, restoring backups, or executing DDL/DML
> automation flows.

## 1. Destructive Operation Catalogue

| Operation | Risk class | Default gate |
|-----------|-----------|--------------|
| `DeleteInstance` | irreversible (data) | final snapshot + cross-region copy + confirmation |
| `RestoreInstance` | overwrite existing | dry-run + maintenance window |
| `ResetPassword` | credential rotation | confirm new password delivery |
| `Failover` | brief downtime | low-traffic window + smoke test |
| `Switchover` | planned HA drill | maintenance window + rollback plan |

## 2. Safety Gate Workflow

1. **Inventory**: list affected DB instance IDs
2. **Pre-snapshot**: trigger manual backup + cross-region copy
3. **Confirm**: collect `{{user.confirm_destructive}}` per instance
4. **Execute**: dry-run, then apply via SDK / CLI
5. **Verify**: poll `status` until `ACTIVE`; run smoke queries
6. **Audit**: emit CTS event + surface `{{output.rds_change_record}}`

## 3. Cross-Skill Delegation

- `huaweicloud-rds-ops → huaweicloud-cbr-ops` for backup / restore
- `huaweicloud-rds-ops → huaweicloud-dcs-ops` for cache invalidation
- `huaweicloud-rds-ops → huaweicloud-iam-ops` for credential rotation
- `huaweicloud-rds-ops → huaweicloud-cts-ops` for audit trail

> **Security-Sensitive**: destructive operations above MUST pass the Safety
> Gate. Always pre-snapshot and cross-region copy before delete; never run
> DDL automation against production without a maintenance window.