# CLI Usage — Huawei Cloud RDS

> **Purpose:** Maps RDS operations to hcloud CLI commands.
> **Version:** 1.0.0
> **Last Updated:** 2026-05-20

---

## Table of Contents

1. [CLI Overview](#1-cli-overview)
2. [Instance Management Commands](#2-instance-management-commands)
3. [Backup Management Commands](#3-backup-management-commands)
4. [Database & User Commands](#4-database--user-commands)
5. [Parameter Management Commands](#5-parameter-management-commands)
6. [CLI Coverage Gap Analysis](#6-cli-coverage-gap-analysis)

---

## 1. CLI Overview

### 1.1 CLI Installation

```bash
# Linux/macOS one-click install
curl -sSL https://cn-north-4.myhuaweicloud.com/cli/latest/hcloud_install.sh -o ./hcloud_install.sh && bash ./hcloud_install.sh -y

# Verify installation
hcloud version
# Current KooCLI version: 4.1.6
```

### 1.2 Authentication

```bash
export HW_ACCESS_KEY_ID="your-access-key-id"
export HW_SECRET_ACCESS_KEY="your-secret-access-key"
export HW_REGION_ID="cn-north-4"
export HW_PROJECT_ID="your-project-id"
```

### 1.3 CLI Syntax

```bash
hcloud rds <command> [options]
```

### 1.4 Global Options

| Option | Description |
|--------|-------------|
| `--region` | Huawei Cloud region ID |
| `--project-id` | Project ID |
| `--access-key` | Access Key ID |
| `--secret-key` | Secret Access Key |
| `--output` | Output format: json, table (default: table) |

---

## 2. Instance Management Commands

### 2.1 List Instances

**Command:**
```bash
hcloud rds list [options]
```

**Options:**

| Option | Required | Description |
|--------|----------|-------------|
| `--region` | Yes | Region ID |
| `--limit` | No | Maximum records to return (1-100) |
| `--offset` | No | Offset for pagination |
| `--name` | No | Filter by instance name (fuzzy) |
| `--status` | No | Filter by status |

**Example:**
```bash
hcloud rds list --region cn-north-4 --limit 20 --output json
```

**Sample Output:**
```json
{
  "count": 2,
  "instances": [
    {
      "id": "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx",
      "name": "rds-mysql-prod-01",
      "status": "ACTIVE",
      "datastore_type": "MySQL",
      "datastore_version": "8.0",
      "flavor_ref": "rds.mysql.s1.large",
      "volume_size": 100,
      "vpc_id": "vpc-xxx",
      "private_ips": ["192.168.0.100"],
      "port": "3306"
    }
  ]
}
```

### 2.2 Show Instance

**Command:**
```bash
hcloud rds show --instance-id <id> [options]
```

**Options:**

| Option | Required | Description |
|--------|----------|-------------|
| `--instance-id` | Yes | Instance ID |
| `--region` | Yes | Region ID |

**Example:**
```bash
hcloud rds show --instance-id xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx --region cn-north-4
```

### 2.3 Create Instance

**Command:**
```bash
hcloud rds create [options]
```

**Required Options:**

| Option | Description |
|--------|-------------|
| `--name` | Instance name |
| `--engine` | Database engine (MySQL, PostgreSQL, SQLServer) |
| `--engine-version` | Engine version |
| `--flavor-ref` | Flavor specification code |
| `--vpc-id` | VPC ID |
| `--subnet-id` | Subnet ID |
| `--security-group-id` | Security group ID |
| `--availability-zone` | Availability zone |
| `--volume-size` | Storage size in GB |

**Optional Options:**

| Option | Description |
|--------|-------------|
| `--volume-type` | Storage type (ULTRAHIGH, ULTRAHIGHPRO) |
| `--password` | Database root password |
| `--port` | Database port |
| `--backup-start-time` | Backup window (HH:MM-HH:MM) |
| `--backup-keep-days` | Backup retention days (1-35) |
| `--ha-mode` | HA mode (Ha for primary/standby) |
| `--region` | Region ID |

**Example:**
```bash
hcloud rds create \
  --region cn-north-4 \
  --name rds-mysql-prod-01 \
  --engine MySQL \
  --engine-version 8.0 \
  --flavor-ref rds.mysql.s1.large \
  --vpc-id vpc-xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx \
  --subnet-id subnet-xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx \
  --security-group-id sg-xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx \
  --availability-zone cn-north-4a \
  --volume-size 100 \
  --volume-type ULTRAHIGH \
  --password "SecurePassword123!" \
  --backup-start-time "02:00-03:00" \
  --backup-keep-days 7 \
  --ha-mode Ha
```

### 2.4 Delete Instance

**Command:**
```bash
hcloud rds delete --instance-id <id> [options]
```

**Options:**

| Option | Required | Description |
|--------|----------|-------------|
| `--instance-id` | Yes | Instance ID |
| `--keep-backup` | No | Retain automated backups |
| `--region` | Yes | Region ID |

**Example:**
```bash
hcloud rds delete \
  --instance-id xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx \
  --region cn-north-4 \
  --keep-backup
```

**⚠️ Safety Gate:** This command requires confirmation:
```
Warning: This will permanently delete the RDS instance and all associated data.
Instance: rds-mysql-prod-01 (xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx)
This action is irreversible.

Do you want to proceed? (yes/no): yes
```

### 2.5 Modify Instance

**Command:**
```bash
hcloud rds modify --instance-id <id> [options]
```

**Options:**

| Option | Description |
|--------|-------------|
| `--name` | New instance name |
| `--port` | New database port |

**Example:**
```bash
hcloud rds modify \
  --instance-id xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx \
  --name rds-mysql-prod-01-renamed \
  --region cn-north-4
```

### 2.6 Resize Instance

**Command:**
```bash
hcloud rds resize --instance-id <id> --flavor-ref <flavor> [options]
```

**Options:**

| Option | Required | Description |
|--------|----------|-------------|
| `--instance-id` | Yes | Instance ID |
| `--flavor-ref` | Yes | New flavor specification |
| `--auto-pay` | No | Auto-pay for billing (default: true) |
| `--region` | Yes | Region ID |

**Example:**
```bash
hcloud rds resize \
  --instance-id xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx \
  --flavor-ref rds.mysql.s1.xlarge \
  --auto-pay \
  --region cn-north-4
```

### 2.7 Expand Volume

**Command:**
```bash
hcloud rds expand-volume --instance-id <id> --size <size> [options]
```

**Options:**

| Option | Required | Description |
|--------|----------|-------------|
| `--instance-id` | Yes | Instance ID |
| `--size` | Yes | New storage size in GB |
| `--auto-pay` | No | Auto-pay for billing |
| `--region` | Yes | Region ID |

**Example:**
```bash
hcloud rds expand-volume \
  --instance-id xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx \
  --size 200 \
  --auto-pay \
  --region cn-north-4
```

### 2.8 Restart Instance

**Command:**
```bash
hcloud rds restart --instance-id <id> [options]
```

**Options:**

| Option | Required | Description |
|--------|----------|-------------|
| `--instance-id` | Yes | Instance ID |
| `--region` | Yes | Region ID |

**Example:**
```bash
hcloud rds restart \
  --instance-id xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx \
  --region cn-north-4
```

### 2.9 Start/Stop Instance

**Commands:**
```bash
# Stop instance
hcloud rds stop --instance-id <id> --region <region>

# Start instance
hcloud rds start --instance-id <id> --region <region>
```

---

## 3. Backup Management Commands

### 3.1 Create Manual Backup

**Command:**
```bash
hcloud rds create-manual-backup --instance-id <id> --name <name> [options]
```

**Options:**

| Option | Required | Description |
|--------|----------|-------------|
| `--instance-id` | Yes | Instance ID |
| `--name` | Yes | Backup name |
| `--description` | No | Backup description |
| `--region` | Yes | Region ID |

**Example:**
```bash
hcloud rds create-manual-backup \
  --instance-id xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx \
  --name "manual-backup-$(date +%Y%m%d)" \
  --description "Manual backup before upgrade" \
  --region cn-north-4
```

### 3.2 List Backups

**Command:**
```bash
hcloud rds list-backup [options]
```

**Options:**

| Option | Description |
|--------|-------------|
| `--instance-id` | Filter by instance ID |
| `--backup-type` | Filter by type (auto, manual) |
| `--limit` | Maximum records |
| `--region` | Region ID |

**Example:**
```bash
hcloud rds list-backup \
  --instance-id xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx \
  --region cn-north-4 \
  --output json
```

### 3.3 Delete Backup

**Command:**
```bash
hcloud rds delete-backup --backup-id <id> [options]
```

**Options:**

| Option | Required | Description |
|--------|----------|-------------|
| `--backup-id` | Yes | Backup ID |
| `--region` | Yes | Region ID |

**Example:**
```bash
hcloud rds delete-backup \
  --backup-id backup-xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx \
  --region cn-north-4
```

### 3.4 Restore from Backup

**Command:**
```bash
hcloud rds restore --instance-id <id> --backup-id <backup-id> [options]
```

**Options:**

| Option | Required | Description |
|--------|----------|-------------|
| `--instance-id` | Yes | Target instance ID |
| `--backup-id` | Yes | Backup ID to restore from |
| `--auto-pay` | No | Auto-pay |
| `--region` | Yes | Region ID |

**⚠️ Safety Gate:** This command overwrites current data:
```
Warning: This will overwrite the current data on the instance.
Target: rds-mysql-prod-01
Source: manual-backup-20260520
This action is irreversible.

Do you want to proceed? (yes/no): yes
```

**Example:**
```bash
hcloud rds restore \
  --instance-id xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx \
  --backup-id backup-xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx \
  --auto-pay \
  --region cn-north-4
```

### 3.5 Restore to New Instance

**Command:**
```bash
hcloud rds restore-to-new --backup-id <id> --name <name> [options]
```

**Options:**

| Option | Required | Description |
|--------|----------|-------------|
| `--backup-id` | Yes | Backup ID |
| `--name` | Yes | New instance name |
| `--region` | Yes | Region ID |

**Example:**
```bash
hcloud rds restore-to-new \
  --backup-id backup-xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx \
  --name rds-mysql-restored \
  --region cn-north-4
```

---

## 4. Database & User Commands

### 4.1 Create Database

**Command:**
```bash
hcloud rds create-database --instance-id <id> --name <name> [options]
```

**Options:**

| Option | Required | Description |
|--------|----------|-------------|
| `--instance-id` | Yes | Instance ID |
| `--name` | Yes | Database name |
| `--character-set` | No | Character set (default: utf8mb4) |
| `--region` | Yes | Region ID |

**Example:**
```bash
hcloud rds create-database \
  --instance-id xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx \
  --name myapp_database \
  --character-set utf8mb4 \
  --region cn-north-4
```

### 4.2 List Databases

**Command:**
```bash
hcloud rds list-database --instance-id <id> [options]
```

**Options:**

| Option | Required | Description |
|--------|----------|-------------|
| `--instance-id` | Yes | Instance ID |
| `--region` | Yes | Region ID |

**Example:**
```bash
hcloud rds list-database \
  --instance-id xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx \
  --region cn-north-4
```

### 4.3 Delete Database

**Command:**
```bash
hcloud rds delete-database --instance-id <id> --database-name <name> [options]
```

**Options:**

| Option | Required | Description |
|--------|----------|-------------|
| `--instance-id` | Yes | Instance ID |
| `--database-name` | Yes | Database name |
| `--region` | Yes | Region ID |

**⚠️ Safety Gate:** Deletes all data in the database:
```
Warning: This will delete the database and all its data.
Database: myapp_database on instance rds-mysql-prod-01
This action is irreversible.

Do you want to proceed? (yes/no): yes
```

**Example:**
```bash
hcloud rds delete-database \
  --instance-id xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx \
  --database-name myapp_database \
  --region cn-north-4
```

### 4.4 Create User

**Command:**
```bash
hcloud rds create-user --instance-id <id> --name <name> --password <password> [options]
```

**Options:**

| Option | Required | Description |
|--------|----------|-------------|
| `--instance-id` | Yes | Instance ID |
| `--name` | Yes | Username |
| `--password` | Yes | User password |
| `--database` | No | Database permissions (format: dbname:readonly) |
| `--region` | Yes | Region ID |

**Example:**
```bash
hcloud rds create-user \
  --instance-id xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx \
  --name app_user \
  --password "UserPassword123!" \
  --database "myapp_database:rw,myapp_database_read:ro" \
  --region cn-north-4
```

### 4.5 List Users

**Command:**
```bash
hcloud rds list-user --instance-id <id> [options]
```

**Options:**

| Option | Required | Description |
|--------|----------|-------------|
| `--instance-id` | Yes | Instance ID |
| `--region` | Yes | Region ID |

**Example:**
```bash
hcloud rds list-user \
  --instance-id xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx \
  --region cn-north-4
```

### 4.6 Delete User

**Command:**
```bash
hcloud rds delete-user --instance-id <id> --user-name <name> [options]
```

**Options:**

| Option | Required | Description |
|--------|----------|-------------|
| `--instance-id` | Yes | Instance ID |
| `--user-name` | Yes | Username |
| `--region` | Yes | Region ID |

**Example:**
```bash
hcloud rds delete-user \
  --instance-id xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx \
  --user-name app_user \
  --region cn-north-4
```

---

## 5. Parameter Management Commands

### 5.1 List Parameters

**Command:**
```bash
hcloud rds list-parameter --instance-id <id> [options]
```

**Options:**

| Option | Description |
|--------|-------------|
| `--name` | Filter by parameter name |
| `--region` | Region ID |

**Example:**
```bash
hcloud rds list-parameter \
  --instance-id xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx \
  --name "max_connections" \
  --region cn-north-4
```

### 5.2 Modify Parameter

**Command:**
```bash
hcloud rds modify-parameter --instance-id <id> --name <name> --value <value> [options]
```

**Options:**

| Option | Required | Description |
|--------|----------|-------------|
| `--instance-id` | Yes | Instance ID |
| `--name` | Yes | Parameter name |
| `--value` | Yes | New value |
| `--region` | Yes | Region ID |

**Example:**
```bash
hcloud rds modify-parameter \
  --instance-id xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx \
  --name "max_connections" \
  --value "500" \
  --region cn-north-4
```

### 5.3 Reset Parameter to Default

**Command:**
```bash
hcloud rds reset-parameter --instance-id <id> --name <name> [options]
```

**Options:**

| Option | Required | Description |
|--------|----------|-------------|
| `--instance-id` | Yes | Instance ID |
| `--name` | Yes | Parameter name |
| `--region` | Yes | Region ID |

**Example:**
```bash
hcloud rds reset-parameter \
  --instance-id xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx \
  --name "max_connections" \
  --region cn-north-4
```

---

## 6. CLI Coverage Gap Analysis

### 6.1 Fully Supported Operations

| Operation | CLI Command | Status |
|-----------|-------------|--------|
| List Instances | `hcloud rds list` | ✅ Full |
| Show Instance | `hcloud rds show` | ✅ Full |
| Create Instance | `hcloud rds create` | ✅ Full |
| Delete Instance | `hcloud rds delete` | ✅ Full |
| Modify Instance | `hcloud rds modify` | ✅ Full |
| Resize Instance | `hcloud rds resize` | ✅ Full |
| Expand Volume | `hcloud rds expand-volume` | ✅ Full |
| Restart Instance | `hcloud rds restart` | ✅ Full |
| Start/Stop Instance | `hcloud rds start/stop` | ✅ Full |
| Create Backup | `hcloud rds create-manual-backup` | ✅ Full |
| List Backups | `hcloud rds list-backup` | ✅ Full |
| Delete Backup | `hcloud rds delete-backup` | ✅ Full |
| Restore from Backup | `hcloud rds restore` | ✅ Full |
| Restore to New | `hcloud rds restore-to-new` | ✅ Full |
| List Parameters | `hcloud rds list-parameter` | ✅ Full |
| Modify Parameter | `hcloud rds modify-parameter` | ✅ Full |
| Reset Parameter | `hcloud rds reset-parameter` | ✅ Full |

### 6.2 SDK-Only Operations (CLI Gap)

| Operation | CLI | SDK | Notes |
|-----------|-----|-----|-------|
| Create Database | ❌ | ✅ | Use Go SDK |
| List Databases | ❌ | ✅ | Use Go SDK |
| Delete Database | ❌ | ✅ | Use Go SDK |
| Create User | ❌ | ✅ | Use Go SDK |
| List Users | ❌ | ✅ | Use Go SDK |
| Delete User | ❌ | ✅ | Use Go SDK |
| Batch Operations | ❌ | ✅ | Use Go SDK |
| Custom Parameter Group | Partial | ✅ | Limited CLI support |
| Cross-Region Replication | ❌ | ✅ | Use Go SDK |

### 6.3 Fallback Strategy

For CLI-gap operations, use JIT Go SDK:

```go
// Example: Create database via SDK
func createDatabase(client *rds.RdsClient, instanceId string, dbName string) error {
    request := &rds_model.CreateDatabaseRequest{
        InstanceId: instanceId,
        Body: &rds_model.CreateDatabaseRequestBody{
            Name:         dbName,
            CharacterSet: "utf8mb4",
        },
    }
    
    response, err := client.CreateDatabase(request)
    if err != nil {
        return err
    }
    fmt.Printf("Database created: %s\n", response.Database.Name)
    return nil
}
```

---

*This document defines CLI usage patterns for RDS operations. Refer to official Huawei Cloud hcloud documentation for the latest command reference.*
