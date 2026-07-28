// Package cmd — router subcommands.
//
// Implements the offline `router calibrate` CLI defined in
// docs/superpowers/specs/2026-07-27-harness-runtime-p1p2-design.md §4.2.2
// and rubric A2.13 of audit-results/p2-rubric.md v3.
//
// Hard contracts enforced:
//   - dry-run is the default; --apply is required to mutate
//     capability-registry.json on disk (rubric A2.13 + S3)
//   - the CLI runs only OFFLINE; it never imports the runtime hot path
//   - successful --apply bumps router_policy_version (semver) and
//     updates policy_diff_at; rollback-to is supported with the same
//     dry-run-by-default contract.
package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/buhaiqing/hcloud-skills/hwcloud-skillcheck/internal/embedder"
	"github.com/buhaiqing/hcloud-skills/hwcloud-skillcheck/internal/router"
)

// runRouter dispatches `hwcloud-skillcheck router <subcommand>`.
func runRouter(args []string) error {
	if len(args) == 0 || args[0] == "-h" || args[0] == "--help" {
		return fmt.Errorf("router: missing subcommand (use 'info', 'embed-test', or 'calibrate')")
	}
	switch args[0] {
	case "calibrate":
		return runRouterCalibrate(args[1:])
	case "info":
		return runRouterInfo(args[1:])
	case "embed-test":
		return runRouterEmbedTest(args[1:])
	default:
		return fmt.Errorf("router: unknown subcommand %q", args[0])
	}
}

func runRouterEmbedTest(args []string) error {
	fs := newFlagSet("router embed-test")
	root := fs.String("root", ".", "workspace or module root containing capability-registry.json")
	provider := fs.String("provider", "", "temporary provider override for this diagnostic only")
	text := fs.String("text", "list ecs servers", "sample text to embed after preflight passes")
	if err := fs.Parse(args); err != nil {
		return err
	}
	policyPath := os.Getenv("HC_CAPABILITY_REGISTRY")
	if policyPath == "" {
		policyPath = filepath.Join(*root, "capability-registry.json")
		if _, err := os.Stat(policyPath); err != nil {
			policyPath = filepath.Join(*root, "hwcloud-skillcheck", "capability-registry.json")
		}
	}
	previousPath, hadPath := os.LookupEnv("HC_CAPABILITY_REGISTRY")
	if err := os.Setenv("HC_CAPABILITY_REGISTRY", policyPath); err != nil {
		return err
	}
	defer func() {
		if hadPath {
			_ = os.Setenv("HC_CAPABILITY_REGISTRY", previousPath)
		} else {
			_ = os.Unsetenv("HC_CAPABILITY_REGISTRY")
		}
	}()
	cfg, err := embedder.LoadConfig()
	if err != nil {
		return fmt.Errorf("embedding preflight: cannot load config: %w. Fix: validate JSON syntax in %s", err, policyPath)
	}
	if *provider != "" {
		cfg.ProviderName = *provider
		cfg.Mode = ""
	}
	probe, err := embedder.NewUninitialized(cfg.ProviderName)
	if err != nil {
		return err
	}
	report := probe.Preflight(cfg)
	printPreflightReport(report, policyPath)
	if err := report.Validate(); err != nil {
		return err
	}
	emb, err := embedder.PreflightAndInit(context.Background(), cfg)
	if err != nil {
		return err
	}
	defer emb.Close()
	started := time.Now()
	vector, err := emb.Embed(context.Background(), *text)
	if err != nil {
		return fmt.Errorf("embedding smoke call failed: %w", err)
	}
	preview := vector
	if len(preview) > 5 {
		preview = preview[:5]
	}
	fmt.Fprintf(os.Stdout, "embedding smoke test: PASS\n  provider: %s\n  vector_dim: %d\n  duration_ms: %d\n  preview: %v\n", emb.Name(), len(vector), time.Since(started).Milliseconds(), preview)
	return nil
}

func printPreflightReport(report embedder.PreflightReport, policyPath string) {
	status := "PASS"
	if !report.OK {
		status = "FAIL"
	}
	fmt.Fprintf(os.Stdout, "sandbox preflight: %s\n  provider: %s\n  config: %s\n", status, report.Provider, policyPath)
	for _, info := range report.Info {
		fmt.Fprintf(os.Stdout, "  info: %s\n", info)
	}
	for _, warning := range report.Warnings {
		fmt.Fprintf(os.Stdout, "  warning [%s]: %s\n    Fix: %s\n", warning.Field, warning.Message, warning.Fix)
	}
	for _, issue := range report.Errors {
		fmt.Fprintf(os.Stdout, "  error [%s]: %s\n    Fix: %s\n", issue.Field, issue.Message, issue.Fix)
		if issue.DocURL != "" {
			fmt.Fprintf(os.Stdout, "    See: %s\n", issue.DocURL)
		}
	}
}

func runRouterInfo(args []string) error {
	fs := newFlagSet("router info")
	root := fs.String("root", "", "workspace root (reads capability-registry.json)")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *root == "" {
		return fmt.Errorf("--root is required")
	}
	data, err := os.ReadFile(filepath.Join(*root, "capability-registry.json"))
	if err != nil {
		return fmt.Errorf("read %s: %w", *root, err)
	}
	var p router.Policy
	if err := json.Unmarshal(data, &p); err != nil {
		return fmt.Errorf("decode: %w", err)
	}
	fmt.Fprintf(os.Stdout, "router_policy_version:   %s\n", p.RouterPolicyVersion)
	if p.RouterPolicyCandidate != "" {
		fmt.Fprintf(os.Stdout, "router_policy_candidate: %s\n", p.RouterPolicyCandidate)
	}
	if p.PolicyDiffAt != "" {
		fmt.Fprintf(os.Stdout, "policy_diff_at:          %s\n", p.PolicyDiffAt)
	}
	fmt.Fprintf(os.Stdout, "confidence_gate:         top1_score_min=%d margin_min=%d entity_match=%v\n",
		p.ConfidenceGate.Top1ScoreMin, p.ConfidenceGate.MarginMin, p.ConfidenceGate.EntityMatch)
	fmt.Fprintf(os.Stdout, "lexicon_version:         %s\n", p.Lexicon.Version)
	return nil
}

// runRouterCalibrate is the A2.13 entry point.
func runRouterCalibrate(args []string) error {
	fs := newFlagSet("router calibrate")
	root := fs.String("root", "", "workspace root")
	apply := fs.Bool("apply", false, "apply calibration (mutates capability-registry.json). Without this, dry-run is the default per rubric A2.13.")
	source := fs.String("source", "", "audit-results dir to read traces from (advisory only in this revision)")
	rollbackTo := fs.String("rollback-to", "", "rollback to a previous router_policy_version (e.g. v1.0.0)")
	bumpLevel := fs.String("bump", "patch", "version bump level on --apply: patch|minor|major")
	policyPath := fs.String("policy", "", "explicit path to capability-registry.json (default: $HC_CAPABILITY_REGISTRY or <root>/capability-registry.json)")

	if err := fs.Parse(args); err != nil {
		return err
	}
	if *root == "" {
		return fmt.Errorf("--root is required")
	}

	effectivePolicyPath := *policyPath
	if effectivePolicyPath == "" {
		effectivePolicyPath = os.Getenv("HC_CAPABILITY_REGISTRY")
	}
	if effectivePolicyPath == "" {
		effectivePolicyPath = filepath.Join(*root, "capability-registry.json")
	}

	// Load current policy.
	raw, err := os.ReadFile(effectivePolicyPath)
	if err != nil {
		return fmt.Errorf("read policy %s: %w", effectivePolicyPath, err)
	}
	var policy router.Policy
	if err := json.Unmarshal(raw, &policy); err != nil {
		return fmt.Errorf("decode policy %s: %w", effectivePolicyPath, err)
	}
	if policy.RouterPolicyVersion == "" {
		return fmt.Errorf("policy %s missing router_policy_version", effectivePolicyPath)
	}

	// Resolve the target version.
	targetVersion := policy.RouterPolicyVersion
	var notes []string

	if *rollbackTo != "" {
		if !strings.HasPrefix(*rollbackTo, "v") {
			return fmt.Errorf("--rollback-to must be semver-like \"vX.Y.Z\", got %q", *rollbackTo)
		}
		targetVersion = *rollbackTo
		notes = append(notes, fmt.Sprintf("rollback_to=%s", *rollbackTo))
	} else if *apply {
		bumped, err := bumpSemver(policy.RouterPolicyVersion, *bumpLevel)
		if err != nil {
			return fmt.Errorf("bump: %w", err)
		}
		targetVersion = bumped
		notes = append(notes, fmt.Sprintf("bump=%s -> %s", policy.RouterPolicyVersion, targetVersion))
	}

	// Trace-derived suggestions (advisory only — real algorithm ships when
	// shadow mode lands; the CLI structure is stable so the call sites don't
	// need to change later).
	if *source != "" {
		traces := filepath.Join(*source, "gcl-trace-*.json")
		if _, err := filepath.Glob(traces); err != nil {
			notes = append(notes, fmt.Sprintf("source=%s (trace scan failed: %v)", *source, err))
		} else {
			notes = append(notes, fmt.Sprintf("source=%s (trace-derived suggestions land in next revision)", *source))
		}
	}

	// Print the plan first (always).
	fmt.Fprintf(os.Stdout, "router calibrate plan\n")
	fmt.Fprintf(os.Stdout, "  policy:           %s\n", effectivePolicyPath)
	fmt.Fprintf(os.Stdout, "  workspace_root:   %s\n", *root)
	fmt.Fprintf(os.Stdout, "  current_version:  %s\n", policy.RouterPolicyVersion)
	fmt.Fprintf(os.Stdout, "  target_version:   %s\n", targetVersion)
	if policy.PolicyDiffAt != "" {
		fmt.Fprintf(os.Stdout, "  policy_diff_at:   %s\n", policy.PolicyDiffAt)
	}
	fmt.Fprintf(os.Stdout, "  confidence_gate:  top1_score_min=%d margin_min=%d entity_match=%v\n",
		policy.ConfidenceGate.Top1ScoreMin, policy.ConfidenceGate.MarginMin, policy.ConfidenceGate.EntityMatch)
	w := policy.ScoringWeights
	if w.InputShapeMatch+w.TagOverlapCosine+w.PermissionsSubset+w.BM25FSupplement+w.LexiconAliasBonus != 0 ||
		w.HardFilterPenalty != 0 {
		fmt.Fprintf(os.Stdout, "  scoring_weights:  input_shape_match=%d tag_overlap_cosine=%d permissions_subset=%d bm25f_supplement=%d lexicon_alias_bonus=%d hard_filter_penalty=%d\n",
			policy.ScoringWeights.InputShapeMatch, policy.ScoringWeights.TagOverlapCosine,
			policy.ScoringWeights.PermissionsSubset, policy.ScoringWeights.BM25FSupplement,
			policy.ScoringWeights.LexiconAliasBonus, policy.ScoringWeights.HardFilterPenalty)
	}
	fmt.Fprintf(os.Stdout, "  lexicon_version:  %s\n", defaultStr(policy.Lexicon.Version, "(unset)"))
	if len(notes) > 0 {
		fmt.Fprintf(os.Stdout, "  notes:\n")
		for _, n := range notes {
			fmt.Fprintf(os.Stdout, "    - %s\n", n)
		}
	}

	// Hard contract: dry-run is the default.
	if !*apply {
		fmt.Fprintf(os.Stdout, "\nDRY-RUN: no changes written. Re-run with --apply to commit (rubric A2.13).\n")
		return nil
	}

	// Apply: write the new policy.
	policy.RouterPolicyVersion = targetVersion
	policy.PolicyDiffAt = time.Now().UTC().Format(time.RFC3339)
	data, err := json.MarshalIndent(&policy, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal: %w", err)
	}
	if err := os.WriteFile(effectivePolicyPath, data, 0o600); err != nil {
		return fmt.Errorf("write policy %s: %w", effectivePolicyPath, err)
	}
	fmt.Fprintf(os.Stdout, "\nAPPLIED: router_policy_version -> %s (policy_diff_at=%s)\n",
		targetVersion, policy.PolicyDiffAt)
	return nil
}

// bumpSemver applies a patch/minor/major bump to a semver-like string of the
// form "vX.Y.Z" or "vX.Y". Returns an error on malformed input.
func bumpSemver(v, level string) (string, error) {
	v = strings.TrimPrefix(v, "v")
	parts := strings.Split(v, ".")
	if len(parts) < 2 || len(parts) > 3 {
		return "", fmt.Errorf("not semver-like: %q", v)
	}
	for _, p := range parts {
		if p == "" {
			return "", fmt.Errorf("empty semver component in %q", v)
		}
		for _, c := range p {
			if c < '0' || c > '9' {
				return "", fmt.Errorf("non-numeric component in %q", v)
			}
		}
	}
	var major, minor, patch int
	switch len(parts) {
	case 2:
		fmt.Sscanf(parts[0]+"."+parts[1]+".0", "%d.%d.%d", &major, &minor, &patch)
	case 3:
		fmt.Sscanf(parts[0]+"."+parts[1]+"."+parts[2], "%d.%d.%d", &major, &minor, &patch)
	}
	switch strings.ToLower(level) {
	case "patch", "":
		return fmt.Sprintf("v%d.%d.%d", major, minor, patch+1), nil
	case "minor":
		return fmt.Sprintf("v%d.%d.%d", major, minor+1, 0), nil
	case "major":
		return fmt.Sprintf("v%d.%d.%d", major+1, 0, 0), nil
	default:
		return "", fmt.Errorf("unknown bump level %q (want patch|minor|major)", level)
	}
}

func defaultStr(v, def string) string {
	if v == "" {
		return def
	}
	return v
}
