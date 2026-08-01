package cmd

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/buhaiqing/hcloud-skills/hwcloud-skillcheck/internal/gcl"
)

// printGCLRunJSON/Human 的预算超时可观测性：预算超时（BudgetExceeded 非空）必须与
// 真安全违规（SAFETY_VIOLATION）在 CLI 输出上可区分，便于运维判断"可重试"vs"需立即处理"。

func TestPrintGCLRunJSONBudgetExceeded(t *testing.T) {
	var buf bytes.Buffer
	res := gcl.RunResult{ExitCode: gcl.ExitSafety, TracePath: "trace.json", BudgetExceeded: "wall_clock"}
	printGCLRunJSON(&buf, "test-skill", res)

	var got map[string]any
	if err := json.Unmarshal(buf.Bytes(), &got); err != nil {
		t.Fatalf("invalid JSON: %v\n%s", err, buf.String())
	}
	if got["status"] != "BUDGET_EXCEEDED" {
		t.Errorf("预算超时 status 应为 BUDGET_EXCEEDED，got %v", got["status"])
	}
	if got["budget_exceeded"] != "wall_clock" {
		t.Errorf("JSON 应含 budget_exceeded=wall_clock，got %v", got["budget_exceeded"])
	}
	if got["exit_code"].(float64) != float64(gcl.ExitSafety) {
		t.Errorf("exit_code 应保持 ExitSafety(3)（fail-closed），got %v", got["exit_code"])
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
	if !strings.Contains(out, "BUDGET_EXCEEDED") || !strings.Contains(out, "budget_exceeded=tokens") {
		t.Errorf("human 输出应标注 BUDGET_EXCEEDED(budget_exceeded=tokens)，got %q", out)
	}
	if strings.Contains(out, "SAFETY_VIOLATION") {
		t.Errorf("预算超时不应显示 SAFETY_VIOLATION，got %q", out)
	}
}

func TestPrintGCLRunHumanRealSafetyViolation(t *testing.T) {
	var buf bytes.Buffer
	res := gcl.RunResult{ExitCode: gcl.ExitSafety, TracePath: "trace.json"} // 无 BudgetExceeded
	printGCLRunHuman(&buf, "test-skill", res)
	if !strings.Contains(buf.String(), "SAFETY_VIOLATION") {
		t.Errorf("真安全违规应显示 SAFETY_VIOLATION，got %q", buf.String())
	}
}
