package cmd

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/buhaiqing/hcloud-skills/hwcloud-skillcheck/internal/l4"
)

// campaignMetrics is the raw-count input of `learning campaign record`.
//
// Rates are derived here rather than passed in: a hand-written 0.47 cannot be
// checked against its own numerator and denominator, and the point of these
// numbers is that a later session can falsify them.
//
// One metric from the original design was deliberately dropped. A single
// "critic_false_positive_rate" cannot distinguish a Critic being wrong from an
// Adjudicator wrongly overturning a Critic — opposite signals in one rate. Both
// counts are kept raw here so either rate can be defined later without losing
// data, but neither ships as a rate until their numerators are unambiguous.
type campaignMetrics struct {
	PostFixRereview string   `json:"post_fix_rereview"` // reviewed | waived
	Waivers         []waiver `json:"waivers"`
	Dispatches      struct {
		Failed int `json:"failed"`
		Total  int `json:"total"`
	} `json:"dispatches"`
	Critic struct {
		Findings                  *int `json:"findings"`
		AcknowledgedFalsePositive *int `json:"acknowledged_false_positive"` // Critic self-retracted
	} `json:"critic"`
	Adjudicator struct {
		Overturns *int `json:"overturns"` // Critic findings overturned after review
	} `json:"adjudicator"`
	Fix struct {
		// TouchedInLaterRounds counts files from the first round's write set
		// that a later fix round modified again. The denominator is the first
		// round's write set, not the union across rounds: a union would make
		// the rate grow with the number of rounds by construction.
		TouchedInLaterRounds int `json:"touched_in_later_rounds"`
		FirstRoundFiles      int `json:"first_round_files"`
	} `json:"fix"`
	Rounds struct {
		Rereviewed int `json:"rereviewed"`
		Total      int `json:"total"`
	} `json:"rounds"`
}

// waiver is the evidence a fix-round waiver must carry (docs/gcl-spec.md §4.1):
// without all four fields the waiver is prose, and prose is bypassable.
type waiver struct {
	FindingID string `json:"finding_id"`
	FileLine  string `json:"file_line"`
	Command   string `json:"command"`
	Result    string `json:"result"`
}

func (w waiver) complete() bool {
	return w.FindingID != "" && w.FileLine != "" && w.Command != "" && w.Result != ""
}

// campaignRecord is the on-disk campaign artifact. audit-results is gitignored
// and outcome memory is pruned after 90 days, so these numbers are local
// runtime state: inheritable by later sessions on the same working copy only.
type campaignRecord struct {
	ExperimentID    string          `json:"experiment_id"`
	Skill           string          `json:"skill"`
	Action          string          `json:"action"`
	Outcome         string          `json:"outcome"`
	GCLDecision     string          `json:"gcl_decision"`
	PostFixRereview string          `json:"post_fix_rereview"`
	Waivers         []waiver        `json:"waivers,omitempty"`
	Raw             campaignMetrics `json:"raw"`
	RecordedAt      string          `json:"recorded_at"`
}

// rate divides and rejects impossible inputs instead of clamping them: a
// numerator above its denominator means the counts were mis-entered, and a
// silent clamp would bake the error into a metric meant to be falsifiable.
func rate(name string, numerator, denominator int) (float64, error) {
	if denominator <= 0 {
		return 0, fmt.Errorf("%s: denominator must be > 0, got %d", name, denominator)
	}
	if numerator < 0 || numerator > denominator {
		return 0, fmt.Errorf("%s: numerator %d out of range for denominator %d", name, numerator, denominator)
	}
	return float64(numerator) / float64(denominator), nil
}

func runLearningCampaign(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: hwcloud-skillcheck learning campaign <record|report> ...")
	}
	switch args[0] {
	case "record":
		return runLearningCampaignRecord(args[1:])
	case "report":
		return runLearningCampaignReport(args[1:])
	default:
		return fmt.Errorf("unknown campaign subcommand %q; use record|report", args[0])
	}
}

func runLearningCampaignRecord(args []string) error {
	fs := newFlagSet("hwcloud-skillcheck learning campaign record")
	root := fs.String("root", ".", "repo root (default: current directory)")
	id := fs.String("id", "", "GCL experiment id (required)")
	skill := fs.String("skill", "huaweicloud-skillcheck", "skill or component the campaign targeted")
	action := fs.String("action", "gcl-batch", "action label recorded in outcome memory")
	outcome := fs.String("outcome", "", "success | failure | blocked (required)")
	gclDecision := fs.String("gcl-decision", "PASS", "GCL verdict recorded in outcome memory")
	risk := fs.String("risk", "high", "risk level recorded in outcome memory")
	rbac := fs.String("rbac-decision", "allowed", "RBAC decision recorded in outcome memory")
	metricsPath := fs.String("metrics", "", "path to the campaign metrics JSON (required)")
	dryRun := fs.Bool("dry-run", false, "validate inputs and print rates without writing")
	if err := fs.Parse(args); err != nil {
		return err
	}

	if *id == "" || *outcome == "" || *metricsPath == "" {
		return fmt.Errorf("--id, --outcome and --metrics are required")
	}
	switch *outcome {
	case "success", "failure", "blocked":
	default:
		return fmt.Errorf("--outcome must be success|failure|blocked, got %q", *outcome)
	}

	rawMetrics, err := os.ReadFile(*metricsPath)
	if err != nil {
		return fmt.Errorf("read metrics: %w", err)
	}
	var metrics campaignMetrics
	if err := json.Unmarshal(rawMetrics, &metrics); err != nil {
		return fmt.Errorf("parse metrics: %w", err)
	}

	switch metrics.PostFixRereview {
	case "reviewed":
	case "waived":
		if len(metrics.Waivers) == 0 {
			return fmt.Errorf("post_fix_rereview=waived requires at least one waiver, and every waiver needs finding_id, file_line, command and result")
		}
		for i, w := range metrics.Waivers {
			if !w.complete() {
				return fmt.Errorf("waiver %d incomplete: finding_id, file_line, command and result are all required", i)
			}
		}
	default:
		return fmt.Errorf("metrics.post_fix_rereview must be reviewed|waived, got %q", metrics.PostFixRereview)
	}

	// Pointers, not ints: a missing block must not read as "0 findings found",
	// which is the convenient way to make the critic signal look clean. The
	// checks run in a fixed order so the reported field is deterministic.
	for _, required := range []struct {
		name  string
		value *int
	}{
		{"critic.findings", metrics.Critic.Findings},
		{"critic.acknowledged_false_positive", metrics.Critic.AcknowledgedFalsePositive},
		{"adjudicator.overturns", metrics.Adjudicator.Overturns},
	} {
		if required.value == nil {
			return fmt.Errorf("metrics.%s is required (0 is a valid value; omitting it is not)", required.name)
		}
	}

	rates := map[string]float64{}
	for _, r := range []struct {
		name string
		num  int
		den  int
	}{
		{"dispatch_failure_rate", metrics.Dispatches.Failed, metrics.Dispatches.Total},
		{"fix_rework_rate", metrics.Fix.TouchedInLaterRounds, metrics.Fix.FirstRoundFiles},
		{"post_fix_rereview_coverage", metrics.Rounds.Rereviewed, metrics.Rounds.Total},
	} {
		v, rateErr := rate(r.name, r.num, r.den)
		if rateErr != nil {
			return rateErr
		}
		rates[r.name] = v
	}

	record := campaignRecord{
		ExperimentID:    *id,
		Skill:           *skill,
		Action:          *action,
		Outcome:         *outcome,
		GCLDecision:     *gclDecision,
		PostFixRereview: metrics.PostFixRereview,
		Waivers:         metrics.Waivers,
		Raw:             metrics,
		RecordedAt:      time.Now().UTC().Format(time.RFC3339),
	}

	fmt.Printf("campaign %s: post_fix_rereview=%s dispatch_failure=%.3f fix_rework=%.3f rereview_coverage=%.3f (critic findings=%d self-retracted=%d adjudicator_overturns=%d)\n",
		*id, metrics.PostFixRereview, rates["dispatch_failure_rate"],
		rates["fix_rework_rate"], rates["post_fix_rereview_coverage"],
		*metrics.Critic.Findings, *metrics.Critic.AcknowledgedFalsePositive,
		*metrics.Adjudicator.Overturns)
	if *dryRun {
		fmt.Println("dry-run: nothing written")
		return nil
	}

	campaignDir := filepath.Join(*root, "audit-results", "gcl-campaigns")
	if err := os.MkdirAll(campaignDir, 0o755); err != nil {
		return fmt.Errorf("mkdir %s: %w", campaignDir, err)
	}
	campaignPath := filepath.Join(campaignDir, *id+".json")
	// One record per experiment id: overwriting would silently drop the first
	// record's raw counts while its outcome row survives, so the two stores
	// would disagree.
	if _, statErr := os.Stat(campaignPath); statErr == nil {
		return fmt.Errorf("campaign %s already recorded at %s; pick a new --id (re-recording is not idempotent by design)", *id, campaignPath)
	}
	body, err := json.MarshalIndent(record, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal campaign: %w", err)
	}
	if err := os.WriteFile(campaignPath, append(body, '\n'), 0o644); err != nil {
		return fmt.Errorf("write campaign: %w", err)
	}

	mem, err := l4.NewOutcomeMemory(*root)
	if err != nil {
		return err
	}
	ratePtr := func(name string) *float64 {
		v := rates[name]
		return &v
	}
	// context_hash is the sha256 of the metrics file — the stable input this
	// record was derived from, so MatchOutcomes can find it again.
	sum := sha256.Sum256(rawMetrics)
	recordID, err := uuidV4()
	if err != nil {
		return err
	}
	if err := mem.Record(l4.OutcomeRecord{
		ID:                      recordID,
		Timestamp:               record.RecordedAt,
		TaskID:                  *id,
		Skill:                   *skill,
		Action:                  *action,
		ContextHash:             "sha256:" + hex.EncodeToString(sum[:]),
		Outcome:                 *outcome,
		RetryCount:              0, // schema meaning is per-step retries; dispatch failures stay in the artifact
		DurationMS:              0,
		Risk:                    *risk,
		RBACDecision:            *rbac,
		GCLDecision:             *gclDecision,
		GCLExperimentID:         *id,
		DispatchFailureRate:     ratePtr("dispatch_failure_rate"),
		FixReworkRate:           ratePtr("fix_rework_rate"),
		PostFixRereviewCoverage: ratePtr("post_fix_rereview_coverage"),
	}); err != nil {
		return err
	}

	fmt.Printf("wrote %s and appended outcome row to .l4-memory/outcomes.jsonl\n", campaignPath)
	return nil
}

// runLearningCampaignReport is the reader that makes the rates falsifiable: a
// later session runs one command and sees the measured history instead of
// re-reading campaign transcripts. Rates are recomputed from raw counts here,
// so a hand-edited artifact cannot disagree with its own report.
func runLearningCampaignReport(args []string) error {
	fs := newFlagSet("hwcloud-skillcheck learning campaign report")
	root := fs.String("root", ".", "repo root (default: current directory)")
	asJSON := fs.Bool("json", false, "emit JSON")
	if err := fs.Parse(args); err != nil {
		return err
	}

	campaignDir := filepath.Join(*root, "audit-results", "gcl-campaigns")
	entries, err := os.ReadDir(campaignDir)
	if err != nil {
		if os.IsNotExist(err) {
			fmt.Printf("no campaigns recorded under %s (campaign metrics are local runtime state, gitignored)\n", campaignDir)
			return nil
		}
		return err
	}

	var records []campaignRecord
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}
		body, readErr := os.ReadFile(filepath.Join(campaignDir, entry.Name()))
		if readErr != nil {
			return readErr
		}
		var rec campaignRecord
		if err := json.Unmarshal(body, &rec); err != nil {
			return fmt.Errorf("parse %s: %w", entry.Name(), err)
		}
		records = append(records, rec)
	}
	// Stable sort with an explicit tiebreaker: two campaigns recorded in the
	// same second must not have their order decided by an unstable sort.
	sort.SliceStable(records, func(i, j int) bool {
		if records[i].RecordedAt != records[j].RecordedAt {
			return records[i].RecordedAt < records[j].RecordedAt
		}
		return records[i].ExperimentID < records[j].ExperimentID
	})

	type campaignRates struct {
		ExperimentID     string  `json:"experiment_id"`
		RecordedAt       string  `json:"recorded_at"`
		Outcome          string  `json:"outcome"`
		GCLDecision      string  `json:"gcl_decision"`
		PostFixRereview  string  `json:"post_fix_rereview"`
		DispatchFailure  float64 `json:"dispatch_failure_rate"`
		FixRework        float64 `json:"fix_rework_rate"`
		RereviewCover    float64 `json:"post_fix_rereview_coverage"`
		CriticFindings   int     `json:"critic_findings"`
		CriticRetracted  int     `json:"critic_acknowledged_false_positive"`
		AdjudicatorTurns int     `json:"adjudicator_overturns"`
	}
	report := make([]campaignRates, 0, len(records))
	for _, rec := range records {
		// A malformed count pair degrades to -1 rather than aborting the whole
		// history: a bad row must not hide the campaigns around it.
		dispatch, dErr := rate("dispatch_failure_rate", rec.Raw.Dispatches.Failed, rec.Raw.Dispatches.Total)
		rework, rErr := rate("fix_rework_rate", rec.Raw.Fix.TouchedInLaterRounds, rec.Raw.Fix.FirstRoundFiles)
		coverage, cErr := rate("post_fix_rereview_coverage", rec.Raw.Rounds.Rereviewed, rec.Raw.Rounds.Total)
		row := campaignRates{
			ExperimentID: rec.ExperimentID, RecordedAt: rec.RecordedAt, Outcome: rec.Outcome,
			GCLDecision: rec.GCLDecision, PostFixRereview: rec.PostFixRereview,
			CriticFindings:   derefOrNeg1(rec.Raw.Critic.Findings),
			CriticRetracted:  derefOrNeg1(rec.Raw.Critic.AcknowledgedFalsePositive),
			AdjudicatorTurns: derefOrNeg1(rec.Raw.Adjudicator.Overturns),
		}
		row.DispatchFailure, row.FixRework, row.RereviewCover = -1, -1, -1
		if dErr == nil {
			row.DispatchFailure = dispatch
		}
		if rErr == nil {
			row.FixRework = rework
		}
		if cErr == nil {
			row.RereviewCover = coverage
		}
		report = append(report, row)
	}

	if *asJSON {
		out, err := json.MarshalIndent(report, "", "  ")
		if err != nil {
			return err
		}
		fmt.Println(string(out))
		return nil
	}
	if len(report) == 0 {
		fmt.Printf("no campaigns recorded under %s\n", campaignDir)
		return nil
	}
	fmt.Printf("%-22s %-9s %-17s %8s %8s %8s  %s\n",
		"experiment", "outcome", "post_fix_rereview", "dispatch", "rework", "rereview", "critic f/r/overturn")
	for _, row := range report {
		fmt.Printf("%-22s %-9s %-17s %8.3f %8.3f %8.3f  %d/%d/%d\n",
			row.ExperimentID, row.Outcome, row.PostFixRereview,
			row.DispatchFailure, row.FixRework, row.RereviewCover,
			row.CriticFindings, row.CriticRetracted, row.AdjudicatorTurns)
	}
	return nil
}

// derefOrNeg1 keeps a hand-edited artifact (critic counts set to null) from
// panicking the report; -1 marks "absent or impossible" in both places.
func derefOrNeg1(v *int) int {
	if v == nil {
		return -1
	}
	return *v
}

// uuidV4 keeps the outcome schema's "id MUST be a uuid v4" rule true without
// pulling in a dependency: 16 random bytes with the version/variant bits set.
// (internal/l4 has its own ID helpers, but they are unexported and the existing
// outcome writer emits 16-hex rather than a uuid — pre-existing, out of scope.)
func uuidV4() (string, error) {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", fmt.Errorf("uuid: %w", err)
	}
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16]), nil
}
