// Package cmd: hwcloud-skillcheck learning subcommand.
//
//	hwcloud-skillcheck learning gen --root <dir>            # regenerate failure_patterns.json
//	                                                # + remediation-playbooks.json for all
//	                                                # top-frequency skills (RDS/VPC/ELB/CCE).
//	hwcloud-skillcheck learning trace aggregate --root <dir> --skill <skill> [--since-hours N] [--dry-run]
//	hwcloud-skillcheck learning trace learn    --root <dir> --skill <skill> --trace <path> [--dry-run]
//	hwcloud-skillcheck learning trace report   --root <dir> --skill <skill> [--json]
package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/buhaiqing/hcloud-skills/hwcloud-skillcheck/internal/learning"
)

func runLearning(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: hwcloud-skillcheck learning <gen|trace> ...")
	}
	switch args[0] {
	case "gen":
		return runLearningGen(args[1:])
	case "trace":
		return runLearningTrace(args[1:])
	case "-h", "--help", "help":
		fmt.Fprint(os.Stderr, learningHelp)
		return nil
	default:
		return fmt.Errorf("unknown learning subcommand %q", args[0])
	}
}

const learningHelp = `hwcloud-skillcheck learning — knowledge base + GCL trace utilities

Usage:
  hwcloud-skillcheck learning gen [--root <dir>]
  hwcloud-skillcheck learning trace <aggregate|learn|report> [--root <dir>] [--skill <id>] [--since-hours N] [--dry-run] [--json]
`

func runLearningGen(args []string) error {
	fs := newFlagSet("hwcloud-skillcheck learning gen")
	root := fs.String("root", ".", "repo root (default: current directory)")
	if err := fs.Parse(args); err != nil {
		return err
	}
	count, err := learning.GenerateAll(*root)
	if err != nil {
		return err
	}
	fmt.Printf("learning gen: wrote assets for %d skills under %s\n", count, *root)
	return nil
}

func runLearningTrace(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: hwcloud-skillcheck learning trace <aggregate|learn|report> ...")
	}
	switch args[0] {
	case "aggregate":
		return runTraceAggregate(args[1:])
	case "learn":
		return runTraceLearn(args[1:])
	case "report":
		return runTraceReport(args[1:])
	default:
		return fmt.Errorf("unknown trace subcommand %q", args[0])
	}
}

func runTraceAggregate(args []string) error {
	fs := newFlagSet("hwcloud-skillcheck learning trace aggregate")
	root := fs.String("root", ".", "repo root")
	skill := fs.String("skill", "", "skill id (e.g. huaweicloud-ecs-ops)")
	since := fs.Int("since-hours", 0, "only traces newer than N hours (0 = all)")
	dry := fs.Bool("dry-run", false, "print results without writing")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *skill == "" {
		return fmt.Errorf("--skill is required")
	}
	var sincePtr *int
	if *since > 0 {
		sincePtr = since
	}
	res, err := learning.Aggregate(*root, *skill, sincePtr, *dry)
	if err != nil {
		return err
	}
	fmt.Printf("Traces scanned: %d\n  New patterns: %d\n  Updated patterns: %d\n  Skipped (no failure): %d\n",
		res.Scanned, res.NewCount, res.UpdatedCount, res.SkippedCount)
	if res.WrittenTo != "" {
		fmt.Printf("\nWritten: %s\n", res.WrittenTo)
	}
	return nil
}

func runTraceLearn(args []string) error {
	fs := newFlagSet("hwcloud-skillcheck learning trace learn")
	root := fs.String("root", ".", "repo root")
	skill := fs.String("skill", "", "skill id")
	tracePath := fs.String("trace", "", "path to gcl-trace-*.json")
	dry := fs.Bool("dry-run", false, "print without writing")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *skill == "" || *tracePath == "" {
		return fmt.Errorf("--skill and --trace are required")
	}
	// Reuse Aggregate() with a 1-trace scan: load the file, inject as a
	// single-trace scan, run the same dedup/merge path.
	if !filepath.IsAbs(*tracePath) {
		*tracePath = filepath.Join(*root, *tracePath)
	}
	raw, err := os.ReadFile(*tracePath)
	if err != nil {
		return fmt.Errorf("read trace: %w", err)
	}
	var data map[string]any
	if err := json.Unmarshal(raw, &data); err != nil {
		return fmt.Errorf("parse trace: %w", err)
	}
	fp := learning.ExtractPatternFromTrace(data)
	if fp == nil {
		fmt.Println("No failure_pattern in trace (likely a PASS). Nothing to learn.")
		return nil
	}
	existing := learning.LoadFailurePatterns(*root, *skill)
	patterns, _ := existing["patterns"].([]any)
	key := fmt.Sprintf("%v|%v|%v", fp["category"], fp["error"], firstTokenString(fp["command"]))
	matched := false
	for _, p := range patterns {
		pm, ok := p.(map[string]any)
		if !ok {
			continue
		}
		cat, _ := pm["category"].(string)
		sig, _ := pm["signature"].(map[string]any)
		errStr, _ := sig["error_message_regex"].(string)
		cmdPat, _ := sig["command_pattern"].(string)
		k := fmt.Sprintf("%s|%s|%s", cat, errStr, cmdPat)
		if k == key {
			learning.MergePattern(pm, fp, filepath.Base(*tracePath))
			fmt.Printf("Updated existing pattern: %s\n", pm["id"])
			matched = true
			break
		}
	}
	if !matched {
		nextNum := learning.MaxPatternID(patterns) + 1
		entry := learning.CreatePatternEntry(fp, *skill, nextNum, filepath.Base(*tracePath))
		patterns = append(patterns, entry)
		fmt.Printf("Created new pattern: %s (%s)\n", entry["id"], entry["category"])
	}
	existing["patterns"] = patterns
	if *dry {
		fmt.Println("[DRY-RUN] No changes written.")
		return nil
	}
	out, err := learning.SaveFailurePatterns(*root, *skill, existing)
	if err != nil {
		return err
	}
	fmt.Printf("Written: %s\n", out)
	return nil
}

func runTraceReport(args []string) error {
	fs := newFlagSet("hwcloud-skillcheck learning trace report")
	root := fs.String("root", ".", "repo root")
	skill := fs.String("skill", "", "skill id")
	jsonOut := fs.Bool("json", false, "emit JSON")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *skill == "" {
		return fmt.Errorf("--skill is required")
	}
	data := learning.LoadFailurePatterns(*root, *skill)
	patterns, _ := data["patterns"].([]any)
	meta, _ := data["meta"].(map[string]any)
	if *jsonOut {
		out := map[string]any{
			"skill":    *skill,
			"patterns": patterns,
			"meta":     meta,
		}
		buf, _ := json.MarshalIndent(out, "", "  ")
		fmt.Println(string(buf))
		return nil
	}
	fmt.Printf("=== Failure Knowledge Base: %s ===\n", *skill)
	fmt.Printf("Total patterns: %d\n", len(patterns))
	if v, ok := meta["last_aggregation"].(string); ok {
		fmt.Printf("Last aggregation: %s\n", v)
	}
	if v, ok := meta["source_traces_analyzed"].(float64); ok {
		fmt.Printf("Source traces analyzed: %d\n", int(v))
	}
	byCat := map[string]int{}
	for _, p := range patterns {
		pm, ok := p.(map[string]any)
		if !ok {
			continue
		}
		c, _ := pm["category"].(string)
		byCat[c]++
	}
	fmt.Println("\nBy category:")
	for c, n := range byCat {
		fmt.Printf("  %s: %d\n", c, n)
	}
	return nil
}

func firstTokenString(v any) string {
	s, _ := v.(string)
	if s == "" {
		return ""
	}
	for i, r := range s {
		if r == ' ' || r == '\t' {
			return s[:i]
		}
	}
	return s
}
