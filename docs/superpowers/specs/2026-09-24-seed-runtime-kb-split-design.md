# Spec — Seed / Runtime Knowledge-Base Split (fix P0-3)

- Date: 2026-09-24
- Status: approved-by-user (Q2 decision "拆成两个文件，双权威" + "fix it, use gcl")
- Closes: findings.md P0-3
- Anchors: #T1..#T7 (tests reference these)

## Problem Statement

`hwcloud-skillcheck learning gen` is a **hard pre-commit/CI gate** (`check_precommit.go` gate #6) yet it *writes* files. It overwrites the runtime knowledge state:

1. `assets/failure_patterns.json` — gitignored runtime artifact (`.gitignore:241-242`, commit `7eaed41 "untrack failure_patterns.json (runtime-generated)"`). For the 4 `Products` skills (rds/vpc/elb/cce) every Go commit rewrites it to the static seed: `source_traces_analyzed` back to 0, all `learned_from` wiped.
2. `assets/remediation-playbooks.json` — **tracked**; gen rewrites the 4 Products entries and drops `metadata` (incl. `success_rate`), which runtime `RecordPlaybookOutcome` is supposed to accumulate.

Seed definitions and runtime observations share the same files with no authority split, so the generator (regenerated on every gate run) silently wins over learned state. Bonus race: the gate stage runs `learning gen` and `l4 handle` **concurrently** while the latter raw-reads `failure_patterns.json`.

## Solution

Split into two on-disk artifacts per skill with explicit dual authority; loaders merge at read time; the gate becomes check-only.

```text
seed (tracked, generated, canonical)     overlay (runtime, append-only)
failure_patterns.seed.json      ←gen─     failure_patterns.json  (gitignored)
remediation-playbooks.seed.json ←gen─     remediation-playbooks.json (tracked)
        \                                        /
         ------ LoadFailurePatterns / LoadPlaybooks (merge) ------
                              ↓
                    consumers (aggregate, pre-risk gate, autofix, pitfall report)
```

## User Stories

1. As an operator, I run `check --pre-commit` and trust that no gate mutates my working tree or destroys learned counters.
2. As the learner, `learning trace aggregate` persists observations that survive any number of generator runs.
3. As a fresh-clone CI job, curated seed definitions are present (tracked) without any generation step.
4. As a consumer, I cannot tell whether definitions came from seed, overlay, or both — one merged view.

## Implementation Decisions

- **#T1 Seed contract** — `failure_patterns.seed.json` carries `$schema/skill_id/patterns/meta{total_patterns}` only (definitions; no `last_aggregation`, no `source_traces_analyzed`). `remediation-playbooks.seed.json` carries the current generated shape `id/name/trigger/diagnosis/remediation` (no `metadata`, no `escalation`). Byte output of the generator stays structurally identical to today minus runtime keys.
- **#T2 Overlay contract** — existing filenames stay authoritative for runtime state. `failure_patterns.json` remains gitignored (no .gitignore change; the exact-basename patterns do not match `.seed.json`). `remediation-playbooks.json` stays tracked: seed skills reduce to a metadata overlay (`playbooks: []` initially); skills with **no seed file** (22 non-Products skills) keep their current full document — overlay-as-full-doc is the documented backward-compatible mode.
- **#T3 Merge (read)** — seed absent → today's behavior exactly (overlay/scaffold is the document). Seed present → per-ID definition fields from seed, `stats`+`learned_from` (patterns) / `metadata` (playbooks) from overlay when present; overlay-only IDs appended whole (trace-derived, `provenance=trace`). `meta`: overlay meta preferred, else scaffold; `meta.total_patterns` recomputed to the merged length.
- **#T4 Split (save)** — `SaveFailurePatterns` emits, for seed IDs, only `{id, stats?, learned_from?}`; non-seed IDs whole; writes the overlay file only; sets `meta.last_aggregation=now`. Seed files are never written by any path except the generator. `RecordPlaybookOutcome` updates the overlay in place (structure-preserving); if the ID exists in seed but not overlay it appends `{id, metadata}`; unknown ID stays a graceful no-op.
- **#T5 Generator** — `learning gen` writes **only** the two seed files. New `--check` renders in-memory and byte-compares against disk: when the repo marker (`docs/gcl-spec.md`) is present all 4 Products seeds must exist and match (drift/missing → exit 1, no writes); marker absent → vacuous pass (keeps state-tolerant gate semantics on empty roots). The pre-commit gate calls `gen --check`.
- **#T6 Consumers** — `GeneratePitfallReport` and `internal/l4.readFailurePatternsForSkill` (pre-execution risk gate) stop raw-reading and go through `LoadFailurePatterns`. Import direction `internal/l4 → internal/learning` is verified safe (learning imports only stdlib + `internal/embed` + `internal/schema`; grep shows no reverse import). This also removes the gen-writes / l4-reads race in gate stage 2.
- **#T7 Migration** — run `learning gen` to create the 4 seed pairs; reduce the 4 tracked `remediation-playbooks.json` to `{schema, skill_id, playbooks: []}`; gitignored `failure_patterns.json` files need no migration (merge ignores their duplicated definitions and the first Save normalizes them). Fresh clone = seeds only; loader scaffolds overlay on demand.

## Testing Decisions

- Generator writes seeds and provably does **not** touch either overlay filename (P0-3 regression test, #T5).
- `gen --check`: match → 0; seed drift → 1 + name; seed missing (marker present) → 1; no marker → 0; never writes (#T5).
- Merge: seed defs + overlay stats win; overlay-only appended; seed missing = byte-compat with today (#T3).
- Save: seed IDs reduced to stats/learned_from; non-seed full; `last_aggregation` bumped; seed file untouched (#T4).
- Playbook outcome: seed-skill appends partial metadata; non-seed in-place update unchanged; unknown ID no-op (#T4).
- Pitfall report counts seed definitions through merge (#T6); l4 pre-risk gate finds seed patterns when overlay absent (fresh-clone shape) (#T6).
- Existing suites stay green unchanged where they model the no-seed (22-skill) world.

## Out of Scope (explicit)

- P0-1 `--require-evidence` gate wiring (Q5, sequenced after first real evidence).
- P0-2 autofix selection/RBAC/destructive-gate redesign.
- P2-8 fail-open malformed-file handling.
- Untracking `remediation-playbooks.json` or moving runtime `success_rate` out of the tracked file (follow-up; current split stops generator clobber, which is P0-3).
- success_rate bootstrap paradox (0.0 blocks autofix forever) — feeds P0-2 work.

## Further Notes

- Doc-contract anchors in `references/self-healing-spec.md` (`### 5.1`–`### 5.4`, `source_traces_analyzed`) are untouched by this change; `validate doc-contracts` must stay green.
- Authority one-liner: **seed = 定义基线（review 进），overlay = 运行时证据（程序写），同 ID 定义取 seed、统计取 overlay。**
