# GaussDB API Navigation — v3 (openGauss engine)

Base URL: `https://gaussdb-opengauss.{region}.myhuaweicloud.com`

> API paths use pattern `/v3/{project_id}/{resource}`. The recommended (newer) path group is suffixed with " (Recommended)" below.

---

## Instance Management (Recommended APIs)

| API | Method | Path | Go SDK Method |
|-----|--------|------|---------------|
| Create Instance | POST | `/v3/{project_id}/instances` | `CreateInstance()` |
| List Instances | GET | `/v3.3/{project_id}/instances` | `ListInstances()` |
| Show Instance Detail | GET | `/v3.3/{project_id}/instances/{instance_id}` | `ShowInstanceDetail()` |
| Delete Instance | DELETE | `/v3/{project_id}/instances/{instance_id}` | `DeleteInstance()` |
| Update Instance Name | PUT | `/v3/{project_id}/instances/{instance_id}/name` | `UpdateInstanceName()` |
| Resize Instance Flavor | PUT | `/v3/{project_id}/instance/{instance_id}/flavor` | `ResizeInstanceFlavor()` |
| Restart Instance | POST | `/v3/{project_id}/instances/{instance_id}/restart` | `RestartInstance()` |
| Reset Password | POST | `/v3/{project_id}/instances/{instance_id}/password` | `ResetPwd()` |
| Add CN | POST | `/v3/{project_id}/instances/{instance_id}/cn` | `AddInstanceCN()` |
| Expand DN | POST | `/v3/{project_id}/instances/{instance_id}/dn` | `ExpandInstanceDN()` |
| Switch Shard | POST | `/v3/{project_id}/instances/{instance_id}/switch-shard` | `SwitchShard()` |
| Bind EIP | POST | `/v3/{project_id}/instances/{instance_id}/public-ip/bind` | `BindEIP()` |
| Unbind EIP | POST | `/v3/{project_id}/instances/{instance_id}/public-ip/unbind` | `UnbindEIP()` |
| Query Quotas | GET | `/v3/{project_id}/quotas` | `ShowQuotas()` |
| List Flavor Info | GET | `/v3/{project_id}/flavors` | `ListFlavors()` |
| Show SSL Download Link | GET | `/v3/{project_id}/instances/{instance_id}/ssl-cert/download-link` | `ShowSslCertDownloadLink()` |

## Backup & Restoration (Recommended APIs)

| API | Method | Path | Go SDK Method |
|-----|--------|------|---------------|
| Set Backup Policy | PUT | `/v3/{project_id}/instances/{instance_id}/backups/policy` | `SetBackupPolicy()` |
| Show Backup Policy | GET | `/v3/{project_id}/instances/{instance_id}/backups/policy` | `ShowBackupPolicy()` |
| List Backups | GET | `/v3/{project_id}/backups` | `ListBackups()` |
| Create Manual Backup | POST | `/v3/{project_id}/backups` | `CreateManualBackup()` |
| Delete Manual Backup | DELETE | `/v3/{project_id}/backups/{backup_id}` | `DeleteManualBackup()` |
| Restore to New Instance | POST | `/v3/{project_id}/instances` | `RestoreInstance()` |
| Show Restoration Time | GET | `/v3/{project_id}/instances/{instance_id}/restore-time` | `ListRestoreTimes()` |

## Parameter Template Management

| API | Method | Path | Go SDK Method |
|-----|--------|------|---------------|
| List Templates | GET | `/v3/{project_id}/configurations` | `ListConfigurations()` |
| Show Template Detail | GET | `/v3/{project_id}/configurations/{config_id}` | `ShowConfigurationSetting()` |
| Create Template | POST | `/v3/{project_id}/configurations` | `CreateConfigurationTemplate()` |
| Update Template | PUT | `/v3/{project_id}/configurations/{config_id}` | `UpdateConfigurationSetting()` |
| Delete Template | DELETE | `/v3/{project_id}/configurations/{config_id}` | `DeleteConfiguration()` |
| Apply Template | PUT | `/v3/{project_id}/configurations/{config_id}/apply` | `ApplyConfiguration()` |
| Compare Templates | POST | `/v3/{project_id}/configurations/{config_id}/differences` | `ListDiffDetails()` |
| List Applied Instances | GET | `/v3/{project_id}/configurations/{config_id}/applicable-instances` | `ListApplicableInstances()` |
| Show Apply History | GET | `/v3/{project_id}/configurations/{config_id}/apply-history` | `ListAppliedHistories()` |

## Database & Account Management (Recommended APIs)

| API | Method | Path | Go SDK Method |
|-----|--------|------|---------------|
| Create Database | POST | `/v3/{project_id}/instances/{instance_id}/database` | `CreateDatabase()` |
| List Databases | GET | `/v3/{project_id}/instances/{instance_id}/databases` | `ListDatabases()` |
| Delete Database | DELETE | `/v3/{project_id}/instances/{instance_id}/database` | `DeleteDatabase()` |
| Create Db User | POST | `/v3/{project_id}/instances/{instance_id}/db-user` | `CreateDbUser()` |
| List Db Users | GET | `/v3/{project_id}/instances/{instance_id}/db-users` | `ListDbUsers()` |
| Reset Db User Pwd | PUT | `/v3/{project_id}/instances/{instance_id}/db-user/password` | `SetDbUserPwd()` |
| Create Schema | POST | `/v3/{project_id}/instances/{instance_id}/schema` | `CreateSchema()` |
| List Schemas | GET | `/v3/{project_id}/instances/{instance_id}/schemas` | `ListSchemas()` |
| Create Database Role | POST | `/v3/{project_id}/instances/{instance_id}/db-role` | `CreateDbRole()` |
| List Database Roles | GET | `/v3/{project_id}/instances/{instance_id}/db-roles` | `ListDbRoles()` |
| Grant Database Privilege | PUT | `/v3/{project_id}/instances/{instance_id}/db-privilege` | `SetDbUserPrivilege()` |

## Tag Management

| API | Method | Path |
|-----|--------|------|
| List Instance Tags | GET | `/v3/{project_id}/gaussdb/resource_instances/action` |
| Add/Delete Tags | POST | `/v3/{project_id}/instances/{instance_id}/tags/action` |
| List Project Tags | GET | `/v3/{project_id}/gaussdb/tags` |

## Task Management

| API | Method | Path | Go SDK Method |
|-----|--------|------|---------------|
| List Tasks | GET | `/v3/{project_id}/tasks` | `ListTasks()` |
| Delete Task | DELETE | `/v3/{project_id}/tasks/{task_id}` | `DeleteTask()` |

## Enterprise Project Quotas

| API | Method | Path | Go SDK Method |
|-----|--------|------|---------------|
| Show Eps Quotas | GET | `/v3/{project_id}/enterprise-projects/quotas` | `ListEpsQuotas()` |
| Modify Eps Quota | PUT | `/v3/{project_id}/enterprise-projects/quotas` | `ModifyEpsQuota()` |

## Recycle Bin

| API | Method | Path | Go SDK Method |
|-----|--------|------|---------------|
| Set Recycle Policy | PUT | `/v3/{project_id}/instances/recycle-policy` | `CreateRecyclePolicy()` |
| Show Recycle Policy | GET | `/v3/{project_id}/instances/recycle-policy` | `ShowRecyclePolicy()` |
| List Recycled Instances | GET | `/v3/{project_id}/recycle-instances` | `ListRecycleInstances()` |
