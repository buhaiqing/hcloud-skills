# ADR-0010: Real Executor + Subprocess Semantics

- Status: Proposed (2026-07-28)
- Deciders: hcloud-skills maintainers
- Supersedes: —
- Related: ADR-0007 (Outcome Memory + Self-healing), `internal/l4/execution.go`

## Context

ADR-0007's outcome-memory plan introduced an `Executor` interface seam in `RunExecutionLoop` so tests can drive `PostFailureHook` through a retry path via `StubExecutor`. Real subprocess execution (`os/exec.CommandContext`) was deferred to a follow-up because it carries its own design surface:

1. **Timeout policy**: per-step timeout vs overall task timeout.
2. **Environment handling**: passthrough `os.Environ()` vs sanitized secrets.
3. **Output capture**: stdout/stderr buffering vs streaming; cap size to prevent OOM on misbehaving commands.
4. **Signal propagation**: SIGINT → SIGTERM to child on cancel.
5. **Exit code mapping**: distinguish shell exit-1 from `exec.LookPath`-not-found from context deadline.

Without a `RealExecutor`, `RunExecutionLoop` sets `result.Success` purely from the GCL structural critic's verdict on a dry-run payload. There is no `exec.Command`, no exit-code capture, no real failure mode. Self-healing is decorative.

## Decision

Add `RealExecutor` in `hwcloud-skillcheck/internal/l4/execution.go` (alongside the existing `StubExecutor`):

```go
type RealExecutor struct {
    Env      []string        // os.Environ() by default; override for tests
    Timeout  time.Duration   // default 60s; configurable per call
    MaxBytes int             // stdout/stderr cap; default 1<<20 = 1 MB
}

func NewRealExecutor() *RealExecutor {
    return &RealExecutor{
        Env:      os.Environ(),
        Timeout:  60 * time.Second,
        MaxBytes: 1 << 20,
    }
}

func (r *RealExecutor) Run(candidate string, timeout time.Duration) (int, string, error) {
    // 1. ctx, cancel := context.WithTimeout(context.Background(), effectiveTimeout)
    // 2. cmd := exec.CommandContext(ctx, "bash", "-c", candidate)
    // 3. cmd.Env = r.Env
    // 4. var stdout, stderr bytes.Buffer; stdout.Limit = r.MaxBytes; stderr.Limit = r.MaxBytes
    // 5. cmd.Stdout = &stdout; cmd.Stderr = &stderr
    // 6. err := cmd.Run()
    // 7. On *exec.ExitError → exitCode = exitErr.ExitCode()
    // 8. On ctx.Err() == context.DeadlineExceeded → return (0, output, ctx.Err())
    // 9. Return (exitCode, stdout+stderr, err)
}
```

Wire it as the default in `RunExecutionLoop`:

```go
func RunExecutionLoop(...) *TaskState {
    executor := exec  // parameter; nil → NewRealExecutor()
    if executor == nil {
        executor = NewRealExecutor()
    }
    // ... per step: exitCode, output, execErr := executor.Run(candidate, 0)
    // result.Success = execErr == nil && exitCode == 0
    // result.ExitCode = exitCode
    // result.Output = output
    // result.Error = errorString(execErr)
}
```

### Why `bash -c`

`candidate` is the full shell command string (e.g. `hcloud ecs delete-instances --instance-ids i-abc123`). Most users expect shell semantics (pipes, globs, env-var expansion). `bash -c "$candidate"` preserves that, at the cost of one extra fork. Risk: shell injection from user-supplied candidates is already mitigated upstream by the RBAC gate and the destructive-verb gate (ADR-0007); bash here is a convenience, not a security boundary.

### Why default timeout 60s

Empirically, `hcloud` CLI invocations complete in <5s for most operations. 60s is the upper bound that catches infinite hangs without false-positive slow paths. Per-call override via the `Run(candidate, timeout)` second arg for special cases (snapshot deletion can be 2+ minutes).

### Why cap output at 1 MB

`bytes.Buffer.Limit` (Go 1.20+) prevents OOM when a command emits a multi-GB response (e.g., `list-instances` against a noisy project). 1 MB is enough for any single step's diagnostic data; logs larger than this should go to a file, not stdout. The cap is **per stream** (stdout and stderr are capped independently).

### Why env passthrough

The orchestrator runs `hcloud`, which needs `HW_ACCESS_KEY_ID`, `HW_SECRET_ACCESS_KEY`, `HW_REGION_ID`, `HW_PROJECT_ID`. We don't know which subset. Stripping these would break the tool. Sanitizing them into env would require knowing what to redact — out of scope. Pass through; secrets stay on the developer's machine. **Future work**: a `--secrets-redact` flag for the `validate` subcommand, separate ADR.

### Signal propagation

`exec.CommandContext` cancels the child via SIGKILL by default. We preserve that. A future "graceful" mode would SIGTERM-then-SIGKILL; not needed for batch operations.

## Consequences

### Positive

- **Self-healing becomes real.** Transient errors now flow through `PostFailureHook` with `Success: false` triggering `retry`.
- **Outcome memory records actual outcomes.** No more "fake success" from a dry-run critic.
- **Trust scoring has ground truth.** `OpHistory.Outcome = "failure"` now reflects real failures, not GCL-predicted ones.
- **Test seam preserved.** `StubExecutor` remains for fast unit tests; `RealExecutor` covers the integration path.

### Negative

- **Sandbox required.** Without `os/exec`, the orchestrator was a static analyzer. With it, executing the candidate runs `hcloud` for real. CI must run in a sandbox or with mock credentials.
- **No cross-platform signal handling.** `bash -c` is Unix-only; Windows would need `cmd.exe /c`. Out of scope for v1; the Windows binary builds but executor invocation is best-effort no-op.
- **Output cap may truncate useful logs.** A failed `list-instances` against a 100k-record project might emit >1 MB. Operators would have to use `--raw-output` flag (future work).

### Neutral

- New `RealExecutor` adds ~50 LOC.
- One new package `internal/l4/metrics.go` is not needed here (metrics live with the observability ADR).

## Alternatives Considered

1. **Pure-Go subprocess via `os.Exec` directly, no bash wrapper.** Rejected: forces candidates to be single argv forms, loses shell semantics users expect.
2. **Stream output to file rather than buffer.** Rejected: complicates `StepResult.Output` semantics; 1 MB cap covers the diagnostic case.
3. **Use Docker-in-Docker sandbox for execution.** Rejected: too heavy for the orchestrator's role; sandboxing is a deployment concern, not a CLI concern.
4. **Skip real execution entirely; trust GCL critic.** Rejected: self-healing becomes decorative.

## Rollout

1. Implement `RealExecutor` (this ADR) — already gated by `Executor` interface seam.
2. Plumb `executor Executor` parameter through `RunExecutionLoop` (default: `NewRealExecutor()`).
3. Update `TestRunExecutionLoop_*` to use `StubExecutor` (no real `hcloud` calls in CI).
4. Add `execution_real_test.go` with cases: success / failure / timeout / output truncation / env passthrough.
5. Document in README that `hwcloud-skillcheck` now actually executes (and warn re: credentials).

## Open Questions

- **Multi-process invocation**: should a future version support running on a remote `hcloud` host via SSH? Not now.
- **Sandboxing**: should `RealExecutor` integrate with `bubblewrap` or `nspawn`? Not now.
- **Async execution**: should long-running ops (e.g., snapshot delete) be async with a polling UI? Not now.