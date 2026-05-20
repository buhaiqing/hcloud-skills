# GaussDB Security Best Practices (SecOps)

## 1. IAM Policy Hardening

**Minimum required actions by role**:

| Role | Required IAM Action Set |
|------|----------------------|
| Read-only Auditor | `gaussdb:List*`, `gaussdb:Show*` |
| DB Administrator | `gaussdb:Create*`, `gaussdb:Update*`, `gaussdb:Delete*`, `gaussdb:List*`, `gaussdb:Show*` |
| Backup Operator | `gaussdb:Create*Backup*`, `gaussdb:ListBackups`, `gaussdb:Delete*Backup*` |
| Security Admin | Add `kms:Decrypt`, `kms:Encrypt` for disk encryption |

**IAM policy example** (read-only):
```json
{
  "Version": "1.1",
  "Statement": [
    {
      "Effect": "Allow",
      "Action": [
        "gaussdb:List*",
        "gaussdb:Show*"
      ]
    }
  ]
}
```

## 2. Network Security

- **Deploy in private subnet**: Never expose GaussDB directly to the internet.
- **EIP management**: Bind EIP only temporarily for migration, unbind immediately after.
- **Security group rules**: Restrict ingress to port 8000 from known application CIDRs only.
- **VPC peering**: Use VPC peering or Direct Connect for cross-VPC access.

## 3. Data Encryption

- **At-rest encryption**: Enable `disk_encryption_id` (KMS key) during instance creation.
  ```bash
  hcloud GaussDB CreateInstance \
    --name="prod-gauss" \
    --flavor_ref="gaussdb.opengauss.4xlarge.x864.8" \
    --disk_encryption_id="{{env.KMS_KEY_ID}}" \
    --volume.type="ULTRAHIGH" --volume.size=200
  ```
- **In-transit encryption**: Enforce SSL for all client connections.
  ```bash
  hcloud GaussDB ShowSslCertDownloadLink --instance_id="{{env.GAUSSDB_INSTANCE_ID}}"
  ```

## 4. Database Account Security

- **Least privilege per account**: Create separate DB users for read-only, read-write, DDL operations.
  ```sql
  -- Example: Read-only user
  CREATE USER reader WITH PASSWORD '{{env.READER_PASSWORD}}';
  GRANT SELECT ON ALL TABLES IN SCHEMA public TO reader;
  ```
- **Regular password rotation**: Use `SetDbUserPwd()` to rotate passwords.
- **Remove orphaned accounts**: Query `ListDbUsers()` and delete unused accounts.

## 5. Backup Security

- **Encrypt backups**: Backups inherit instance encryption settings.
- **Access control**: Restrict `DeleteManualBackup` and `CreateManualBackup` to admin roles.
- **Cross-region backup** (if supported): Enable for disaster recovery compliance.

## 6. Audit & Monitoring

- **Enable CTS (Cloud Trace Service)**: All GaussDB API calls are logged.
- **Monitor via CloudEye**:
  - `gaussdb_instance_status` → alarm on non-ACTIVE
  - `gaussdb_disk_usage` → alarm at 85%
  - `gaussdb_connections` → alarm at 90% of max_connections
- **Review task history**: `ListTasks()` reveals recent operations on the instance.

## 7. Incident Response Playbook

| Incident | Detection | Immediate Action | Recovery |
|----------|-----------|-----------------|----------|
| Unauthorized access | CTS logs show unknown IP | Rotate all DB passwords via `SetDbUserPwd()` | Review audit logs, revoke unknown grants |
| Data deletion | Backup count drops | Stop sync, isolate instance | Restore from latest backup |
| Ransomware / DB locked | Suspicious queries | Snapshot instance, revoke connections | Restore from pre-incident backup |
| Instance hijacked | IAM activity anomaly | Rotate AK/SK immediately | Review IAM policies, reset all credentials |

## 8. Compliance Checklist

- [ ] Disk encryption enabled on all production instances
- [ ] SSL enforced for all client connections
- [ ] IAM policies follow least privilege
- [ ] No GaussDB instances with public EIP permanently bound
- [ ] Automatic backups configured with 7-day minimum retention
- [ ] CTS enabled for data plane audit
- [ ] CloudEye alarms configured for disk, connections, status
- [ ] DB user accounts audited quarterly for stale accounts
