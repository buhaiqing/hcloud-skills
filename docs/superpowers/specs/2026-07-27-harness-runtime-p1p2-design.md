# Spec: Harness Runtime v2 — P1 Trustworthy Evidence + P2 Efficiency

> Status: **DRAFT** — pending user review
> Last updated: 2026-07-27
> Build dependency: P0 trust boundary (commit `2b935ea`, `99996a2`, `deaf3b8` — done)

## 1. Background & Goals

The `hwcloud-skillcheck` Harness today loads every `huaweicloud-*-ops/SKILL.md` eagerly, runs the L4 Orchestrator on a hand-written fault, and produces a GCL trace per Generator invocation. P0 closed the trust boundary (Critic request leak, retry feedback, schema validation, destructive auto-detection, confirmation tokens). P0 alone is not enough: the system has no ground-truth evidence layer, no runtime resource budget, and no observable routing quality. This spec defines the next two layers — one for evidence (P1), one for efficiency (P2) — written as a single design because the P2 Router's ground truth is the P1 Golden oracle and the P1 Manifest is the P2 Router's index.

### Goals

- **P1**: every executable skill ships ≥5 golden scenarios, an E2E sandbox (mock hcloud CLI) and a CI gate that runs them. CLI stdout/stderr for every subcommand is locked as a fixture. Telemetry is partitioned by lane (self-test / sandbox / production) with no cross-lane writes. Per-skill `capability_manifest.json` is auto-generated; a per-skill maturity report rolls them up.
- **P2**: skill loading is frontmatter-only-first (no body read until the skill is selected). Router is two-stage (manifest filter → ONNX rerank). Each run has hard caps on context tokens, tool calls, and wall-clock. Intent confusion is captured in the trace and analysed offline. Local dev entry is a single `task` binary.

### Suggested metrics → spec acceptance

| # | Metric | Target | Data source | Measured by |
|---|---|---|---|---|
| M1 | Critic raw-request leak | 0 occurrences | P0 `SanitizeRequest` | `TestSanitizeRequest_FailClosedOn*` |
| M2 | Destructive Guardrail coverage | 100% of destructive commands | P0 `preExecutionGate` + l4 sync test | `TestHighRiskVerbsInSync` |
| M3 | Golden scenario coverage | every executable skill ≥5 | P1 golden fixtures | `golden run --root .` |
| M4 | Skill-change regression coverage | 100% of touched skills re-run on PR | P1 sandbox E2E + A/B | `ab compare` PR gate |
| M5 | Trace provenance completeness | 100% of fields populated | P0 `GCLTrace` schema required fields | `trace-schema` gate |
| M7 | Router Top-1 accuracy / misroute / fallback | continuously observable | P2 Router decision block in trace | `telemetry confusion` |
| M8 | P95 latency / mean tokens / tool calls / cost-per-task | continuously observable | P0 trace + P1 telemetry | CES `CUSTOM.GCL` |

M1–M2 are already satisfied by P0 commits; this spec freezes them under load (M3, M4, M6 are the P1 evidence surface) and instruments the rest (M7, M8 are the P2 observability surface).

## 2. Architecture

```
┌────────────────────────────────────────────────────────────────┐
│                    Harness Runtime v2                            │
│                                                                  │
│  ┌────────────────────────────────────────────────────────────┐ │
│  │  Skill Registry (P2)                                        │ │
│  │  - frontmatter index (loaded once at boot, O(1) lookup)     │ │
│  │  - lazy body loader (reads SKILL.md only when selected)     │ │
│  │  - references indexed by ID (loaded on demand)              │ │
│  └────────────────────┬───────────────────────────────────────┘ │
│                       │                                         │
│  ┌────────────────────▼───────────────────────────────────────┐ │
│  │  Router (P2)                                                │ │
│  │  - input: (user_request, op_intent)                        │ │
│  │  - stage 1: manifest filter (P1) — match by inputs/outputs/  │ │
│  │    side_effect_class/required_permissions                 │ │
│  │  - stage 2: ONNX rerank — top-k from stage 1,              │ │
│  │    score with local all-MiniLM-L6-v2 / bge-small-en-v1.5     │ │
│  │  - output: top-1 skill + score + 5-candidate matrix         │ │
│  │  - emits router_decision block at top of trace             │ │
│  └────────────────────┬───────────────────────────────────────┘ │
│                       │                                         │
│  ┌────────────────────▼───────────────────────────────────────┐ │
│  │  Pre-execution Gate (P0) + Resource Budgets (P2)            │ │
│  │  - P0 destructive detection (in sync with l4 via test)      │ │
│  │  - token budget: hard cap (per-run)                        │ │
│  │  - tool-call budget: hard cap (per-run)                    │ │
│  │  - wall-clock budget: hard cap (per-run)                   │ │
│  │  - over-budget → emit budget_exceeded → SAFETY_FAIL         │ │
│  └────────────────────┬───────────────────────────────────────┘ │
│                       │                                         │
│  ┌────────────────────▼───────────────────────────────────────┐ │
│  │  Executor + Critic (P0)                                     │ │
│  │  - Generator: hcloud CLI or mockhcloud (P1 sandbox)         │ │
│  │  - Critic: structural + LLM (P0)                            │ │
│  │  - per-iter: trace block (P0 schema, P1 sandbox lane)        │ │
│  └────────────────────┬───────────────────────────────────────┘ │
│                       │                                         │
│  ┌────────────────────▼───────────────────────────────────────┐ │
│  │  Telemetry Lanes (P1)                                       │ │
│  │  - self-test | sandbox | production — no cross-lane writes  │ │
│  │  - per-iter: route_score, gate_decision, budget, score     │ │
│  │  - router_decision + intent_confusion matrix (P2, derived)  │ │
│  └────────────────────────────────────────────────────────────┘ │
└────────────────────────────────────────────────────────────────┘
```

P0 trust boundary (Critic request leak / retry / pre-execution gate / confirmation) is unchanged; the P1 evidence layer writes into the P0 trace schema; P2 reads from the P1 Manifest and writes the router_decision block at the top of the same P0 trace.

## 3. P1 batch — Trustworthy Evidence Layer

### 3.1 Golden scenarios

**Contract**: every executable skill (i.e. every `huaweicloud-*-ops` except `huaweicloud-skill-generator`, which is meta) ships ≥5 golden scenarios. A scenario is one input/output pair for the skill's primary operation path.

**Where**: `internal/golden/testdata/<skill>/<scenario>.json`. Each fixture has:

```json
{
  "name": "ecs-list-servers-empty",
  "command": "hcloud ecs list-servers --region cn-north-4",
  "expected_stdout_excerpt": "[]",
  "expected_stderr_excerpt": "",
  "expected_exit_code": 0,
  "tags": ["read-only", "list"]
}
```

**Cross-product** scenarios live in `internal/golden/testdata/cross-product/<name>.json` and tag ≥2 skills (e.g. `huaweicloud-vpc-ops + huaweicloud-ecs-ops`). At least 8 cross-product scenarios are required.

**Per-skill + per-product** scope: each skill gets 3–5 in-skill scenarios + each Huawei Cloud product group (VPC, ECS, RDS, OBS, …) gets 1–2 cross-product scenarios. Total: ~60–100 fixtures.

### 3.2 CLI response fixtures + sandbox E2E

**Mock hcloud CLI** is a standalone Go binary at `internal/mockhcloud/` (sibling of `internal/embed/`, both consumed via `embed.FS`). It:

- reads `--script <path-to-fixture>` at startup
- matches each invocation's `<verb> <args>` against the script (longest prefix wins) and returns the scripted response
- logs every call to stdout as JSON
- exits 0 unless the script says otherwise
- **never** calls the real Huawei Cloud API; **never** reads credentials

The sandbox E2E test sets `HCLOUD_BIN=/path/to/mockhcloud` and runs `hcloud …` via the real `hcloud` binary which the mock shadows via PATH.

CLI response fixtures live at `internal/cli_fixtures/<cmd>.json` and lock the stdout/stderr of every subcommand (golden, lint, validate, …) for byte-level regression detection.

### 3.3 CI gates

Four gates become **required** in `scripts/pre_commit_check.sh` and `.github/workflows/validate-skills.yml`:

1. `golden run --root .` — every scenario passes
2. `ab compare --root . --old <git-ref>` — runs on PR; pass or have intentional-delta allowlist
3. `telemetry check --root .` — lane separation, self-test purity
4. `check advanced-coverage --root .` — TE-7 (existing, promoted to required)

The four gates form the A-class total entry (A1 = A1 here is the P1 evidence surface, A2 = A2 here is the P2 routing surface — see §6).

### 3.4 Telemetry lanes (self-test | sandbox | production)

Every event is tagged with `lane` at emission:

| Lane | When | Storage | Retention |
|---|---|---|---|
| `self-test` | inside `go test ./...` | `audit-results/self-test/` (gitignored) | 7 days |
| `sandbox` | mock-driven E2E | `audit-results/sandbox/` (gitignored) | 30 days |
| `production` | real runs (L4 / L5) | `audit-results/production/` (gitignored) | permanent |

Lane is determined by env var `HC_TELEMETRY_LANE` (default: `production`). Self-test sets it to `self-test`; mockhcloud runs set it to `sandbox`. A CI gate (`telemetry check`) asserts that no `self-test`-tagged events exist in `production/` paths (M6).

### 3.5 Capability Manifest + Maturity report

**Capability Manifest** is auto-generated per skill:

```json
{
  "schema_version": "1.0",
  "skill": "huaweicloud-ecs-ops",
  "name": "ECS Operations",
  "description": "Manage ECS instances, disks, images, snapshots on Huawei Cloud.",
  "version": "1.4.0",
  "inputs": [
    { "name": "server_id", "type": "string", "required": true, "sensitive": false },
    { "name": "action",    "type": "enum[list|get|start|stop|reboot|delete|resize]", "required": true }
  ],
  "outputs": [
    { "name": "stdout",     "type": "string" },
    { "name": "exit_code",  "type": "integer" }
  ],
  "side_effect_class": "destructive",
  "required_permissions": [
    "ecs:listServers", "ecs:getServer", "ecs:startServer", "ecs:stopServer",
    "ecs:rebootServer", "ecs:deleteServer", "ecs:resizeServer"
  ],
  "telemetry_emitted": [
    "gcl.trace.iteration", "gcl.critic.score", "l4.fault.handled"
  ],
  "maturity": {
    "golden_scenarios": 5,
    "te7_pass": true,
    "manifest_complete": true,
    "telemetry_lane_clean": true
  }
}
```

Generator: `hwcloud-skillcheck manifest gen --root . --out audit-results/sandbox/manifests/`.

Maturity report: `hwcloud-skillcheck maturity report --root .` rolls all manifests up to a per-skill score:

```
score = 0.3·golden_pass_rate + 0.3·te7_pass + 0.2·manifest_complete + 0.2·telemetry_lane_clean
```

## 4. P2 batch — Harness Efficiency

### 4.1 Skill Registry

**Boot cost**: read the frontmatter of every `huaweicloud-*-ops/SKILL.md` once. The full body, the `references/`, the `assets/` are **not** read until a skill is selected. Each SKILL.md body is loaded on first reference; references are loaded on first request by `{{ref:id}}`.

**Indexed fields** (load-time, O(1) lookup at runtime): `name`, `description`, `version`, `cli_applicability`, `side_effect_class_max` (max across all `operation_intent` samples), `required_permissions` (set), `telemetry_emitted` (set), `inputs` (list of `{name, type, required, sensitive}`), `outputs` (list of `{name, type}`).

### 4.2 Router (two-stage)

**Stage 1 — manifest filter**:

Inputs: `(user_request: string, op_intent: map[string]any)`.
- Drop every skill whose `side_effect_class_max` exceeds the requested
  `op_intent.safety_class`. Rationale: a read-only intent must not
  surface a destructive-only skill, even if its input shape matches.
- For each surviving skill, score on:
  - input shape match: how many of the request's typed tokens (typed by a tiny local intent tagger) match the skill's `inputs[*].name`?
  - required_permissions subset: does the caller's principal have every required permission? (defaults to "yes" in dev; uses l4 trust in prod)
  - tag overlap: cosine of the request embedding vs. the skill's `description` embedding (1.0 if identical, 0.0 if disjoint).
- Take top-k = 5.

**Stage 2 — ONNX rerank**:

- For each of the 5 candidates, embed the request + the skill's `description + inputs[*].name + operation_intent.example` and score.
- Re-rank by combined score `0.5·manifest_score + 0.5·onnx_cosine`.
- Return top-1 + the 5-candidate matrix for the trace.

**Embedding model**: ONNX-exported `bge-small-en-v1.5` or `all-MiniLM-L6-v2` (final choice in plan phase). The `.onnx` and `tokenizer.json` are pinned in `deps/embed/` with checksums. Inference: `github.com/yalue/onnxruntime_go` (pure Go, requires libonnxruntime shared library bundled in the Docker image).

**Trace block** at the top of every GCL trace:

```json
"router_decision": {
  "request": "delete the broken ECS",
  "op_intent": { "operation": "delete", "safety_class": "destructive" },
  "candidates": [
    { "skill": "huaweicloud-ecs-ops", "manifest_score": 0.82, "onnx_cosine": 0.91, "rank": 1 },
    { "skill": "huaweicloud-billing-ops", "manifest_score": 0.31, "onnx_cosine": 0.55, "rank": 2 }
  ],
  "chosen": "huaweicloud-ecs-ops",
  "fallback_used": false,
  "duration_ms": 12
}
```

### 4.3 Resource Budgets (all hard)

| Budget | Default | Override | Over-budget behavior |
|---|---|---|---|
| context tokens (in + out) | 200,000 | `--budget-tokens` | SAFETY_FAIL with `budget_exceeded=tokens` |
| tool calls | 50 | `--budget-tool-calls` | SAFETY_FAIL with `budget_exceeded=tool_calls` |
| wall-clock | 120s | `--budget-wall-clock` | SAFETY_FAIL with `budget_exceeded=wall_clock` |

All three are **hard** caps. Over-budget means the run ends immediately with SAFETY_FAIL and the trace carries the `budget_exceeded` reason. Soft caps are explicitly out of scope (would hide slow degradation; the user picks hard cap so l4 Orchestrator does not get stuck).

### 4.4 Intent Confusion Matrix

**Storage**: in-trace only. Every Router decision is recorded as `router_decision.candidates[].skill + .rank` in the trace. The Confusion Matrix is **derived**: the L4 Orchestrator's existing `aggregate trace` job reads trace files, builds the matrix `intended_skill × actual_top1`, and exposes the result via `hwcloud-skillcheck telemetry confusion --root .`.

**No new persistent storage**. Trace files are the only source of truth; the matrix is a view.

**Why in-trace, not in a side ledger**: P0 traces are already append-only, mode-0600, and gitignored. Adding a side ledger would double the write paths and create a consistency problem. The trace is the trace.

### 4.5 DevEx convergence

- **`taskfile.yml`** at the repo root. Common commands: `task test`, `task golden`, `task sandbox`, `task lint`, `task build`, `task deps:lock`.
- **Single dev entry**: `task dev` brings up the docker-compose stack + seeds `audit-results/self-test/` and `audit-results/sandbox/` directories.
- **Dependency lock**: `deps/embed/` contains pinned `.onnx` + `tokenizer.json` with `SHA256SUMS`. CI fails if any file's hash drifts from the lockfile.
- **`Makefile` is removed** in this spec; existing `make` users get a deprecation note in CHANGELOG. (`scripts/pre_commit_check.sh` keeps running in CI unchanged; the binary is the single source of truth — see AGENTS.md TE-6.)
- **Local dev entry is `task` only**. No second way in. README's Quick Start section shows `go install github.com/go-task/task/v3/cmd/task@latest` as the bootstrap.

## 5. Data Flow

```
Caller
  │
  ▼
[1] Skill Registry boots
      - reads frontmatter of every skill (fast, O(n_skills))
      - bodies untouched
  │
  ▼
[2] Router
      - input: (request, op_intent)
      - stage 1: manifest filter → top-5
      - stage 2: ONNX rerank → top-1
      - emits router_decision block at top of trace
  │
  ▼
[3] Pre-execution Gate (P0) + Resource Budgets (P2)
      - destructive auto-detection (in sync with l4)
      - token / tool-call / wall-clock caps
      - over-budget → SAFETY_FAIL (emit budget_exceeded reason)
  │
  ▼
[4] Executor + Critic (P0)
      - Generator: hcloud CLI or mockhcloud
      - Critic: structural + LLM
      - per-iter: trace block (P0 schema, P1 sandbox lane)
  │
  ▼
[5] Telemetry Lanes (P1)
      - per-iter: route_score, gate_decision, budget, score
      - router_decision + intent_confusion matrix (P2, derived)
```

## 6. Acceptance Criteria (DoD)

P1 + P2 ship in two separate plans. P1 must be complete and verified before P2 plan begins (P2 Router depends on P1 Manifest). Each acceptance criterion below maps to either A1 (P1 evidence) or A2 (P2 efficiency). A spec-level "all green" is required before either plan can start; P1 plan verifies A1 criteria, P2 plan verifies A2 criteria.

### A1 — P1 evidence criteria

| # | Criterion | Test |
|---|---|---|
| A1.1 | Every executable skill has ≥5 golden scenarios | `TestGoldenScenarioCoverage` |
| A1.2 | ≥8 cross-product scenarios exist | `TestCrossProductScenarioCount` |
| A1.3 | Every CLI subcommand has a fixture | `TestCLISubcommandFixtureCoverage` |
| A1.4 | mockhcloud never opens a real network socket | `TestMockhcloudNoNetwork` |
| A1.5 | `golden run` exits 0 on green | `TestGoldenRunPass` |
| A1.6 | `ab compare` detects a deliberate stdout diff | `TestABDetectsStdoutDiff` |
| A1.7 | `telemetry check` fails when self-test events leak into `production/` | `TestTelemetryLaneSeparation` |
| A1.8 | `manifest gen` produces a valid manifest for every skill | `TestManifestGeneration` |
| A1.9 | `maturity report` rolls up per-skill scores | `TestMaturityReportRollup` |
| A1.10 | All four P1 gates are listed in pre_commit_check.sh and CI | `TestP1GatesWired` |

### A2 — P2 efficiency criteria

| # | Criterion | Test |
|---|---|---|
| A2.1 | Registry boot reads only frontmatter, not body | `TestRegistryBootDoesNotReadBody` |
| A2.2 | Router stage 1 returns top-5 within 10ms on a 20-skill repo | `TestRouterManifestFilterLatency` |
| A2.3 | Router stage 2 (ONNX rerank) returns top-1 within 50ms | `TestRouterONNXRerankLatency` |
| A2.4 | Router emits `router_decision` block with all 5 candidates | `TestRouterEmitsDecisionBlock` |
| A2.5 | Token budget: over-budget → SAFETY_FAIL with `budget_exceeded=tokens` | `TestBudgetTokensHard` |
| A2.6 | Tool-call budget: over-budget → SAFETY_FAIL with `budget_exceeded=tool_calls` | `TestBudgetToolCallsHard` |
| A2.7 | Wall-clock budget: over-budget → SAFETY_FAIL with `budget_exceeded=wall_clock` | `TestBudgetWallClockHard` |
| A2.8 | Confusion matrix derivable from trace files | `TestConfusionMatrixDerivable` |
| A2.9 | `task` is the single dev entry; `Makefile` removed | `TestDevExSingleEntry` |
| A2.10 | `deps/embed/SHA256SUMS` covers the ONNX model + tokenizer | `TestDependencyLock` |

### M-metrics acceptance

- M1: `TestSanitizeRequest_FailClosedOn*` is green in CI (already shipped via P0).
- M2: `TestHighRiskVerbsInSync` + `TestHighRiskCommandsInSync` are green (already shipped via P0).
- M3: `TestGoldenScenarioCoverage` passes with per-skill count ≥5.
- M4: `ab compare` PR gate is green; "no skill changed but no golden re-run" is a build failure.
- M5: `trace-schema` gate fails on a trace missing any required field (M5 is enforced by the same gate as A1.5's reporter).
- M6: `TestTelemetryLaneSeparation` is green; the gate fails on cross-lane writes.
- M7: `telemetry confusion --root .` exits 0 and the report shows non-empty `top1_vs_intended` rows (continuous; no test gate, just an operational dashboard).
- M8: CES `CUSTOM.GCL` namespace exports `p95_latency_ms`, `mean_tokens`, `mean_tool_calls`, `cost_per_task_usd` (continuous; no test gate).

## 7. Risks & Trade-offs

| Risk | Mitigation |
|---|---|
| ONNX embedding is a new runtime dep; Docker image grows ~30MB | pin to a single ONNX model with a checksum; lazy-load the model in `Registry.Boot`; allow `--no-rerank` flag that skips stage 2 and falls back to stage 1 (degraded but deterministic) |
| Manifest generation needs to keep up with skill authors | `manifest gen` is a CLI that any CI run can call; the maturity score surfaces drift; `manifest_complete` field gives a low-friction "this skill needs re-gen" signal |
| Hard caps on token budget can spike false-positive SAFETY_FAILs for legitimate long tasks | Default 200,000 tokens is generous; per-skill override via `huaweicloud-*-ops/assets/example-config.yaml` lets skill authors declare a higher cap for their known-heavy flows |
| `task` is one more binary to install | README's Quick Start gets a one-liner; CI installs via `go install …@latest`; the cost of a single Go binary is dwarfed by the cost of a fragmented Makefile + scripts/ surface |
| Two-stage Router adds latency (10 + 50 = 60ms) | both stages are local + cached; P95 budget is < 100ms total; the latency is dwarfed by the Generator's network call |
| Embedding drift: ONNX model version mismatch | `SHA256SUMS` lockfile + CI step that fails on hash drift; offline pre-embed step `task deps:lock` |
| Cross-product scenarios require multi-skill fixtures | the `huaweicloud-skill-generator` becomes the canonical author of cross-product fixtures; CI lint enforces every cross-product scenario tags ≥2 skills |
| Telemetry lane mix-up | env var is the single source of truth; `telemetry check` CI gate fails on cross-lane writes; tests run with `HC_TELEMETRY_LANE=self-test` baked in |

## 8. Out of Scope

- Multi-agent GCL fan-out (L5 P3).
- Persistent confirmation ledger (P0 already out of scope; P1 doesn't introduce one).
- Auto-remediation closed-loop (L5).
- Persistent telemetry backend (CES export is enough; no InfluxDB / ClickHouse).
- Embedding model fine-tuning for Huawei Cloud domain — pinned to a stock model.
- Online learning of router weights (offline-only confusion matrix).

## 9. Dependencies

- **P0 trust boundary** (commits `2b935ea`, `99996a2`, `deaf3b8`) — done.
- **`github.com/yalue/onnxruntime_go`** — new; pure Go binding to libonnxruntime. Pinned to a specific version in go.mod.
- **`libonnxruntime`** — shared library, bundled in the Docker image. CI's `validate-skills.yml` uses the prebuilt Docker image; local dev installs via `task dev`.
- **`github.com/go-task/task/v3/cmd/task@latest`** — dev-only; not a runtime dep.
- **No Python deps**. The all-MiniLM / bge-small ONNX export keeps the toolchain Go-only.

## 10. Changelog

| Version | Date | Change |
|---|---|---|
| 0.1.0 | 2026-07-27 | Initial P1+P2 design — single spec, two-plan split (P1 evidence → P2 efficiency). Pins P0 metrics M1/M2; instruments M3–M8. |
