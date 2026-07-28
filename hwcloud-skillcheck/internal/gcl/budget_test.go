package gcl

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestBudgetTokensHard(t *testing.T) {
	marker := filepath.Join(t.TempDir(), "ran")
	result := Run(RunConfig{
		Skill:   "huaweicloud-ecs-ops",
		Request: "list all ECS servers in the project",
		Command: "touch " + marker,
		Root:    t.TempDir(),
		Budget:  ResourceBudget{Tokens: 1},
	})
	if result.ExitCode != ExitSafety || result.BudgetExceeded != "tokens" {
		t.Fatalf("got %+v, want token SAFETY_FAIL", result)
	}
	if _, err := os.Stat(marker); !os.IsNotExist(err) {
		t.Fatal("generator ran after token budget was exceeded")
	}
}

func TestBudgetToolCallsHard(t *testing.T) {
	marker := filepath.Join(t.TempDir(), "ran")
	result := Run(RunConfig{
		Skill:   "huaweicloud-ecs-ops",
		Request: "smoke",
		Command: "touch " + marker,
		Root:    t.TempDir(),
		Budget:  ResourceBudget{ToolCalls: -1},
	})
	if result.ExitCode != ExitSafety || result.BudgetExceeded != "tool_calls" {
		t.Fatalf("got %+v, want tool_calls SAFETY_FAIL", result)
	}
	if _, err := os.Stat(marker); !os.IsNotExist(err) {
		t.Fatal("generator ran after tool-call budget was exceeded")
	}
}

func TestBudgetWallClockHard(t *testing.T) {
	result := Run(RunConfig{
		Skill:   "huaweicloud-ecs-ops",
		Request: "smoke",
		Command: "sleep 1",
		Root:    t.TempDir(),
		Budget:  ResourceBudget{WallClock: 20 * time.Millisecond},
	})
	if result.ExitCode != ExitSafety || result.BudgetExceeded != "wall_clock" {
		t.Fatalf("got %+v, want wall_clock SAFETY_FAIL", result)
	}
}
