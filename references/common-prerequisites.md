# Common Prerequisites — Huawei Cloud Ops Skills

> Shared prerequisites for all `huaweicloud-*-ops` skills. Each skill's SKILL.md
> links here instead of duplicating installation scripts.

## 1. Install KooCLI

```bash
curl -sSL https://cn-north-4.myhuaweicloud.com/cli/latest/hcloud_install.sh -o ./hcloud_install.sh && bash ./hcloud_install.sh -y
hcloud version
```

## 2. Bootstrap Go Runtime (JIT SDK Fallback)

Required when `cli_applicability=dual-path` or `sdk-only`. Skip if CLI-only.

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

## 3. Configure Credentials

```bash
export HW_ACCESS_KEY_ID="{{env.HW_ACCESS_KEY_ID}}"
export HW_SECRET_ACCESS_KEY="{{env.HW_SECRET_ACCESS_KEY}}"
export HW_REGION_ID="{{env.HW_REGION_ID}}"
export HW_PROJECT_ID="{{env.HW_PROJECT_ID}}"
test -n "$HW_SECRET_ACCESS_KEY" && echo "✅ Credentials configured"
```

> **Security**: Never echo or log credential values. Verify existence only.
