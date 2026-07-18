package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/buhaiqing/hcloud-skills/skillcheck/internal/embed"
)

// runAggregate dispatches the `skillcheck aggregate` subcommands.
func runAggregate(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("aggregate: missing subcommand (use 'trace')")
	}
	switch args[0] {
	case "trace":
		return runAggregateTrace(args[1:])
	case "-h", "--help", "help":
		fmt.Fprintln(os.Stdout, "skillcheck aggregate trace --root <dir> [--since-hours N] [--output FILE]")
		return nil
	default:
		return fmt.Errorf("aggregate: unknown subcommand %q", args[0])
	}
}

const (
	rubricDims    = "correctness,safety,idempotency,traceability,spec_compliance"
	finalStatuses = "PASS,SAFETY_FAIL,MAX_ITER"
)

// runAggregateTrace aggregates audit-results/gcl-trace-*.json into a quality
// summary, mirroring scripts/gcl_trace_aggregate.py. When no trace files exist
// it WARNs and returns nil (exit 0) per Spec §4: trace files are produced by
// the runtime runner, so an external user may legitimately have none.
func runAggregateTrace(args []string) error {
	fs := newFlagSet("skillcheck aggregate trace")
	root := fs.String("root", ".", "skill repository root")
	sinceHours := fs.Int("since-hours", -1, "only traces modified within N hours")
	output := fs.String("output", "", "write summary to FILE instead of stdout")
	selfCheck := fs.Bool("self-check", false, "aggregate the embedded trace fixture instead of the repo")
	if err := fs.Parse(args); err != nil {
		return err
	}

	if *selfCheck {
		return runAggregateSelfCheck(*output)
	}

	rootDir, err := filepath.Abs(*root)
	if err != nil {
		return err
	}

	auditDir := filepath.Join(rootDir, "audit-results")
	var paths []string
	if entries, gErr := filepath.Glob(filepath.Join(auditDir, "gcl-trace-*.json")); gErr == nil {
		paths = entries
	}
	sort.Strings(paths)

	if *sinceHours >= 0 {
		cutoff := time.Now().Add(-time.Duration(*sinceHours) * time.Hour)
		var filtered []string
		for _, p := range paths {
			info, sErr := os.Stat(p)
			if sErr != nil {
				continue
			}
			if info.ModTime().After(cutoff) {
				filtered = append(filtered, p)
			}
		}
		paths = filtered
	}

	if len(paths) == 0 {
		fmt.Fprintln(os.Stderr, "WARN: no gcl-trace files found; skipping aggregate (trace files are produced by the runtime runner)")
		return nil
	}

	var traces []map[string]any
	for _, p := range paths {
		rel, _ := filepath.Rel(rootDir, p)
		trace, perr := parseAggregateTrace(p)
		if perr != nil {
			fmt.Fprintf(os.Stderr, "WARN: skip %s: %v\n", rel, perr)
			continue
		}
		trace["_source_path"] = rel
		traces = append(traces, trace)
	}
	if len(traces) == 0 {
		fmt.Fprintln(os.Stderr, "WARN: no valid traces parsed; skipping aggregate")
		return nil
	}

	summary := aggregateTraces(traces)

	var out []byte
	out, err = json.MarshalIndent(summary, "", "  ")
	if err != nil {
		return err
	}
	out = append(out, '\n')

	if *output != "" {
		outPath := *output
		if !filepath.IsAbs(outPath) {
			outPath = filepath.Join(rootDir, outPath)
		}
		if wErr := os.WriteFile(outPath, out, 0o644); wErr != nil {
			return wErr
		}
		fmt.Printf("Wrote quality summary to %s (total_runs=%d, pass_rate=%.4f)\n",
			outPath, intOf(summary["totals"].(map[string]any)["total_runs"]), summary["pass_rate"].(float64))
		return nil
	}
	os.Stdout.Write(out)
	return nil
}

// runAggregateSelfCheck aggregates the embedded healthy trace fixture and
// verifies the resulting summary is well-formed (total_runs >= 1, pass_rate in
// [0,1]). This proves the aggregation path is wired correctly inside the binary
// without requiring repo trace files.
func runAggregateSelfCheck(output string) error {
	var trace map[string]any
	if err := json.Unmarshal(embed.TraceHealthy, &trace); err != nil {
		return fmt.Errorf("self-check: bad embedded trace fixture: %w", err)
	}
	traces := []map[string]any{trace}
	summary := aggregateTraces(traces)

	totalRuns, _ := summary["totals"].(map[string]any)["total_runs"].(int)
	passRate, _ := summary["pass_rate"].(float64)
	if totalRuns < 1 {
		return fmt.Errorf("self-check: aggregated summary reported total_runs=%d", totalRuns)
	}
	if passRate < 0 || passRate > 1 {
		return fmt.Errorf("self-check: aggregated summary reported pass_rate=%v", passRate)
	}

	out, err := json.MarshalIndent(summary, "", "  ")
	if err != nil {
		return err
	}
	out = append(out, '\n')

	if output != "" {
		if wErr := os.WriteFile(output, out, 0o644); wErr != nil {
			return wErr
		}
		fmt.Printf("Wrote self-check quality summary to %s (total_runs=%d, pass_rate=%.4f)\n", output, totalRuns, passRate)
		return nil
	}
	os.Stdout.Write(out)
	return nil
}

func parseAggregateTrace(path string) (map[string]any, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var trace map[string]any
	if err := json.Unmarshal(data, &trace); err != nil {
		return nil, err
	}
	if _, ok := trace["skill"]; !ok {
		return nil, fmt.Errorf("missing skill")
	}
	if _, ok := trace["final"]; !ok {
		return nil, fmt.Errorf("missing final")
	}
	return trace, nil
}

// aggregateTraces reduces a set of traces into a quality summary with totals,
// pass_rate, per-dimension average scores, and per-skill buckets.
func aggregateTraces(traces []map[string]any) map[string]any {
	dims := splitOnComma(rubricDims)
	statuses := splitOnComma(finalStatuses)

	totals := map[string]any{}
	for _, s := range statuses {
		totals[s] = 0
	}
	totals["total_runs"] = 0

	bySkill := map[string]any{}
	scoreSums := map[string]float64{}
	scoreCount := 0
	for _, d := range dims {
		scoreSums[d] = 0
	}

	for _, trace := range traces {
		skill, _ := trace["skill"].(string)
		if skill == "" {
			skill = "unknown"
		}
		final, _ := trace["final"].(map[string]any)
		status := "UNKNOWN"
		if final != nil {
			if s, ok := final["status"].(string); ok {
				status = s
			}
		}
		if _, ok := totals[status]; ok {
			totals[status] = intOf(totals[status]) + 1
		}
		totals["total_runs"] = intOf(totals["total_runs"]) + 1

		bucket, _ := bySkill[skill].(map[string]any)
		if bucket == nil {
			bucket = map[string]any{
				"total": 0, "PASS": 0, "SAFETY_FAIL": 0, "MAX_ITER": 0, "avg_iterations": 0.0,
			}
			bySkill[skill] = bucket
		}
		bucket["total"] = intOf(bucket["total"]) + 1
		if _, ok := bucket[status]; ok {
			bucket[status] = intOf(bucket[status]) + 1
		}
		iterCount := lenOfList(trace["iterations"])
		prevAvg := floatOf(bucket["avg_iterations"])
		prevTotal := intOf(bucket["total"])
		bucket["avg_iterations"] = (prevAvg*float64(prevTotal-1) + float64(iterCount)) / float64(prevTotal)

		scores := lastCriticScores(trace)
		if len(scores) > 0 {
			scoreCount++
			for _, d := range dims {
				scoreSums[d] += floatOf(scores[d])
			}
		}
	}

	totalRuns := intOf(totals["total_runs"])
	passRate := 0.0
	if totalRuns > 0 {
		passRate = float64(intOf(totals["PASS"])) / float64(totalRuns)
	}
	avgScores := map[string]any{}
	for _, d := range dims {
		if scoreCount > 0 {
			avgScores[d] = round3(scoreSums[d] / float64(scoreCount))
		} else {
			avgScores[d] = nil
		}
	}

	traceFiles := make([]any, 0, len(traces))
	for _, trace := range traces {
		if sp, ok := trace["_source_path"]; ok {
			traceFiles = append(traceFiles, sp)
		}
	}

	return map[string]any{
		"version":           "1.0",
		"generated_at":      time.Now().UTC().Format(time.RFC3339),
		"cloud":             "huaweicloud",
		"metric_namespace":  "CUSTOM.GCL",
		"window":            map[string]any{"trace_count": totalRuns},
		"totals":            totals,
		"pass_rate":         round4(passRate),
		"avg_rubric_scores": avgScores,
		"by_skill":          bySkill,
		"trace_files":       traceFiles,
	}
}

// lastCriticScores returns the critic scores from the final iteration.
func lastCriticScores(trace map[string]any) map[string]float64 {
	iterations, ok := trace["iterations"].([]any)
	if !ok || len(iterations) == 0 {
		return nil
	}
	last, ok := iterations[len(iterations)-1].(map[string]any)
	if !ok {
		return nil
	}
	critic, ok := last["critic"].(map[string]any)
	if !ok {
		return nil
	}
	scores, ok := critic["scores"].(map[string]any)
	if !ok {
		return nil
	}
	out := map[string]float64{}
	for k, v := range scores {
		out[k] = floatOf(v)
	}
	return out
}

func splitOnComma(s string) []string {
	var out []string
	cur := ""
	for _, r := range s {
		if r == ',' {
			out = append(out, cur)
			cur = ""
			continue
		}
		cur += string(r)
	}
	out = append(out, cur)
	return out
}

func intOf(v any) int {
	switch n := v.(type) {
	case int:
		return n
	case int64:
		return int(n)
	case float64:
		return int(n)
	default:
		return 0
	}
}

func floatOf(v any) float64 {
	switch n := v.(type) {
	case float64:
		return n
	case int:
		return float64(n)
	case int64:
		return float64(n)
	default:
		return 0
	}
}

func lenOfList(v any) int {
	if l, ok := v.([]any); ok {
		return len(l)
	}
	return 0
}

func round3(f float64) float64 {
	return float64(int(f*1000+0.5)) / 1000
}

func round4(f float64) float64 {
	return float64(int(f*10000+0.5)) / 10000
}
