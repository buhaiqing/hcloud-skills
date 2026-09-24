package learning

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// --- #T5: the generator must never touch runtime overlays (P0-3 regression) ---

func TestWriteSkillAssets_DoesNotTouchOverlays(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, "huaweicloud-rds-ops", "assets")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	sentinelFP := []byte(`{"sentinel":"runtime-state","patterns":[{"id":"KEEP-ME"}]}`)
	sentinelRP := []byte(`{"sentinel":"runtime-state","playbooks":[{"id":"KEEP-ME","metadata":{"success_rate":0.9}}]}`)
	if err := os.WriteFile(filepath.Join(dir, "failure_patterns.json"), sentinelFP, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "remediation-playbooks.json"), sentinelRP, 0o644); err != nil {
		t.Fatal(err)
	}

	if err := WriteSkillAssets(root, "rds"); err != nil {
		t.Fatalf("WriteSkillAssets: %v", err)
	}

	gotFP, err := os.ReadFile(filepath.Join(dir, "failure_patterns.json"))
	if err != nil || string(gotFP) != string(sentinelFP) {
		t.Errorf("generator mutated failure_patterns overlay: err=%v got=%s", err, gotFP)
	}
	gotRP, err := os.ReadFile(filepath.Join(dir, "remediation-playbooks.json"))
	if err != nil || string(gotRP) != string(sentinelRP) {
		t.Errorf("generator mutated remediation-playbooks overlay: err=%v got=%s", err, gotRP)
	}
}

// --- #T5: gen --check tri-state (vacuous / match / drift / missing) ---

func TestCheckGeneratedAssets(t *testing.T) {
	t.Run("no marker vacuous pass", func(t *testing.T) {
		if err := CheckGeneratedAssets(t.TempDir()); err != nil {
			t.Fatalf("marker-less root must pass vacuously, got %v", err)
		}
	})

	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "docs"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "docs", "gcl-spec.md"), []byte("# gcl"), 0o644); err != nil {
		t.Fatal(err)
	}

	t.Run("marker without seeds fails", func(t *testing.T) {
		if err := CheckGeneratedAssets(root); err == nil {
			t.Fatal("marker root without seeds must fail")
		}
	})

	if _, err := GenerateAll(root); err != nil {
		t.Fatalf("GenerateAll: %v", err)
	}

	t.Run("fresh seeds match", func(t *testing.T) {
		if err := CheckGeneratedAssets(root); err != nil {
			t.Fatalf("fresh seeds must match, got %v", err)
		}
	})

	t.Run("drifted seed fails", func(t *testing.T) {
		p := filepath.Join(root, "huaweicloud-rds-ops", "assets", "failure_patterns.seed.json")
		raw, err := os.ReadFile(p)
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, append(raw, ' '), 0o644); err != nil {
			t.Fatal(err)
		}
		if err := CheckGeneratedAssets(root); err == nil {
			t.Fatal("drifted seed must fail gen --check")
		}
	})

	t.Run("removed seed fails", func(t *testing.T) {
		p := filepath.Join(root, "huaweicloud-vpc-ops", "assets", "remediation-playbooks.seed.json")
		if err := os.Remove(p); err != nil {
			t.Fatal(err)
		}
		if err := CheckGeneratedAssets(root); err == nil {
			t.Fatal("missing seed must fail gen --check")
		}
	})
}

// --- #T3: merged read semantics ---

func TestLoadFailurePatterns_MergesSeedAndOverlay(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, "huaweicloud-rds-ops", "assets")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	seed := `{
  "$schema": "failure-patterns/v1",
  "skill_id": "huaweicloud-rds-ops",
  "patterns": [
    {"id":"RDS-FP001","category":"resource_state","root_cause":"seed cause","fix":{"strategy":"retry","action":"seed fix"},"prevention":"seed prev"},
    {"id":"RDS-FP002","category":"runtime","root_cause":"second","fix":{"strategy":"halt","action":"x"},"prevention":"p"}
  ],
  "meta": {"total_patterns": 2}
}`
	overlay := `{
  "$schema": "failure-patterns/v1",
  "skill_id": "huaweicloud-rds-ops",
  "patterns": [
    {"id":"RDS-FP001","root_cause":"STALE overlay cause","stats":{"occurrence_count":7},"learned_from":["gcl-trace-1.json"]},
    {"id":"RDS-TRACE-001","category":"runtime","provenance":"trace","root_cause":"derived","stats":{"occurrence_count":1},"learned_from":["t2"]}
  ],
  "meta": {"last_aggregation":"2026-09-20T00:00:00Z","source_traces_analyzed":4}
}`
	if err := os.WriteFile(filepath.Join(dir, "failure_patterns.seed.json"), []byte(seed), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "failure_patterns.json"), []byte(overlay), 0o644); err != nil {
		t.Fatal(err)
	}

	got := LoadFailurePatterns(root, "huaweicloud-rds-ops")
	pats, _ := got["patterns"].([]any)
	if len(pats) != 3 {
		t.Fatalf("merged patterns = %d, want 3 (2 seed + 1 trace-derived)", len(pats))
	}
	byID := map[string]map[string]any{}
	for _, p := range pats {
		pm := p.(map[string]any)
		id, _ := pm["id"].(string)
		byID[id] = pm
	}
	// Definitions from seed even though the overlay carries a stale root_cause.
	if rc := byID["RDS-FP001"]["root_cause"]; rc != "seed cause" {
		t.Errorf("RDS-FP001 root_cause = %v, want seed authority (seed cause)", rc)
	}
	// Runtime stats from overlay.
	st, _ := byID["RDS-FP001"]["stats"].(map[string]any)
	if st == nil || st["occurrence_count"].(float64) != 7 {
		t.Errorf("RDS-FP001 stats = %+v, want occurrence_count 7 from overlay", st)
	}
	if lf, _ := byID["RDS-FP001"]["learned_from"].([]any); len(lf) != 1 {
		t.Errorf("RDS-FP001 learned_from = %v, want 1 entry from overlay", lf)
	}
	// Second seed entry untouched.
	if _, ok := byID["RDS-FP002"]; !ok {
		t.Error("RDS-FP002 missing from merged view")
	}
	// Overlay-only (trace-derived) entry appended whole.
	if prov := byID["RDS-TRACE-001"]["provenance"]; prov != "trace" {
		t.Errorf("RDS-TRACE-001 provenance = %v, want trace", prov)
	}
	// Meta: overlay's counters preserved, total recomputed.
	meta, _ := got["meta"].(map[string]any)
	if meta["source_traces_analyzed"].(float64) != 4 {
		t.Errorf("source_traces_analyzed = %v, want 4 (overlay meta)", meta["source_traces_analyzed"])
	}
	if meta["total_patterns"].(float64) != 3 {
		t.Errorf("total_patterns = %v, want 3 (merged length)", meta["total_patterns"])
	}
}

func TestLoadFailurePatterns_SeedWithoutOverlay(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, "huaweicloud-rds-ops", "assets")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	seed := `{"$schema":"failure-patterns/v1","skill_id":"huaweicloud-rds-ops","patterns":[{"id":"RDS-FP001","category":"runtime","root_cause":"only seed"}],"meta":{"total_patterns":1}}`
	if err := os.WriteFile(filepath.Join(dir, "failure_patterns.seed.json"), []byte(seed), 0o644); err != nil {
		t.Fatal(err)
	}

	got := LoadFailurePatterns(root, "huaweicloud-rds-ops")
	pats, _ := got["patterns"].([]any)
	if len(pats) != 1 {
		t.Fatalf("fresh clone (seed only) patterns = %d, want 1", len(pats))
	}
	meta, _ := got["meta"].(map[string]any)
	if meta["source_traces_analyzed"].(float64) != 0 {
		t.Errorf("source_traces_analyzed = %v, want 0 scaffold", meta["source_traces_analyzed"])
	}
	if meta["total_patterns"].(float64) != 1 {
		t.Errorf("total_patterns = %v, want 1", meta["total_patterns"])
	}
}

func TestLoadFailurePatterns_NoSeedLegacyCompat(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, "huaweicloud-ecs-ops", "assets")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	// Legacy full-document overlay (the 22 non-Products skills): no seed at all.
	legacy := `{"$schema":"failure-patterns/v1","skill_id":"huaweicloud-ecs-ops","patterns":[{"id":"ECS-FP001","category":"runtime","root_cause":"legacy","stats":{"occurrence_count":2}}],"meta":{"source_traces_analyzed":9,"total_patterns":1}}`
	if err := os.WriteFile(filepath.Join(dir, "failure_patterns.json"), []byte(legacy), 0o644); err != nil {
		t.Fatal(err)
	}

	got := LoadFailurePatterns(root, "huaweicloud-ecs-ops")
	pats, _ := got["patterns"].([]any)
	if len(pats) != 1 {
		t.Fatalf("legacy overlay patterns = %d, want 1 (pre-split behavior)", len(pats))
	}
	pm := pats[0].(map[string]any)
	if pm["root_cause"] != "legacy" {
		t.Errorf("root_cause = %v, want legacy whole-document serve", pm["root_cause"])
	}
	meta, _ := got["meta"].(map[string]any)
	if meta["source_traces_analyzed"].(float64) != 9 {
		t.Errorf("source_traces_analyzed = %v, want 9 (overlay meta untouched)", meta["source_traces_analyzed"])
	}
}

// --- #T4: split write semantics ---

func TestSaveFailurePatterns_ReducesSeedEntries(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, "huaweicloud-rds-ops", "assets")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	seed := `{"$schema":"failure-patterns/v1","skill_id":"huaweicloud-rds-ops","patterns":[{"id":"RDS-FP001","category":"runtime","root_cause":"seed cause"},{"id":"RDS-FP002","category":"runtime","root_cause":"second"}],"meta":{"total_patterns":2}}`
	seedPath := filepath.Join(dir, "failure_patterns.seed.json")
	if err := os.WriteFile(seedPath, []byte(seed), 0o644); err != nil {
		t.Fatal(err)
	}
	seedBefore, err := os.ReadFile(seedPath)
	if err != nil {
		t.Fatal(err)
	}

	data := map[string]any{
		"$schema":  "failure-patterns/v1",
		"skill_id": "huaweicloud-rds-ops",
		"patterns": []any{
			map[string]any{
				"id":           "RDS-FP001",
				"root_cause":   "seed cause",
				"stats":        map[string]any{"occurrence_count": float64(3)},
				"learned_from": []any{"t1"},
			},
			map[string]any{
				"id":         "RDS-TRACE-001",
				"category":   "runtime",
				"root_cause": "derived whole entry",
			},
		},
		"meta": map[string]any{"source_traces_analyzed": float64(3)},
	}
	written, err := SaveFailurePatterns(root, "huaweicloud-rds-ops", data)
	if err != nil {
		t.Fatalf("SaveFailurePatterns: %v", err)
	}
	raw, err := os.ReadFile(written)
	if err != nil {
		t.Fatal(err)
	}
	var out map[string]any
	if err := json.Unmarshal(raw, &out); err != nil {
		t.Fatalf("overlay not valid JSON: %v", err)
	}
	pats, _ := out["patterns"].([]any)
	if len(pats) != 2 {
		t.Fatalf("overlay patterns = %d, want 2", len(pats))
	}
	seedPartial := pats[0].(map[string]any)
	if _, ok := seedPartial["root_cause"]; ok {
		t.Errorf("seed-owned entry must be reduced to runtime fields, got %+v", seedPartial)
	}
	if id := seedPartial["id"]; id != "RDS-FP001" {
		t.Errorf("overlay[0].id = %v, want RDS-FP001", id)
	}
	if st, _ := seedPartial["stats"].(map[string]any); st == nil {
		t.Errorf("seed partial lost stats: %+v", seedPartial)
	}
	traceEntry := pats[1].(map[string]any)
	if traceEntry["root_cause"] != "derived whole entry" {
		t.Errorf("non-seed entry must be written whole, got %+v", traceEntry)
	}
	meta, _ := out["meta"].(map[string]any)
	if meta["last_aggregation"] == nil || meta["last_aggregation"] == "" {
		t.Errorf("last_aggregation not set on save: %+v", meta)
	}
	if meta["source_traces_analyzed"].(float64) != 3 {
		t.Errorf("source_traces_analyzed = %v, want 3 preserved", meta["source_traces_analyzed"])
	}
	// Seed file untouched by save.
	seedAfter, err := os.ReadFile(seedPath)
	if err != nil || string(seedAfter) != string(seedBefore) {
		t.Errorf("save mutated the seed file: err=%v", err)
	}
}

// --- #T3/#T4: playbook merge + seed-only metadata append ---

func TestLoadPlaybooks_MergesSeedDefinitionsWithOverlayMetadata(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, "huaweicloud-ecs-ops", "assets")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	seed := `{"$schema":"remediation-playbooks/v1","skill_id":"huaweicloud-ecs-ops","playbooks":[{"id":"ECS-R001","name":"seed name","remediation":{"risk_level":"low","auto_execute_threshold":0.7,"execute":"seed-cmd"}}]}`
	overlay := `{"$schema":"remediation-playbooks/v1","skill_id":"huaweicloud-ecs-ops","playbooks":[{"id":"ECS-R001","name":"STALE","metadata":{"success_rate":0.95}},{"id":"ECS-RX","name":"overlay only"}]}`
	if err := os.WriteFile(filepath.Join(dir, "remediation-playbooks.seed.json"), []byte(seed), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "remediation-playbooks.json"), []byte(overlay), 0o644); err != nil {
		t.Fatal(err)
	}

	pbs, err := LoadPlaybooks(root, "huaweicloud-ecs-ops")
	if err != nil {
		t.Fatalf("LoadPlaybooks: %v", err)
	}
	if len(pbs) != 2 {
		t.Fatalf("merged playbooks = %d, want 2 (1 seed + 1 overlay-only)", len(pbs))
	}
	byID := map[string]RemediationPlaybook{}
	for _, pb := range pbs {
		byID[pb.ID] = pb
	}
	x := byID["ECS-R001"]
	if x.Name != "seed name" {
		t.Errorf("ECS-R001 name = %q, want seed authority (seed name)", x.Name)
	}
	if x.Remediation.Execute != "seed-cmd" {
		t.Errorf("ECS-R001 execute = %q, want seed-cmd", x.Remediation.Execute)
	}
	if x.Metadata == nil || x.Metadata["success_rate"].(float64) != 0.95 {
		t.Errorf("ECS-R001 metadata = %+v, want success_rate 0.95 from overlay", x.Metadata)
	}
	if y := byID["ECS-RX"]; y.Name != "overlay only" {
		t.Errorf("overlay-only playbook = %+v, want served whole", y)
	}
}

func TestRecordPlaybookOutcome_SeedOnlyAppendsPartial(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, "huaweicloud-ecs-ops", "assets")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	seed := `{"$schema":"remediation-playbooks/v1","skill_id":"huaweicloud-ecs-ops","playbooks":[{"id":"ECS-R001","name":"seed","remediation":{"risk_level":"low","execute":"x"}}]}`
	seedPath := filepath.Join(dir, "remediation-playbooks.seed.json")
	if err := os.WriteFile(seedPath, []byte(seed), 0o644); err != nil {
		t.Fatal(err)
	}
	seedBefore, _ := os.ReadFile(seedPath)

	if err := RecordPlaybookOutcome(root, "huaweicloud-ecs-ops", "ECS-R001", true); err != nil {
		t.Fatalf("first RecordPlaybookOutcome: %v", err)
	}
	pbs, err := LoadPlaybooks(root, "huaweicloud-ecs-ops")
	if err != nil || len(pbs) != 1 {
		t.Fatalf("after append load: pbs=%d err=%v, want 1", len(pbs), err)
	}
	rate1 := pbs[0].Metadata["success_rate"].(float64)
	if rate1 <= 0 || rate1 > 0.11 {
		t.Errorf("first bootstrap rate = %v, want ~0.1", rate1)
	}

	if err := RecordPlaybookOutcome(root, "huaweicloud-ecs-ops", "ECS-R001", true); err != nil {
		t.Fatalf("second RecordPlaybookOutcome: %v", err)
	}
	pbs, _ = LoadPlaybooks(root, "huaweicloud-ecs-ops")
	rate2 := pbs[0].Metadata["success_rate"].(float64)
	if rate2 <= rate1 {
		t.Errorf("second rate = %v, want EWMA rise over %v", rate2, rate1)
	}
	// Definition still from seed after overlay writes.
	if pbs[0].Name != "seed" {
		t.Errorf("name after outcome = %q, want seed", pbs[0].Name)
	}
	// Seed file untouched.
	seedAfter, _ := os.ReadFile(seedPath)
	if string(seedAfter) != string(seedBefore) {
		t.Error("RecordPlaybookOutcome mutated the seed file")
	}

	// Unknown ID with no seed knowledge stays a no-op.
	if err := RecordPlaybookOutcome(root, "huaweicloud-ecs-ops", "NOPE", true); err != nil {
		t.Fatalf("unknown id must no-op: %v", err)
	}
}

// --- #T6: pitfall report reads through the merged loader ---

func TestGeneratePitfallReport_ReadsSeedWithoutOverlay(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, "huaweicloud-ecs-ops", "assets")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	// Fresh-clone shape: tracked seed only, gitignored overlay absent.
	seed := `{"$schema":"failure-patterns/v1","skill_id":"huaweicloud-ecs-ops","patterns":[{"id":"ECS-FP001","category":"runtime","root_cause":"seed lesson cause","prevention":"Seed lesson to avoid reuse"}],"meta":{"total_patterns":1}}`
	if err := os.WriteFile(filepath.Join(dir, "failure_patterns.seed.json"), []byte(seed), 0o644); err != nil {
		t.Fatal(err)
	}

	count, err := GeneratePitfallReport(root)
	if err != nil {
		t.Fatalf("GeneratePitfallReport: %v", err)
	}
	if count != 1 {
		t.Fatalf("pitfall entries = %d, want 1 (seed-served)", count)
	}
	report, err := os.ReadFile(filepath.Join(root, "huaweicloud-skill-generator", "references", "common-pitfalls.md"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(report), "Seed lesson to avoid reuse") {
		t.Errorf("report missing seed prevention:\n%s", report)
	}
}
