# Integration — Huawei Cloud IAM

> **Purpose:** SDK setup, cross-skill delegation, and environment configuration for IAM.
> **Version:** 1.0.0
> **Last Updated:** 2026-05-20

---

## Table of Contents

1. [Integration Overview](#1-integration-overview)
2. [Cross-Skill Delegation Matrix](#2-cross-skill-delegation-matrix)
3. [IAM as Security Foundation](#3-iam-as-security-foundation)
4. [JIT SDK Integration](#4-jit-sdk-integration)
5. [CLI Integration](#5-cli-integration)
6. [CTS Integration](#6-cts-integration)
7. [KMS Integration](#7-kms-integration)
8. [Federation Integration](#8-federation-integration)
9. [Enterprise Integration Patterns](#9-enterprise-integration-patterns)
10. [CI/CD Integration](#10-cicd-integration)

---

## 1. Integration Overview

IAM is the **security foundation** for all Huawei Cloud services. Every other skill depends on IAM for authentication and authorization. This document defines how IAM integrates with other skills and external systems.

| Integration Type | Primary Method | Secondary Method |
|-----------------|----------------|------------------|
| Cross-Skill Delegation | Permission check → IAM skill | Direct IAM API call |
| CLI Integration | `hcloud iam` commands | JIT Go SDK fallback |
| CTS Integration | Audit trail queries | Event-driven alerts |
| KMS Integration | Credential encryption | Key rotation |
| Federation | SAML 2.0 / OIDC | SCIM provisioning |

---

## 2. Cross-Skill Delegation Matrix

### 2.1 Other Skills Delegating to IAM

| Source Skill | Delegation Trigger | IAM Action | Notes |
|-------------|-------------------|------------|-------|
| huaweicloud-ecs-ops | 403 on ECS operation | Check user's ECS permissions | Delegate permission diagnosis |
| huaweicloud-rds-ops | 403 on RDS operation | Check user's RDS permissions | Delegate permission diagnosis |
| huaweicloud-vpc-ops | 403 on VPC operation | Check user's VPC permissions | Delegate permission diagnosis |
| huaweicloud-elb-ops | 403 on ELB operation | Check user's ELB permissions | Delegate permission diagnosis |
| huaweicloud-ces-ops | 403 on CES operation | Check user's CES permissions | Delegate permission diagnosis |
| huaweicloud-cts-ops | 403 on CTS operation | Check user's CTS permissions | Delegate permission diagnosis |
| huaweicloud-functiongraph-ops | 403 on FunctionGraph operation | Check user's FG permissions | Delegate permission diagnosis |

### 2.2 IAM Delegating to Other Skills

| Delegation Target | Trigger | Target Action | Notes |
|-------------------|---------|---------------|-------|
| huaweicloud-cts-ops | Need audit trail for IAM events | Query CTS traces | Security investigation |
| huaweicloud-ces-ops | Need to set up IAM-related alerts | Configure CES alarm rules | Monitoring integration |
| huaweicloud-kms-ops | Need key management for credential encryption | Create/manage KMS keys | Credential security |

### 2.3 Delegation Protocol

```
[Permission Denied Error in Resource Skill]
    │
    ├── 1. Resource skill catches 403 error
    ├── 2. Resource skill delegates to huaweicloud-iam-ops
    ├── 3. IAM skill checks user's policies and groups
    ├── 4. IAM skill identifies missing permission
    ├── 5. IAM skill recommends or applies fix (with approval)
    └── 6. Resource skill retries operation
```

---

## 3. IAM as Security Foundation

### 3.1 Every Skill's Dependency on IAM

Every skill in the hcloud-skills project depends on IAM for:

1. **Authentication** — AK/SK or token for API calls
2. **Authorization** — Policies granting access to resources
3. **Audit** — CTS tracing of all operations
4. **Credential Management** — AK/SK lifecycle management

### 3.2 Common Permission Patterns

| Pattern | Required IAM Permission | Typical Policy |
|---------|------------------------|----------------|
| Read-only access | `<service>:*:get`, `<service>:*:list` | `ReadOnlyAccess` |
| Operator access | Read + `<service>:*:create`, `<service>:*:delete` | Custom policy |
| Admin access | `<service>:*` | `<Service>FullAccess` |
| Cross-account access | Agency with scoped policies | Custom agency |

---

## 4. JIT SDK Integration

### 4.1 Go SDK Package

```
github.com/huaweicloud/huaweicloud-sdk-go-v3/services/iam/v3
```

### 4.2 Client Initialization

```go
package main

import (
    "fmt"
    "os"
    
    "github.com/huaweicloud/huaweicloud-sdk-go-v3/core/auth/basic"
    "github.com/huaweicloud/huaweicloud-sdk-go-v3/core/config"
    iam "github.com/huaweicloud/huaweicloud-sdk-go-v3/services/iam/v3"
)

func newIamClient() *iam.IamClient {
    ak := os.Getenv("HW_ACCESS_KEY_ID")
    sk := os.Getenv("HW_SECRET_ACCESS_KEY")
    domainId := os.Getenv("HW_DOMAIN_ID")
    
    httpConfig := config.DefaultHttpConfig().
        WithTimeout(120).
        WithMaxRetryCount(3)
    
    credential := basic.NewCredentialsBuilder().
        WithAk(ak).
        WithSk(sk).
        WithDomainId(domainId).
        Build()
    
    client := iam.IamClientBuilder().
        WithEndpoint("https://iam.myhuaweicloud.com").
        WithCredential(credential).
        WithHttpConfig(httpConfig).
        Build()
    
    return client
}
```

### 4.3 JIT Script Template

```go
// iam_jit_template.go
package main

import (
    "encoding/json"
    "fmt"
    "os"
    
    "github.com/huaweicloud/huaweicloud-sdk-go-v3/core/auth/basic"
    "github.com/huaweicloud/huaweicloud-sdk-go-v3/core/config"
    iam "github.com/huaweicloud/huaweicloud-sdk-go-v3/services/iam/v3"
    iam_model "github.com/huaweicloud/huaweicloud-sdk-go-v3/services/iam/v3/model"
)

func main() {
    client := initClient()
    cmd := os.Args[1]
    
    switch cmd {
    case "list-users":
        listUsers(client)
    case "create-user":
        createUser(client, os.Args[2])
    case "check-permissions":
        checkPermissions(client, os.Args[2])
    default:
        fmt.Printf("Unknown command: %s\n", cmd)
        os.Exit(1)
    }
}

func initClient() *iam.IamClient {
    ak := os.Getenv("HW_ACCESS_KEY_ID")
    sk := os.Getenv("HW_SECRET_ACCESS_KEY")
    domainId := os.Getenv("HW_DOMAIN_ID")
    
    return iam.IamClientBuilder().
        WithEndpoint("https://iam.myhuaweicloud.com").
        WithCredential(basic.NewCredentialsBuilder().
            WithAk(ak).WithSk(sk).WithDomainId(domainId).Build()).
        WithHttpConfig(config.DefaultHttpConfig().
            WithTimeout(120).WithMaxRetryCount(3)).
        Build()
}
```

---

## 5. CLI Integration

### 5.1 CLI Commands for Integration

```bash
# Verify IAM credentials work
hcloud iam list-users --domain-id {{env.HW_DOMAIN_ID}} --limit 1

# Check current user's permissions
hcloud iam list-user-policies --user-id <user-id> --domain-id {{env.HW_DOMAIN_ID}}
```

---

## 6. CTS Integration

### 6.1 IAM Events in CTS

IAM events are automatically recorded in CTS. Other skills can query CTS for IAM-related security events.

```bash
# Query IAM events via CTS
hcloud cts list-traces --service-type IAM --from "2026-05-20T00:00:00" --to "2026-05-20T23:59:59"
```

### 6.2 Event Correlation

| CTS Event | Cross-Skill Correlation | Action |
|-----------|------------------------|--------|
| `attachPolicy` (admin) | Check if target user has ECS/RDS access | Verify least privilege |
| `createAccessKey` | Check if key is used for resource operations | Monitor usage patterns |
| `deleteUser` | Check if user had active resources | Prevent orphaned resources |

---

## 7. KMS Integration

### 7.1 Credential Encryption

| Use Case | KMS Integration | Implementation |
|----------|----------------|----------------|
| AK/SK storage | Encrypt at rest with KMS | Application-level KMS API call |
| Password vault | KMS envelope encryption | Store encrypted password in config |
| Token encryption | KMS data key | Encrypt tokens before storage |

### 7.2 Key Rotation

```bash
# Create KMS key for IAM credential encryption
hcloud kms create-key --alias "iam-credential-key" --domain-id {{env.HW_DOMAIN_ID}}

# Rotate KMS key annually
hcloud kms rotate-key --key-id <kms-key-id>
```

---

## 8. Federation Integration

### 8.1 SAML 2.0 Federation

```bash
# Create identity provider
hcloud iam create-provider \
  --domain-id {{env.HW_DOMAIN_ID}} \
  --name "enterprise-idp" \
  --metadata @saml-metadata.xml

# Create federation group mapping
hcloud iam create-group \
  --domain-id {{env.HW_DOMAIN_ID}} \
  --name "federated-users" \
  --description "Auto-provisioned from IdP"
```

### 8.2 OIDC Federation

```bash
# Configure OIDC provider
hcloud iam create-provider \
  --domain-id {{env.HW_DOMAIN_ID}} \
  --name "oidc-provider" \
  --protocol OIDC \
  --client-id "xxxxxxxx" \
  --authorization-endpoint "https://idp.example.com/authorize"
```

---

## 9. Enterprise Integration Patterns

### 9.1 SCIM User Provisioning

| Pattern | Description | Implementation |
|---------|-------------|----------------|
| JIT Provisioning | Create user on first SSO login | IAM federation auto-creation |
| SCIM Sync | Synchronize users from IdP | Custom SCIM bridge script |
| Manual Provisioning | Admin creates users in IAM | CLI/API automation |

### 9.2 Permission as Code

```yaml
# permissions.yaml — Declarative permission management
groups:
  - name: developers
    policies:
      - ECS-ReadOnly-Custom
      - RDS-ReadOnly-Custom
      - VPC-ReadOnly-Custom
    projects:
      - cn-north-4

  - name: operators
    policies:
      - ECS-Operator-Custom
      - RDS-Operator-Custom
      - CES-Operator-Custom
    projects:
      - cn-north-4
      - cn-east-2

policies:
  - name: ECS-ReadOnly-Custom
    document:
      Version: "1.1"
      Statement:
        - Effect: Allow
          Action:
            - ecs:servers:get
            - ecs:servers:list
          Resource: ["*"]
```

---

## 10. CI/CD Integration

### 10.1 Pipeline Credential Management

```yaml
# .gitlab-ci.yml — IAM credential management in CI/CD
stages:
  - deploy

deploy:
  stage: deploy
  variables:
    # Credentials injected from CI/CD secrets
    HW_ACCESS_KEY_ID: $CI_HW_AK
    HW_SECRET_ACCESS_KEY: $CI_HW_SK
    HW_DOMAIN_ID: $CI_HW_DOMAIN_ID
  script:
    # Verify credentials
    - hcloud iam list-users --domain-id $HW_DOMAIN_ID --limit 1
    
    # Deploy using scoped credentials
    - hcloud ecs create --region cn-north-4 ...
    
    # Audit: log credential usage
    - echo "Deployment completed with AK prefix ${HW_ACCESS_KEY_ID:0:8}..."
```

### 10.2 Service Account Pattern

```bash
# Create dedicated service account for CI/CD
hcloud iam create-user \
  --domain-id {{env.HW_DOMAIN_ID}} \
  --name "svc-cicd-pipeline" \
  --description "Service account for CI/CD pipeline"

# Create minimal policy
hcloud iam create-policy \
  --domain-id {{env.HW_DOMAIN_ID}} \
  --name "CICD-Deploy-Policy" \
  --policy-document '{
    "Version": "1.1",
    "Statement": [
      {
        "Effect": "Allow",
        "Action": [
          "ecs:servers:create",
          "ecs:servers:delete",
          "ecs:servers:get",
          "rds:instance:get"
        ],
        "Resource": ["*"],
        "Condition": {
          "StringEquals": {
            "hw:project": "cn-north-4"
          }
        }
      }
    ]
  }'

# Create AK/SK for service account
hcloud iam create-access-key \
  --user-id <svc-user-id> \
  --description "AK/SK for CI/CD pipeline"
```

---

*This document defines integration patterns for IAM operations. Refer to official Huawei Cloud SDK documentation for the latest integration details.*
