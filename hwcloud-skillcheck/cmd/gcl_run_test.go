package cmd

import (
	"bytes"
	"encoding/json"
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
