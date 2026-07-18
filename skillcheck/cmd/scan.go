package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/buhaiqing/hcloud-skills/skillcheck/internal/embed"
	"github.com/buhaiqing/hcloud-skills/skillcheck/internal/security"
)

// runScan dispatches the `skillcheck scan` subcommands.
func runScan(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("scan: missing subcommand (use 'secret')")
	}
	switch args[0] {
	case "secret":
		return runScanSecret(args[1:])
	case "-h", "--help", "help":
		fmt.Fprintln(os.Stdout, "skillcheck scan secret <trace|summary|alarm-plan> --root <dir> [--latest] [--include-fixture] [--json] [--self-check]")
		return nil
	default:
		return fmt.Errorf("scan: unknown subcommand %q", args[0])
	}
}

const (
	globTrace      = "audit-results/gcl-trace-*.json"
	globSummary    = "audit-results/gcl-quality-summary-*.json"
	globAlarmPlan  = "audit-results/gcl-alarm-plan-*.json"
	fixtureSummary = "scripts/fixtures/gcl-quality-summary-healthy.json"
	fixtureAlarm   = "scripts/fixtures/gcl-alarm-plan-healthy.json"
)

// scanSecretResult describes one scanned artifact.
type scanSecretResult struct {
	artifact string // display path
	ok       bool
	leaks    int
	error    string
}

// runScanSecret scans GCL artifacts (trace / summary / alarm-plan) for secret
// leaks, mirroring scripts/check_gcl_{trace,summary,alarm_plan}_security.py.
// When no explicit inputs and no matching files exist, it returns ok (the
// Python scripts default to --allow-empty behavior via exit 0).
func runScanSecret(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("scan secret: missing artifact kind (trace|summary|alarm-plan)")
	}
	kind := args[0]
	rest := args[1:]

	fs := newFlagSet("skillcheck scan secret " + kind)
	root := fs.String("root", ".", "skill repository root")
	latest := fs.Bool("latest", false, "scan only the latest artifact when no explicit input")
	includeFixture := fs.Bool("include-fixture", false, "also scan the healthy fixture (CI smoke)")
	jsonOut := fs.Bool("json", false, "emit JSON report")
	selfCheck := fs.Bool("self-check", false, "scan embedded fixtures instead of the repo")
	if err := fs.Parse(rest); err != nil {
		return err
	}

	if *selfCheck {
		return runScanSelfCheck(kind, *jsonOut)
	}

	rootDir, err := filepath.Abs(*root)
	if err != nil {
		return err
	}

	var glob string
	switch kind {
	case "trace":
		glob = globTrace
	case "summary":
		glob = globSummary
	case "alarm-plan":
		glob = globAlarmPlan
	default:
		return fmt.Errorf("scan secret: unknown artifact kind %q", kind)
	}

	// Explicit positional inputs (treated as relative to root unless absolute).
	var inputs []string
	for _, a := range fs.Args() {
		p := a
		if !filepath.IsAbs(p) {
			p = filepath.Join(rootDir, a)
		}
		inputs = append(inputs, p)
	}

	paths := collectScanPaths(rootDir, inputs, glob, *latest)
	// --include-fixture appends the healthy fixture to the scan set.
	if len(inputs) == 0 && *includeFixture {
		fp := filepath.Join(rootDir, fixtureFor(kind))
		if fileExists(fp) {
			paths = append(paths, fp)
		}
	}

	results := scanArtifactPaths(paths)

	// No artifacts and no explicit input => treat as ok (allow-empty).
	if len(results) == 0 {
		if *jsonOut {
			fmt.Println(`{"ok":true,"results":[]}`)
		} else {
			fmt.Printf("OK: no GCL %s files found\n", kind)
		}
		return nil
	}

	ok := true
	for _, r := range results {
		if !r.ok {
			ok = false
		}
	}

	if *jsonOut {
		printScanJSON(kind, ok, results)
	} else {
		for _, r := range results {
			status := "OK"
			if !r.ok {
				status = "FAIL"
			}
			fmt.Printf("%s: %s\n", status, r.artifact)
			if r.error != "" {
				fmt.Printf("  - error: %s\n", r.error)
			}
		}
	}

	if !ok {
		return fmt.Errorf("scan secret %s: %d artifact(s) leaked secrets", kind, countLeaks(results))
	}
	return nil
}

func fixtureFor(kind string) string {
	switch kind {
	case "summary":
		return fixtureSummary
	case "alarm-plan":
		return fixtureAlarm
	default:
		return ""
	}
}

// collectScanPaths resolves the file set to scan: explicit inputs (kept only if
// they exist), otherwise the glob under root, optionally restricted to the
// latest when --latest is set.
func collectScanPaths(root string, inputs []string, glob string, latest bool) []string {
	if len(inputs) > 0 {
		var out []string
		for _, p := range inputs {
			if fileExists(p) {
				out = append(out, p)
			}
		}
		sort.Strings(out)
		return out
	}
	matches, _ := filepath.Glob(filepath.Join(root, glob))
	sort.Strings(matches)
	if latest && len(matches) > 0 {
		return matches[len(matches)-1:]
	}
	return matches
}

func scanArtifactPaths(paths []string) []scanSecretResult {
	results := make([]scanSecretResult, 0, len(paths))
	for _, p := range paths {
		res := scanSecretResult{artifact: p}
		data, err := os.ReadFile(p)
		if err != nil {
			res.error = err.Error()
			results = append(results, res)
			continue
		}
		// Verify it is JSON-decodable; a decode error is reported but not a leak.
		var probe any
		if jErr := json.Unmarshal(data, &probe); jErr != nil {
			res.error = "invalid JSON: " + jErr.Error()
			results = append(results, res)
			continue
		}
		findings, _ := security.ScanContent(data)
		res.leaks = len(findings)
		res.ok = len(findings) == 0
		results = append(results, res)
	}
	return results
}

func countLeaks(results []scanSecretResult) int {
	n := 0
	for _, r := range results {
		if !r.ok {
			n++
		}
	}
	return n
}

func printScanJSON(kind string, ok bool, results []scanSecretResult) {
	fmt.Println("{")
	fmt.Printf("  \"ok\": %v,\n", ok)
	fmt.Println("  \"results\": [")
	for i, r := range results {
		fmt.Printf("    {\"%s\": %q, \"ok\": %v, \"leaks\": %d}",
			kind, r.artifact, r.ok, r.leaks)
		if i < len(results)-1 {
			fmt.Println(",")
		} else {
			fmt.Println()
		}
	}
	fmt.Println("  ]")
	fmt.Println("}")
}

// runScanSelfCheck scans the embedded healthy fixtures to verify the binary
// itself is leak-free. trace has no embed fixture (repo ships no
// gcl-trace-healthy.json), so it is skipped/allowed.
func runScanSelfCheck(kind string, jsonOut bool) error {
	if kind == "trace" {
		if jsonOut {
			fmt.Println(`{"ok":true,"skipped":["trace"]}`)
		} else {
			fmt.Println("OK: self-check: no embedded trace fixture; skipping (repo has no gcl-trace-healthy.json)")
		}
		return nil
	}

	var fixture []byte
	switch kind {
	case "summary":
		fixture = embed.SummaryHealthy
	case "alarm-plan":
		fixture = embed.AlarmPlanHealthy
	default:
		return fmt.Errorf("scan secret: unknown artifact kind %q", kind)
	}

	findings, _ := security.ScanContent(fixture)
	if len(findings) > 0 {
		if jsonOut {
			fmt.Printf("{\"ok\":false,\"kind\":%q,\"leaks\":%d}\n", kind, len(findings))
		} else {
			fmt.Printf("FAIL: self-check %s fixture leaked %d secret(s)\n", kind, len(findings))
		}
		return fmt.Errorf("self-check %s fixture leaked secrets", kind)
	}
	if jsonOut {
		fmt.Printf("{\"ok\":true,\"kind\":%q}\n", kind)
	} else {
		fmt.Printf("OK: self-check %s fixture is clean\n", kind)
	}
	return nil
}
