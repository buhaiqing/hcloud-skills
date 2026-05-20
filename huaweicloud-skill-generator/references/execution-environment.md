# Execution Environment Setup — Huawei Cloud Skill Generator

> **Purpose:** CLI install, Go JIT download, credential configuration, verification steps. Progressive disclosure — loaded on demand by agent.
> **Status:** Reference document

---

## 1. CLI Installation

### 1.1 KooCLI (Official Binary)

```bash
# Linux one-click install
curl -sSL https://cn-north-4.myhuaweicloud.com/cli/latest/hcloud_install.sh -o ./hcloud_install.sh && bash ./hcloud_install.sh -y

# Verify
hcloud version
# Current KooCLI version: 4.1.6
```

### 1.2 OpenStack CLI (alternative)

```bash
pip install python-openstackclient python-huaweicloudsdk
```

---

## 2. Go Runtime JIT

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
    export GOPATH="/tmp/go-workspace"
    export GOCACHE="/tmp/go-cache"
fi

go version
```

Go version strategy: **JIT download Go 1.24+**, **Script compatibility Go 1.21+**.

---

## 3. Credential Configuration

### 3.1 Environment Variables (Recommended)

```bash
export HW_ACCESS_KEY_ID="{{env.HW_ACCESS_KEY_ID}}"
export HW_SECRET_ACCESS_KEY="{{env.HW_SECRET_ACCESS_KEY}}"
export HW_REGION_ID="{{env.HW_REGION_ID}}"
export HW_PROJECT_ID="{{env.HW_PROJECT_ID}}"
# Security: check existence only, never echo
test -n "$HW_SECRET_ACCESS_KEY" && echo "✅ Credentials configured"
```

### 3.2 CLI Configuration

```bash
hcloud init
# Follow interactive prompts
```

### 3.3 Go SDK Credential

```go
import (
    "os"
    "github.com/huaweicloud/huaweicloud-sdk-go-v3/core/auth/basic"
)

ak := os.Getenv("HW_ACCESS_KEY_ID")
sk := os.Getenv("HW_SECRET_ACCESS_KEY") // SECURITY: never log/print
creds := basic.NewCredentialsBuilder().
    WithAk(ak).WithSk(sk).Build()
```

---

## 4. Verification

```bash
# CLI verification
hcloud ecs describe-instances --region cn-north-4

# Go SDK verification (compile + run test script)
```

---

*This document uses progressive disclosure. Agent loads only the section needed for current task.*
