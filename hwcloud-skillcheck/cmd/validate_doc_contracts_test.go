package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// The fixture bodies below are minimal stand-ins: each contains exactly the
// literal anchors docContracts asserts, so a mutation test fails for the
// intended reason and nothing else. Real-repo coverage comes from
// `hwcloud-skillcheck validate --root .` in the pre-commit gate and CI,
// keeping these tests hermetic per the repo's Test Hermeticity P0 rule.

func canonicalGCLRuntime() string {
	return strings.Join([]string{
		"# GCL Runtime",
		"",
		"## Threshold Calibration",
		"",
		"| Dimension | Bar | Evidence |",
		"|---|---|---|",
		"| Correctness (GCL pass bar) | ≥ 0.5 | none yet |",
		"| Safety | = 1.0 | zero tolerance |",
		"| Idempotency | ≥ 0.5 | none yet |",
		"| Traceability | ≥ 0.5 | none yet |",
		"| Spec Compliance | ≥ 0.5 | none yet |",
		"| confidence (low / mid / high) | 0.70 / 0.85 / 0.95 | empirical constant |",
		"| auto_execute (low / mid / high) | 0.70 / 0.85 / 0.95 | empirical constant |",
		"",
	}, "\n")
}

func canonicalSkillUpdate() string {
	return strings.Join([]string{
		"# Skill Update Rule",
		"",
		"## Round 1",
		"",
		"## Round 2",
		"",
		"## Round 3 — Rubric Auto-Proposal (Closed Loop)",
		"",
		"1. Append findings to `audit-results/reflection-findings.jsonl`.",
		"2. Run `python3 scripts/reflection_findings.py propose --file <jsonl>`.",
		"3. Propose when `finding_type` repeats AND `rubric_item == null`.",
		"",
	}, "\n")
}

func canonicalCAArchive() string {
	return strings.Join([]string{
		"# CA Archive",
		"",
		"## 退役条目",
		"",
		"### CA-A11. worktree .git is a file",
		"**Rule**: shared hooks.",
		"",
		"### CA-A12. init() caches defeat t.Setenv",
		"**Rule**: export a reset function.",
		"",
		"### CA-A13. shell-to-Go shadow behaviour",
		"**Rule**: audit every step.",
		"",
		"### CA-A14. os.Exit kills in-process gates",
		"**Rule**: shell out via exec.Command.",
		"",
	}, "\n")
}

func canonicalSelfHealing() string {
	return strings.Join([]string{
		"# Self-Healing Spec",
		"",
		"## 3. Failure Pattern Contract",
		"",
		"`assets/failure_patterns.json` rows carry `source_traces_analyzed`.",
		"",
		"## 5. Experience Learning",
		"",
		"### 5.1 触发条件",
		"text",
		"",
		"### 5.2 学习流程",
		"text",
		"",
		"### 5.3 Loop Health Observability",
		"text",
		"",
		"### 5.4 GCL Runner 集成",
		"text",
		"",
	}, "\n")
}

func writeDocFixtures(t *testing.T, root string) {
	t.Helper()
	files := map[string]string{
		// Marker file: its presence is what makes doc-contracts applicable.
		"docs/gcl-spec.md":                "# GCL spec\n",
		"references/gcl-runtime.md":       canonicalGCLRuntime(),
		"references/skill-update-rule.md": canonicalSkillUpdate(),
		"references/ca-archive.md":        canonicalCAArchive(),
		"references/self-healing-spec.md": canonicalSelfHealing(),
	}
	for rel, body := range files {
		path := filepath.Join(root, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatalf("mkdir %s: %v", rel, err)
		}
		if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
			t.Fatalf("write %s: %v", rel, err)
		}
	}
}

// mutateDoc replaces the first occurrence of old with replacement in one doc,
// failing the test if the anchor is not present (so a fixture drift surfaces
// as a test bug rather than a silently vacuous mutation).
func mutateDoc(t *testing.T, root, rel, old, replacement string) {
	t.Helper()
	path := filepath.Join(root, filepath.FromSlash(rel))
	body, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", rel, err)
	}
	if !strings.Contains(string(body), old) {
		t.Fatalf("fixture %s does not contain %q", rel, old)
	}
	updated := strings.Replace(string(body), old, replacement, 1)
	if err := os.WriteFile(path, []byte(updated), 0o644); err != nil {
		t.Fatalf("write %s: %v", rel, err)
	}
}

func TestRunValidateDocContracts_CanonicalRepoPasses(t *testing.T) {
	root := t.TempDir()
	writeDocFixtures(t, root)

	if err := runValidateDocContracts([]string{"--root", root}); err != nil {
		t.Fatalf("canonical fixture repo should pass doc-contracts, got: %v", err)
	}
}

func TestRunValidateDocContracts_MissingSectionFails(t *testing.T) {
	root := t.TempDir()
	writeDocFixtures(t, root)
	mutateDoc(t, root, "references/gcl-runtime.md", "## Threshold Calibration", "## Calibration Notes")

	stderr := captureStderr(t, func() {
		_ = runValidateDocContracts([]string{"--root", root})
	})
	if !strings.Contains(stderr, `references/gcl-runtime.md: missing anchor "## Threshold Calibration"`) {
		t.Fatalf("stderr must name the missing section anchor verbatim, got: %q", stderr)
	}
}

func TestRunValidateDocContracts_ThresholdDriftFails(t *testing.T) {
	root := t.TempDir()
	writeDocFixtures(t, root)
	mutateDoc(t, root, "references/gcl-runtime.md",
		"| Correctness (GCL pass bar) | ≥ 0.5 |", "| Correctness (GCL pass bar) | ≥ 0.6 |")

	stderr := captureStderr(t, func() {
		_ = runValidateDocContracts([]string{"--root", root})
	})
	if !strings.Contains(stderr, `missing anchor "| Correctness (GCL pass bar) | ≥ 0.5 |"`) {
		t.Fatalf("threshold drift must be reported with the expected literal, got: %q", stderr)
	}
}

func TestRunValidateDocContracts_TierBandDriftFails(t *testing.T) {
	root := t.TempDir()
	writeDocFixtures(t, root)
	mutateDoc(t, root, "references/gcl-runtime.md",
		"| confidence (low / mid / high) | 0.70 / 0.85 / 0.95 |",
		"| confidence (low / mid / high) | 0.60 / 0.85 / 0.95 |")

	stderr := captureStderr(t, func() {
		_ = runValidateDocContracts([]string{"--root", root})
	})
	if !strings.Contains(stderr, "0.70 / 0.85 / 0.95") {
		t.Fatalf("tier-band drift must be reported, got: %q", stderr)
	}
}

func TestRunValidateDocContracts_Round3CommandDriftFails(t *testing.T) {
	root := t.TempDir()
	writeDocFixtures(t, root)
	mutateDoc(t, root, "references/skill-update-rule.md", "propose --file", "propose --input")

	stderr := captureStderr(t, func() {
		_ = runValidateDocContracts([]string{"--root", root})
	})
	if !strings.Contains(stderr, `references/skill-update-rule.md: missing anchor "propose --file"`) {
		t.Fatalf("Round 3 command drift must be reported, got: %q", stderr)
	}
}

func TestRunValidateDocContracts_BareArchivedCAHeadingFails(t *testing.T) {
	root := t.TempDir()
	writeDocFixtures(t, root)
	mutateDoc(t, root, "references/ca-archive.md", "### CA-A11", "### CA-11")

	stderr := captureStderr(t, func() {
		_ = runValidateDocContracts([]string{"--root", root})
	})
	if !strings.Contains(stderr, "collides with AGENTS.md active numbering") {
		t.Fatalf("bare archived CA heading must be reported as a collision, got: %q", stderr)
	}
	if !strings.Contains(stderr, "### CA-11.") {
		t.Fatalf("collision report must name the offending heading, got: %q", stderr)
	}
}

func TestRunValidateDocContracts_SpecNumberingGapFails(t *testing.T) {
	root := t.TempDir()
	writeDocFixtures(t, root)
	mutateDoc(t, root, "references/self-healing-spec.md", "### 5.4 GCL Runner 集成", "### 5.5 GCL Runner 集成")

	stderr := captureStderr(t, func() {
		_ = runValidateDocContracts([]string{"--root", root})
	})
	if !strings.Contains(stderr, "### 5.4 GCL Runner 集成") {
		t.Fatalf("spec numbering gap must be reported, got: %q", stderr)
	}
}

func TestRunValidateDocContracts_SkipsRepoWithoutMarker(t *testing.T) {
	// A foreign skill repo has none of the hcloud-skills contract docs; the
	// gate must skip instead of failing, or `validate --root <other repo>`
	// becomes a breaking change.
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "references"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "references", "other.md"), []byte("# other\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := runValidateDocContracts([]string{"--root", root}); err != nil {
		t.Fatalf("repo without %s must be skipped, got: %v", "docs/gcl-spec.md", err)
	}
}

func TestRunValidateDocContracts_MarkerWithoutPinnedDocsStillFails(t *testing.T) {
	// Guard against the cheapest bypass: deleting all four pinned docs must
	// fail even though the marker is still present.
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "docs"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "docs", "gcl-spec.md"), []byte("# GCL spec\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	stderr := captureStderr(t, func() {
		_ = runValidateDocContracts([]string{"--root", root})
	})
	if !strings.Contains(stderr, "open references/gcl-runtime.md") {
		t.Fatalf("deleting the pinned docs must fail loudly, got: %q", stderr)
	}
}

func TestRunValidateDocContracts_MissingReferenceDocIsHardError(t *testing.T) {
	root := t.TempDir()
	writeDocFixtures(t, root)
	if err := os.Remove(filepath.Join(root, "references", "gcl-runtime.md")); err != nil {
		t.Fatalf("remove fixture: %v", err)
	}

	stderr := captureStderr(t, func() {
		_ = runValidateDocContracts([]string{"--root", root})
	})
	if !strings.Contains(stderr, "open references/gcl-runtime.md") {
		t.Fatalf("missing mandatory reference doc must be a hard error, got: %q", stderr)
	}
}

func TestRunValidateDocContracts_AggregatesMultipleFailures(t *testing.T) {
	root := t.TempDir()
	writeDocFixtures(t, root)
	mutateDoc(t, root, "references/gcl-runtime.md", "| Safety | = 1.0 |", "| Safety | = 0.9 |")
	mutateDoc(t, root, "references/self-healing-spec.md", "source_traces_analyzed", "traces_analyzed")

	stderr := captureStderr(t, func() {
		_ = runValidateDocContracts([]string{"--root", root})
	})
	for _, want := range []string{"| Safety | = 1.0 |", "source_traces_analyzed"} {
		if !strings.Contains(stderr, want) {
			t.Fatalf("aggregate stderr must report %q, got: %q", want, stderr)
		}
	}
}
