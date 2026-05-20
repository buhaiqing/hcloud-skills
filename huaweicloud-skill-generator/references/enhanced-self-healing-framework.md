# Enhanced Self-Healing Framework for CLI Installation

> **Purpose:** Defines enhanced CLI installation error handling and self-healing capability framework.
> **Version:** 1.0.0
> **Last Updated:** 2026-05-20
> **Status:** MANDATORY — all generated skills MUST follow this self-healing framework

---

## 1. Core Principles

### 1.1 Self-Healing Maturity Model

| Level | Name | Characteristics |
|-------|------|-----------------|
| L1 | 基础重试 | Fixed-count retry, no error classification |
| L2 | 智能重试 | Error classification, targeted retry strategy |
| L3 | 多路径自愈 | Multiple self-healing paths, auto-select optimal |
| L4 | 预防性自愈 | Pre-flight anomaly detection, proactive prevention |
| L5 | 自学习自愈 | Historical data analysis, strategy optimization |

### 1.2 Self-Healing Decision Tree

```
[异常发生]
    │
    ├── Step 1: 错误分类 (网络/权限/资源/配置/未知)
    ├── Step 2: 选择自愈路径 (根据错误类型选择策略)
    ├── Step 3: 执行自愈 (记录结果)
    ├── Step 4: 验证自愈效果 (检查是否已解决)
    ├── Step 5: 自愈失败处理 (尝试下一级或降级)
    └── Step 6: 用户指导 (明确错误信息和修复建议)
```

---

## 2. Error Classification

### 2.1 CLI Installation Errors

| Category | Code | Scenario | Self-Healing |
|----------|------|---------|-------------|
| **网络异常** | `NET_TIMEOUT` | Download timeout | Switch mirror, increase timeout |
| | `NET_DNS_FAIL` | DNS resolution fails | Use IP or alternate domain |
| | `NET_CONNECTION_REFUSED` | Connection refused | Check firewall, switch network |
| **权限异常** | `PERM_WRITE_FAIL` | Write to /usr/local fails | Use user dir, prompt sudo |
| | `PERM_EXEC_FAIL` | Execute permission denied | chmod +x, prompt sudo |
| **资源异常** | `RES_DISK_FULL` | Insufficient disk space | Clean temp files, prompt user |
| | `RES_BINARY_CORRUPT` | Downloaded file corrupted | Delete and re-download |
| **配置异常** | `CONF_PATH_NOT_FOUND` | PATH doesn't include install path | Auto-add PATH, prompt user |
| **未知异常** | `UNKNOWN_ERROR` | Unclassified | Log details, suggest diagnosis |

### 2.2 Go Runtime Errors

| Category | Code | Scenario | Self-Healing |
|----------|------|---------|-------------|
| **下载异常** | `GO_DOWNLOAD_FAIL` | Go download fails | Switch mirror, use China CDN |
| **解压异常** | `GO_EXTRACT_FAIL` | tar extraction fails | Check integrity, re-download |
| **版本异常** | `GO_VERSION_MISMATCH` | Version mismatch | Download compatible version |
| **环境异常** | `GO_PATH_SETUP_FAIL` | PATH setup fails | Use absolute path invocation |

### 2.3 Dependency Download Errors

| Category | Code | Scenario | Self-Healing |
|----------|------|---------|-------------|
| **网络异常** | `DEP_NET_TIMEOUT` | go get timeout | Switch GOPROXY, increase timeout |
| **权限异常** | `DEP_WRITE_FAIL` | Write to GOMODCACHE fails | Use /tmp directory |
| **编译异常** | `DEP_BUILD_FAIL` | Compilation fails | Check Go version, clear cache |

---

## 3. Enhanced Self-Healing Flow

### 3.1 CLI Installation

#### Phase 1: Pre-flight Checks

```bash
# 预检1: 网络连通性
echo "=== Pre-flight: Network ==="
if ! curl -fsSL --connect-timeout 5 https://console.huaweicloud.com/ > /dev/null 2>&1; then
    echo "⚠️  Network check failed, trying alternate endpoints..."
    for endpoint in "https://support.huaweicloud.com/" "https://www.huaweicloud.com/"; do
        if curl -fsSL --connect-timeout 5 "$endpoint" > /dev/null 2>&1; then
            echo "✅ Alternate endpoint available: $endpoint"; break
        fi
    done
fi

# 预检2: 磁盘空间
echo "=== Pre-flight: Disk Space ==="
REQUIRED_MB=50
AVAILABLE_KB=$(df -k /tmp | awk 'NR==2 {print $4}')
AVAILABLE_MB=$((AVAILABLE_KB / 1024))
if [ "$AVAILABLE_MB" -lt "$REQUIRED_MB" ]; then
    rm -rf /tmp/huaweicloud-cli-* /tmp/go-* 2>/dev/null
fi

# 预检3: 安装路径权限
echo "=== Pre-flight: Permissions ==="
INSTALL_PATH="/usr/local/bin"
if [ ! -w "$INSTALL_PATH" ]; then
    USER_BIN="$HOME/.local/bin"
    mkdir -p "$USER_BIN" && INSTALL_PATH="$USER_BIN"
fi
```

#### Phase 2: Multi-Mirror Download

```bash
download_hcloud_cli() {
    local mirrors=(
        "https://obs.cn-north-4.myhuaweicloud.com/hcli/hcloud-cli-latest"
        "https://repo.huaweicloud.com/tools/hcli/"
    )
    for mirror in "${mirrors[@]}"; do
        if curl -fsSL --connect-timeout 10 "$mirror" -o /tmp/hcloud-cli; then
            if [ -s /tmp/hcloud-cli ]; then
                echo "✅ Download successful"; return 0
            fi
        fi
    done
    return 1
}
```

#### Phase 3: Installation Verification

```bash
health_check_hcloud_cli() {
    SCORE=0; MAX=10
    [ -f "$INSTALL_PATH/hcloud" ] && SCORE=$((SCORE+2)) && echo "✅ Binary exists"
    [ -x "$INSTALL_PATH/hcloud" ] && SCORE=$((SCORE+2)) && echo "✅ Executable"
    command -v hcloud &>/dev/null && SCORE=$((SCORE+2)) && echo "✅ In PATH"
    hcloud version &>/dev/null && SCORE=$((SCORE+2)) && echo "✅ Version OK"
    echo "Health Score: $SCORE/$MAX"
    [ "$SCORE" -ge 8 ] && return 0 || return 1
}
```

### 3.2 Go runtime Self-Healing

```bash
bootstrap_go_enhanced() {
    if command -v go &> /dev/null; then
        GO_VER=$(go version | awk '{print $3}')
        echo "✅ Go already installed: $GO_VER"; return 0
    fi
    
    OS=$(uname -s | tr '[:upper:]' '[:lower:]')
    ARCH=$(uname -m)
    [ "$ARCH" = "x86_64" ] && ARCH="amd64"
    [ "$ARCH" = "aarch64" ] && ARCH="arm64"
    
        for mirror in "https://go.dev/dl" "https://golang.google.cn/dl" "https://mirrors.aliyun.com/golang"; do
            if curl -fsSL --connect-timeout 15 "${mirror}/${ver}.${OS}-${ARCH}.tar.gz" -o /tmp/go.tar.gz; then
                mkdir -p /tmp/go-runtime
                tar -xzf /tmp/go-runtime.tar.gz -C /tmp/go-runtime 2>/dev/null
                if [ -f "/tmp/go-runtime/go/bin/go" ]; then
                    export PATH="/tmp/go-runtime/go/bin:$PATH"
                    export GOPROXY="https://goproxy.cn,direct"
                    echo "✅ Go installed: $ver"
                    return 0
                fi
                rm -rf /tmp/go-runtime /tmp/go.tar.gz
            fi
        done
    done
    echo "❌ Go download failed all attempts"
    return 1
}
```

---

## 4. Degradation Path

```
[CLI安装失败]
    │
    ├── 尝试自愈 (最多5次)
    │   ├── 成功 → 继续执行
    │   └── 失败 → 降级
    │
    ├── 降级1: JIT Go SDK模式 (Go可用 → SDK执行)
    ├── 降级2: 控制台手动操作 (提供控制台链接和步骤)
    └── 降级3: 用户手动修复 (提供详细错误信息和修复建议)
```

---

## 5. Self-Healing Metrics

| Metric | Target | Method |
|--------|--------|--------|
| Self-healing success rate | > 80% | Successful heals / total exceptions |
| Mean self-healing time | < 30s | Time from exception to resolution |
| User intervention rate | < 20% | Exceptions requiring manual intervention |
| Degradation path usage | < 10% | Exceptions entering degradation |

---

## 6. Compliance Checklist

- [ ] All CLI installation flows include pre-flight checks
- [ ] Error classification covers all known types (network, permission, resource, config)
- [ ] Each error type has corresponding self-healing strategy
- [ ] Clear degradation path after self-healing exhausted
- [ ] User guidance includes detailed error information and fix suggestions
- [ ] Post-installation health check executed
- [ ] Self-healing events logged
- [ ] Self-healing success rate trackable

---

*This framework is mandatory for all generated skills. Update quarterly based on effectiveness data.*
