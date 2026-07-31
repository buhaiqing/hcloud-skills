# ADR-0012: Agentic AI Maturity Assessment

- Status: Accepted
- Date: 2026-07-31
- Deciders: hcloud-skills maintainers
- Supersedes: —
- Related: ADR-0007 (Outcome Memory), ADR-0008 (Context Memory), ADR-0009 (Trust),
  ADR-0010 (Real Executor), ADR-0011 (Cross-Skill Delegation),
  `docs/gcl-spec.md`, `internal/l4/`, `internal/gcl/`, `AGENTS.md` (glossary L3→L5)

## Context

Stakeholders asked where this repo sits on an Agentic-AI maturity scale. The repo is
agent *tooling / runbook infrastructure* (the `hwcloud-skillcheck` Go engine + Markdown
skills), i.e. the machinery an executing agent consumes — not a live autonomous agent
itself. Nevertheless its capabilities can be graded against a maturity model to show
what is built and what remains deliberately gated.

**Framing caveat (honesty):** Gartner's *Emerging Tech: Agentic AI Maturity Roadmap*
(doc 6825634, 2025-08) specific level definitions are paywalled; public content only
gives the directional arc **today's agents → advanced agents → expert agents**. The
concrete L1–L4 grading below uses the widely-cited industry L1–L4 ladder as a ruler,
with every claim tied to repo evidence (`file:line`). It is NOT verbatim Gartner text.

## Decision

### Maturity verdict

**L3 (collaborative: multi-agent + tools + memory) — achieved and solid.**
**L4 (autonomous: self-evolving + expert decisions + continuous learning) — framework
mostly ready, but intentionally braked by safety HITL; not unconditional L4.**

Maps to Gartner arc: mid-transition from "today's agents" toward "advanced agents".

### Evidence by dimension

| Dimension | Evidence | Location |
|-----------|----------|----------|
| Multi-agent orchestration | L4 Orchestrator closes the loop over RBAC + topology + trust + healing + executor | `internal/l4/orchestrator.go` |
| Autonomy / self-healing | Self-healing hooks; append-only outcome memory; atomic context memory | `self_healing.go`, `outcome_memory.go:38`, `context_memory.go:23` |
| Quality gating / Critic | GCL Generator-Critic-Loop + automated **external** Critic (subprocess JSON) | `internal/gcl/critic.go:18-29`, `docs/gcl-spec.md` |
| Memory / learning | Per-skill `failure_patterns.json` + `remediation-playbooks.json`; trust scoring; `learning trace aggregate` | `assets/*`, ADR-0009 |
| Human-in-the-loop / escalation | `SAFETY_FAIL` abort (no partial output); `halt` decision; destructive-op confirmation nonce; critical playbooks MUST NOT auto-execute | `execution.go:262`, `orchestrator.go:357`, `gcl/confirmation.go:1-15` |
| Tool use / cross-skill delegation | `RealExecutor` shells hcloud CLI via `os/exec`; delegation graph BFS over `DelegatesTo` | `execution.go:65`, `orchestration.go:18` |
| Internal maturity self-label | Explicit L3→L4→L5, Trust Phase 1–4 references | `AGENTS.md:309-312`, ADR-0007~0011 |

### Why L4 is "braked, not absent"

The repo deliberately retains human gates:
- Confirmation store is in-process only (`gcl/confirmation.go:11`) — no durable
  cross-session approval.
- `risk_level: critical` playbooks are forbidden from auto-execution (enforced in
  `AGENTS.md` + `internal/l4/self_healing.go`).

These are **safety design choices**, not capability gaps. They prevent the repo from
claiming "unattended end-to-end autonomy," which is correct for an ops tool touching
production cloud resources.

## Consequences

- **Accurate positioning:** we can state L3-achieved / L4-framework-ready without
  overclaiming "expert agent" status.
- **Roadmap signal:** moving toward true L4 means (a) durable confirmation store,
  (b) safe automated execution for non-critical playbooks, (c) richer trust-phase
  coverage — each a separate ADR when pursued.
- **Honesty guard:** do not cite Gartner level names as if verbatim; the directional
  framing is Gartner's, the L1–L4 ladder is industry-common interpretation.

## Alternatives considered

- *Claim full L4 now:* rejected — contradicts the explicit HITL brakes and would be
  an overstatement.
- *Grade only against Gartner's 3-stage arc:* rejected as too coarse to be useful for
  internal roadmap; the L1–L4 ladder adds actionable granularity while the caveat above
  preserves honesty.
