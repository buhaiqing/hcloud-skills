# Deployment & Operations Guide

> **Scope**: This guide covers the full lifecycle of deploying, operating, and maintaining an
> `hcloud-skills` environment — from first-time setup to production operations and CI/CD pipelines.
>
> **Audience**: Platform engineers, SREs, and ops teams operating Huawei Cloud skills.
>
> **Last updated**: 2026-07-28 (P2 end-of-cycle)

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

## 1.5 Router Policy Registry (P2, partial)

The Skill Router resolves its decision parameters from a versioned JSON registry
specified by the environment. The binary exposes **no setter** for these
parameters; changes only land via `hwcloud-skillcheck router calibrate --apply`,
which the runbook sandbox does not expose to operator UIs.

| Env var | Default | Purpose |
|---|---|---|
| `HC_CAPABILITY_REGISTRY` | `capability-registry.json` (cwd) | Path to the versioned policy file. Used by `Router.Route()` at every dispatch. |

**Defaults** shipped in the canonical file (rubric A2.14 pins them):

```json
{
  "router_policy_version": "v1.0.0",
  "confidence_gate": {
    "top1_score_min": 7500,
    "margin_min": 1500,
    "entity_match": ["strong"]
  }
}
```

**Pre-deployment check** (any environment lacking `HC_CAPABILITY_REGISTRY` will
fall back to these hard-coded defaults at runtime):

```bash
test -f "$HC_CAPABILITY_REGISTRY" &&   python3 -c "import json,sys; d=json.load(open('$HC_CAPABILITY_REGISTRY')); assert d['router_policy_version'].startswith('v') and d['confidence_gate']['top1_score_min']==7500"   && echo "capability-registry OK"
```

If `HC_CAPABILITY_REGISTRY` is unset, `Route()` reads `./capability-registry.json`
relative to the process CWD; in the Go binary's installed layout this resolves to
`/usr/local/bin/capability-registry.json`, which does not exist. Operators MUST
set the env var explicitly in production. The fallback exists for tests and
local development only.

Three embedding modes are mutually compatible: **local** (default
`local-fasttext`), **cloud** (`huaweicloud-modelarts`), and **off** (`none`).
`fallback_chain` is honoured at runtime: if the primary provider fails to
initialise, the Router walks the chain and records `fallback_used=true` with
the active provider name in the trace.


### Calibration workflow (offline only)

The `--apply` flag of `router calibrate` is the only path through which the
binary mutates `capability-registry.json`. Recommended operational sequence:

```bash
# Step 1: dry-run; review the plan, do NOT touch files
hwcloud-skillcheck router calibrate --root /etc/hcloud-skills     --source /var/log/hcloud-skills/audit-results/     --bump patch

# Step 2: re-run with --apply once the diff is approved
hwcloud-skillcheck router calibrate --root /etc/hcloud-skills     --source /var/log/hcloud-skills/audit-results/     --apply --bump patch

# Step 3 (rollback): revert to a previously-good version
hwcloud-skillcheck router calibrate --root /etc/hcloud-skills     --apply --rollback-to v1.0.0
```

The CLI exits 0 on success and writes the new policy to the same path
(`$HC_CAPABILITY_REGISTRY` if set, else `<root>/capability-registry.json`).
The trace-derived suggestions currently print only the source path; the
calibration algorithm that consumes traces ships in a follow-up revision.

**Hard rule (rubric A2.13 + S3)**: `--apply` is the ONLY way to mutate the
policy. Do NOT edit `capability-registry.json` by hand; bypassing the CLI
skips the version bump, breaks `router_policy_diff_at` audit, and invalidates
the `--rollback-to` semantics.

See `docs/superpowers/specs/2026-07-27-harness-runtime-p1p2-design.md` §4.2.1
for the policy versioning contract and §6.1 for the runtime-immutability
guarantees.

## 1.6 Embedding Sandbox Setup and Preflight

The Router uses a provider pattern, so local and cloud execution share one interface.
The **default is local process mode** (`local-fasttext`): it runs inside
`hwcloud-skillcheck`, requires no network, no model download and no credential. Cloud mode
is opt-in and never activates merely because cloud environment variables happen to exist.

Every provider has a `Preflight()` stage. It runs before model initialization or network
access, reports all configuration problems in one pass, and gives each problem a concrete
`Fix:`. Use it during installation, after a registry edit, and before restarting production:

```bash
export HC_CAPABILITY_REGISTRY="$(pwd)/hwcloud-skillcheck/capability-registry.json"
hwcloud-skillcheck router embed-test --root . --text "list ecs servers"
```

A healthy local result looks like:

```text
sandbox preflight: PASS
  provider: local-fasttext
embedding smoke test: PASS
  provider: local-fasttext
  vector_dim: 384
```

### Local process mode (recommended default)

```json
"embedding": {
  "mode": "local",
  "provider_name": "local-fasttext",
  "dim": 384,
  "timeout_ms": 500,
  "fallback_chain": ["local-fasttext"]
}
```

Security controls include context cancellation, a 64 KiB input cap, a per-process QPS cap,
panic containment, finite-value/L2 normalization, return-copy isolation, and no network
egress. Fields such as `endpoint`, `auth_env`, and `project_id` are unnecessary in local
mode; preflight reports them as warnings with removal instructions.

### Huawei Cloud ModelArts mode (explicit opt-in)

Store only the **environment variable name** in the registry. Never write the AK/SK or IAM
token itself to JSON:

```json
"embedding": {
  "mode": "cloud",
  "provider_name": "huaweicloud-modelarts",
  "endpoint": "https://modelarts.<region>.myhuaweicloud.com/v1/infers/<inference-id>",
  "auth_env": "HC_MODELARTS_AUTH",
  "project_id": "<project-id>",
  "dim": 384,
  "timeout_ms": 500,
  "fallback_chain": ["local-fasttext"],
  "extra": {"model_id": "<deployed-model-id>"}
}
```

Choose one credential form, then run preflight:

```bash
# IAM token
export HC_MODELARTS_AUTH='<iam-token>'

# Or AK/SK; the separator is a literal pipe
export HC_MODELARTS_AUTH='<access-key>|<secret-key>'

export HC_EMBED_PROVIDER=huaweicloud-modelarts
hwcloud-skillcheck router embed-test --root . --text "list ecs servers"
```

`HC_EMBED_PROVIDER` overrides only the provider for that process. The rest of the settings
still come from `HC_CAPABILITY_REGISTRY`. Remove the variable to return to registry-driven
selection. Provider changes take effect after restart; runtime requests cannot mutate them.

### Common preflight messages

| Message | Meaning | Fix |
|---|---|---|
| `endpoint is required` | Cloud provider has no inference URL | Add the deployed ModelArts HTTPS inference URL to `embedding.endpoint`. |
| `HTTPS is required` | Endpoint uses `http://` or lacks a scheme | Change it to a verified `https://` URL. |
| `auth_env is required` | Registry does not name a credential variable | Set `embedding.auth_env` to an env-var **name**, such as `HC_MODELARTS_AUTH`. |
| `env var ... is unset or empty` | The named credential is absent from the process | Export it before starting the service; verify service-manager environment propagation. |
| `contains whitespace control characters` | Credential includes a copied newline or tab | Re-export the credential without newline/tab characters. |
| `dim ... outside allowed range` | Vector size is outside 64–4096 | Use the deployed model's dimension; default is 384. |
| `timeout_ms ... exceeds ... 500ms` | Cloud call can violate the Stage-2 budget | Set `timeout_ms` to 500 or less. |
| `unknown embedding provider` | Typo or unavailable provider | Use `local-fasttext`, `huaweicloud-modelarts`, `none`, or provision the documented ONNX runtime first. |
| `embedding=none ... endpoint is unused` | No-sandbox mode selected but old cloud fields remain | Remove `endpoint`, `auth_env`, and `fallback_chain` from the registry. |
| `fallback_chain is ignored` | No-sandbox mode and fallback chain are both set | Drop the chain or pick a non-none primary. |

Preflight is configuration-only and does not prove cloud reachability. After it passes,
`router embed-test` performs one real embedding call; connection, 401, 404 and 429 failures
also include a remediation hint. For locked-down production networks, allow egress only to
the configured HTTPS host. If cloud is unavailable, switch the registry back to
`local-fasttext`; automatic fallback metadata is reserved for the runtime fallback path.

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
task all           # lint + test + build
task self-check    # verify binary against embedded fixtures

# Tag and push (CI will build and publish artifacts)
task release VERSION=0.2.0
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
task all

# 2. Verify binary health
task self-check

# 3. Check skill-generator drift (dual-copy trap)
hwcloud-skillcheck drift check --root .

# 4. Validate GCL artifacts for all skills
hwcloud-skillcheck validate gcl-conformance --root .
```

### 5.2 Pre-Release Checklist

- [ ] `task all` passes (lint + test + build)
- [ ] `task self-check` passes (binary exercises embedded fixtures)
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
- `Taskfile.yml` — build targets reference
- `docs/superpowers/` — architecture plans and specs


### 1.6.1 No-sandbox mode (`provider_name: none`)

The no-sandbox provider skips Stage-2 rerank entirely; the Router still emits
trace metadata so audits can reason about decisions made without an embedding
model. It is the explicit opt-out for:

- locked-down production environments without egress or local model files
- CI runs that must validate Stage-1 determinism in isolation
- air-gapped operator workstations verifying the binary shape

```json
"embedding": {
  "mode": "off",
  "provider_name": "none",
  "dim": 384
}
```

`router embed-test` reports `rerank_mode=skipped` and exits 0 after one call
that returns an empty vector; provider metadata is `none` with `fallback_used=false`.
Warnings surface for leftover cloud fields (endpoint, auth_env, fallback_chain)
so old configurations are visible during migration.
