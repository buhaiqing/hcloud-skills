# L4 闭环优化 Spec + Plan

> **Status:** 🗄 **SUPERSEDED** (2026-07-28)
> **Superseded by:** Go `hwcloud-skillcheck/internal/l4/` + ADR-0007…0011
> **Do not implement** the Python paths below (`scripts/runtime_orchestrator.py`,
> `critic_v1.py`, `progressive_trust.py`, `topology_graph.py`). They describe the
> pre-migration design.

## Supersession map

| This plan (Python) | Current source of truth |
|--------------------|-------------------------|
| Phase 1 knowledge base (RDS/VPC/ELB/CCE patterns) | `huaweicloud-*-ops/assets/failure_patterns.json` + `remediation-playbooks.json` (shipped) |
| Phase 2 `runtime_orchestrator.py` | `hwcloud-skillcheck l4 handle` → `internal/l4/orchestrator.go` (`HandleFault`) |
| Phase 3 `critic_v1.py` | `internal/gcl` structural critic + optional external Critic (`docs/gcl-spec.md`) |
| Phase 4 trust-state.json | ADR-0009 Phase 4 — trust single source = OutcomeMemory (no curator / OpHistory) |
| Phase 5 topology dynamic | `internal/l4/topology.go` + ADR-0011 cross-skill delegation |
| Phase 6 verify + CADL | `go test ./...`, `scripts/pre_commit_check.sh`, AGENTS.md §CADL |

Related ADRs: `docs/architecture/0007-*.md` … `0011-cross-skill-delegation-protocol.md`.
Living backlog: `TODOS.md` (open P0–P3 empty as of 2026-07-28).

---

## Historical text (frozen)

The sections below are retained for archaeology only. Status markers are not
maintained.

# L4 闭环优化 Spec + Plan (original)

> **目标**：从 L3+ 推进到完整 L4 运行成熟度
> **设计成熟度**：已是 L4（4 引擎 + 规范 + 验证全部就绪）
> **运行成熟度**：当时 L3（生产闭环未形成，仅 ECS 有完整自愈链）— **已由 Go L4 闭环取代**

### 1.1 目标（历史）

将 4 个独立 L4 引擎（dynamic_orchestration / predictive_ops / topology_graph / progressive_trust）
从"可独立运行的 CLI 工具"升级为"统一调度闭环中的协作模块"。

### 验收（历史 → 现状）

| 历史验收项 | 现状 |
|--------|------|
| `runtime_orchestrator.py handle --json` 六段输出 | `hwcloud-skillcheck l4 handle` → topology / predictive / orchestration / gcl / trust / learning |
| RDS/VPC/ELB/CCE knowledge | skill `assets/` 已落地 |
| Critic v1 | GCL Go critic path |
| Trust 持久化 | OutcomeMemory JSONL（ADR-0007/0009） |
| Topology 动态 | `DiscoverTransitiveSkills` + `ExpandMatchedWithDelegates`（T-3 / ADR-0011） |

每个 Phase 完成时同步更新此文档的 ✅/⏳ 状态。 → **停止更新**；改 ADR / TODOS / AGENTS 术语表。
