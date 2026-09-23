package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// makeAgentsMD writes a synthetic AGENTS.md into root with the given body.
// Caller controls the body so each test can craft pass/fail shapes; the file
// name is fixed because the validator resolves AGENTS.md by name at the
// repo root.
func makeAgentsMD(t *testing.T, root, body string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(root, "AGENTS.md"), []byte(body), 0o644); err != nil {
		t.Fatalf("write AGENTS.md: %v", err)
	}
}

// --- Check 1: line budget ---

func TestRunValidateAgents_LinesPass(t *testing.T) {
	root := t.TempDir()
	// 400 lines is well under the 500-line cap; the body content is irrelevant
	// for this check beyond total line count.
	body := strings.Repeat("padding line\n", 400)
	makeAgentsMD(t, root, body)
	if err := runValidateAgents([]string{"--root", root}); err != nil {
		t.Fatalf("400-line AGENTS.md should pass, got: %v", err)
	}
}

func TestRunValidateAgents_LinesFail(t *testing.T) {
	root := t.TempDir()
	// 501 lines — exactly one over the cap — should fail with a clear budget
	// message. We don't try to drive the CA count, so leave no CA headings.
	body := strings.Repeat("padding line\n", 501)
	makeAgentsMD(t, root, body)
	stderr := captureStderr(t, func() {
		_ = runValidateAgents([]string{"--root", root})
	})
	if !strings.Contains(stderr, "line count 501") {
		t.Fatalf("stderr should report line count 501, got: %q", stderr)
	}
	if !strings.Contains(stderr, "exceeds limit 500") {
		t.Fatalf("stderr should mention the 500-line cap, got: %q", stderr)
	}
}

func TestRunValidateAgents_LinesExactlyAtBudget(t *testing.T) {
	root := t.TempDir()
	// Boundary: exactly 500 lines must pass (<= budget, not <).
	body := strings.Repeat("x\n", 500)
	makeAgentsMD(t, root, body)
	if err := runValidateAgents([]string{"--root", root}); err != nil {
		t.Fatalf("500-line AGENTS.md should pass (budget is <=500), got: %v", err)
	}
}

// --- Check 2: CA entry count ---

// buildCanonicalCA returns a minimal AGENTS.md body with `n` CA entries in a
// contiguous sequence starting at 1, with enough padding lines to stay under
// the 500-line cap. Used to drive only the CA-count check (and as a baseline
// for the other CA tests).
func buildCanonicalCA(n int) string {
	var sb strings.Builder
	// Filler so line count stays small even when n is large; the count check
	// is independent of line count.
	for i := 0; i < 10; i++ {
		sb.WriteString("filler\n")
	}
	for i := 1; i <= n; i++ {
		fmt.Fprintf(&sb, "### CA-%d. rule\n**Rule**: example.\n\n", i)
	}
	return sb.String()
}

func TestRunValidateAgents_CACountPass(t *testing.T) {
	root := t.TempDir()
	makeAgentsMD(t, root, buildCanonicalCA(12))
	if err := runValidateAgents([]string{"--root", root}); err != nil {
		t.Fatalf("12 CA entries should pass (budget is 12), got: %v", err)
	}
}

func TestRunValidateAgents_CACountFail(t *testing.T) {
	root := t.TempDir()
	makeAgentsMD(t, root, buildCanonicalCA(13))
	stderr := captureStderr(t, func() {
		_ = runValidateAgents([]string{"--root", root})
	})
	// The error should include the entry list so operators can see which
	// entries triggered the gate without re-reading the file.
	if !strings.Contains(stderr, "13 CA entries") {
		t.Fatalf("stderr should report entry count, got: %q", stderr)
	}
	if !strings.Contains(stderr, "CA-13") {
		t.Fatalf("stderr should list CA-13 among offenders, got: %q", stderr)
	}
}

// --- Check 3: contiguous, gap-free, duplicate-free numbering ---

func TestRunValidateAgents_CANumberingGap(t *testing.T) {
	root := t.TempDir()
	// CA-1, CA-2, CA-4 — CA-3 is missing, which breaks contiguity.
	body := "filler\n" +
		"### CA-1. r\n**Rule**: a\n\n" +
		"### CA-2. r\n**Rule**: a\n\n" +
		"### CA-4. r\n**Rule**: a\n\n"
	makeAgentsMD(t, root, body)
	stderr := captureStderr(t, func() {
		_ = runValidateAgents([]string{"--root", root})
	})
	if !strings.Contains(stderr, "non-contiguous") {
		t.Fatalf("stderr should mention non-contiguous, got: %q", stderr)
	}
	if !strings.Contains(stderr, "CA-2..CA-4") {
		t.Fatalf("stderr should describe the gap, got: %q", stderr)
	}
}

func TestRunValidateAgents_CANumberingDuplicate(t *testing.T) {
	root := t.TempDir()
	// Two CA-3 entries — duplicate number.
	body := "filler\n" +
		"### CA-1. r\n**Rule**: a\n\n" +
		"### CA-3. r\n**Rule**: a\n\n" +
		"### CA-3. r\n**Rule**: a\n\n"
	makeAgentsMD(t, root, body)
	stderr := captureStderr(t, func() {
		_ = runValidateAgents([]string{"--root", root})
	})
	if !strings.Contains(stderr, "duplicate CA numbers") {
		t.Fatalf("stderr should mention duplicate, got: %q", stderr)
	}
	if !strings.Contains(stderr, "[3]") {
		t.Fatalf("stderr should name the duplicated number, got: %q", stderr)
	}
}

func TestRunValidateAgents_CANumberingPass(t *testing.T) {
	root := t.TempDir()
	// Non-1-starting but contiguous sequence: CA-2..CA-5 is still a valid
	// contiguous block (no internal gaps, no duplicates). The gate only
	// enforces contiguity, not "starts at 1" — that is a separate editorial
	// concern handled by humans, not the mechanical gate.
	body := "filler\n" +
		"### CA-2. r\n**Rule**: a\n\n" +
		"### CA-3. r\n**Rule**: a\n\n" +
		"### CA-4. r\n**Rule**: a\n\n" +
		"### CA-5. r\n**Rule**: a\n\n"
	makeAgentsMD(t, root, body)
	if err := runValidateAgents([]string{"--root", root}); err != nil {
		t.Fatalf("contiguous non-1-starting sequence should pass, got: %v", err)
	}
}

// --- Missing AGENTS.md ---

func TestRunValidateAgents_MissingFile(t *testing.T) {
	root := t.TempDir()
	// Empty tempdir: no AGENTS.md written. The file is mandatory, so a
	// missing one must be a hard error (not a silent pass).
	stderr := captureStderr(t, func() {
		_ = runValidateAgents([]string{"--root", root})
	})
	if !strings.Contains(stderr, "open") {
		t.Fatalf("stderr should mention open/read failure, got: %q", stderr)
	}
	if !strings.Contains(stderr, "AGENTS.md") {
		t.Fatalf("stderr should name the missing file, got: %q", stderr)
	}
}

// --- Multi-failure aggregation ---

func TestRunValidateAgents_AllChecksFail(t *testing.T) {
	root := t.TempDir()
	// 600 lines (over budget) + 14 CA entries (over budget) + a gap
	// (CA-1, CA-2, ..., CA-13, CA-15 — CA-14 missing). One bad file
	// exercises all three error paths in a single run.
	var sb strings.Builder
	for i := 0; i < 590; i++ {
		sb.WriteString("pad\n")
	}
	for _, n := range []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 15} {
		fmt.Fprintf(&sb, "### CA-%d. r\n**Rule**: a\n\n", n)
	}
	makeAgentsMD(t, root, sb.String())
	stderr := captureStderr(t, func() {
		_ = runValidateAgents([]string{"--root", root})
	})
	for _, want := range []string{"line count", "CA entries", "non-contiguous"} {
		if !strings.Contains(stderr, want) {
			t.Errorf("stderr should mention %q, got: %q", want, stderr)
		}
	}
}
