# Deployment & Operations Guide

> **Scope**: This guide covers the full lifecycle of deploying, operating, and maintaining an
> `hcloud-skills` environment — from first-time setup to production operations and CI/CD pipelines.
>
> **Audience**: Platform engineers, SREs, and ops teams operating Huawei Cloud skills.
>
> **Last updated**: 2026-07-26

---

## 1. Environment Setup

### 1.1 Prerequisites

| Requirement | Version | Notes |
|-------------|---------|-------|
| Huawei Cloud account | — | With IAM credentials (Access Key ID + Secret Access Key) |
| `hcloud` CLI | latest | Primary execution path for all skill operations |
| `hwcloud-skillcheck` binary | ≥ v0.1.4 | Standalone validator (no Go toolchain needed) |
| Docker (optional) | ≥ 24 | For sandboxed execution environment |
| Go (optional) | ≥ 1.26 | Only needed for local development / SDK fallback |

### 1.2 Install Huawei Cloud CLI (KooCLI)

```bash
# One-click install (Linux/macOS)
curl -sSL https://cn-north-4.myhuaweicloud.com/cli/latest/hcloud_install.sh | bash

# Verify
hcloud version
```

See [Huawei Cloud CLI official docs](https://support.huaweicloud.com/intl/en-us/cli-latest/) for alternative installation methods.

### 1.3 Configure Credentials

```bash
# Set credentials via environment variables (recommended for CI/automation)
export HW_ACCESS_KEY_ID="your-access-key-id"
export HW_SECRET_ACCESS_KEY="your-secret-access-key"
export HW_REGION_ID="cn-north-4"

# Or use hcloud CLI configure command
hcloud configure set --cli-profile default \
  --access-key-id "$HW_ACCESS_KEY_ID" \
  --secret-access-key "$HW_SECRET_ACCESS_KEY" \
  --region "$HW_REGION_ID"
```

> **Security**: Never commit credentials to git. Use environment variables, secret managers,
> or `.env` files (gitignored). See §6 Security Baseline.

### 1.4 Install hwcloud-skillcheck

The `hwcloud-skillcheck` binary is a **standalone, statically linked Go binary** with zero
external dependencies — no Python, no Go toolchain required.

```bash
# Download the latest release for your platform
# See https://github.com/buhaiqing/hcloud-skills/releases for all versions

# Linux (amd64)
VERSION="v0.1.4"
curl -sSLO "https://github.com/buhaiqing/hcloud-skills/releases/download/${VERSION}/hwcloud-skillcheck-linux-amd64"
chmod +x hwcloud-skillcheck-linux-amd64
sudo mv hwcloud-skillcheck-linux-amd64 /usr/local/bin/hwcloud-skillcheck

# macOS (Apple Silicon)
curl -sSLO "https://github.com/buhaiqing/hcloud-skills/releases/download/${VERSION}/hwcloud-skillcheck-darwin-arm64"
chmod +x hwcloud-skillcheck-darwin-arm64
sudo mv hwcloud-skillcheck-darwin-arm64 /usr/local/bin/hwcloud-skillcheck

# Windows (PowerShell)
$version = "v0.1.4"
$url = "https://github.com/buhaiqing/hcloud-skills/releases/download/$version/hwcloud-skillcheck-windows-amd64"
$out = "$env:USERPROFILE\.local\bin\hwcloud-skillcheck.exe"
Invoke-WebRequest -Uri $url -OutFile $out

# Verify
hwcloud-skillcheck --help
```

Available platform binaries:

| Asset | Platform | Architecture |
|-------|----------|-------------|
| `hwcloud-skillcheck-linux-amd64` | Linux | x86_64 |
| `hwcloud-skillcheck-linux-arm64` | Linux | ARM64 |
| `hwcloud-skillcheck-darwin-amd64` | macOS | Intel |
| `hwcloud-skillcheck-darwin-arm64` | macOS | Apple Silicon |
| `hwcloud-skillcheck-windows-amd64` | Windows | x86_64 |
| `hwcloud-skillcheck-windows-arm64` | Windows | ARM64 |

---

## 2. Deployment Options

### 2.1 Bare Metal / VM Deployment

For production use where direct CLI access is needed:

```bash
# 1. Install hcloud CLI + configure credentials (see §1.2-1.3)

# 2. Install hwcloud-skillcheck (see §1.4)

# 3. Clone the skills repository
git clone https://github.com/buhaiqing/hcloud-skills.git /opt/hcloud-skills

# 4. Validate the environment
cd /opt/hcloud-skills
hwcloud-skillcheck validate --root .
# Expected: all A-class checks pass, exit 0

# 5. Use skills interactively
# Each skill's SKILL.md contains executable operations
cat huaweicloud-ecs-ops/SKILL.md
```

### 2.2 Docker Sandbox

For isolated execution environments (development, testing, or CI):

```bash
# 1. Configure credentials
cp .env.example .env
# Edit .env with your HW_ACCESS_KEY_ID, HW_SECRET_ACCESS_KEY, HW_REGION_ID

# 2. Build and start
docker-compose build
docker-compose up hcloud-skills

# Inside the container:
check-env          # Verify HW_* env vars
skill-list          # List all available skills
skill-read <name>   # Read a skill's SKILL.md
hc <product> <op>   # Alias for hcloud CLI
```

Available services:

| Service | Purpose | Profile |
|---------|---------|---------|
| `hcloud-skills` | Interactive CLI sandbox | default |
| `hcloud-worker` | Background task execution | default |
| `hcloud-test` | Isolated test environment | test |
| `hcloud-sdk-builder` | Go SDK compilation | build |

### 2.3 CI/CD Integration

The repository includes two CI workflows that serve as reference for external integration:

**validate-skills.yml** — runs on every push/PR to main:
- Builds `hwcloud-skillcheck` from source
- Runs A-class validation suite (frontmatter, eval-queries, example-config, markdown-links, etc.)
- Runs GCL surface (aggregate trace, alarm-wire smoke)
- Runs learning + L4 subcommands
- Runs drift guard (sync + check canonical/runtime copy equality)
- Runs critic scorer smoke test

**build-skillcheck.yml** — runs on push/PR + tags `v*`:
- Cross-platform build matrix (6 platform/arch combinations)
- Tests + lint on ubuntu-latest
- On tag push: publishes artifacts to GitHub Release

External consumers can integrate `hwcloud-skillcheck validate --root <repo>` into their own CI
pipelines — the binary is standalone and needs no runtime dependencies.

---

## 3. Release Process

### 3.1 Creating a Release

```bash
# Ensure working tree is clean
git status

# Build, test, and verify locally
make all          # fmt + vet + test + build
make self-check   # verify binary against embedded fixtures

# Tag and push (CI will build and publish artifacts)
make release VERSION=0.2.0
# This runs: git tag v0.2.0 && git push origin v0.2.0
# CI workflow build-skillcheck.yml picks up the tag,
# builds 6 platform binaries, and publishes to GitHub Releases
```

### 3.2 Release Artifacts

Each release produces 6 platform-specific binaries:

```
hwcloud-skillcheck-linux-amd64      # 3.7 MB
hwcloud-skillcheck-linux-arm64      # 3.5 MB
hwcloud-skillcheck-darwin-amd64     # 3.7 MB
hwcloud-skillcheck-darwin-arm64     # 3.5 MB
hwcloud-skillcheck-windows-amd64    # 3.9 MB
hwcloud-skillcheck-windows-arm64    # 3.5 MB
```

All binaries are:
- **Statically linked** (`CGO_ENABLED=0`) — no libc dependencies
- **Stripped** (`-ldflags="-s -w"`) — minimal binary size
- **Deterministic** — `-trimpath` removes local filesystem paths

### 3.3 Version Scheme

| Pattern | Example | Use Case |
|---------|---------|----------|
| `vMAJOR.MINOR.PATCH` | `v0.1.4` | Production release |
| `vMAJOR.MINOR.PATCH-rc.N` | `v0.2.0-rc.1` | Release candidate |
| `dev` | `dev` | Local development (git describe fails) |

---

## 4. CI/CD Pipeline Reference

### 4.1 Workflow: validate-skills.yml

**File**: `.github/workflows/validate-skills.yml`
**Triggers**: `push` to `main`/`master`, `pull_request`

| Step | Command | Purpose | Fails On |
|------|---------|---------|----------|
| Build | `go build -trimpath` | Compile hwcloud-skillcheck binary | Compile error |
| gofmt | `gofmt -l .` | Format check | Unformatted Go files |
| go vet | `go vet ./...` | Static analysis | Suspicious constructs |
| Go test | `go test ./...` | Unit + integration tests | Test failure |
| A-class validate | `hwcloud-skillcheck validate --root .` | Full validation suite | Any A-class check |
| audit-results | `hwcloud-skillcheck check audit-results` | Audit persistence guard | Gitignore/permission drift |
| aggregate trace | `hwcloud-skillcheck aggregate trace --require-traces` | Trace parsing path | Schema regression |
| alarm-wire | `hwcloud-skillcheck gcl alarm-wire` | Alarm wiring smoke | (allowed to fail) |
| learning gen | `hwcloud-skillcheck learning gen` | Knowledge base regeneration | Generation error |
| l4 handle | `hwcloud-skillcheck l4 handle --fault smoke` | L4 orchestrator smoke | Runtime error |
| drift sync | `hwcloud-skillcheck drift sync --dry-run` | Drift guard dry-run | Flag regression |
| drift check | `hwcloud-skillcheck drift sync --apply && drift check` | Canonical/runtime parity | Copy divergence |
| critic score | `hwcloud-skillcheck critic score` | Critic scorer smoke | (allowed to fail) |

### 4.2 Workflow: build-skillcheck.yml

**File**: `.github/workflows/build-skillcheck.yml`
**Triggers**: `push` to `main`/`master`, `pull_request`, tags `v*`

| Job | Depends On | Purpose |
|-----|-----------|---------|
| `test` | — | gofmt + vet + test (fast feedback) |
| `build` | `test` | Cross-platform matrix build (6 variants) |
| `release` | `build` | Publish to GitHub Releases (tag push only) |

---

## 5. Operations Checklist

### 5.1 Daily / Per-Change Checks

```bash
# 1. Run full validation
make all

# 2. Verify binary health
make self-check

# 3. Check skill-generator drift (dual-copy trap)
hwcloud-skillcheck drift check --root .

# 4. Validate GCL artifacts for all skills
hwcloud-skillcheck validate gcl-conformance --root .
```

### 5.2 Pre-Release Checklist

- [ ] `make all` passes (fmt + vet + test + build)
- [ ] `make self-check` passes (binary exercises embedded fixtures)
- [ ] `hwcloud-skillcheck validate --root .` — all A-class checks pass
- [ ] `hwcloud-skillcheck drift check --root .` — no canonical/runtime drift
- [ ] Working tree is clean (`git status`)
- [ ] CHANGELOG or release notes drafted

### 5.3 Post-Release Verification

- [ ] GitHub Release page shows 6 artifacts
- [ ] Download one platform binary and verify `--help`
- [ ] Run `hwcloud-skillcheck validate --root <external-repo>` on a clean machine

### 5.4 Troubleshooting

| Symptom | Likely Cause | Fix |
|---------|-------------|-----|
| `hcloud: command not found` | CLI not installed | Re-run install script from §1.2 |
| `credential not found` | Env vars not set | Check `HW_ACCESS_KEY_ID` / `HW_SECRET_ACCESS_KEY` |
| `validate: frontmatter FAIL` | SKILL.md YAML frontmatter broken | Check for dangling lines, missing fields |
| `drift check FAIL` | Canonical/runtime copy out of sync | Run `hwcloud-skillcheck drift sync --apply --root .` |
| `aggregate trace: no traces found` | Fresh checkout with no audit history | Run a GCL cycle first, or ignore if first run |
| Release artifacts missing | Tag not pushed, or CI failed | Check `build-skillcheck.yml` run in Actions tab |

---

## 6. Security Baseline

### 6.1 Credential Management

| Practice | Implementation |
|----------|---------------|
| Never commit credentials | `.env` in `.gitignore`; use `{{env.*}}` placeholders in skills |
| Use environment variables | `HW_ACCESS_KEY_ID`, `HW_SECRET_ACCESS_KEY` |
| Mask in output | All skill output masks credentials as `***` (enforced by `internal/gcl/sanitizer.go`) |
| Rotate regularly | IAM key rotation per organizational policy |

### 6.2 IAM Least Privilege

Each skill documents its required IAM permissions in `references/iam-permissions.md`.
For production deployments, grant only the permissions needed for the specific skills in use:

```bash
# Example: read-only ECS operations
hwcloud ecs list-servers      # needs ecs:servers:list
hwcloud ecs show-server       # needs ecs:servers:get

# Destructive operations require explicit confirmation
hwcloud ecs delete-server     # blocked by safety gate
```

### 6.3 Audit Trail

- All GCL execution traces are persisted to `audit-results/gcl-trace-*.json` (gitignored)
- Traces contain sanitized operation_intent (no raw credentials or PII)
- CES alarm plans are persisted as `audit-results/gcl-alarm-plan-*.json`
- The `check audit-results` gate verifies directory mode (0700) and gitignore coverage

### 6.4 Safety Gates

| Gate | Trigger | Action |
|------|---------|--------|
| Credential leak | Raw credential in output | `SAFETY_FAIL` — abort immediately |
| Destructive operation | `safety_class: destructive` | Human approval required |
| Trust tier < threshold | Low trust score on high-risk op | Escalate to human |
| Audit persistence | Missing audit-results/ contract | CI gate failure |

---

## 7. Reference

- `docs/manual/hwcloud-skillcheck.md` — CLI command reference
- `docs/gcl-spec.md` — GCL runtime specification
- `docker/README.md` — Docker sandbox details
- `.github/workflows/validate-skills.yml` — CI validation pipeline
- `.github/workflows/build-skillcheck.yml` — CI build + release pipeline
- `Makefile` — build targets reference
- `docs/superpowers/` — architecture plans and specs
