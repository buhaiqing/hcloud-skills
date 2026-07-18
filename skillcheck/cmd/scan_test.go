package cmd

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/buhaiqing/hcloud-skills/skillcheck/internal/embed"
)

// --- scan secret trace ---

func TestScanSecretTraceClean(t *testing.T) {
	root := t.TempDir()
	writeTrace(t, root, "gcl-trace-20260701-000000.json", `{"skill":"huaweicloud-ecs-ops","final":{"status":"PASS"}}`)
	if err := runScan([]string{"secret", "trace", "--root", root}); err != nil {
		t.Fatalf("clean trace should pass, got: %v", err)
	}
}

func TestScanSecretTraceLeak(t *testing.T) {
	root := t.TempDir()
	// SK= base64-style secret must be flagged.
	writeTrace(t, root, "gcl-trace-20260701-000000.json", `{"skill":"huaweicloud-ecs-ops","request":"SK=ABCDEFGHIJKLMNOPQRSTUVWX","final":{"status":"PASS"}}`)
	if err := runScan([]string{"secret", "trace", "--root", root}); err == nil {
		t.Fatal("leaking trace should fail scan")
	}
}

func TestScanSecretTraceLatest(t *testing.T) {
	root := t.TempDir()
	// Only the latest file leaks; --latest must still detect it.
	writeTrace(t, root, "gcl-trace-20260701-000000.json", `{"skill":"huaweicloud-ecs-ops","final":{"status":"PASS"}}`)
	writeTrace(t, root, "gcl-trace-20260702-000000.json", `{"skill":"huaweicloud-ecs-ops","request":"SK=ABCDEFGHIJKLMNOPQRSTUVWX","final":{"status":"PASS"}}`)
	if err := runScan([]string{"secret", "trace", "--root", root, "--latest"}); err == nil {
		t.Fatal("--latest should still scan the (latest) leaking trace")
	}
}

func TestScanSecretTraceNoFiles(t *testing.T) {
	root := t.TempDir()
	// No trace files => ok (no error), mirroring --allow-empty default in CI.
	if err := runScan([]string{"secret", "trace", "--root", root}); err != nil {
		t.Fatalf("no trace files should pass (allow-empty), got: %v", err)
	}
}

// --- scan secret summary ---

func TestScanSecretSummaryClean(t *testing.T) {
	root := t.TempDir()
	writeSummary(t, root, "gcl-quality-summary-20260701-000000.json", string(embed.SummaryHealthy))
	if err := runScan([]string{"secret", "summary", "--root", root}); err != nil {
		t.Fatalf("healthy summary should pass, got: %v", err)
	}
}

func TestScanSecretSummaryLeak(t *testing.T) {
	root := t.TempDir()
	writeSummary(t, root, "gcl-quality-summary-20260701-000000.json",
		`{"version":"1.0","note":"HW_SECRET_ACCESS_KEY=AKIA_LEAKED_VALUE"}`)
	if err := runScan([]string{"secret", "summary", "--root", root}); err == nil {
		t.Fatal("leaking summary should fail scan")
	}
}

func TestScanSecretSummaryIncludeFixture(t *testing.T) {
	root := t.TempDir()
	// --include-fixture adds the healthy fixture; it must be clean (no leak).
	if err := runScan([]string{"secret", "summary", "--root", root, "--include-fixture"}); err != nil {
		t.Fatalf("healthy fixture scan should pass, got: %v", err)
	}
}

// --- scan secret alarm-plan ---

func TestScanSecretAlarmPlanClean(t *testing.T) {
	root := t.TempDir()
	writeAlarmPlan(t, root, "gcl-alarm-plan-20260701-000000-plan.json", string(embed.AlarmPlanHealthy))
	if err := runScan([]string{"secret", "alarm-plan", "--root", root}); err != nil {
		t.Fatalf("healthy alarm-plan should pass, got: %v", err)
	}
}

func TestScanSecretAlarmPlanLeak(t *testing.T) {
	root := t.TempDir()
	writeAlarmPlan(t, root, "gcl-alarm-plan-20260701-000000-plan.json",
		`{"note":"Bearer aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"}`)
	if err := runScan([]string{"secret", "alarm-plan", "--root", root}); err == nil {
		t.Fatal("leaking alarm-plan should fail scan")
	}
}

func TestScanSecretAlarmPlanIncludeFixture(t *testing.T) {
	root := t.TempDir()
	if err := runScan([]string{"secret", "alarm-plan", "--root", root, "--include-fixture"}); err != nil {
		t.Fatalf("healthy fixture alarm-plan scan should pass, got: %v", err)
	}
}

// --- self-check ---

func TestScanSelfCheck(t *testing.T) {
	// --self-check scans the embedded fixtures (summary + alarm-plan); both are
	// healthy, so it must pass. trace has no embed fixture and is skipped.
	if err := runScan([]string{"secret", "trace", "--self-check"}); err != nil {
		t.Fatalf("self-check trace (no fixture) should skip cleanly, got: %v", err)
	}
	if err := runScan([]string{"secret", "summary", "--self-check"}); err != nil {
		t.Fatalf("self-check summary fixture should be clean, got: %v", err)
	}
	if err := runScan([]string{"secret", "alarm-plan", "--self-check"}); err != nil {
		t.Fatalf("self-check alarm-plan fixture should be clean, got: %v", err)
	}
}

// --- helpers ---

func writeTrace(t *testing.T, root, name, content string) {
	t.Helper()
	dir := filepath.Join(root, "audit-results")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func writeSummary(t *testing.T, root, name, content string) {
	writeTrace(t, root, name, content)
}

func writeAlarmPlan(t *testing.T, root, name, content string) {
	writeTrace(t, root, name, content)
}
