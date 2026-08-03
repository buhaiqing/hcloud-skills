package gcl

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func asDefault(d HallucinationDetector) *DefaultHallucinationDetector {
	return d.(*DefaultHallucinationDetector)
}

// ---- L1: CLI parameter existence -----------------------------------------

func TestL1_ValidFlags(t *testing.T) {
	dir := t.TempDir()
	cliDir := filepath.Join(dir, "huaweicloud-ecs-ops", "references")
	if err := os.MkdirAll(cliDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(cliDir, "cli-usage.md"), []byte(`
## hcloud ecs start-server
--server-id   string   required  ECS instance ID.
--wait        flag     optional Wait for server to reach running state.
`), 0644); err != nil {
		t.Fatal(err)
	}

	detector := NewHallucinationDetector(filepath.Join(dir, "huaweicloud-ecs-ops"))
	result, err := asDefault(detector).checkCLIParams("hcloud ecs start-server --server-id xxx --wait")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Blocked {
		t.Errorf("expected not blocked, got blocked with invalid flags: %v", result.InvalidFlags)
	}
}

func TestL1_InvalidFlag(t *testing.T) {
	dir := t.TempDir()
	cliDir := filepath.Join(dir, "huaweicloud-ecs-ops", "references")
	if err := os.MkdirAll(cliDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(cliDir, "cli-usage.md"), []byte(`--server-id   string   required`), 0644); err != nil {
		t.Fatal(err)
	}

	detector := NewHallucinationDetector(filepath.Join(dir, "huaweicloud-ecs-ops"))
	result, err := asDefault(detector).checkCLIParams("hcloud ecs start-server --server-id xxx --no-wait")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Blocked {
		t.Error("expected blocked for invalid --no-wait flag")
	}
	if len(result.InvalidFlags) != 1 || result.InvalidFlags[0] != "--no-wait" {
		t.Errorf("expected [--no-wait], got %v", result.InvalidFlags)
	}
}

func TestL1_NoFlags(t *testing.T) {
	detector := NewHallucinationDetector(t.TempDir())
	result, err := asDefault(detector).checkCLIParams("hcloud ecs list-servers")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Blocked {
		t.Error("expected not blocked for no flags")
	}
}

// ---- L2: JSON structure compliance ---------------------------------------

func TestL2_NoOutput(t *testing.T) {
	detector := NewHallucinationDetector("/nonexistent")
	result, err := asDefault(detector).checkJSONStructure("")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Blocked {
		t.Error("expected not blocked for empty output")
	}
}

func TestL2_ValidJSON(t *testing.T) {
	dir := t.TempDir()
	schemaDir := filepath.Join(dir, "huaweicloud-ecs-ops", "references")
	if err := os.MkdirAll(schemaDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(schemaDir, "openapi-schema.json"),
		[]byte(`{"type":"object"}`), 0644); err != nil {
		t.Fatal(err)
	}

	detector := NewHallucinationDetector(filepath.Join(dir, "huaweicloud-ecs-ops"))
	result, err := asDefault(detector).checkJSONStructure(`{"server_id":"ecs-xxx","status":"SHUTOFF"}`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Blocked {
		t.Errorf("expected not blocked, got errors: %v", result.Errors)
	}
}

// ---- L3: WAF compliance -------------------------------------------------

func TestL3_NoViolation(t *testing.T) {
	d := NewHallucinationDetector("/nonexistent")
	result, err := asDefault(d).checkWAF(GeneratorOutput{
		Command:       "hcloud ecs start-server --server-id xxx --wait",
		ResultExcerpt: `{"server_id":"ecs-xxx","status":"ACTIVE"}`,
	}, &GCLTrace{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Violations) != 0 {
		t.Errorf("expected no violations, got: %v", result.Violations)
	}
}

func TestL3_CredentialExposure(t *testing.T) {
	d := NewHallucinationDetector("/nonexistent")
	result, err := asDefault(d).checkWAF(GeneratorOutput{
		Command:       "hcloud ecs describe",
		ResultExcerpt: `AK=ABCDEFGHIJKLMNOPQRSTUVWXYZ1234567890`,
	}, &GCLTrace{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Violations) == 0 {
		t.Error("expected credential exposure violation")
	}
}

func TestL3_DangerousVerbWithoutGuard(t *testing.T) {
	d := NewHallucinationDetector("/nonexistent")
	result, err := asDefault(d).checkWAF(GeneratorOutput{
		Command:       "hcloud ecs delete-server --server-id xxx",
		ResultExcerpt: `{}`,
	}, &GCLTrace{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	var found bool
	for _, v := range result.Violations {
		if v.Rule == "dangerous-verb-without-guard" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected dangerous-verb violation, got: %v", result.Violations)
	}
}

func TestL3_DangerousVerbWithGuard(t *testing.T) {
	d := NewHallucinationDetector("/nonexistent")
	result, err := asDefault(d).checkWAF(GeneratorOutput{
		Command:       "hcloud ecs delete-server --server-id xxx --dry-run",
		ResultExcerpt: `{}`,
	}, &GCLTrace{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	for _, v := range result.Violations {
		if v.Rule == "dangerous-verb-without-guard" {
			t.Errorf("expected no dangerous-verb violation with guard, got: %v", result.Violations)
			return
		}
	}
}

func TestL3_HighCostResource(t *testing.T) {
	d := NewHallucinationDetector("/nonexistent")
	result, err := asDefault(d).checkWAF(GeneratorOutput{
		Command:       "hcloud ecs stop-server --server-id xxx",
		ResultExcerpt: `{}`,
	}, &GCLTrace{
		ResourceContext: map[string]any{
			"monthly_cost_usd": float64(2000),
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	for _, v := range result.Violations {
		if v.Rule == "high-cost-resource" {
			return
		}
	}
	t.Errorf("expected high-cost-resource violation, got: %v", result.Violations)
}

func TestL3_MultiResourceWithoutRollback(t *testing.T) {
	d := NewHallucinationDetector("/nonexistent")
	result, err := asDefault(d).checkWAF(GeneratorOutput{
		Command:       "hcloud ecs stop-servers --all",
		ResultExcerpt: `{}`,
	}, &GCLTrace{
		OperationIntent: map[string]any{
			"blast_radius": "multi-resource",
			// no rollback_plan
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	for _, v := range result.Violations {
		if v.Rule == "no-rollback-plan" {
			return
		}
	}
	t.Errorf("expected no-rollback-plan violation, got: %v", result.Violations)
}

// ---- HallucinationResult.BlockedBySafety --------------------------------

func TestBlockedBySafety_L1(t *testing.T) {
	h := &HallucinationResult{L1: &L1Result{Blocked: true}, L2: &L2Result{Blocked: false}}
	if !h.BlockedBySafety() {
		t.Error("expected blocked by safety when L1 is blocked")
	}
}

func TestBlockedBySafety_L2(t *testing.T) {
	h := &HallucinationResult{L1: &L1Result{Blocked: false}, L2: &L2Result{Blocked: true}}
	if !h.BlockedBySafety() {
		t.Error("expected blocked by safety when L2 is blocked")
	}
}

func TestBlockedBySafety_L3Only(t *testing.T) {
	h := &HallucinationResult{L1: &L1Result{Blocked: false}, L2: &L2Result{Blocked: false}, L3: &L3Result{Blocked: true}}
	if h.BlockedBySafety() {
		t.Error("L3 violations should not auto-block by safety")
	}
}

func TestBlockedBySafety_Nil(t *testing.T) {
	h := &HallucinationResult{}
	if h.BlockedBySafety() {
		t.Error("nil layers should not be blocked")
	}
}

// ---- Full integration: Run -----------------------------------------------

func TestHallucinationDetector_Run(t *testing.T) {
	dir := t.TempDir()
	cliDir := filepath.Join(dir, "huaweicloud-ecs-ops", "references")
	if err := os.MkdirAll(cliDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(cliDir, "cli-usage.md"),
		[]byte(`--server-id --wait --dry-run`), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(cliDir, "openapi-schema.json"),
		[]byte(`{"type":"object","properties":{"server_id":{"type":"string"}}}`), 0644); err != nil {
		t.Fatal(err)
	}

	detector := NewHallucinationDetector(filepath.Join(dir, "huaweicloud-ecs-ops"))
	result, err := detector.Run(context.Background(), GeneratorOutput{
		Command:       "hcloud ecs start-server --server-id ecs-xxx --wait",
		ResultExcerpt: `{"server_id":"ecs-xxx","status":"ACTIVE"}`,
	}, &GCLTrace{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Blocked {
		t.Errorf("expected not blocked: %s", result.Summary)
	}
	if result.L1.Blocked || result.L2.Blocked {
		t.Errorf("L1/L2 should not be blocked: L1=%v L2=%v", result.L1.Blocked, result.L2.Blocked)
	}
}
