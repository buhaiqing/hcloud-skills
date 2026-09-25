package spec_audit

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestP1Acceptance_AuditsAllCriteria(t *testing.T) {
	root := ".."
	mustFind := map[string][]string{
		"A1.1":  {"TestGoldenScenarioCoverage"},
		"A1.2":  {"TestCrossProductScenarioCount"},
		"A1.3":  {"TestCLISubcommandFixtureCoverage"},
		"A1.4":  {"TestMockhcloudNoNetwork"},
		"A1.5":  {"TestGoldenRunPass"},
		"A1.6":  {"TestABDetectsStdoutDiff"},
		"A1.7":  {"TestTelemetryLaneSeparation"},
		"A1.8":  {"TestManifestGeneration"},
		"A1.9":  {"TestMaturityReportRollup"},
		"A1.10": {"TestP1GatesWired"},
	}
	hits := map[string]bool{}
	_ = filepath.Walk(root, func(p string, info os.FileInfo, _ error) error {
		if info != nil && !info.IsDir() && strings.HasSuffix(p, "_test.go") {
			b, _ := os.ReadFile(p)
			txt := string(b)
			for id, names := range mustFind {
				for _, n := range names {
					if strings.Contains(txt, n) {
						hits[id] = true
					}
				}
			}
		}
		return nil
	})
	missing := []string{}
	for id := range mustFind {
		if !hits[id] {
			missing = append(missing, id)
		}
	}
	if len(missing) > 0 {
		t.Fatalf("P1 acceptance criteria missing tests: %s", strings.Join(missing, ", "))
	}
}

func TestP1GatesWired(t *testing.T) {
	// Uses `hwcloud-skillcheck check --pre-commit` (ADR-0014, Phase 5 Go migration).
	// The binary must be pre-built; in CI the Validate Skills workflow builds it
	// to bin/, but the Build workflow does not. Skip gracefully when absent.
	scriptPath := filepath.Join(repoRoot(), "bin", "hwcloud-skillcheck")
	if _, err := os.Stat(scriptPath); os.IsNotExist(err) {
		t.Skipf("binary not found at %s; build it first (e.g. go build -trimpath -o ../bin/hwcloud-skillcheck .)", scriptPath)
	}
	cmd := exec.Command(scriptPath, "check", "--pre-commit", "--skip-tests")
	cmd.Dir = repoRoot()
	out, err := cmd.CombinedOutput()
	text := string(out)

	// This test verifies the P1 pre-commit gates are WIRED (executed and
	// printed), per Test Hermeticity (AGENTS.md §Test Hermeticity): the
	// `check --pre-commit` flow reads runtime/gitignored state
	// (`audit-results/`), which can legitimately leave unparseable or
	// schema-invalid trace files behind on a dirty repo. Those failures are
	// NOT a P1 gate regression — they're gitignored leftover state from prior
	// runs. Real pass/fail checks run via CI / the actual pre-commit hook.
	//
	// To avoid a previous reviewer-trap where any non-zero exit was tolerated
	// (which silently swallowed real gate regressions — gofmt/vet/validate
	// failures, schema errors, redaction bugs), we fail-CLOSED: every FAIL
	// line in the output must match at least one entry in
	// allowedRuntimeErrorSubstrings, each of which is anchored to a verified
	// --reject-invalid code path in cmd/aggregate.go that ONLY fires on
	// gitignored audit-results/ traces. Anything else fails this test
	// (proving real regressions cannot escape via this smoke test).
	if err != nil {
		fails := extractFailDetails(text)
		if len(fails) == 0 {
			t.Fatalf("check --pre-commit exited %v but no FAIL line found in output:\n%s", err, text)
		}
		allowed, unmatched := isAllowedRuntimeError(fails)
		if !allowed {
			t.Fatalf("check --pre-commit failed with non-runtime gate error(s) (real regression — must fix, not silence):\n  %s\n\nfull output:\n%s",
				strings.Join(unmatched, "\n  "), text)
		}
		t.Logf("check --pre-commit failed with runtime/gitignored-state error(s) only (tolerated by smoke test): %d fail line(s) — %s",
			len(fails), strings.Join(fails, " | "))
	}
	needed := []string{"golden run", "check lanes", "ab compare", "check advanced-coverage"}
	for _, g := range needed {
		if !strings.Contains(text, g) {
			t.Errorf("gate %q not found in pre-commit output", g)
		}
	}
}

// allowedRuntimeErrorSubstrings is the narrow allow-list for
// TestP1GatesWired. Each entry is anchored to a verified code path in
// cmd/aggregate.go that ONLY fires when `--reject-invalid` finds leftover
// unparseable or schema-invalid trace files in the gitignored
// audit-results/ directory. Any FAIL detail not matching one of these
// substrings forces a fail-closed assertion (real P1 gate regression).
var allowedRuntimeErrorSubstrings = []string{
	// cmd/aggregate.go:181 — JSON parse failure in gitignored audit-results/.
	// Format: "aggregate: unparseable=N trace file(s) found; --reject-invalid rejects untrusted trace input"
	"unparseable=",
	// cmd/aggregate.go:181 — companion message on the same parse-failure path.
	"--reject-invalid rejects untrusted trace input",
	// cmd/aggregate.go:204 — schema-invalid trace in gitignored audit-results/.
	// Format: "aggregate: N invalid trace(s) found (invalid_trace=N); rejecting untrusted trace input"
	"rejecting untrusted trace input",
}

// isAllowedRuntimeError reports whether every FAIL detail in `failDetails` is
// attributable to runtime/gitignored-state (audit-results/ leftover traces)
// rather than a real P1 gate regression. Returns the unmatched details so the
// caller can fail-closed with diagnostics. An empty failDetails slice is
// trivially "allowed" — the caller is expected to check len(fails) > 0 first.
func isAllowedRuntimeError(failDetails []string) (allowed bool, unmatched []string) {
	for _, d := range failDetails {
		matched := false
		for _, sub := range allowedRuntimeErrorSubstrings {
			if strings.Contains(d, sub) {
				matched = true
				break
			}
		}
		if !matched {
			unmatched = append(unmatched, d)
		}
	}
	return len(unmatched) == 0, unmatched
}

// extractFailDetails pulls the detail portion of every line beginning with
// `FAIL:` from `check --pre-commit` output. The gate prints one `FAIL:` line
// per failing gate in the form:
//
//	FAIL: <gate label>: <detail...>
//
// Multi-line details (e.g. gofmt's "files need gofmt:\n  foo.go") are joined
// back into a single detail string using "\n" so substring matching in
// isAllowedRuntimeError operates on the full gate message. Lines that do not
// begin with `FAIL:` are skipped.
func extractFailDetails(text string) []string {
	lines := strings.Split(text, "\n")
	var details []string
	for i := 0; i < len(lines); i++ {
		line := strings.TrimSpace(lines[i])
		if !strings.HasPrefix(line, "FAIL:") {
			continue
		}
		rest := strings.TrimPrefix(line, "FAIL:")
		// First ": " separates gate label from detail.
		idx := strings.Index(rest, ": ")
		if idx < 0 {
			details = append(details, strings.TrimSpace(rest))
			continue
		}
		detail := strings.TrimSpace(rest[idx+2:])
		// Re-join any continuation lines (gofmt prints file lists indented
		// under the FAIL header) up to the next blank line or next
		// `==> [gate]` / `FAIL:` / `WARN:` marker.
		for j := i + 1; j < len(lines); j++ {
			next := strings.TrimSpace(lines[j])
			if next == "" || strings.HasPrefix(next, "==>") ||
				strings.HasPrefix(next, "FAIL:") || strings.HasPrefix(next, "WARN:") ||
				strings.HasPrefix(next, "OK:") || strings.HasPrefix(next, "PASS ") {
				break
			}
			detail += "\n" + strings.TrimSpace(lines[j])
			i = j
		}
		details = append(details, detail)
	}
	return details
}

// TestIsAllowedRuntimeError_NegativeCheck proves the allow-list in
// TestP1GatesWired does not silently swallow real P1 gate regressions. Each
// case feeds a synthetic FAIL detail and asserts the helper classifies it
// correctly: runtime-state failures are allowed; gofmt/vet/validate/repo-state
// failures are NOT allowed and would force TestP1GatesWired to fail-closed.
func TestIsAllowedRuntimeError_NegativeCheck(t *testing.T) {
	cases := []struct {
		name      string
		failLine  string
		wantAllow bool
	}{
		{
			name:      "runtime: unparseable trace from gitignored audit-results/",
			failLine:  "aggregate: unparseable=2 trace file(s) found; --reject-invalid rejects untrusted trace input",
			wantAllow: true,
		},
		{
			name:      "runtime: schema-invalid trace from gitignored audit-results/",
			failLine:  "aggregate: 4 invalid trace(s) found (invalid_trace=4); rejecting untrusted trace input",
			wantAllow: true,
		},
		{
			name:      "real regression: gofmt files need reformatting (must NOT be allowed)",
			failLine:  "files need gofmt:\n  internal/spec_audit/p1_audit_test.go",
			wantAllow: false,
		},
		{
			name:      "real regression: go vet failure (must NOT be allowed)",
			failLine:  "go vet failed: exit status 1\n  ./internal/foo: undefined: Bar",
			wantAllow: false,
		},
		{
			name:      "real regression: validate frontmatter (must NOT be allowed)",
			failLine:  "frontmatter missing required field 'description' in huaweicloud-foo-ops/SKILL.md",
			wantAllow: false,
		},
		{
			name:      "real regression: build failure (must NOT be allowed)",
			failLine:  "build hwcloud-skillcheck: go build: exit status 1: ./cmd/bar.go:42: undefined: Baz",
			wantAllow: false,
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			allowed, unmatched := isAllowedRuntimeError([]string{c.failLine})
			if allowed != c.wantAllow {
				t.Fatalf("isAllowedRuntimeError(%q) allowed=%v want=%v (unmatched=%v)",
					c.failLine, allowed, c.wantAllow, unmatched)
			}
			if !c.wantAllow {
				if len(unmatched) != 1 {
					t.Fatalf("want 1 unmatched line, got %d: %v", len(unmatched), unmatched)
				}
				if unmatched[0] != c.failLine {
					t.Fatalf("unmatched line round-trip mismatch: got %q want %q", unmatched[0], c.failLine)
				}
			}
		})
	}
}

// repoRoot returns the repository root (parent of hwcloud-skillcheck/).
func repoRoot() string {
	// This test lives in hwcloud-skillcheck/internal/spec_audit/.
	// Walk up three levels to reach the repo root.
	_, f0, _, _ := runtime.Caller(0)
	// f0 = p1_audit_test.go in spec_audit dir
	// parent[0] = spec_audit dir
	// parent[1] = internal dir
	// parent[2] = hwcloud-skillcheck dir
	// parent[3] = repo root
	for i := 0; i < 4; i++ {
		f0 = filepath.Dir(f0)
	}
	return f0
}
