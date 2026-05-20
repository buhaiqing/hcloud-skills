# Integration — Huawei Cloud RDS

> **Purpose:** SDK setup, cross-skill delegation, and environment configuration.
> **Version:** 1.0.0
> **Last Updated:** 2026-05-20

---

## Table of Contents

1. [Go SDK Setup](#1-go-sdk-setup)
2. [JIT SDK Bootstrap](#2-jit-sdk-bootstrap)
3. [Cross-Skill Delegation Matrix](#3-cross-skill-delegation-matrix)
4. [Environment Variables](#4-environment-variables)
5. [Dependency Configuration](#5-dependency-configuration)
6. [Version Compatibility](#6-version-compatibility)

---

## 1. Go SDK Setup

### 1.1 SDK Installation

```bash
# Create project
mkdir rds-ops && cd rds-ops
go mod init rds-ops

# Install RDS SDK
go get github.com/huaweicloud/huaweicloud-sdk-go-v3@v1.6.0

# Verify
go list -m github.com/huaweicloud/huaweicloud-sdk-go-v3
```

### 1.2 SDK Package Structure

```
huaweicloud-sdk-go-v3/
├── core/                    # Core SDK (config, auth, HTTP)
├── services/
│   └── rds/
│       └── v3/              # RDS API v3
│           ├── client.go    # RDS client
│           ├── model/       # Request/response models
│           └── region/       # Regional endpoints
```

### 1.3 Client Initialization

```go
package main

import (
    "fmt"
    "os"
    
    "github.com/huaweicloud/huaweicloud-sdk-go-v3/core/auth/basic"
    "github.com/huaweicloud/huaweicloud-sdk-go-v3/core/config"
    "github.com/huaweicloud/huaweicloud-sdk-go-v3/core/httphandler"
    "rds" "github.com/huaweicloud/huaweicloud-sdk-go-v3/services/rds/v3"
)

func newRdsClient(region string) *rds.RdsClient {
    // Environment-based credentials
    ak := os.Getenv("HW_ACCESS_KEY_ID")
    sk := os.Getenv("HW_SECRET_ACCESS_KEY")
    
    // Custom HTTP config for retry and timeout
    httpConfig := config.DefaultHttpConfig().
        WithTimeout(120).          // 120s timeout
        WithMaxRetryCount(3).      // 3 retries
        WithTransport(getTransport())
    
    // Credential builder
    credential := basic.NewCredentialsBuilder().
        WithAk(ak).
        WithSk(sk).
        Build()
    
    // Regional endpoint
    endpoint := fmt.Sprintf("rds.%s.myhuaweicloud.com", region)
    
    // Build client
    client := rds.RdsClientBuilder().
        WithEndpoint(endpoint).
        WithCredential(credential).
        WithHttpConfig(httpConfig).
        Build()
    
    return client
}

func getTransport() *http.Transport {
    return &http.Transport{
        MaxIdleConns:       10,
        MaxIdleConnsPerHost: 10,
        IdleConnTimeout:     90 * time.Second,
    }
}
```

---

## 2. JIT SDK Bootstrap

### 2.1 JIT Runtime Detection

```bash
#!/bin/bash
# jittify.sh — Bootstrap Go runtime for JIT SDK execution

detect_go() {
    if command -v go &> /dev/null; then
        GO_VERSION=$(go version | grep -oP 'go\K[0-9]+\.[0-9]+')
        echo "Found Go: $GO_VERSION"
        return 0
    else
        echo "Go not found, installing..."
        return 1
    fi
}

install_go() {
    OS=$(uname -s | tr '[:upper:]' '[:lower:]')
    ARCH=$(uname -m)
    
    case $ARCH in
        x86_64) ARCH="amd64" ;;
        aarch64|arm64) ARCH="arm64" ;;
        *) echo "Unsupported architecture: $ARCH"; exit 1 ;;
    esac
    
    GO_VERSION="1.25.0"
    GO_URL="https://go.dev/dl/go${GO_VERSION}.${OS}-${ARCH}.tar.gz"
    
    mkdir -p /tmp/go-runtime
    curl -fsSL "$GO_URL" | tar -xz -C /tmp/go-runtime
    export PATH="/tmp/go-runtime/go/bin:$PATH"
    export GOPROXY="https://goproxy.cn,direct"
    
    echo "Installed Go $GO_VERSION to /tmp/go-runtime"
}

# Main bootstrap
if ! detect_go; then
    install_go
fi

# Verify
go version
```

### 2.2 JIT Script Execution

```bash
#!/bin/bash
# run_rds_jit.sh — Execute RDS JIT Go script

set -e

# Bootstrap runtime
source ./jittify.sh

# Run Go script
go run rds_operations.go "$@"
```

### 2.3 JIT Script Template

```go
// rds_jit_template.go
// Template for JIT RDS operations

package main

import (
    "encoding/json"
    "fmt"
    "os"
    "time"
    
    "github.com/huaweicloud/huaweicloud-sdk-go-v3/core/auth/basic"
    "github.com/huaweicloud/huaweicloud-sdk-go-v3/core/config"
    rds "github.com/huaweicloud/huaweicloud-sdk-go-v3/services/rds/v3"
    "rds_model" "github.com/huaweicloud/huaweicloud-sdk-go-v3/services/rds/v3/model"
)

func main() {
    // Initialize client
    region := os.Getenv("HW_REGION_ID")
    projectId := os.Getenv("HW_PROJECT_ID")
    client := initClient(region)
    
    // Parse command
    cmd := os.Args[1]
    
    switch cmd {
    case "create":
        createInstance(client, projectId)
    case "describe":
        describeInstance(client, os.Args[2])
    case "delete":
        deleteInstance(client, os.Args[2])
    default:
        fmt.Printf("Unknown command: %s\n", cmd)
        os.Exit(1)
    }
}

func initClient(region string) *rds.RdsClient {
    ak := os.Getenv("HW_ACCESS_KEY_ID")
    sk := os.Getenv("HW_SECRET_ACCESS_KEY")
    
    return rds.RdsClientBuilder().
        WithEndpoint(fmt.Sprintf("rds.%s.myhuaweicloud.com", region)).
        WithCredential(basic.NewCredentialsBuilder().
            WithAk(ak).WithSk(sk).Build()).
        WithHttpConfig(config.DefaultHttpConfig().
            WithTimeout(120).
            WithMaxRetryCount(3)).
        Build()
}

// Operations implemented as functions
// ... (see api-sdk-usage.md for details)
```

---

## 3. Cross-Skill Delegation Matrix

### 3.1 Alarm-to-Skill Routing

| Alarm Type | Metric | Primary Skill | Secondary Skill | Delegation |
|-----------|--------|--------------|-----------------|------------|
| CPU High | rds001_cpu_usage | huaweicloud-rds-ops | huaweicloud-ces-ops | Optional |
| Memory High | rds002_mem_usage | huaweicloud-rds-ops | — | — |
| Connections High | rds003_connections_usage | huaweicloud-rds-ops | huaweicloud-vpc-ops | Optional |
| Disk Full | rds004_disk_usage | huaweicloud-rds-ops | — | — |
| Slow Queries | rds043_slow_queries | huaweicloud-rds-ops | — | Required |
| Replication Lag | rds006_replication_lag | huaweicloud-rds-ops | — | — |
| Backup Failed | rds050_backup_failures | huaweicloud-rds-ops | huaweicloud-obs-ops | Required |
| HA Down | rds007_replication_status | huaweicloud-rds-ops | — | — |

### 3.2 Delegation Protocol

```
[告警触发]
    │
    ├── 1. 识别Alarm Type + Metric
    ├── 2. 查矩阵确定主诊断Skill
    ├── 3. 调用huaweicloud-rds-ops检查资源状态
    ├── 4. 若资源异常 → 调用Secondary Skill（如需要）
    ├── 5. 若delegation=Required → 始终调用secondary
    └── 6. 汇总所有输出生成统一报告
```

### 3.3 Skill Dependencies

| Resource | Required Skill | Action |
|----------|---------------|--------|
| VPC | huaweicloud-vpc-ops | Must exist before RDS creation |
| Subnet | huaweicloud-vpc-ops | Must exist before RDS creation |
| Security Group | huaweicloud-vpc-ops | Must exist before RDS creation |
| CES Metrics | huaweicloud-ces-ops | For monitoring integration |
| OBS Backup | huaweicloud-obs-ops | For backup storage |
| IAM Permissions | huaweicloud-iam-ops | For permission management |

### 3.4 Delegation Templates

```markdown
## Delegation: VPC Check Before RDS Creation

When creating RDS instance:
1. Call huaweicloud-vpc-ops to verify VPC exists
2. If not exists → HALT with message "Create VPC first"
3. Call huaweicloud-vpc-ops to verify subnet exists
4. If not exists → HALT with message "Create subnet first"
5. Call huaweicloud-vpc-ops to verify security group exists
6. If not exists → HALT with message "Create security group first"
7. Proceed with RDS creation

## Delegation: OBS Check Before Backup

When creating backup:
1. Call huaweicloud-obs-ops to check OBS quota
2. If insufficient → HALT with message "Clean up OBS or expand quota"
3. Proceed with backup creation
```

---

## 4. Environment Variables

### 4.1 Required Variables

| Variable | Description | Example | Source |
|----------|-------------|---------|--------|
| `HW_ACCESS_KEY_ID` | Huawei Cloud Access Key | `XXXXXXXXXXXXXXXXXXXX` | User environment |
| `HW_SECRET_ACCESS_KEY` | Huawei Cloud Secret Key | `XXXXXXXXXXXXXXXXXXXX` | User environment |
| `HW_REGION_ID` | Region ID | `cn-north-4` | User environment |
| `HW_PROJECT_ID` | Project ID | `xxxxxxxxxxxxxxxxxxxx` | User environment |

### 4.2 Optional Variables

| Variable | Description | Default | Notes |
|----------|-------------|---------|-------|
| `HW_ENDPOINT` | Custom endpoint | Auto | For CN site or private cloud |
| `HW_TIMEOUT` | Request timeout (seconds) | `120` | SDK timeout |
| `HW_MAX_RETRIES` | Max retry count | `3` | SDK retry |
| `HW_LOG_LEVEL` | Log level | `WARN` | DEBUG/INFO/WARN/ERROR |

### 4.3 Environment Setup Script

```bash
#!/bin/bash
# setup_env.sh — Configure RDS skill environment

# Required credentials
export HW_ACCESS_KEY_ID="${HW_ACCESS_KEY_ID:?Access Key required}"
export HW_SECRET_ACCESS_KEY="${HW_SECRET_ACCESS_KEY:?Secret Key required}"
export HW_REGION_ID="${HW_REGION_ID:?Region required}"
export HW_PROJECT_ID="${HW_PROJECT_ID:?Project ID required}"

# Optional configurations
export HW_TIMEOUT="${HW_TIMEOUT:-120}"
export HW_MAX_RETRIES="${HW_MAX_RETRIES:-3}"
export HW_LOG_LEVEL="${HW_LOG_LEVEL:-WARN}"

# Verify setup
echo "=== Huawei Cloud RDS Environment ==="
echo "Region: $HW_REGION_ID"
echo "Project: $HW_PROJECT_ID"
echo "Timeout: $HW_TIMEOUT"
echo "Max Retries: $HW_MAX_RETRIES"

# Test connectivity
hcloud rds list --region $HW_REGION_ID --limit 1
```

---

## 5. Dependency Configuration

### 5.1 Go Module Configuration

```go
// go.mod
module rds-ops

go 1.21

require (
    github.com/huaweicloud/huaweicloud-sdk-go-v3 v1.6.0
)

replace (
    github.com/huaweicloud/huaweicloud-sdk-go-v3 => github.com/huaweicloud/huaweicloud-sdk-go-v3 v1.6.0
)
```

### 5.2 Dependency Injection Pattern

```go
// dependency injection for testability

type RDSClient interface {
    CreateInstance(request *CreateInstanceRequest) (*CreateInstanceResponse, error)
    ShowInstance(request *ShowInstanceRequest) (*ShowInstanceResponse, error)
    DeleteInstance(request *DeleteInstanceRequest) (*DeleteInstanceResponse, error)
    ListInstances(request *ListInstancesRequest) (*ListInstancesResponse, error)
}

type RdsSkillDependencies struct {
    Client     RDSClient
    ProjectId  string
    Region     string
}

func NewRdsSkill(deps RdsSkillDependencies) *RdsSkill {
    return &RdsSkill{
        client:    deps.Client,
        projectId: deps.ProjectId,
        region:    deps.Region,
    }
}
```

---

## 6. Version Compatibility

### 6.1 SDK Version Matrix

| SDK Version | Go Version | RDS API Version | Status |
|-------------|------------|-----------------|--------|
| v1.5.x | 1.18+ | v3 | Legacy |
| v1.6.x | 1.21+ | v3 | Current |
| v1.7.x | 1.21+ | v3.1 | Beta |

### 6.2 API Version Support

| API Version | Status | Notes |
|-------------|--------|-------|
| v1 | Deprecated | Avoid for new development |
| v3 | Recommended | Current stable version |
| v3.1 | Recommended | Latest features, same stability |

### 6.3 Breaking Changes

| Version | Change | Migration |
|---------|--------|-----------|
| v1 → v3 | Authentication | Use AK/SK directly |
| v3 → v3.1 | Response format | Update response parsing |

---

## 7. Configuration Management

### 7.1 Multi-Environment Config

```yaml
# config.yaml
environments:
  production:
    region: cn-north-4
    project_id: prod-project-xxx
    timeout: 120
    max_retries: 3
    backup_retention: 7
    ha_mode: true

  staging:
    region: cn-north-4
    project_id: staging-project-xxx
    timeout: 60
    max_retries: 2
    backup_retention: 3
    ha_mode: true

  development:
    region: cn-north-4
    project_id: dev-project-xxx
    timeout: 30
    max_retries: 1
    backup_retention: 1
    ha_mode: false
```

### 7.2 Feature Flags

```go
type FeatureFlags struct {
    EnableAutoBackup    bool
    EnableBackupEncryption bool
    EnableTDE           bool
    EnableSSL           bool
    EnableReadReplica   bool
}

// Per-environment overrides
var DefaultFlags = FeatureFlags{
    EnableAutoBackup:       true,
    EnableBackupEncryption:  true,
    EnableTDE:               false,
    EnableSSL:               true,
    EnableReadReplica:       false,
}
```

---

*This document defines integration patterns for RDS operations. Refer to official Huawei Cloud SDK documentation for the latest integration details.*