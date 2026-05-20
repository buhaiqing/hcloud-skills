# CLI Usage — Huawei Cloud IAM

> **Purpose:** Maps IAM operations to hcloud CLI commands.
> **Version:** 1.0.0
> **Last Updated:** 2026-05-20

---

## Table of Contents

1. [CLI Overview](#1-cli-overview)
2. [User Management Commands](#2-user-management-commands)
3. [Group Management Commands](#3-group-management-commands)
4. [Policy Management Commands](#4-policy-management-commands)
5. [Role Management Commands](#5-role-management-commands)
6. [Agency Management Commands](#6-agency-management-commands)
7. [Project Management Commands](#7-project-management-commands)
8. [Credential Management Commands](#8-credential-management-commands)
9. [Provider Commands](#9-provider-commands)
10. [Common Workflows](#10-common-workflows)
11. [CLI vs SDK Feature Matrix](#11-cli-vs-sdk-feature-matrix)

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
export HW_DOMAIN_ID="your-domain-id"
```

> **Note:** IAM uses `HW_DOMAIN_ID` instead of `HW_PROJECT_ID` for most operations since it is a global service.

### 1.3 CLI Syntax

```bash
hcloud iam <command> [options]
```

### 1.4 Global Options

| Option | Description |
|--------|-------------|
| `--domain-id` | Domain ID (account ID) |
| `--access-key` | Access Key ID |
| `--secret-key` | Secret Access Key |
| `--output` | Output format: json, table (default: table) |

---

## 2. User Management Commands

### 2.1 List Users

**Command:**
```bash
hcloud iam list-users [options]
```

**Options:**

| Option | Required | Description |
|--------|----------|-------------|
| `--domain-id` | Yes | Domain ID |
| `--limit` | No | Maximum records to return |
| `--offset` | No | Offset for pagination |
| `--name` | No | Filter by user name |
| `--enabled` | No | Filter by enabled status |

**Example:**
```bash
hcloud iam list-users --domain-id {{env.HW_DOMAIN_ID}} --output json
```

### 2.2 Show User

**Command:**
```bash
hcloud iam show-user --user-id <id> [options]
```

**Example:**
```bash
hcloud iam show-user --user-id 07609fb9xxxxxxxxxxxxxxxx --domain-id {{env.HW_DOMAIN_ID}}
```

### 2.3 Create User

**Command:**
```bash
hcloud iam create-user [options]
```

**Required Options:**

| Option | Description |
|--------|-------------|
| `--domain-id` | Domain ID |
| `--name` | User name |
| `--password` | User password |

**Optional Options:**

| Option | Description |
|--------|-------------|
| `--email` | Email address |
| `--phone` | Phone number |
| `--description` | User description |
| `--enabled` | Enable status (default: true) |

**Example:**
```bash
hcloud iam create-user \
  --domain-id {{env.HW_DOMAIN_ID}} \
  --name "app-service-user" \
  --password "SecurePassword123!" \
  --email "user@example.com" \
  --description "Service account for application"
```

### 2.4 Update User

**Command:**
```bash
hcloud iam update-user --user-id <id> [options]
```

**Example:**
```bash
hcloud iam update-user \
  --user-id 07609fb9xxxxxxxxxxxxxxxx \
  --email "new-email@example.com" \
  --description "Updated description"
```

### 2.5 Delete User

**Command:**
```bash
hcloud iam delete-user --user-id <id> --domain-id <domain-id>
```

**⚠️ Safety Gate:** This command requires confirmation:
```
Warning: This will permanently delete the IAM user and all associated credentials.
User: app-service-user (07609fb9xxxxxxxxxxxxxxxx)
All AK/SK will be invalidated. All group memberships will be removed.

Do you want to proceed? (yes/no): yes
```

---

## 3. Group Management Commands

### 3.1 List Groups

**Command:**
```bash
hcloud iam list-groups --domain-id <domain-id>
```

### 3.2 Create Group

**Command:**
```bash
hcloud iam create-group --domain-id <domain-id> --name <name> [options]
```

**Example:**
```bash
hcloud iam create-group \
  --domain-id {{env.HW_DOMAIN_ID}} \
  --name "developers" \
  --description "Development team group"
```

### 3.3 Add User to Group

**Command:**
```bash
hcloud iam add-user-to-group --group-id <group-id> --user-id <user-id>
```

### 3.4 Remove User from Group

**Command:**
```bash
hcloud iam remove-user-from-group --group-id <group-id> --user-id <user-id>
```

---

## 4. Policy Management Commands

### 4.1 List Policies

**Command:**
```bash
hcloud iam list-policies --domain-id <domain-id> [options]
```

**Options:**

| Option | Description |
|--------|-------------|
| `--type` | Filter: `AX` (custom), `role` (system) |
| `--name` | Filter by policy name |
| `--limit` | Maximum records |
| `--offset` | Pagination offset |

**Example:**
```bash
# List custom policies
hcloud iam list-policies --domain-id {{env.HW_DOMAIN_ID}} --type AX --output json

# List system policies
hcloud iam list-policies --domain-id {{env.HW_DOMAIN_ID}} --type role --output json
```

### 4.2 Create Custom Policy

**Command:**
```bash
hcloud iam create-policy --domain-id <domain-id> --name <name> --policy-document <json> [options]
```

**Example:**
```bash
hcloud iam create-policy \
  --domain-id {{env.HW_DOMAIN_ID}} \
  --name "ECS-ReadOnly-Custom" \
  --description "Custom read-only policy for ECS" \
  --policy-document '{
    "Version": "1.1",
    "Statement": [
      {
        "Effect": "Allow",
        "Action": ["ecs:servers:get", "ecs:servers:list"],
        "Resource": ["*"]
      }
    ]
  }'
```

### 4.3 Attach Policy to User

**Command:**
```bash
hcloud iam attach-policy-to-user --user-id <user-id> --policy-id <policy-id> --domain-id <domain-id>
```

### 4.4 Attach Policy to Group

**Command:**
```bash
hcloud iam attach-policy-to-group --group-id <group-id> --policy-id <policy-id> --domain-id <domain-id>
```

### 4.5 Detach Policy from User

**Command:**
```bash
hcloud iam detach-policy-from-user --user-id <user-id> --policy-id <policy-id> --domain-id <domain-id>
```

### 4.6 Delete Custom Policy

**Command:**
```bash
hcloud iam delete-policy --policy-id <policy-id> --domain-id <domain-id>
```

**⚠️ Safety Gate:** Policy must not be attached to any user or group.

---

## 5. Role Management Commands

### 5.1 List System Roles

**Command:**
```bash
hcloud iam list-roles --domain-id <domain-id>
```

### 5.2 Assign Role to Group on Project

**Command:**
```bash
hcloud iam assign-role --group-id <group-id> --role-id <role-id> --project-id <project-id>
```

---

## 6. Agency Management Commands

### 6.1 List Agencies

**Command:**
```bash
hcloud iam list-agencies --domain-id <domain-id>
```

### 6.2 Create Agency

**Command:**
```bash
hcloud iam create-agency --domain-id <domain-id> --name <name> --trust-domain-id <id> [options]
```

**Example:**
```bash
hcloud iam create-agency \
  --domain-id {{env.HW_DOMAIN_ID}} \
  --name "cross-account-delegation" \
  --trust-domain-id "f39a8dcexxxxxxxxxxxxxxxx" \
  --description "Delegation for cross-account access" \
  --duration "FOREVER"
```

### 6.3 Delete Agency

**Command:**
```bash
hcloud iam delete-agency --agency-id <agency-id> --domain-id <domain-id>
```

---

## 7. Project Management Commands

### 7.1 List Projects

**Command:**
```bash
hcloud iam list-projects --domain-id <domain-id>
```

### 7.2 Show Project

**Command:**
```bash
hcloud iam show-project --project-id <project-id>
```

---

## 8. Credential Management Commands

### 8.1 Create Access Key (AK/SK)

**Command:**
```bash
hcloud iam create-access-key --user-id <user-id> [options]
```

**Options:**

| Option | Required | Description |
|--------|----------|-------------|
| `--user-id` | Yes | Target user ID |
| `--description` | No | Key description |

**⚠️ CRITICAL Safety Gate:**
```
Warning: Creating an AK/SK generates permanent credentials.
- Store the secret key securely; it CANNOT be retrieved later.
- Rotate AK/SK every 90 days.
- Delete unused keys immediately.

Do you want to proceed? (yes/no): yes
```

**Example:**
```bash
hcloud iam create-access-key \
  --user-id 07609fb9xxxxxxxxxxxxxxxx \
  --description "AK/SK for deployment pipeline"
```

### 8.2 List Access Keys

**Command:**
```bash
hcloud iam list-access-keys --user-id <user-id>
```

### 8.3 Delete Access Key

**Command:**
```bash
hcloud iam delete-access-key --user-id <user-id> --access-key-id <key-id>
```

---

## 9. Provider Commands

### 9.1 List Identity Providers

**Command:**
```bash
hcloud iam list-providers --domain-id <domain-id>
```

### 9.2 Create SAML Provider

**Command:**
```bash
hcloud iam create-provider --domain-id <domain-id> --name <name> --metadata <xml>
```

---

## 10. Common Workflows

### Workflow 1: Onboard New Team Member

```bash
# Step 1: Create user
hcloud iam create-user \
  --domain-id {{env.HW_DOMAIN_ID}} \
  --name "zhang-san" \
  --password "TempPass123!" \
  --email "zhangsan@example.com"

# Step 2: Add to group
hcloud iam add-user-to-group \
  --group-id <developers-group-id> \
  --user-id <new-user-id>

# Step 3: Create AK/SK (if needed)
hcloud iam create-access-key \
  --user-id <new-user-id> \
  --description "AK/SK for zhang-san"

# Step 4: User enables MFA via console
```

### Workflow 2: Create Service Account

```bash
# Step 1: Create service user
hcloud iam create-user \
  --domain-id {{env.HW_DOMAIN_ID}} \
  --name "svc-deployment" \
  --password "AutoGenPass!2026"

# Step 2: Create custom policy
hcloud iam create-policy \
  --domain-id {{env.HW_DOMAIN_ID}} \
  --name "Deployment-Policy" \
  --policy-document '{
    "Version": "1.1",
    "Statement": [
      {
        "Effect": "Allow",
        "Action": [
          "ecs:servers:list",
          "ecs:servers:create",
          "ecs:servers:delete"
        ],
        "Resource": ["*"]
      }
    ]
  }'

# Step 3: Attach policy to user
hcloud iam attach-policy-to-user \
  --user-id <service-user-id> \
  --policy-id <policy-id> \
  --domain-id {{env.HW_DOMAIN_ID}}

# Step 4: Create AK/SK
hcloud iam create-access-key \
  --user-id <service-user-id> \
  --description "AK/SK for CI/CD pipeline"
```

### Workflow 3: Set Up Cross-Account Access

```bash
# Step 1: Create agency
hcloud iam create-agency \
  --domain-id {{env.HW_DOMAIN_ID}} \
  --name "audit-delegation" \
  --trust-domain-id "f39a8dcexxxxxxxxxxxxxxxx"

# Step 2: Attach policy to agency
hcloud iam attach-policy-to-agency \
  --agency-id <agency-id> \
  --policy-id <audit-policy-id>
```

### Workflow 4: Permission Audit

```bash
# Step 1: List all users
hcloud iam list-users --domain-id {{env.HW_DOMAIN_ID}} --output json

# Step 2: For each user, list their policies
for user_id in $(hcloud iam list-users --domain-id {{env.HW_DOMAIN_ID}} --output json | jq -r '.users[].id'); do
  echo "User: $user_id"
  hcloud iam list-user-policies --user-id "$user_id" --domain-id {{env.HW_DOMAIN_ID}} --output json
done

# Step 3: List all groups and their policies
hcloud iam list-groups --domain-id {{env.HW_DOMAIN_ID}} --output json

# Step 4: List all AK/SK (check for stale keys)
for user_id in $(hcloud iam list-users --domain-id {{env.HW_DOMAIN_ID}} --output json | jq -r '.users[].id'); do
  echo "Keys for user: $user_id"
  hcloud iam list-access-keys --user-id "$user_id" --output json
done
```

### Workflow 5: AK/SK Rotation

```bash
# Step 1: Create new AK/SK
hcloud iam create-access-key \
  --user-id <user-id> \
  --description "Rotated key - $(date +%Y%m%d)"

# Step 2: Update application configuration with new AK/SK
# (Manual step: update application config files)

# Step 3: Verify new key works
hcloud ecs list --region cn-north-4 --access-key <new-ak> --secret-key <new-sk>

# Step 4: Delete old AK/SK
hcloud iam delete-access-key \
  --user-id <user-id> \
  --access-key-id <old-access-key-id>
```

### Workflow 6: Least Privilege Policy Creation

```bash
# Create a minimal policy for specific operations
hcloud iam create-policy \
  --domain-id {{env.HW_DOMAIN_ID}} \
  --name "RDS-Operator-Minimal" \
  --policy-document '{
    "Version": "1.1",
    "Statement": [
      {
        "Effect": "Allow",
        "Action": [
          "rds:instance:get",
          "rds:instance:list",
          "rds:backup:create",
          "rds:backup:list"
        ],
        "Resource": ["acs:rds:*:*:instance/*"],
        "Condition": {
          "StringEquals": {
            "hw:project": "cn-north-4"
          }
        }
      }
    ]
  }'
```

---

## 11. CLI vs SDK Feature Matrix

### 11.1 Fully Supported Operations

| Operation | CLI Command | Status |
|-----------|-------------|--------|
| List Users | `hcloud iam list-users` | ✅ Full |
| Show User | `hcloud iam show-user` | ✅ Full |
| Create User | `hcloud iam create-user` | ✅ Full |
| Update User | `hcloud iam update-user` | ✅ Full |
| Delete User | `hcloud iam delete-user` | ✅ Full |
| List Groups | `hcloud iam list-groups` | ✅ Full |
| Create Group | `hcloud iam create-group` | ✅ Full |
| Add User to Group | `hcloud iam add-user-to-group` | ✅ Full |
| List Policies | `hcloud iam list-policies` | ✅ Full |
| Create Policy | `hcloud iam create-policy` | ✅ Full |
| Attach Policy | `hcloud iam attach-policy-to-user/group` | ✅ Full |
| List Access Keys | `hcloud iam list-access-keys` | ✅ Full |
| Create Access Key | `hcloud iam create-access-key` | ✅ Full |
| Delete Access Key | `hcloud iam delete-access-key` | ✅ Full |

### 11.2 SDK-Only Operations (CLI Gap)

| Operation | CLI | SDK | Notes |
|-----------|-----|-----|-------|
| MFA Device Management | ❌ | ✅ | Use Go SDK |
| Federation Configuration | Partial | ✅ | Limited CLI support |
| Password Policy | ❌ | ✅ | Use Go SDK |
| Token Operations | ❌ | ✅ | SDK handles internally |
| Bulk Permission Check | ❌ | ✅ | Use Go SDK |

---

*This document defines CLI usage patterns for IAM operations. Refer to official Huawei Cloud hcloud documentation for the latest command reference.*
