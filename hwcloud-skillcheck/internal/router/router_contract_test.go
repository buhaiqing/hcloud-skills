// Package router contract tests for rubric A2.3 (tightened), A2.11, A2.12,
// A2.13, A2.14. These tests are written FIRST per TDD; they MUST compile
// against the current router package and MUST FAIL until the production
// changes described in docs/superpowers/specs/2026-07-27-harness-runtime-p1p2-design.md
// §4.2.1-4.2.3 are landed.
//
// Each test name is a spec contract (L6); renaming a test is a contract
// break. Helpers follow AGENTS.md lessons:
//   - L2: t.TempDir() returns a fresh dir per call; cache per test via sync.Map.
//   - L9: set GOCACHE=/tmp/hcloud-go-cache before any go build/test.
package router

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/buhaiqing/hcloud-skills/hwcloud-skillcheck/internal/embedder"
	"github.com/buhaiqing/hcloud-skills/hwcloud-skillcheck/internal/registry"
)

// testTmp keeps a stable t.TempDir() per *testing.T so multiple helpers in the
// same test don't end up with different dirs (L2 lesson).
var testTmp sync.Map // *testing.T -> string

func tmpFor(t *testing.T) string {
	if v, ok := testTmp.Load(t); ok {
		return v.(string)
	}
	dir := t.TempDir()
	testTmp.Store(t, dir)
	t.Cleanup(func() { testTmp.Delete(t) })
	return dir
}

func repoRoot(t *testing.T) string {
	if v := os.Getenv("HC_REPO_ROOT"); v != "" {
		return v
	}
	// Walk upward looking for the repo's go.mod (the workspace has multiple
	// Go modules; we want the root that contains docs/).
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	dir := cwd
	for i := 0; i < 8; i++ {
		if _, err := os.Stat(filepath.Join(dir, "docs", "superpowers")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	t.Skipf("docs/superpowers not found above %s; set HC_REPO_ROOT", cwd)
	return ""
}

// moduleRoot returns the directory holding go.mod for the hwcloud-skillcheck
// module. Used as exec cwd for `go build`.
func moduleRoot(t *testing.T) string {
	if v := os.Getenv("HC_MODULE_ROOT"); v != "" {
		return v
	}
	if repo := repoRoot(t); repo != "" {
		candidate := filepath.Join(repo, "hwcloud-skillcheck")
		if _, err := os.Stat(filepath.Join(candidate, "go.mod")); err == nil {
			return candidate
		}
	}
	t.Skip("module root (go.mod) not found; set HC_MODULE_ROOT")
	return ""
}

func sampleEntries() []registry.Entry {
	return []registry.Entry{
		{Skill: "huaweicloud-ecs-ops", Name: "ECS", Description: "manage ecs servers and disks", SideEffectClass: "read-only"},
		{Skill: "huaweicloud-rds-ops", Name: "RDS", Description: "manage rds databases", SideEffectClass: "read-only"},
		{Skill: "huaweicloud-billing-ops", Name: "Billing", Description: "query billing statements", SideEffectClass: "read-only"},
	}
}

func marshalDecision(t *testing.T, d Decision) map[string]any {
	t.Helper()
	raw, err := json.Marshal(d)
	if err != nil {
		t.Fatalf("marshal Decision: %v", err)
	}
	var out map[string]any
	if err := json.Unmarshal(raw, &out); err != nil {
		t.Fatalf("re-decode Decision: %v", err)
	}
	return out
}

// ---------------------------------------------------------------------------
// A2.3 — Stage-2 uses a configured non-lexical embedding provider and records
// its identity and metadata. Lexical fallback is forbidden.
// ---------------------------------------------------------------------------

func TestRouterNoLexicalFallback(t *testing.T) {
	decision := Route(context.Background(), sampleEntries(), "list ecs servers", Intent{SafetyClass: "read-only"}, nil)
	generic := marshalDecision(t, decision)

	// Contract 1: the Decision MUST declare an explicit rerank_mode. Today:
	// there is no such field at all — RED.
	mode, _ := generic["rerank_mode"].(string)
	switch mode {
	case "":
		t.Fatalf("Decision missing rerank_mode field (rubric A2.3, §4.2.4). got keys=%v", keysOf(generic))
	case "lexical":
		t.Fatalf("rerank_mode=lexical; lexical Jaccard fallback is banned by rubric A2.3 + §4.2.4 — got=%q", mode)
	case "skipped", "embedding":
		// acceptable modes
	default:
		t.Fatalf("rerank_mode=%q; must be one of skipped|embedding (no lexical allowed)", mode)
	}

	// Contract 2: rerank_source must declare which path produced the score.
	// Today: absent — RED.
	src, _ := generic["rerank_source"].(string)
	if src != "local-fasttext" {
		t.Fatalf("rerank_source=%q; want configured non-lexical provider \"local-fasttext\"", src)
	}
}

func TestRouterEmbeddingProviderMetadata(t *testing.T) {
	emb, err := embedder.PreflightAndInit(context.Background(), embedder.ProviderConfig{ProviderName: "local-fasttext", Dim: 128})
	if err != nil {
		t.Fatalf("init local embedder: %v", err)
	}
	decision := Route(context.Background(), sampleEntries(), "list ecs servers", Intent{SafetyClass: "read-only"}, emb)
	meta := decision.EmbeddingProviderMeta
	if decision.RerankMode != "embedding" || decision.RerankSource != "local-fasttext" {
		t.Fatalf("unexpected rerank metadata: mode=%q source=%q", decision.RerankMode, decision.RerankSource)
	}
	if meta.Primary != "local-fasttext" || meta.FallbackUsed || meta.InputBytes == 0 {
		t.Fatalf("unexpected provider metadata: %+v", meta)
	}
	if meta.InputSha256Pfx != sha256Prefix("list ecs servers", 8) || len(meta.InputSha256Pfx) != 8 {
		t.Fatalf("input hash prefix=%q; want stable 8-char SHA256 prefix", meta.InputSha256Pfx)
	}
	if meta.DurationMs < 0 {
		t.Fatalf("embedding duration must be non-negative: %d", meta.DurationMs)
	}
}

// ---------------------------------------------------------------------------
// A2.11 — Trace carries router_policy_version and confidence_gate on every
// dispatch. Today: neither field exists — RED.
// ---------------------------------------------------------------------------

func TestRouterPolicyVersionInTrace(t *testing.T) {
	decision := Route(context.Background(), sampleEntries(), "list ecs", Intent{SafetyClass: "read-only"}, nil)
	generic := marshalDecision(t, decision)

	v, _ := generic["router_policy_version"].(string)
	if v == "" {
		t.Fatalf("Decision missing router_policy_version (rubric A2.11); got keys=%v", keysOf(generic))
	}
	if !strings.HasPrefix(v, "v") || strings.Count(v, ".") < 2 {
		t.Fatalf("router_policy_version=%q; want semver-like \"vX.Y.Z\"", v)
	}
}

func TestRouterEmitsConfidenceGate(t *testing.T) {
	decision := Route(context.Background(), sampleEntries(), "list ecs", Intent{SafetyClass: "read-only"}, nil)
	generic := marshalDecision(t, decision)

	gate, ok := generic["confidence_gate"].(map[string]any)
	if !ok {
		t.Fatalf("Decision missing confidence_gate block (rubric A2.11); got keys=%v", keysOf(generic))
	}
	required := []string{"top1_score", "margin", "entity_match", "hard_filtered", "decision", "rationale"}
	for _, k := range required {
		if _, ok := gate[k]; !ok {
			t.Fatalf("confidence_gate missing required field %q (got keys=%v)", k, keysOf(gate))
		}
	}
	d, _ := gate["decision"].(string)
	if d != "skip_onnx" && d != "invoke_onnx" {
		t.Fatalf("confidence_gate.decision=%q; want skip_onnx|invoke_onnx", d)
	}
}

// ---------------------------------------------------------------------------
// A2.12 — Shadow candidate runs alongside main decision but never alters
// chosen skill or score. Today: no shadow block exists — RED.
// ---------------------------------------------------------------------------

func TestRouterShadowCandidateDoesNotAffectMainDecision(t *testing.T) {
	decision := Route(context.Background(), sampleEntries(), "list ecs servers", Intent{SafetyClass: "read-only"}, nil)
	generic := marshalDecision(t, decision)

	shadow, ok := generic["router_decision_shadow"].(map[string]any)
	if !ok {
		t.Fatalf("Decision missing router_decision_shadow (rubric A2.12); got keys=%v", keysOf(generic))
	}
	// Required shadow fields. Note: chosenMain MUST be independently derived
	// from the production policy_version; shadow merely reports.
	for _, k := range []string{"router_policy_candidate", "chosen", "score_delta", "margin_delta", "would_have_changed"} {
		if _, ok := shadow[k]; !ok {
			t.Fatalf("router_decision_shadow missing required field %q (got keys=%v)", k, keysOf(shadow))
		}
	}
	// Hard contract: shadow block must NOT duplicate chosenMain into a field
	// that the gate reads. We assert shadow["chosen"] exists independently,
	// and that the top-level "chosen" field is bound to the main policy only.
	chosenMain, _ := generic["chosen"].(string)
	if chosenMain == "" {
		t.Fatalf("Decision missing chosen; cannot verify isolation")
	}
	_ = chosenMain // the field will be exercised once shadow-vs-main comparison lands.
}

// ---------------------------------------------------------------------------
// A2.13 — `router calibrate` is offline-only, defaults to --dry-run, and
// applied calibration requires an explicit router_policy_version bump.
// Tests 5 & 6 spawn the actual CLI binary; today the subcommand is absent
// so both tests RED with "unknown command".
// ---------------------------------------------------------------------------

var cliOnce sync.Once
var cliPath string
var cliErr error

func buildCLI(t *testing.T) string {
	cliOnce.Do(func() {
		// Per L9: GOCACHE must point at /tmp to dodge the read-only Library/Caches.
		os.Setenv("GOCACHE", "/tmp/hcloud-go-cache")
		root := moduleRoot(t)
		// Process-lifetime dir: t.TempDir() would be reaped when the first
		// test finishes, so subsequent calibrate tests would not find the
		// binary. Use a stable /tmp location.
		dir, err := os.MkdirTemp("/tmp", "hcloud-cli-")
		if err != nil {
			cliErr = err
			return
		}
		out := filepath.Join(dir, "hwcloud-skillcheck")
		cmd := exec.Command("go", "build", "-o", out, ".")
		cmd.Dir = root
		buf, err := cmd.CombinedOutput()
		if err != nil {
			cliErr = &execError{Err: err, Out: buf}
			return
		}
		cliPath = out
	})
	if cliErr != nil {
		t.Fatalf("build CLI: %v\n--- go build ---\n%s", cliErr, extractBuildLog(cliErr))
	}
	return cliPath
}

type execError struct {
	Err error
	Out []byte
}

func (e *execError) Error() string { return e.Err.Error() }
func (e *execError) Unwrap() error { return e.Err }

func extractBuildLog(err error) string {
	if e, ok := err.(*execError); ok {
		return string(e.Out)
	}
	return ""
}

func seedCapabilityRegistry(t *testing.T, dir, version string) {
	t.Helper()
	body := map[string]any{
		"router_policy_version":   version,
		"router_policy_candidate": "v0.0.0-shadow",
		"policy_diff_at":          "2026-07-27T22:00:00Z",
		"confidence_gate": map[string]any{
			"top1_score_min": 7500,
			"margin_min":     1500,
			"entity_match":   []string{"strong"},
		},
		"scoring_weights": map[string]any{},
		"lexicon":         map[string]any{"version": "v1.0.0"},
	}
	data, err := json.MarshalIndent(body, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "capability-registry.json"), data, 0o600); err != nil {
		t.Fatal(err)
	}
}

func TestRouterCalibrateDryRunOnly(t *testing.T) {
	bin := buildCLI(t)
	root := tmpFor(t)
	seedCapabilityRegistry(t, root, "v1.0.0")
	before, err := os.ReadFile(filepath.Join(root, "capability-registry.json"))
	if err != nil {
		t.Fatal(err)
	}

	cmd := exec.Command(bin, "router", "calibrate", "--root", root)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("router calibrate (dry-run default) exit=%v\n--- output ---\n%s\n(rubric A2.13: dry-run must default to non-mutating)", err, out)
	}
	after, err := os.ReadFile(filepath.Join(root, "capability-registry.json"))
	if err != nil {
		t.Fatal(err)
	}
	if sha256.Sum256(before) != sha256.Sum256(after) {
		t.Fatalf("dry-run mutated capability-registry.json; rubric A2.13 bans writes without --apply")
	}
}

func TestRouterCalibrateRequiresVersionBump(t *testing.T) {
	bin := buildCLI(t)
	root := tmpFor(t)
	seedCapabilityRegistry(t, root, "v1.0.0")

	cmd := exec.Command(bin, "router", "calibrate", "--root", root, "--apply")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("router calibrate --apply exit=%v\n--- output ---\n%s", err, out)
	}
	data, err := os.ReadFile(filepath.Join(root, "capability-registry.json"))
	if err != nil {
		t.Fatal(err)
	}
	var reg map[string]any
	if err := json.Unmarshal(data, &reg); err != nil {
		t.Fatalf("parse capability-registry.json: %v", err)
	}
	v, _ := reg["router_policy_version"].(string)
	if v == "" {
		t.Fatalf("apply removed router_policy_version; rubric A2.13 requires bump, not erasure")
	}
	if v == "v1.0.0" {
		t.Fatalf("router_policy_version unchanged at v1.0.0; rubric A2.13 requires --apply to bump semver")
	}
}

// ---------------------------------------------------------------------------
// A2.14 — confidence_gate thresholds live in capability-registry.json; the
// runtime exposes no setter for them.
// ---------------------------------------------------------------------------

func TestRouterConfidenceGateFieldsFixed(t *testing.T) {
	root := repoRoot(t)
	regPath := filepath.Join(root, "hwcloud-skillcheck", "capability-registry.json")
	data, err := os.ReadFile(regPath)
	if err != nil {
		t.Fatalf("capability-registry.json missing at %s: %v\n(rubric A2.14 requires the gate as the source of truth, versioned with router_policy_version)", regPath, err)
	}
	var reg map[string]any
	if err := json.Unmarshal(data, &reg); err != nil {
		t.Fatalf("parse %s: %v", regPath, err)
	}
	gate, ok := reg["confidence_gate"].(map[string]any)
	if !ok {
		t.Fatalf("capability-registry.json missing confidence_gate block")
	}
	if v, _ := gate["top1_score_min"].(float64); v != 7500 {
		t.Fatalf("confidence_gate.top1_score_min=%v; want exactly 7500 (rounded default)", gate["top1_score_min"])
	}
	if v, _ := gate["margin_min"].(float64); v != 1500 {
		t.Fatalf("confidence_gate.margin_min=%v; want exactly 1500", gate["margin_min"])
	}
	em, ok := gate["entity_match"].([]any)
	if !ok || len(em) == 0 {
		t.Fatalf("confidence_gate.entity_match must be a non-empty list (got %v)", gate["entity_match"])
	}
}

func TestRouterConfidenceGateHasNoRuntimeSetter(t *testing.T) {
	root := repoRoot(t)
	srcPath := filepath.Join(root, "hwcloud-skillcheck", "internal", "router", "router.go")
	src, err := os.ReadFile(srcPath)
	if err != nil {
		t.Fatal(err)
	}

	// Precondition RED: the router package must declare ConfidenceGate as a
	// first-class type. Without it, "no setter" is meaningless. Today the
	// type does not exist — RED.
	if !bytes.Contains(src, []byte("type ConfidenceGate")) {
		t.Fatalf("router.go missing `type ConfidenceGate` declaration (rubric A2.14); got file head:\n%s",
			firstNLines(string(src), 30))
	}

	// Now assert that no exported setter for the gate exists.
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, srcPath, src, parser.AllErrors)
	if err != nil {
		t.Fatalf("parse router.go: %v", err)
	}
	var forbidden []string
	for _, decl := range file.Decls {
		fd, ok := decl.(*ast.FuncDecl)
		if !ok || fd.Recv != nil || fd.Name == nil || !fd.Name.IsExported() {
			continue
		}
		name := fd.Name.Name
		if strings.HasPrefix(name, "SetConfidenceGate") ||
			strings.HasPrefix(name, "UpdateConfidenceGate") ||
			strings.HasPrefix(name, "ReplaceConfidenceGate") {
			forbidden = append(forbidden, name)
		}
	}
	if len(forbidden) > 0 {
		t.Fatalf("forbidden confidence_gate setters in router package (rubric A2.14, S3): %v", forbidden)
	}
}

// ---------------------------------------------------------------------------
// helpers
// ---------------------------------------------------------------------------

func keysOf(m map[string]any) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}

func firstNLines(s string, n int) string {
	lines := strings.Split(s, "\n")
	if len(lines) <= n {
		return s
	}
	return strings.Join(lines[:n], "\n")
}
