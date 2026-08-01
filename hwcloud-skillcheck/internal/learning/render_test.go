package learning

import (
	"strings"
	"testing"
)

func TestRenderOutput_ReplacesPlaceholders(t *testing.T) {
	outputs := map[string]string{
		"instance_id": "i-abc123",
		"region":      "cn-north-4",
	}
	in := "hcloud RDS showInstance --instance_id {{output.instance_id}} --region {{output.region}}"
	got, ok, err := RenderOutput(in, outputs)
	if err != nil {
		t.Fatalf("RenderOutput: %v", err)
	}
	if !ok {
		t.Fatal("RenderOutput returned ok=false for fully-renderable input")
	}
	if strings.Contains(got, "{{") {
		t.Errorf("unresolved placeholder remains: %q", got)
	}
	if !strings.Contains(got, "i-abc123") || !strings.Contains(got, "cn-north-4") {
		t.Errorf("placeholder not substituted: %q", got)
	}
}

func TestRenderOutput_MissingKeyBlocks(t *testing.T) {
	outputs := map[string]string{}
	_, ok, err := RenderOutput("hcloud X --id {{output.id}}", outputs)
	if err == nil {
		t.Fatal("expected error for missing {{output.id}}")
	}
	if ok {
		t.Fatal("ok must be false when a placeholder is missing")
	}
}

func TestRenderOutput_EnvPlaceholder(t *testing.T) {
	t.Setenv("HW_REGION_ID", "cn-north-9")
	in := "hcloud ECS listServers --region {{env.HW_REGION_ID}}"
	got, ok, err := RenderOutput(in, nil)
	if err != nil {
		t.Fatalf("RenderOutput: %v", err)
	}
	if !ok {
		t.Fatal("env placeholder should resolve")
	}
	if !strings.Contains(got, "cn-north-9") {
		t.Errorf("env placeholder not substituted: %q", got)
	}
}

func TestRenderOutput_NoPlaceholderPassthrough(t *testing.T) {
	in := "hcloud ECS listServers"
	got, ok, err := RenderOutput(in, nil)
	if err != nil {
		t.Fatalf("RenderOutput: %v", err)
	}
	if !ok || got != in {
		t.Fatalf("no-placeholder input must pass through unchanged, ok=%v got=%q", ok, got)
	}
}

func TestEvalPreconditions_AllPass(t *testing.T) {
	// runFn returns (0, "", nil) for any command → all preconditions pass.
	run := func(cmd string) (int, string, error) { return 0, "", nil }
	ok, failed := EvalPreconditions([]string{"hcloud A", "hcloud B"}, nil, run)
	if !ok {
		t.Fatalf("expected all preconditions pass, failed=%v", failed)
	}
	if len(failed) != 0 {
		t.Fatalf("expected 0 failed, got %v", failed)
	}
}

func TestEvalPreconditions_OneFails(t *testing.T) {
	run := func(cmd string) (int, string, error) {
		if strings.Contains(cmd, "B") {
			return 1, "boom", nil
		}
		return 0, "", nil
	}
	ok, failed := EvalPreconditions([]string{"hcloud A", "hcloud B"}, nil, run)
	if ok {
		t.Fatal("expected ok=false when one precondition fails")
	}
	if len(failed) != 1 || failed[0] != "hcloud B" {
		t.Fatalf("expected [hcloud B] failed, got %v", failed)
	}
}
