# Integration — Huawei Cloud GaussDB

> **Purpose:** SDK setup, cross-skill delegation, and environment configuration.
> **Version:** 1.0.0
> **Last Updated:** 2026-07-18

---

## Table of Contents

1. [Go SDK Setup](#1-go-sdk-setup)
2. [JIT SDK Bootstrap](#2-jit-sdk-bootstrap)
3. [Cross-Skill Delegation Matrix](#3-cross-skill-delegation-matrix)
4. [Environment Variables](#4-environment-variables)
5. [Dependency Configuration](#5-dependency-configuration)

---

## 1. Go SDK Setup

### 1.1 SDK Installation

```bash
mkdir gaussdb-ops && cd gaussdb-ops
go mod init gaussdb-ops
go get github.com/huaweicloud/huaweicloud-sdk-go-v3@v1.6.0
go list -m github.com/huaweicloud/huaweicloud-sdk-go-v3
```

### 1.2 SDK Package Structure

```
huaweicloud-sdk-go-v3/
├── core/                    # Core SDK (config, auth, HTTP)
├── services/gaussdb/        # GaussDB openGauss / MySQL SDK
```

### 1.3 Client Construction

```go
package main

import (
    "fmt"
    "net/http"
    "time"

   "github.com/huaweicloud/huaweicloud-sdk-go-v3/core/auth/global"
    gaussdb "github.com/huaweicloud/huaweicloud-sdk-go-v3/services/gaussdb/v5"
    "github.com/huaweicloud/huaweicloud-sdk-go-v3/core"
)

func newGaussDBClient(region string) *gaussdb.GaussDBClient {
    credential := global.NewCredentialsBuilder().
        WithAk("{{env.HW_ACCESS_KEY_ID}}").
        WithSk("{{env.HW_SECRET_ACCESS_KEY}}").
        WithProjectId("{{env.HW_PROJECT_ID}}").
        Build()

    endpoint := fmt.Sprintf("gaussdb.%s.myhuaweicloud.com", region)

    client := gaussdb.NewGaussDBClientBuilder().
        WithEndpoint(endpoint).
        WithCredential(credential).
        WithHttpConfig(&core.HttpConfig{Timeout: 60 * time.Second}).
        Build()

    return client
}
```

---

## 2. JIT SDK Bootstrap

### 2.1 JIT Runtime Detection

```bash
#!/bin/bash
detect_go() {
    if command -v go &> /dev/null; then
        GO_VERSION=$(go version | grep -oP 'go\K[0-9]+\.[0-9]+')
        echo "Found Go: $GO_VERSION"
        return 0
    fi
    return 1
}

install_go() {
    OS=$(uname -s | tr '[:upper:]' '[:lower:]')
    ARCH=$(uname -m)
    case "$ARCH" in x86_64) ARCH="amd64" ;; aarch64|arm64) ARCH="arm64" ;; esac
    GO_VERSION="1.25.0"
    GO_URL="https://go.dev/dl/go${GO_VERSION}.${OS}-${ARCH}.tar.gz"
    mkdir -p /tmp/go-runtime
    curl -fsSL "$GO_URL" | tar -xz -C /tmp/go-runtime
    export PATH="/tmp/go-runtime/go/bin:$PATH"
}
```

---

## 3. Cross-Skill Delegation Matrix

### 3.1 Alarm-to-Skill Routing

| Alarm Type | Metric | Primary Skill | Secondary Skill | Delegation |
|-----------|--------|--------------|-----------------|------------|
| CPU High | gaussdb001_cpu_usage | huaweicloud-gaussdb-ops | huaweicloud-ces-ops | Optional |
| Memory High | gaussdb002_mem_usage | huaweicloud-gaussdb-ops | — | — |
| Connections High | gaussdb003_connections_usage | huaweicloud-gaussdb-ops | huaweicloud-vpc-ops | Optional |
| Disk Full | gaussdb004_disk_usage | huaweicloud-gaussdb-ops | — | — |
| Slow Queries | gaussdb005_slow_queries | huaweicloud-gaussdb-ops | — | Required |
| Replication Lag | gaussdb006_replication_lag | huaweicloud-gaussdb-ops | — | — |
| Backup Failed | gaussdb007_backup_failures | huaweicloud-gaussdb-ops | huaweicloud-obs-ops | Required |
| Transaction Lock Wait | gaussdb008_lock_wait_timeout | huaweicloud-gaussdb-ops | — | — |

### 3.2 Delegation Protocol

```
[告警触发]
    │
    ├── 1. 识别Alarm Type + Metric
    ├── 2. 查矩阵确定主诊断Skill
    ├── 3. 调用huaweicloud-gaussdb-ops检查资源状态
    ├── 4. 若资源异常 → 调用Secondary Skill（如需要）
    ├── 5. 若delegation=Required → 始终调用secondary
    └── 6. 汇总所有输出生成统一报告
```

### 3.3 Skill Dependencies

| Resource | Required Skill | Action |
|----------|---------------|--------|
| VPC | huaweicloud-vpc-ops | Must exist before GaussDB creation |
| Subnet | huaweicloud-vpc-ops | Must exist before GaussDB creation |
| Security Group | huaweicloud-vpc-ops | Must exist before GaussDB creation |
| CES Metrics | huaweicloud-ces-ops | For monitoring integration |
| OBS Backup | huaweicloud-obs-ops | For backup storage |
| IAM Permissions | huaweicloud-iam-ops | For permission management |

---

## 4. Environment Variables

### 4.1 Required Variables

| Variable | Description | Example | Source |
|----------|-------------|---------|--------|
| `HW_ACCESS_KEY_ID` | Huawei Cloud Access Key | `XXXXXXXXXXXXXXXXXXXX` | User environment |
| `HW_SECRET_ACCESS_KEY` | Huawei Cloud Secret Key | `XXXXXXXXXXXXXXXXXXXX` | User environment |
| `HW_REGION_ID` | Region ID | `cn-north-4` | User environment |
| `HW_PROJECT_ID` | Project ID | `xxxxxxxxxxxxxxxxxxxx` | User environment |

---

## 5. Dependency Configuration

### 5.1 GaussDB Instance Dependencies

```bash
# Before creating GaussDB, verify dependencies
huaweicloud-vpc-ops:check-vpc --vpc-id {{user.vpc_id}}
huaweicloud-vpc-ops:check-subnet --subnet-id {{user.subnet_id}}
huaweicloud-vpc-ops:check-security-group --sg-id {{user.sg_id}}
```
