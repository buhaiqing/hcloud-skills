// Package cmd: hwcloud-skillcheck trend subcommand.
//
//	hwcloud-skillcheck trend report --root <dir> [--window-days 7] [--json]
//
// Emits the recurrence metric across both trace families (gcl + L4
// orchestrator). The default text summary shows the recurrence rate and
// the top recurring patterns; --json emits the frozen TrendReport schema
// for downstream consumers.
package cmd

import (
	"encoding/json"
	"fmt"
	"sort"

	"github.com/buhaiqing/hcloud-skills/hwcloud-skillcheck/internal/learning"
)

// runTrend dispatches `hwcloud-skillcheck trend <subcommand>`. Currently
// only `report` is exposed — it is the only consumer of ComputeTrendReport.
func runTrend(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: hwcloud-skillcheck trend <report> ...")
	}
	switch args[0] {
	case "report":
		return runTrendReport(args[1:])
	default:
		return fmt.Errorf("unknown trend subcommand: %s; use 'report'", args[0])
	}
}

// runTrendReport prints the cross-trace recurrence summary for the
// given root. Default output is a human-readable text summary; --json
// emits the strict frozen schema.
func runTrendReport(args []string) error {
	fs := newFlagSet("hwcloud-skillcheck trend report")
	root := fs.String("root", ".", "repo root (default: current directory)")
	window := fs.Int("window-days", 7, "recurrence window in days (0 = all traces)")
	jsonOut := fs.Bool("json", false, "emit frozen TrendReport JSON schema")
	if err := fs.Parse(args); err != nil {
		return err
	}

	rep, err := learning.ComputeTrendReport(*root, *window)
	if err != nil {
		return fmt.Errorf("trend report: %w", err)
	}

	if *jsonOut {
		buf, err := json.MarshalIndent(rep, "", "  ")
		if err != nil {
			return fmt.Errorf("marshal trend: %w", err)
		}
		fmt.Println(string(buf))
		return nil
	}

	fmt.Printf("=== Trend Report (window=%dd, root=%s) ===\n", rep.WindowDays, *root)
	fmt.Printf("Total traces:          %d\n", rep.TotalTraces)
	fmt.Printf("Total findings:        %d\n", rep.TotalFindings)
	fmt.Printf("Recurring findings:    %d\n", rep.RecurringFindings)
	fmt.Printf("Recurrence rate:       %.4f\n", rep.RecurrenceRate)

	if len(rep.TracesByCriticType) > 0 {
		fmt.Println("\nTraces by critic_type:")
		// Stable order (alphabetical) so the text summary is diff-friendly
		// across runs — same rationale as the JSON set→slice conversion.
		keys := sortedKeys(rep.TracesByCriticType)
		for _, k := range keys {
			fmt.Printf("  %s: %d\n", k, rep.TracesByCriticType[k])
		}
	}

	if len(rep.ByPattern) == 0 {
		fmt.Println("\nNo findings in window.")
		return nil
	}
	fmt.Println("\nTop recurring patterns:")
	shown := 0
	for _, p := range rep.ByPattern {
		if p.Count < 2 {
			// Already sorted DESC by count, so first time Count<2 we're
			// past the recurring tail — non-recurring unique findings are
			// not the headline this summary exists to surface.
			break
		}
		fmt.Printf("  [%d×] %s  (first=%s last=%s)\n",
			p.Count, p.PatternKey, p.FirstSeen, p.LastSeen)
		shown++
	}
	if shown == 0 {
		fmt.Println("  (none — every finding was unique in the window)")
	}
	return nil
}

// sortedKeys returns the keys of m in ascending order, so the text
// summary is diff-friendly across runs (same rationale as the JSON
// set→slice conversion in internal/learning/trend.go).
func sortedKeys(m map[string]int) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}
