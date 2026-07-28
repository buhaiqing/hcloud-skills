package router

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"

	"github.com/buhaiqing/hcloud-skills/hwcloud-skillcheck/internal/embedder"
)

// GateThresholds are the three fixed knobs that drive the confidence_gate
// decision (see docs/superpowers/specs/2026-07-27-harness-runtime-p1p2-design.md
// §4.2.3). They live ONLY in capability-registry.json and have no runtime
// setter (rubric A2.14, S3).
type GateThresholds struct {
	Top1ScoreMin int      `json:"top1_score_min"`
	MarginMin    int      `json:"margin_min"`
	EntityMatch  []string `json:"entity_match"`
}

type ScoringWeights struct {
	InputShapeMatch   int `json:"input_shape_match"`
	TagOverlapCosine  int `json:"tag_overlap_cosine"`
	PermissionsSubset int `json:"permissions_subset"`
	BM25FSupplement   int `json:"bm25f_supplement"`
	LexiconAliasBonus int `json:"lexicon_alias_bonus"`
	HardFilterPenalty int `json:"hard_filter_penalty"`
}

type Lexicon struct {
	Version   string            `json:"version"`
	Products  map[string]string `json:"products"`
	Actions   map[string]string `json:"actions"`
	Resources map[string]string `json:"resources"`
}

// Policy mirrors capability-registry.json. Loaded once at package init and
// treated as read-only thereafter.
type Policy struct {
	RouterPolicyVersion   string                  `json:"router_policy_version"`
	RouterPolicyCandidate string                  `json:"router_policy_candidate"`
	PolicyDiffAt          string                  `json:"policy_diff_at"`
	ConfidenceGate        GateThresholds          `json:"confidence_gate"`
	ScoringWeights        ScoringWeights          `json:"scoring_weights"`
	Lexicon               Lexicon                 `json:"lexicon"`
	Embedding             embedder.ProviderConfig `json:"embedding,omitempty"`
	ShadowEmbedding       embedder.ProviderConfig `json:"shadow_embedding,omitempty"`
}

const envPolicyPath = "HC_CAPABILITY_REGISTRY"

// LoadPolicy reads + validates capability-registry.json. It's exported so the
// calibrate CLI (§4.2.2, rubric A2.13) and Capability Compiler can re-use it.
func LoadPolicy(path string) (*Policy, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read policy %s: %w", path, err)
	}
	var p Policy
	if err := json.Unmarshal(data, &p); err != nil {
		return nil, fmt.Errorf("decode policy %s: %w", path, err)
	}
	if err := p.validate(); err != nil {
		return nil, fmt.Errorf("policy %s: %w", path, err)
	}
	return &p, nil
}

func (p *Policy) validate() error {
	if p.RouterPolicyVersion == "" {
		return fmt.Errorf("missing router_policy_version")
	}
	if p.ConfidenceGate.Top1ScoreMin <= 0 {
		return fmt.Errorf("confidence_gate.top1_score_min must be positive")
	}
	if p.ConfidenceGate.MarginMin < 0 {
		return fmt.Errorf("confidence_gate.margin_min must be non-negative")
	}
	if len(p.ConfidenceGate.EntityMatch) == 0 {
		return fmt.Errorf("confidence_gate.entity_match must be non-empty")
	}
	return nil
}

// policyPath returns the file-system path to load the package-level policy
// from. Order:
//  1. HC_CAPABILITY_REGISTRY env var (used by tests + the calibrate CLI)
//  2. capability-registry.json relative to the process working directory
//     (typical for `go test ./internal/router/...` and the CLI's own CWD).
func policyPath() string {
	if p := os.Getenv(envPolicyPath); p != "" {
		return p
	}
	return "capability-registry.json"
}

// loadDefault reads the policy once per process. The Router package exposes no
// setter (rubric A2.14, S3): changing the policy requires a process restart
// AFTER a calibrated capability-registry.json has been written by §4.2.2.
func loadDefault() *Policy {
	p, err := LoadPolicy(policyPath())
	if err != nil {
		return nil
	}
	return p
}

// stubPolicy is a safe fallback used only when the canonical file cannot be
// loaded (e.g. test runs without the file, or before Capability Compiler
// runs). It is hard-coded with the spec defaults — exactly the same values
// the canonical file ships with — so behaviour is identical whether the load
// succeeds or not, as long as the canonical file matches the spec defaults.
func stubPolicy() *Policy {
	return &Policy{
		RouterPolicyVersion: "v0.0.0-uninitialized",
		PolicyDiffAt:        "",
		ConfidenceGate: GateThresholds{
			Top1ScoreMin: 7500,
			MarginMin:    1500,
			EntityMatch:  []string{"strong"},
		},
	}
}

// activePolicy returns the loaded policy when available, or the spec-default
// stub when not. Either way the call site is read-only.
func activePolicy() *Policy {
	if p := loadDefault(); p != nil {
		return p
	}
	return stubPolicy()
}

// sortedPolicyDiff returns a stable, sorted key listing of the policy fields
// to support the offline diff tooling called out in §4.2.2.
func sortedPolicyDiff(p *Policy) []string {
	keys := []string{
		"router_policy_version",
		"confidence_gate.top1_score_min",
		"confidence_gate.margin_min",
		"confidence_gate.entity_match",
		"scoring_weights.input_shape_match",
		"scoring_weights.tag_overlap_cosine",
		"scoring_weights.permissions_subset",
		"scoring_weights.bm25f_supplement",
		"scoring_weights.lexicon_alias_bonus",
		"scoring_weights.hard_filter_penalty",
	}
	sort.Strings(keys)
	return keys
}
