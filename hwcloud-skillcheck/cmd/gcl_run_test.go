package cmd

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/buhaiqing/hcloud-skills/hwcloud-skillcheck/internal/gcl"
)

// printGCLRunJSON/Human 的预算超时可观测性：预算超时（BudgetExceeded 非空）时，
// 状态保持 SAFETY_VIOLATION（fail-closed，与 runner/alarm 的 SAFETY_FAIL 契约一致），
// 但附加 budget_exceeded 细节字段，让运维能从 CLI 区分"可重试的资源耗尽"vs"需立即
// 处理的安全违规"。

func TestPrintGCLRunJSONBudgetExceeded(t *testing.T) {
	var buf bytes.Buffer
	res := gcl.RunResult{ExitCode: gcl.ExitSafety, TracePath: "trace.json", BudgetExceeded: "wall_clock"}
	printGCLRunJSON(&buf, "test-skill", res)

	var got map[string]any
	if err := json.Unmarshal(buf.Bytes(), &got); err != nil {
		t.Fatalf("invalid JSON: %v\n%s", err, buf.String())
	}
	// 状态保持 SAFETY_VIOLATION（fail-closed 契约不变），budget_exceeded 为细节
	if got["status"] != "SAFETY_VIOLATION" {
		t.Errorf("预算超时 status 应保持 SAFETY_VIOLATION，got %v", got["status"])
	}
	if got["budget_exceeded"] != "wall_clock" {
		t.Errorf("JSON 应含 budget_exceeded=wall_clock，got %v", got["budget_exceeded"])
	}
	if got["exit_code"].(float64) != float64(gcl.ExitSafety) {
		t.Errorf("exit_code 应保持 ExitSafety(3)（fail-closed），got %v", got["exit_code"])
	}
}

func TestPrintGCLRunJSONBudgetToolCallsKind(t *testing.T) {
	// 覆盖 tool_calls kind（runner 三种 kind：tokens/tool_calls/wall_clock）
	var buf bytes.Buffer
	res := gcl.RunResult{ExitCode: gcl.ExitSafety, TracePath: "trace.json", BudgetExceeded: "tool_calls"}
	printGCLRunJSON(&buf, "test-skill", res)

	var got map[string]any
	if err := json.Unmarshal(buf.Bytes(), &got); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if got["budget_exceeded"] != "tool_calls" {
		t.Errorf("JSON 应含 budget_exceeded=tool_calls，got %v", got["budget_exceeded"])
	}
}

func TestPrintGCLRunJSONRealSafetyViolation(t *testing.T) {
	var buf bytes.Buffer
	res := gcl.RunResult{ExitCode: gcl.ExitSafety, TracePath: "trace.json"} // 无 BudgetExceeded
	printGCLRunJSON(&buf, "test-skill", res)

	var got map[string]any
	if err := json.Unmarshal(buf.Bytes(), &got); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if got["status"] != "SAFETY_VIOLATION" {
		t.Errorf("真安全违规 status 应为 SAFETY_VIOLATION，got %v", got["status"])
	}
	if _, present := got["budget_exceeded"]; present {
		t.Error("真安全违规不应含 budget_exceeded 字段")
	}
}

func TestPrintGCLRunHumanBudgetExceeded(t *testing.T) {
	var buf bytes.Buffer
	res := gcl.RunResult{ExitCode: gcl.ExitSafety, TracePath: "trace.json", BudgetExceeded: "tokens"}
	printGCLRunHuman(&buf, "test-skill", res)
	out := buf.String()
	// 状态保持 SAFETY_VIOLATION，但附加 budget_exceeded 细节
	if !strings.Contains(out, "SAFETY_VIOLATION") || !strings.Contains(out, "budget_exceeded=tokens") {
		t.Errorf("human 输出应含 SAFETY_VIOLATION + budget_exceeded=tokens，got %q", out)
	}
}

func TestPrintGCLRunHumanRealSafetyViolation(t *testing.T) {
	var buf bytes.Buffer
	res := gcl.RunResult{ExitCode: gcl.ExitSafety, TracePath: "trace.json"} // 无 BudgetExceeded
	printGCLRunHuman(&buf, "test-skill", res)
	if !strings.Contains(buf.String(), "SAFETY_VIOLATION") {
		t.Errorf("真安全违规应显示 SAFETY_VIOLATION，got %q", buf.String())
	}
	if strings.Contains(buf.String(), "budget_exceeded") {
		t.Errorf("真安全违规不应含 budget_exceeded，got %q", buf.String())
	}
}

// TestCriticArgsValueSet guards the flag.Value adapter against a nil slice:
// a zero-value criticArgsValue (e.g. used as a bare var) must return an error
// instead of panicking on append.
func TestCriticArgsValueSet(t *testing.T) {
	var zero criticArgsValue
	if err := zero.Set("x"); err == nil {
		t.Fatal("Set on zero-value criticArgsValue should return an error")
	}

	var dst []string
	v := criticArgsValue{slice: &dst}
	if err := v.Set("a"); err != nil {
		t.Fatalf("Set(a) unexpected error: %v", err)
	}
	if err := v.Set("b"); err != nil {
		t.Fatalf("Set(b) unexpected error: %v", err)
	}
	if len(dst) != 2 || dst[0] != "a" || dst[1] != "b" {
		t.Fatalf("got %v, want [a b]", dst)
	}
	if v.String() != "[a b]" {
		t.Fatalf("String() = %q, want %q", v.String(), "[a b]")
	}
}

// ---- --structural-critic-only semantics ----------------------------------

// writeSmokeSkill scaffolds the minimum skill dir the `gcl run` CLI needs in
// order to route: <root>/huaweicloud-ecs-ops/SKILL.md with a name frontmatter.
func writeSmokeSkill(t *testing.T, root string) string {
	t.Helper()
	dir := filepath.Join(root, "huaweicloud-ecs-ops")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	body := "---\nname: huaweicloud-ecs-ops\ndescription: manage ECS servers\nside_effect_class_max: read-only\nmetadata:\n  version: 1\n---\n"
	if err := os.WriteFile(filepath.Join(dir, "SKILL.md"), []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	return dir
}

// TestGCLRunStructuralCriticOnlyRejectsCriticCmd pins the mutual exclusion
// between --structural-critic-only and --critic-cmd. Both flags name the
// Critic to use, so honouring one and silently dropping the other is a
// footgun; the CLI must fail up front instead. The stub critic touches a
// marker file when executed, so an absent marker proves no critic subprocess
// was spawned on the rejected path.
func TestGCLRunStructuralCriticOnlyRejectsCriticCmd(t *testing.T) {
	tmp := t.TempDir()
	marker := filepath.Join(tmp, "critic-spawned")
	script := filepath.Join(tmp, "critic.sh")
	body := "#!/bin/sh\ntouch " + marker + "\necho '{\"scores\":{\"correctness\":1,\"safety\":1,\"idempotency\":1,\"traceability\":1,\"spec_compliance\":1}}'\n"
	if err := os.WriteFile(script, []byte(body), 0o700); err != nil {
		t.Fatal(err)
	}
	skillDir := writeSmokeSkill(t, filepath.Join(tmp, "repo"))

	err := runGCLRun([]string{"--root", skillDir, "--structural-critic-only", "--critic-cmd", script})
	if err == nil {
		t.Fatal("both --structural-critic-only and --critic-cmd set: want an error, got nil")
	}
	if !strings.Contains(err.Error(), "structural-critic-only and critic-cmd are mutually exclusive") {
		t.Errorf("error = %q, want the mutual-exclusion message", err)
	}
	if _, statErr := os.Stat(marker); !os.IsNotExist(statErr) {
		t.Error("critic subprocess ran even though the flag combination was rejected")
	}
}

// TestGCLRunStructuralCriticOnlySmokePath runs the CLI end-to-end with
// --structural-critic-only and no --critic-cmd: the default `echo ok` smoke
// command must still PASS, and the persisted trace must record the
// in-process structural critic rather than an external one.
func TestGCLRunStructuralCriticOnlySmokePath(t *testing.T) {
	bin := buildSkillcheckBinary(t)
	skillDir := writeSmokeSkill(t, t.TempDir())

	cmd := exec.Command(bin, "gcl", "run", "--root", skillDir, "--structural-critic-only", "--quiet")
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil || cmd.ProcessState.ExitCode() != 0 {
		t.Fatalf("gcl run --structural-critic-only failed: %v (exit %v)\nstdout: %s\nstderr: %s",
			err, cmd.ProcessState.ExitCode(), out, stderr.String())
	}

	tracePath := strings.TrimSpace(string(out))
	if !strings.Contains(tracePath, "gcl-trace-") {
		t.Fatalf("--quiet should print the trace path, got %q (stderr: %s)", tracePath, stderr.String())
	}
	data, readErr := os.ReadFile(tracePath)
	if readErr != nil {
		t.Fatalf("read trace %q: %v", tracePath, readErr)
	}
	var trace struct {
		Final *struct {
			Status     string `json:"status"`
			CriticType string `json:"critic_type"`
		} `json:"final"`
	}
	if jsonErr := json.Unmarshal(data, &trace); jsonErr != nil {
		t.Fatalf("trace is not valid JSON: %v", jsonErr)
	}
	if trace.Final == nil {
		t.Fatal("persisted trace has no final block")
	}
	if trace.Final.Status != "PASS" {
		t.Errorf("final.status = %q, want PASS", trace.Final.Status)
	}
	if trace.Final.CriticType != "structural" {
		t.Errorf("final.critic_type = %q, want structural", trace.Final.CriticType)
	}
}
