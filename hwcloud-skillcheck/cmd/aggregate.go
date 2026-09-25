package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"time"

	"github.com/buhaiqing/hcloud-skills/hwcloud-skillcheck/internal/embed"
	"github.com/buhaiqing/hcloud-skills/hwcloud-skillcheck/internal/learning"
	"golang.org/x/sync/errgroup"
)

// runAggregate dispatches the `hwcloud-skillcheck aggregate` subcommands.
func runAggregate(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("aggregate: missing subcommand (use 'trace')")
	}
	switch args[0] {
	case "trace":
		return runAggregateTrace(args[1:])
	case "-h", "--help", "help":
		fmt.Fprintln(os.Stdout, "hwcloud-skillcheck aggregate trace --root <dir> [--since-hours N] [--output FILE] [--require-traces] [--require-evidence] [--reject-invalid] [--self-check]")
		return nil
	default:
		return fmt.Errorf("aggregate: unknown subcommand %q", args[0])
	}
}

const (
	rubricDims    = "correctness,safety,idempotency,traceability,spec_compliance"
	finalStatuses = "PASS,SAFETY_FAIL,MAX_ITER"
)

// runAggregateTrace aggregates audit-results traces into a quality summary,
// mirroring scripts/gcl_trace_aggregate.py. Both writers' output is read:
// gcl-trace-*.json (internal/gcl.PersistTrace) and orchestrator-trace-*.json
// (internal/l4.HandleFault) — see learning.TraceFilePatterns.
//
// Every candidate is classified in the frozen order parse → schema-invalid →
// smoke → evidence (learning.ClassifyTrace): only evidence traces move a
// metric, and evidence_runs is exactly their number (pass_rate's denominator).
// A trace that fails read/JSON parsing is counted as unparseable; a trace that
// fails canonical-schema validation is counted in invalid_trace. Both are
// untrusted input, WARNed by file name, and excluded from pass_rate, rubric
// averages, by_skill, by_source, by_critic_type, and evidence_runs. When no
// trace files exist it WARNs and returns nil (exit 0) per Spec §4 by default —
// trace files are produced by the runtime runner, so an external user may
// legitimately have none. Pass --require-traces to fail when no trace files
// exist. Callers use --reject-invalid to fail closed on untrusted input while
// allowing zero evidence.
func runAggregateTrace(args []string) error {
	fs := newFlagSet("hwcloud-skillcheck aggregate trace")
	root := fs.String("root", ".", "skill repository root")
	sinceHours := fs.Int("since-hours", -1, "only traces modified within N hours")
	output := fs.String("output", "", "write summary to FILE instead of stdout")
	selfCheck := fs.Bool("self-check", false, "aggregate the embedded trace fixture instead of the repo")
	requireTraces := fs.Bool("require-traces", false, "fail (exit 1) instead of warning when no trace files exist")
	requireEvidence := fs.Bool("require-evidence", false, "fail (exit 1) when no trace carried a verification signal (all smoke or schema-invalid)")
	rejectInvalid := fs.Bool("reject-invalid", false, "fail (exit 1) when any trace is schema-invalid or unparseable")
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
	for _, pat := range learning.TraceFilePatterns {
		entries, gErr := filepath.Glob(filepath.Join(auditDir, pat))
		if gErr != nil {
			continue
		}
		paths = append(paths, entries...)
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
		// With --require-traces, fall back to the embedded fixture so the gate
		// still exercises the parsing/aggregation path. This prevents a fresh
		// CI checkout from failing the gate while still guaranteeing the
		// binary's aggregate code is run end-to-end on every pre-commit / CI.
		if *requireTraces {
			fmt.Fprintf(os.Stderr, "INFO: no trace files under %s (--require-traces set; falling back to embedded fixture self-check)\n", auditDir)
			return runAggregateSelfCheck(*output)
		}
		if *requireEvidence {
			// Zero trace files is zero verification signal; the explicit
			// --require-evidence gate must not pass on it.
			return fmt.Errorf("aggregate: --require-evidence set but no trace files found under %s", auditDir)
		}
		fmt.Fprintln(os.Stderr, "WARN: no trace files found; skipping aggregate (trace files are produced by the runtime runner)")
		return nil
	}

	// Fan-out parseAggregateTrace across paths with bounded concurrency.
	// Each trace file is read+decoded+classified in its own goroutine; the
	// wall-clock win is roughly NumCPU on a CI box with 100+ trace files (was
	// serial). Results are collected into per-index slots so we never need a
	// mutex — errgroup.Wait gives us happens-before across all of them.
	// A nil slot is a file that did not parse at all.
	var (
		slots         = make([]*consumedTrace, len(paths))
		skips         = make([]string, len(paths))
		parseFailures = make([]string, len(paths))
	)
	g, gCtx := errgroup.WithContext(context.Background())
	g.SetLimit(runtime.NumCPU())
	for i, p := range paths {
		i, p := i, p
		g.Go(func() error {
			select {
			case <-gCtx.Done():
				return gCtx.Err()
			default:
			}
			trace, raw, perr := parseAggregateTrace(p)
			rel, _ := filepath.Rel(rootDir, p)
			if perr != nil {
				parseFailures[i] = fmt.Sprintf("skip %s: %v", rel, perr)
				return nil
			}
			class, schemaErrs := learning.ClassifyTrace(raw, trace)
			trace["_source_path"] = rel
			slots[i] = &consumedTrace{Trace: trace, Class: class}
			if class == learning.TraceInvalid {
				// One WARN per rejected file, naming it and the first
				// violation: a crafted audit-results/ file must be visible,
				// not silently dropped.
				skips[i] = fmt.Sprintf("%s: invalid trace (excluded from every metric and from learning): %s", rel, schemaErrs[0])
			}
			return nil
		})
	}
	if wErr := g.Wait(); wErr != nil {
		return wErr
	}
	// Drain skip warnings now (after all goroutines done) to avoid interleaved stderr.
	for i, s := range skips {
		if s != "" {
			fmt.Fprintln(os.Stderr, "WARN:", s)
		}
		if parseFailures[i] != "" {
			fmt.Fprintln(os.Stderr, "WARN:", parseFailures[i])
		}
	}
	unparseable := 0
	for _, failure := range parseFailures {
		if failure != "" {
			unparseable++
		}
	}
	if *rejectInvalid && unparseable > 0 {
		return fmt.Errorf("aggregate: unparseable=%d trace file(s) found; --reject-invalid rejects untrusted trace input", unparseable)
	}

	// Compact: drop nil slots left by unparseable traces.
	parsed := make([]*consumedTrace, 0, len(slots))
	for _, t := range slots {
		if t != nil {
			parsed = append(parsed, t)
		}
	}
	if len(parsed) == 0 {
		if *requireTraces {
			return fmt.Errorf("aggregate: no valid traces parsed under %s (--require-traces set)", auditDir)
		}
		if *requireEvidence {
			return fmt.Errorf("aggregate: --require-evidence set but no trace under %s parsed", auditDir)
		}
		fmt.Fprintln(os.Stderr, "WARN: no valid traces parsed; skipping aggregate")
		return nil
	}
	summary := aggregateTraces(parsed)
	invalidTrace := intOf(summary["invalid_trace"])
	if *rejectInvalid && invalidTrace > 0 {
		return fmt.Errorf("aggregate: %d invalid trace(s) found (invalid_trace=%d); rejecting untrusted trace input", invalidTrace, invalidTrace)
	}
	evidenceRuns := intOf(summary["evidence_runs"])
	if evidenceRuns == 0 {
		if *requireEvidence {
			return fmt.Errorf("aggregate: --require-evidence set but no trace carried a verification signal (parsed=%d, skipped_smoke=%d, invalid_trace=%d)",
				len(parsed), intOf(summary["skipped_smoke"]), invalidTrace)
		}
		fmt.Fprintf(os.Stderr, "WARN: 0 evidence: %d trace file(s) parsed but none carried a verification signal (skipped_smoke=%d, invalid_trace=%d); use --require-evidence for release validation\n",
			len(parsed), intOf(summary["skipped_smoke"]), invalidTrace)
	}

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
		byCriticType, _ := summary["by_critic_type"].(map[string]int)
		fmt.Printf("Wrote quality summary to %s (evidence_runs=%d, pass_rate=%.4f, skipped_smoke=%d, invalid_trace=%d, by_critic_type={structural:%d, external:%d, unknown:%d}, l2_skipped_no_schema=%d)\n",
			outPath, evidenceRuns, summary["pass_rate"].(float64), intOf(summary["skipped_smoke"]), intOf(summary["invalid_trace"]),
			byCriticType[criticTypeStructural], byCriticType[criticTypeExternal], byCriticType[criticTypeUnknown], intOf(summary["l2_skipped_no_schema"]))
		return nil
	}
	os.Stdout.Write(out)
	return nil
}

// consumedTrace is one parsed trace file plus its consumption class (always set
// by learning.ClassifyTrace). The operator-facing WARN for a rejected file is
// emitted by the reader, which is where the schema errors are available.
type consumedTrace struct {
	Trace map[string]any
	Class learning.TraceClass
}

// critic_type vocabulary. The frozen contract for final.critic_type is
// {structural, external}: structural is the deterministic dry-run proxy,
// external is a real Critic. Anything else — including an absent field, or a
// self-declared critic name nothing in the repo implements — is counted as
// "unknown" so a fake critic_type cannot inflate a trusted bucket.
const (
	criticTypeStructural = "structural"
	criticTypeExternal   = "external"
	criticTypeUnknown    = "unknown"
)

// criticTypeOf normalizes a trace's final.critic_type into the frozen
// vocabulary.
func criticTypeOf(final map[string]any) string {
	if final == nil {
		return criticTypeUnknown
	}
	switch s, _ := final["critic_type"].(string); s {
	case criticTypeStructural, criticTypeExternal:
		return s
	default:
		return criticTypeUnknown
	}
}

// runAggregateSelfCheck aggregates the embedded healthy trace fixture and
// verifies the resulting summary is well-formed (evidence_runs >= 1, pass_rate
// in [0,1]). This proves the aggregation path is wired correctly inside the
// binary without requiring repo trace files — and, since the fixture goes
// through the same classification as any real trace, that the fixture still
// conforms to the canonical schema and still counts as evidence.
func runAggregateSelfCheck(output string) error {
	var trace map[string]any
	if err := json.Unmarshal(embed.TraceHealthy, &trace); err != nil {
		return fmt.Errorf("self-check: bad embedded trace fixture: %w", err)
	}
	class, schemaErrs := learning.ClassifyTrace(embed.TraceHealthy, trace)
	if class != learning.TraceEvidence {
		return fmt.Errorf("self-check: embedded trace fixture is not usable evidence (class=%d, violations=%v)", class, schemaErrs)
	}
	trace["_source_path"] = "embedded:fixtures/gcl-trace-healthy.json"
	summary := aggregateTraces([]*consumedTrace{{Trace: trace, Class: class}})

	totalRuns := intOf(summary["totals"].(map[string]any)["total_runs"])
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

// parseAggregateTrace decodes one trace file, returning the decoded payload
// plus the raw bytes (the raw bytes are what canonical-schema validation runs
// on — validating the re-encoded map would silently "repair" a crafted file,
// e.g. by normalizing a float where the schema demands an integer). Only
// `skill` is required here: a trace without it cannot be bucketed at all. A
// missing `final` block is NOT a parse error — such a trace is classified as
// schema-invalid (it no longer conforms to the canonical contract) and surfaces
// under invalid_trace instead of being folded into an indistinguishable "skip".
func parseAggregateTrace(path string) (map[string]any, []byte, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, nil, err
	}
	var trace map[string]any
	if err := json.Unmarshal(data, &trace); err != nil {
		return nil, nil, err
	}
	if _, ok := trace["skill"]; !ok {
		return nil, nil, fmt.Errorf("missing skill")
	}
	return trace, data, nil
}

// aggregateTraces reduces classified traces into a quality summary with totals,
// pass_rate, per-dimension average scores, per-skill buckets, and the trust
// counters (invalid_trace / skipped_smoke / evidence_runs / by_critic_type).
//
// The classification order is frozen (learning.ClassifyTrace — applied by the
// reader, since unparseable files never reach here):
//
//	parse (reader) → schema-invalid (invalid_trace) → smoke (skipped_smoke) → evidence
//
// A schema-invalid trace is untrusted input: it is counted in invalid_trace and
// excluded from every metric. Smoke traces (see learning.IsSmokeTrace) carry no
// verification signal, so folding them in would either fake a pass rate or tank
// it with unverified dry runs. evidence_runs counts the traces that did
// contribute (schema-valid, non-smoke) and is pass_rate's denominator. by_source
// splits those contributing traces by writer (gcl vs l4) — the observability
// hook for "is the L4 loop actually feeding the aggregator?" — and
// by_critic_type records which Critic produced their scores (a self-declared
// critic name outside {structural, external} is counted as "unknown").
func aggregateTraces(traces []*consumedTrace) map[string]any {
	dims := splitOnComma(rubricDims)
	statuses := splitOnComma(finalStatuses)

	totals := map[string]any{}
	for _, s := range statuses {
		totals[s] = 0
	}
	totals["total_runs"] = 0

	bySkill := map[string]any{}
	bySource := map[string]int{learning.TraceSourceGCL: 0, learning.TraceSourceL4: 0}
	byCriticType := map[string]int{criticTypeStructural: 0, criticTypeExternal: 0, criticTypeUnknown: 0}
	scoreSums := map[string]float64{}
	scoreCount := 0
	skippedSmoke := 0
	invalidTrace := 0
	l2SkippedNoSchema := 0
	for _, d := range dims {
		scoreSums[d] = 0
	}

	for _, item := range traces {
		if item == nil {
			// Defensive: an unclassified slot must not move a metric.
			invalidTrace++
			continue
		}
		switch item.Class {
		case learning.TraceInvalid:
			invalidTrace++
			continue
		case learning.TraceSmoke:
			skippedSmoke++
			continue
		}
		trace := item.Trace
		// "source" is absent on traces written before the field existed; those
		// are gcl traces, per learning.TraceSource.
		src := learning.TraceSource(trace)
		bySource[src]++

		if l2StatusOf(trace) == "skipped_no_schema" {
			l2SkippedNoSchema++
		}

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
		} else {
			// Defensive only: a schema-valid trace always carries an enum
			// status, so an out-of-enum value lands in invalid_trace before it
			// can reach this loop. If the contract's enum ever gains a value
			// the aggregator does not know, it must still stay visible: it
			// counts in total_runs, so leaving it out of every bucket would
			// depress pass_rate with nothing to explain it.
			totals["UNKNOWN"] = intOf(totals["UNKNOWN"]) + 1
		}
		totals["total_runs"] = intOf(totals["total_runs"]) + 1
		byCriticType[criticTypeOf(final)]++
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
		} else {
			bucket["UNKNOWN"] = intOf(bucket["UNKNOWN"]) + 1
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
	for _, item := range traces {
		if item == nil {
			continue
		}
		// Every parsed input is listed, including smoke and schema-invalid
		// ones: the summary names the window it was computed from, and the
		// per-file reasons live in the WARN lines.
		if sp, ok := item.Trace["_source_path"]; ok {
			traceFiles = append(traceFiles, sp)
		}
	}

	return map[string]any{
		"version":          "1.0",
		"generated_at":     time.Now().UTC().Format(time.RFC3339),
		"cloud":            "huaweicloud",
		"metric_namespace": "CUSTOM.GCL",
		"window":           map[string]any{"trace_count": totalRuns},
		"totals":           totals,
		// evidence_runs = totals.total_runs = the contributing (schema-valid,
		// non-smoke) traces; it is pass_rate's denominator.
		"evidence_runs":        totalRuns,
		"pass_rate":            round4(passRate),
		"avg_rubric_scores":    avgScores,
		"by_skill":             bySkill,
		"by_source":            bySource,
		"by_critic_type":       byCriticType,
		"skipped_smoke":        skippedSmoke,
		"invalid_trace":        invalidTrace,
		"l2_skipped_no_schema": l2SkippedNoSchema,
		"trace_files":          traceFiles,
	}
}

// l2StatusOf returns the L2 hallucination-check outcome recorded on a trace
// (L2Result.Outcome, JSON field "status"), or "" when the trace carries none.
// Traces written before that field existed encode the status as the
// "<status>: <detail>" prefix of l2.details, which is parsed here as a fallback.
func l2StatusOf(trace map[string]any) string {
	hd, _ := trace["hallucination_detection"].(map[string]any)
	if hd == nil {
		return ""
	}
	l2, _ := hd["l2"].(map[string]any)
	if l2 == nil {
		return ""
	}
	if s, ok := l2["status"].(string); ok {
		return s
	}
	details, _ := l2["details"].(string)
	if token, _, ok := strings.Cut(details, ": "); ok {
		return token
	}
	return ""
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
