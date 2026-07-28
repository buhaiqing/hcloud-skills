# AGENTS.md Glossary Update — Spec

> Date: 2026-07-28
> Purpose: spec for the implementer who will apply this diff to `AGENTS.md` once the two worktrees land (ADR-0007 outcome memory + ADR-0008 context memory). The ADR-0009 trust-phase section is co-shipped here so the glossary is coherent at merge time.
> Sources: `docs/architecture/0007-…md`, `docs/architecture/0008-…md`, `docs/architecture/0009-…md`, `docs/superpowers/specs/outcome-memory-self-healing.md`.

## What to add

Three layers of content:

1. **Layer 1 (new terms)** — definitions + ADR path for every new type/function/concept the two worktrees introduce.
2. **Layer 2 (implementation notes)** — four ground-truth rules the code review must enforce. These are reviewer-facing, not user-facing.
3. **Layer 3 (open follow-ups)** — the four items the glossary should point at so reviewers don't reinvent them.

At the end of this file is a **diff** that shows what the AGENTS.md glossary section looks like after the change. Apply it as a single edit.

---

## Layer 1 — new terms

> Existing entries for `OutcomeRecord`, `OutcomeMemory.RecentOutcomes`, `MatchOutcomes`, `PruneOlderThan`, `HealingPolicy`, `HealingDecision`, `PreExecHook`, `PostFailureHook`, `DestructiveVerbs`, `Transient Pattern`, `ContextMemory`, `TaskSummary`, `ErrorSummary`, `Session Rotation`, `ContextMemory.RecordTask / RecordError / SetPreference / CloseTask` are already present in AGENTS.md (`## 术语表 (Glossary)` §Outcome Memory 与 Self-healing and §Cross-call Memory). **No definition rewrites are needed for those** — only the rows listed below are NEW.

### Outcome Memory 与 Self-healing (ADR-0007) — NEW rows

| 术语 | 定义 | 文档 |
|------|------|------|
| **`OutcomeMemory` (type)** | struct holding `path` (`<root>/.l4-memory/outcomes.jsonl`) + `sync.Mutex`. Constructed by `NewOutcomeMemory(root)` which calls `PruneOlderThan(now-90d)` before returning | ADR-0007 §Decision, spec §6 |
| **`OutcomeMemory.Record(r)`** | 追加一行 `OutcomeRecord` 到 JSONL。原子按行写入。调用方负责 `r` 字段填充（含 `id = uuidv4()`, `ts = NowISO()`） | ADR-0007 §Decision, spec §FR-1 |
| **Zero-value safety** | `OutcomeMemory=nil` 或 `HealingPolicy{}`（零值）时两个 hook 都返回 `proceed`。这是 spec 强制约束，不是约定 | ADR-0007 §Consequences, spec §FR-3 |

### Cross-call Memory (ADR-0008) — NEW rows

| 术语 | 定义 | 文档 |
|------|------|------|
| **`ContextMemory` (type)** | struct holding `path` (`<root>/.l4-memory/context.json`) + `sync.Mutex` + in-memory `Context` document。`Load()` / `save()` 内部化。`Load` 触发 Session Rotation 判定 | ADR-0008 §Decision |
| **`ContextMemory.Load()`** | 启动时调用一次。读 JSON，缺失字段取默认值；`created_at` 超过 24h 时 rotate session | ADR-0008 §Decision |
| **Schema versioning** | `schema="context-memory/v1"`。新字段一律带默认值；版本只升不降。无 migration machinery | ADR-0008 §Decision |
| **Atomic write** | 写策略：tmp 文件 + `os.Rename`。即使进程在写中途被 kill，旧文件完好 | ADR-0008 §Decision |

### Trust from Outcome Memory (ADR-0009) — NEW section

> 现有 AGENTS.md 没有这一节。该节需新增。

| 术语 | 定义 | 文档 |
|------|------|------|
| **Trust Phase 1 (coexist)** | 同一 release 中并存 `ComputeTrustScore([]OpHistory)`（curated）与 `ComputeTrustScoreFromOutcome([]OutcomeRecord)`（新增）。新调用点走 outcome-memory 路径 | ADR-0009 §Migration |
| **Trust Phase 2 (cutover)** | 默认新调用点走 outcome-memory 路径。配指标 `trust_source{from="outcome_memory"}` 观察切换 | ADR-0009 §Migration |
| **Trust Phase 3 (deprecate)** | `ComputeTrustScore([]OpHistory)` 标 deprecated。curator pipeline 转为 back-fill | ADR-0009 §Migration |
| **Trust Phase 4 (remove)** | 移除 curator pipeline。trust 单一来源 = outcome memory | ADR-0009 §Migration |
| **error_recovery weight (new formula)** | 旧式：curator 推断。新式：`count(OutcomeRecord.RetryCount > 0 AND Outcome == "success") / count(RetryCount > 0)`。curator 路径之前无法区分 | ADR-0009 §Compute algorithm |
| **trustCache** | 进程内 `map[skill]*TrustScore`。`Record()` 增量更新，不依赖后台扫描。cache key 含 policy hash 以应对 `HealingPolicy` 跨 invocation 变化 | ADR-0009 §Decision |
| **Outcome → trust inputs mapping** | `Outcome` → outcome（`blocked` 算失败）；`Timestamp` → ts；`Risk` → risk_level；`RetryCount > 0` → had_retry；`error_class` **不映射**（不惩罚 transient → recovered） | ADR-0009 §Data flow |

### 架构成熟度 (updated) — one new row

| 术语 | 定义 | 文档 |
|------|------|------|
| **L4 → L5 跃迁** | 下一阶段：trust score 由 outcome memory 实时驱动，curated `OpHistory` 退役。判据：`trust_source{from="outcome_memory"}` 持续 1 个 release | ADR-0009 |

### 其他常用术语 (updated) — one updated row

| 术语 | 定义 | 文档 |
|------|------|------|
| **Trust Score** | history-derived score（success_rate / consistency / recency / complexity_mastery / error_recovery，权重 0.35 / 0.20 / 0.20 / 0.15 / 0.10）。Phase 1 之后：从 outcome memory 读取；之前：从 curated `OpHistory` 读取 | `internal/l4/trust.go`, ADR-0009 |

---

## Layer 2 — implementation notes

> These four rules are written for the code reviewer. They are NOT user-facing glossary terms — they live in AGENTS.md only because reviewers cross-check the spec against the diff. Apply them as a single new subsection under `## 术语表 (Glossary)` titled `### 实现注意事项 (Implementation notes — reviewer-facing)`.

1. **`p.IsZero()` is the only zero-value gate.** `HealingPolicy{}` (zero value) must be safe — both `PreExecHook` and `PostFailureHook` short-circuit to `proceed` when `p.IsZero()`. A sum-based check (e.g. `if p.MaxRetries == 0 && p.RetryBackoff == 0`) is brittle: adding a new field to the struct silently breaks the gate. The struct's `IsZero()` method is the contract.
2. **`Executor` interface seam.** `internal/l4/execution.go` already accepts an `Executor` interface. In this iteration the only concrete type is `StubExecutor` (returns canned success; for unit tests). The real Hive CLI / subprocess binding lives in ADR-0010 — that ADR adds `RealExecutor` and the seam is already there. Do not new up `RealExecutor` in this branch; do not change the interface signature.
3. **`ExtractHighRiskVerbs()` is the single source of truth.** The destructive verb list (`delete / terminate / destroy / drop / remove`) is exported by a single helper `ExtractHighRiskVerbs()` (already present in `execution.go:226-230`). `HealingPolicy.DestructiveVerbs` defaults to a copy of that list at struct construction. Do not re-spell the list inside `PreExecHook` / `PostFailureHook` — call the helper. If the list grows, both call sites pick it up from one place.
4. **`Verb` field on `TaskStep`.** Destructive matching is `step.Verb` matched against `DestructiveVerbs` (prefix-match on the structured first token), **NOT** substring on `step.Action`. `Action` is the human-readable command string; `Verb` is the structured first token. This avoids `reboot-instances` matching `boot` as a false negative and `update-snapshot-delete` matching `update` as a false positive.

---

## Layer 3 — open follow-ups

> Apply as a new subsection `### Open follow-ups` under `## 术语表 (Glossary)`. Reviewers use this to know which items are intentionally deferred.

| # | Item | Why deferred |
|---|------|--------------|
| 1 | **ADR-0010: Real Executor + subprocess semantics** | Hive CLI binding + timeout policy + structured output capture. The `Executor` interface seam is already in place; ADR-0010 only adds the production implementation. |
| 2 | **L5 trajectory: trust score from outcome memory** | Driven by ADR-0009 phases. Phase 1 ships with this branch; Phases 2-4 are follow-up ADRs (~1 release cadence). |
| 3 | **Healing decision observability** | Separate ADR needed for: structured logs of `HealingDecision`, metric `healing_decisions_total{action,reason}`, and a `hwcloud-skillcheck memory inspect` subcommand. Today the decisions are on disk only. |
| 4 | **`EnsureMemoryDir()` shared helper** | Both `outcome_memory.go` (ADR-0007) and `context_memory.go` (ADR-0008) `mkdir .l4-memory` with mode 0700. Extract to a shared helper when a third caller appears (YAGNI until then). |

---

## Diff (old → new)

> Apply this diff to `AGENTS.md` lines 285–336. The diff is unified format; `---`/`+++` markers are omitted for legibility.

```diff
@@ 架构成熟度 (lines 289-294) @@
 | **L3 → L4 跃迁** | 本项目当前目标：从「等人类触发才执行」进化到「领域内自治 + 自愈」。判据：跨调用 outcome memory + healing hooks | ADR-0007 / ADR-0008 |
+| **L4 → L5 跃迁** | 下一阶段：trust score 由 outcome memory 实时驱动，curated `OpHistory` 退役。判据：`trust_source{from="outcome_memory"}` 持续 1 个 release | ADR-0009 |

@@ Outcome Memory 与 Self-healing (lines 296-311) — no body change, append 3 rows @@
 | **Transient Pattern** | `isTransient(errMsg)` 内部识别的子串集合（大小写不敏感）：`timeout / token expired / 401 / 429 / 503 / connection reset` |
+| **`OutcomeMemory` (type)** | struct holding `path` (`<root>/.l4-memory/outcomes.jsonl`) + `sync.Mutex`。构造时 `PruneOlderThan(now-90d)` | ADR-0007 §Decision |
+| **`OutcomeMemory.Record(r)`** | 追加一行 `OutcomeRecord` 到 JSONL。原子按行写入。调用方填 `id` (uuid v4) + `ts` (ISO-8601 UTC) | ADR-0007 §FR-1 |
+| **Zero-value safety** | `OutcomeMemory=nil` 或 `HealingPolicy{}`（零值）时两个 hook 必须返回 `proceed` | ADR-0007 §Consequences |

@@ NEW section after Cross-call Memory (after line 325) @@
+### Trust from Outcome Memory（ADR-0009）
+
+| 术语 | 定义 | 文档 |
+|------|------|------|
+| **Trust Phase 1 (coexist)** | 同 release 并存 `ComputeTrustScore([]OpHistory)` 与 `ComputeTrustScoreFromOutcome([]OutcomeRecord)`。新调用点走 outcome-memory 路径 | ADR-0009 §Migration |
+| **Trust Phase 2 (cutover)** | 默认新调用点走 outcome-memory 路径。配指标 `trust_source{from="outcome_memory"}` 监控切换 | ADR-0009 §Migration |
+| **Trust Phase 3 (deprecate)** | `ComputeTrustScore([]OpHistory)` 标 deprecated。curator pipeline 转为 back-fill | ADR-0009 §Migration |
+| **Trust Phase 4 (remove)** | 移除 curator pipeline。trust 单一来源 = outcome memory | ADR-0009 §Migration |
+| **error_recovery weight (new formula)** | 旧：curator 推断。新：`count(RetryCount > 0 AND Outcome == "success") / count(RetryCount > 0)` | ADR-0009 §Compute algorithm |
+| **trustCache** | 进程内 `map[skill]*TrustScore`。`Record()` 增量更新。cache key 含 policy hash | ADR-0009 §Decision |
+| **Outcome → trust inputs mapping** | `Outcome` → outcome（`blocked` 算失败）；`Timestamp` → ts；`Risk` → risk_level；`RetryCount > 0` → had_retry；`error_class` **不映射** | ADR-0009 §Data flow |

@@ Cross-call Memory (lines 313-325) — no body change, append 4 rows @@
 | **ContextMemory.CloseTask(taskID)** | 从 `open_tasks` 移除指定 taskID |
+| **`ContextMemory` (type)** | struct holding `path` (`<root>/.l4-memory/context.json`) + `sync.Mutex` + in-memory `Context` 文档 | ADR-0008 §Decision |
+| **`ContextMemory.Load()`** | 启动时调用一次。读 JSON；`created_at` 超过 24h 时 rotate session | ADR-0008 §Decision |
+| **Schema versioning** | `schema="context-memory/v1"`。新字段一律带默认值；版本只升不降。无 migration machinery | ADR-0008 §Decision |
+| **Atomic write** | tmp 文件 + `os.Rename`。进程在写中途被 kill 时旧文件完好 | ADR-0008 §Decision |

@@ 其他常用术语 (lines 327-336) — update Trust Score row, leave others unchanged @@
 | **Trust Score** | 历史 success_rate/consistency/recency/complexity_mastery/error_recovery 加权得分。映射到 L0_new ~ L4_autonomous。详见 `internal/l4/trust.go` |
-| **Trust Score** | history-derived score（success_rate / consistency / recency / complexity_mastery / error_recovery，权重 0.35 / 0.20 / 0.20 / 0.15 / 0.10）。Phase 1 之后：从 outcome memory 读取；之前：从 curated `OpHistory` 读取 | `internal/l4/trust.go`, ADR-0009 |

@@ NEW subsection at end of ## 术语表 (after line 336, before the `---` separator at line 337) @@
+### 实现注意事项 (Implementation notes — reviewer-facing)
+
+1. **`p.IsZero()` is the only zero-value gate.** `HealingPolicy{}` 零值必须安全 — 两个 hook 都 `p.IsZero()` → `proceed`。sum-based check 加新字段就 silently break。
+2. **`Executor` interface seam.** 本迭代只交付 `StubExecutor`（测试用）。真实 Hive CLI 绑定在 ADR-0010。不要 new `RealExecutor`，不要改 interface 签名。
+3. **`ExtractHighRiskVerbs()` is the single source of truth.** destructive verb 列表统一从一个 helper 取。`HealingPolicy.DestructiveVerbs` 默认值是该列表的拷贝。hook 内不要重写。
+4. **`Verb` field on `TaskStep`.** destructive 匹配走 `step.Verb`，不走 `step.Action` 的 substring。Action 是 command 字符串，Verb 是结构化的首 token。
+
+### Open follow-ups
+
+| # | Item | Why deferred |
+|---|------|--------------|
+| 1 | **ADR-0010: Real Executor + subprocess semantics** | Hive CLI 绑定 + timeout + 结构化输出捕获。Seam 已就位 |
+| 2 | **L5 trajectory: trust score from outcome memory** | ADR-0009 phases 驱动。Phase 1 同 branch 交付；2-4 后续 ADR |
+| 3 | **Healing decision observability** | 单独 ADR：结构化日志 + `healing_decisions_total{action,reason}` + `memory inspect` 子命令 |
+| 4 | **`EnsureMemoryDir()` shared helper** | ADR-0007 + ADR-0008 都 mkdir `.l4-memory` (0700)。等第三个 caller 出现再抽 (YAGNI) |
```

---

## Verification checklist for the implementer

Once the diff is applied:

- [ ] `grep -n "^### " AGENTS.md | grep -E "Trust from Outcome|实现注意事项|Open follow-ups"` returns 3 hits.
- [ ] The new L4 → L5 row is in §架构成熟度.
- [ ] The Trust Score row in §其他常用术语 is updated (not duplicated).
- [ ] No existing row is removed or rewritten — only additions.
- [ ] Total additions: 1 row in §架构成熟度, 3 rows in §Outcome Memory, 4 rows in §Cross-call Memory, 1 new section (7 rows), 1 updated row in §其他常用术语, 2 new subsections (Layer 2 + Layer 3).
- [ ] Two-round self-review on the diff (see `docs/rules/post-change.md`).
- [ ] `hwcloud-skillcheck validate --root .` passes — frontmatter references unchanged.
