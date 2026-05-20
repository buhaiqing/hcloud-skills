---
name: huaweicloud-obs-ops
description: >-
  Use when the user needs to deploy, configure, troubleshoot, or monitor Huawei Cloud OBS (Object Storage Service) — bucket lifecycle, object operations, ACL permissions, CDN integration, lifecycle rules, versioning, encryption, lifecycle management, and data migration. User mentions OBS, Object Storage, 对象存储, bucket, S3-compatible storage, or describes object storage scenarios (e.g., upload/download files, CDN origin, static website hosting, lifecycle policies, cross-region replication, object versioning, bucket policy, storage class management) even without naming the product directly. Not for billing, IAM-only tasks, or ECS filesystem operations.
license: MIT
compatibility: >-
  Official Huawei Cloud CLI (`hcloud`), obsutil binary (alternative CLI),
  Go 1.21+ runtime (for JIT SDK fallback via huaweicloud-sdk-go-obs),
  valid API credentials, network access to Huawei Cloud OBS endpoints.
metadata:
  author: huaweicloud
  version: "1.0.0"
  last_updated: "2026-05-20"
  runtime: Harness AI Agent, Claude Code, Cursor, or compatible Agent runtimes
  go_version_minimum: "1.21"
  go_version_jit: "1.24+"
  api_profile: "OBS v3 — https://support.huaweicloud.com/api-obs/"
  cli_applicability: "dual-path"
  cli_support_evidence: >-
    OBS operations available via `hcloud obs` commands and obsutil binary:
    bucket CRUD, object upload/download/copy/delete, ACL management, lifecycle rules,
    versioning, CORS, bucket policy, static website hosting.
  environment:
    - HW_ACCESS_KEY_ID
    - HW_SECRET_ACCESS_KEY
    - HW_ENDPOINT
---

> This skill follows the [Agent Skill Open Specification](https://agentskills.io/specification).

# Huawei Cloud OBS (Object Storage Service) Operations Skill

## Overview

Huawei Cloud OBS is an S3-compatible object storage service providing scalable, durable, and secure data storage. This skill is an **operational runbook** for agents: explicit scope, credential rules, pre-flight checks, **dual-path execution** (official CLI and obsutil binary, with JIT Go SDK fallback), response validation, and failure recovery.

### CLI applicability (repository policy)

- **`cli_applicability: dual-path`:** Official CLI (`hcloud obs`) and obsutil binary both support OBS. In each execution flow, document CLI/obsutil steps **and** JIT Go SDK fallback.

## Five Core Standards (Quality Gates)

| # | Standard | How This Skill Fulfills It |
|---|----------|---------------------------|
| 1 | **Clear Boundaries** | SHOULD/SHOULD NOT Use conditions with precise triggers and delegation rules |
| 2 | **Structured I/O** | `{{env.*}}` / `{{user.*}}` / `{{output.*}}` placeholders with typed sources |
| 3 | **Explicit Actionable Steps** | Every operation: Pre-flight → Execute → Validate → Recover |
| 4 | **Complete Failure Strategies** | 13 OBS error codes with HALT vs retry per error type |
| 5 | **Absolute Single Responsibility** | One product (OBS), one resource model (Bucket/Object); cross-product delegation documented |

### Three-Pillar Ops Integration (FinOps + SecOps + AIOps)

| Pillar | Skill Integration | Reference |
|--------|-------------------|-----------|
| **FinOps** | Storage class tiering optimization, idle bucket detection, egress cost tracking | `references/well-architected-assessment.md` §3 |
| **SecOps** | Bucket policy vs ACL, public bucket risk, SSE-KMS encryption, access logging | `references/well-architected-assessment.md` §4 |
| **AIOps** | Access pattern anomaly detection, unusual egress spikes, knowledge base | `references/knowledge-base.md` |

### Well-Architected Framework Integration (卓越架构)

| Pillar | Skill Integration |
|--------|-------------------|
| **安全 (Security)** | Bucket policy, ACL review, IAM fine-grained permissions, VPC Endpoint, SSE-KMS/TLS |
| **稳定 (Stability)** | Cross-region replication, versioning, delete markers, RTO/RPO, DR runbook |
| **成本 (Cost)** | Storage class optimization (Standard→IA→Archive), lifecycle cost savings, egress pricing |
| **效率 (Efficiency)** | Multipart upload, parallel transfer, CDN acceleration, transfer acceleration |
| **性能 (Performance)** | First-byte latency, concurrent connections, CDN caching, key naming patterns |

## Trigger & Scope (Agent-Readable)

### SHOULD Use This Skill When

- User mentions "Huawei Cloud OBS" OR "Object Storage" OR "对象存储" OR "bucket" OR "S3 storage"
- Task involves bucket/object lifecycle operations (create, upload, download, delete, copy, ACL)
- Task keywords: **bucket**, **object**, **upload**, **download**, **ACL**, **lifecycle**, **versioning**, **CORS**, **CDN origin**, **presigned URL**, **static website**, **storage class**, **multipart**, **cross-region replication**
- User asks to deploy, configure, troubleshoot, or monitor OBS via API, SDK, CLI, obsutil, or automation

### SHOULD NOT Use This Skill When

- Task is purely billing/account management → delegate to: billing skill (when present)
- Task is IAM/permission model only → delegate to: `huaweicloud-iam-ops`
- Task is about ECS filesystem/local disk → delegate to: `huaweicloud-ecs-ops`
- Task is about database backup specific to RDS → delegate to: `huaweicloud-rds-ops`

### Delegation Rules

- OBS + CDN: create OBS bucket first, then configure CDN origin (delegate to CDN skill if available)
- OBS access control: bucket policy/ACL handled here; IAM user/role creation delegated to IAM skill
- OBS monitoring metrics: delegate to `huaweicloud-ces-ops` for alarm rules
- OBS access logging → LTS: delegate to `huaweicloud-lts-ops` for log setup
- VPC Endpoint for OBS: delegate to `huaweicloud-vpc-ops` for endpoint creation

## Variable Convention (Agent-Readable)

| Placeholder | Meaning | Agent Action |
|-------------|---------|--------------|
| `{{env.HW_ACCESS_KEY_ID}}` | From runtime environment | NEVER ask the user; fail if unset |
| `{{env.HW_SECRET_ACCESS_KEY}}` | From runtime environment | NEVER ask the user; fail if unset |
| `{{env.HW_ENDPOINT}}` | OBS endpoint (e.g., obs.cn-north-4.myhuaweicloud.com) | Use from env; derive from region if unset |
| `{{user.bucket_name}}` | User-supplied bucket name | Ask once; validate naming rules |
| `{{user.object_key}}` | User-supplied object key/path | Ask once; validate format |
| `{{user.storage_class}}` | Storage class: standard/warm/cold/deep-cold | Default: standard |
| `{{output.etag}}` | From API/CLI response: object ETag | Parse from upload response |
| `{{output.versionId}}` | From API/CLI response: object version ID | Parse if versioning enabled |

> **`{{env.*}}` MUST NOT** be collected from the user. **`{{user.*}}`** MUST be collected interactively when missing.

> **Security Warning (Credential Masking — MANDATORY):** **NEVER** log, print, or expose `HW_SECRET_ACCESS_KEY` or any credential field value.

## API and Response Conventions

- **OBS uses REST API** — HTTP method + path determines operation
- **Errors**: Return HTTP status + XML body with `<Error><Code>...</Code><Message>...</Message><RequestId>...</RequestId></Error>`
- **Timestamps**: HTTP Date header format (RFC 2616)
- **Idempotency**: PUT on existing object overwrites; DELETE on non-existent returns 404 (acceptable)

## Quick Start

### What This Skill Does
Create, manage, and monitor Huawei Cloud OBS buckets and objects using `hcloud` CLI, obsutil, or JIT Go SDK.

### Prerequisites
- [ ] CLI installed (`hcloud`) or obsutil binary downloaded
- [ ] Credentials configured: `HW_ACCESS_KEY_ID`, `HW_SECRET_ACCESS_KEY`
- [ ] Endpoint set: `HW_ENDPOINT`

### Verify Setup
```bash
# List all buckets
hcloud obs list-buckets
```

### Your First Command
```bash
# Create a bucket
hcloud obs create-bucket --bucket "{{user.bucket_name}}" --acl private --storage-class standard
```

### Next Steps
- [Core Concepts](references/core-concepts.md) — OBS architecture, storage classes, versioning
- [Execution Flows](#execution-flows) — Create buckets, upload objects, manage lifecycle
- [Troubleshooting](references/troubleshooting.md) — Fix access denied, upload failures

## Capabilities at a Glance

| Operation | Description | Complexity | Risk Level |
|-----------|-------------|------------|------------|
| Create Bucket | Create an OBS bucket | Low | Low |
| List Buckets | Enumerate all buckets | Low | None |
| Upload Object | Upload file to bucket | Medium | Low (overwrites if exists) |
| Download Object | Download object to local | Low | None |
| Delete Object | Remove object from bucket | Low | **Medium** — irreversible |
| Delete Bucket | Remove empty bucket | Low | **High** — requires empty bucket |
| Set ACL | Configure bucket/object ACL | Low | **High** — public ACL risk |
| Lifecycle Rules | Set lifecycle transition rules | Medium | Low |
| Enable Versioning | Enable/hide versioning on bucket | Low | Low |
| Static Website | Configure static website hosting | Low | **Medium** — public exposure risk |

## Execution Flows

### Operation: Create Bucket

#### Pre-flight Checks

| Check | Method | Expected | On Failure |
|-------|--------|----------|------------|
| CLI / obsutil | `obsutil version` or `hcloud obs help` | Available | Install obsutil or CLI |
| Credentials | Construct credential from env | Non-empty AK/SK | HALT; configure env |
| Bucket name | Validate per naming rules | 3-63 chars, lowercase letters/digits/hyphens | Fix name |
| Region | Determine from endpoint or ask user | Valid region code | Suggest valid region |

#### Execution — CLI

```bash
hcloud obs create-bucket \
  --bucket "{{user.bucket_name}}" \
  --acl private \
  --storage-class standard \
  --region "{{user.region}}"
```

#### Execution — obsutil (Alternative)

```bash
obsutil mb obs://{{user.bucket_name}} -ak="{{env.HW_ACCESS_KEY_ID}}" -sk="{{env.HW_SECRET_ACCESS_KEY}}" -endpoint="{{env.HW_ENDPOINT}}"
```

#### JIT Go SDK

```go
package main

import (
    "fmt"
    "os"
    "github.com/huaweicloud/huaweicloud-sdk-go-obs/obs"
)

func main() {
    client, err := obs.New(
        os.Getenv("HW_ACCESS_KEY_ID"),
        os.Getenv("HW_SECRET_ACCESS_KEY"),
        os.Getenv("HW_ENDPOINT"))
    if err != nil {
        panic(err)
    }

    input := &obs.CreateBucketInput{
        Bucket:       "{{user.bucket_name}}",
        ACL:          obs.AclPrivate,
        StorageClass: obs.StorageClassStandard,
    }
    _, err = client.CreateBucket(input)
    if err != nil {
        panic(err)
    }
    fmt.Printf("Bucket %s created\n", "{{user.bucket_name}}")
}
```

#### Validation

- Call `HeadBucket` or `ListObjects` — returns 200 = success
- Verify bucket location matches target region via `GetBucketLocation`

#### Failure Recovery

| Error | Max retries | Backoff | Agent Action | UX Feedback |
|-------|-------------|---------|--------------|-------------|
| `BucketAlreadyExists` | 0 | — | Use different name | `[ERROR] Bucket name already taken globally. Choose unique name.` |
| `InvalidBucketName` | 0 | — | Fix name | `[ERROR] Bucket name must be 3-63 chars, lowercase letters/digits/hyphens only.` |
| `InvalidAccessKeyId` | 0 | — | HALT | `[ERROR] Invalid AK. Check HW_ACCESS_KEY_ID.` |
| `SignatureDoesNotMatch` | 0 | — | HALT | `[ERROR] SK mismatch. Check HW_SECRET_ACCESS_KEY.` |
| `AccessDenied` | 0 | — | HALT | `[ERROR] Permission denied. Check IAM bucket creation permission.` |
| `QuotaExceeded` | 0 | — | HALT | `[ERROR] Bucket quota exceeded. Delete unused buckets.` |
| Throttling / 429 | 3 | exponential | Back off | `⚠️ Rate limited. Retrying in {backoff}s...` |
| InternalError / 5xx | 3 | 2s, 4s, 8s | Retry; then HALT | `[ERROR] OBS server error. Retry or escalate with RequestId.` |

---

### Operation: Upload Object

#### Pre-flight

- Bucket exists and is accessible
- Local file exists and size < 5TB
- For files > 100MB: use multipart upload

#### Execution — CLI

```bash
hcloud obs cp "{{user.local_file}}" obs://{{user.bucket_name}}/{{user.object_key}} --acl private
```

#### Execution — obsutil

```bash
obsutil cp "{{user.local_file}}" obs://{{user.bucket_name}}/{{user.object_key}} \
  -ak="{{env.HW_ACCESS_KEY_ID}}" -sk="{{env.HW_SECRET_ACCESS_KEY}}" -endpoint="{{env.HW_ENDPOINT}}"
```

#### Large File (Multipart)

```bash
obsutil cp "{{user.local_file}}" obs://{{user.bucket_name}}/{{user.object_key}} \
  -f -threadNum 10 -partSize 100m
```

#### Validation

- Response includes `ETag` and `VersionId` (if versioning enabled)
- `HeadObject` returns 200 with correct size and storage class

---

### Operation: Download Object

#### Execution

```bash
hcloud obs cp obs://{{user.bucket_name}}/{{user.object_key}} "{{user.local_file}}"
```

#### Validation

- File exists at target path with correct size
- MD5 checksum matches (compare with stored hash if available)

---

### Operation: Delete Object

#### Execution

```bash
hcloud obs rm obs://{{user.bucket_name}}/{{user.object_key}}
```

#### Validation

- `HeadObject` returns 404 ObjectNotFound
- If versioning enabled: delete marker created (object persists as version)

---

### Operation: Delete Bucket

#### Pre-flight (Safety Gate)

- **MUST** confirm: "Delete bucket `{{user.bucket_name}}`? Bucket must be empty. All objects must be deleted first."
- Verify bucket is empty: `hcloud obs list obs://{{user.bucket_name}}` returns nothing
- **MUST NOT** proceed if objects remain

#### Execution

```bash
hcloud obs rb obs://{{user.bucket_name}}
```

#### Validation

- `HeadBucket` returns 404
- `ListBuckets` does not include deleted bucket

---

### Operation: Set Bucket ACL

#### Execution

```bash
hcloud obs set-bucket-acl --bucket "{{user.bucket_name}}" --acl private
```

#### Available ACL Values

| ACL | Visibility | Risk |
|-----|-----------|------|
| private | Owner only | Safest — recommended |
| public-read | Anyone can read | **Risk** — data exposure |
| public-read-write | Anyone can read/write | **Extreme risk** — never use in production |

#### Validation

- `GetBucketAcl` returns correct ACL setting
- Test access: try fetching object URL without credentials (should fail for private)

---

### Operation: Lifecycle Rules

#### Create Lifecycle Rule

```bash
hcloud obs set-bucket-lifecycle --bucket "{{user.bucket_name}}" --lifecycle-file lifecycle.json
```

**Example lifecycle.json:**
```json
{
  "Rules": [
    {
      "ID": "transition-to-ia",
      "Status": "Enabled",
      "Prefix": "",
      "Transition": [
        {
          "Days": 30,
          "StorageClass": "WARM"
        },
        {
          "Days": 180,
          "StorageClass": "COLD"
        }
      ],
      "Expiration": {
        "Days": 365
      }
    }
  ]
}
```

#### Validation

- `GetBucketLifecycle` returns configured rules
- Verify rule ID, prefix, transitions, and expiration match expected

---

### Operation: Enable Versioning

#### Execution

```bash
hcloud obs set-bucket-versioning --bucket "{{user.bucket_name}}" --status Enabled
```

#### Validation

- `GetBucketVersioning` returns `Status: Enabled`
- Upload new version of existing object → verify new versionId created

---

### Operation: Static Website Hosting

#### Pre-flight

- Bucket ACL must allow public read (or use CDN in front)
- Index and error documents must exist in bucket

#### Execution

```bash
hcloud obs set-bucket-website \
  --bucket "{{user.bucket_name}}" \
  --index-document "index.html" \
  --error-document "error.html"
```

#### Validation

- `GetBucketWebsite` returns configured index/error documents
- Access website URL in browser → renders correctly

---

## Prerequisites

### 1. Install CLI

```bash
curl -sSL https://cn-north-4.myhuaweicloud.com/cli/latest/hcloud_install.sh -o ./hcloud_install.sh && bash ./hcloud_install.sh -y
hcloud version
```

### 2. Install obsutil (Alternative CLI)

```bash
OS=$(uname -s | tr '[:upper:]' '[:lower:]')
ARCH=$(uname -m)
[ "$ARCH" = "x86_64" ] && ARCH="64" || ARCH="arm64"
curl -fsSL "https://obs-community.obs.cn-north-4.myhuaweicloud.com/obsutil/current/obsutil_${OS}_${ARCH}.tar.gz" | tar -xz
chmod +x obsutil
./obsutil version
```

### 3. Configure obsutil

```bash
./obsutil config -i={{env.HW_ACCESS_KEY_ID}} -k={{env.HW_SECRET_ACCESS_KEY}} -e={{env.HW_ENDPOINT}}
```

### 4. Bootstrap Go Runtime (JIT SDK Fallback)

```bash
if ! command -v go &> /dev/null; then
    OS=$(uname -s | tr '[:upper:]' '[:lower:]')
    ARCH=$(uname -m)
    [ "$ARCH" = "x86_64" ] && ARCH="amd64"
    [ "$ARCH" = "aarch64" ] && ARCH="arm64"
    mkdir -p /tmp/go-runtime
    curl -fsSL "https://go.dev/dl/go1.25.0.${OS}-${ARCH}.tar.gz" | tar -xz -C /tmp/go-runtime
    export PATH="/tmp/go-runtime/go/bin:$PATH"
    export GOPROXY="https://goproxy.cn,direct"
fi
```

## Reference Directory

- [Core Concepts](references/core-concepts.md)
- [API & SDK Usage](references/api-sdk-usage.md)
- [CLI Usage](references/cli-usage.md)
- [Troubleshooting Guide](references/troubleshooting.md)
- [Monitoring & Alerts](references/monitoring.md)
- [Integration & Delegation](references/integration.md)
- [Knowledge Base](references/knowledge-base.md)
- [Well-Architected Assessment](references/well-architected-assessment.md)

## Well-Architected + Three-Pillar Assessment

This skill's operations are evaluated against:
- [Security](references/well-architected-assessment.md#security): Bucket policy, ACL, IAM, encryption
- [Stability](references/well-architected-assessment.md#stability): CRR, versioning, DR runbook
- [Cost](references/well-architected-assessment.md#cost): Storage class optimization, lifecycle savings
- [Efficiency](references/well-architected-assessment.md#efficiency): Multipart, parallel, CDN
- [Performance](references/well-architected-assessment.md#performance): Latency, concurrency, caching
- [FinOps Integration](references/well-architected-assessment.md#finops): Cost visibility, idle detection
- [SecOps Integration](references/well-architected-assessment.md#secops): Public bucket audit, SSE-KMS
- [AIOps Integration](references/knowledge-base.md): Access anomaly detection, egress spike detection
