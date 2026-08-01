// Package cmd: hwcloud-skillcheck check merge-verify — parallel-execution
// collection gate. After multiple subagents modify files in parallel, the
// orchestrator runs this to (a) validate cross-file consistency (markdown
// links / backtick path targets) across every changed file, and (b) emit a
// per-agent execution log summarising who changed what and whether it verifies.
//
//	hwcloud-skillcheck check merge-verify --log <parallel-log.json> [--root <dir>]
package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// MergeEntry is one subagent's contribution to a parallel execution: which
// agent changed which files and with what result.
type MergeEntry struct {
	Agent  string   `json:"agent"`
	Files  []string `json:"files"`
	Status string   `json:"status"` // success | failed
	Result string   `json:"result,omitempty"`
}

// runCheckMergeVerify is the CLI entry for `check merge-verify`.
func runCheckMergeVerify(args []string) error {
	fs := newFlagSet("hwcloud-skillcheck check merge-verify")
	root := fs.String("root", ".", "skill repository root")
	logPath := fs.String("log", "", "path to parallel-execution log JSON")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *logPath == "" {
		return fmt.Errorf("--log is required")
	}
	rootDir, err := filepath.Abs(*root)
	if err != nil {
		return err
	}
	entries, err := loadMergeLog(*logPath)
	if err != nil {
		return err
	}
	report := runMergeVerify(rootDir, entries)
	fmt.Print(report)
	// Cross-file link failures are hard failures for a parallel merge.
	if hasMergeVerifyFailures(entries) {
		return fmt.Errorf("merge-verify failed: %d subagent(s) reported failure", countMergeFailures(entries))
	}
	return nil
}

// loadMergeLog parses the parallel-execution log JSON into entries.
func loadMergeLog(path string) ([]MergeEntry, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read log: %w", err)
	}
	var entries []MergeEntry
	if err := json.Unmarshal(raw, &entries); err != nil {
		return nil, fmt.Errorf("parse log: %w", err)
	}
	return entries, nil
}

// runMergeVerify validates every changed file (cross-file markdown-link /
// backtick-target consistency) and renders a per-agent execution report.
// Returns a human-readable report string.
func runMergeVerify(root string, entries []MergeEntry) string {
	var b strings.Builder
	fmt.Fprintf(&b, "=== parallel merge verification ===\n")
	broken := 0
	for _, e := range entries {
		fmt.Fprintf(&b, "[agent %s] status=%s files=%d\n", e.Agent, e.Status, len(e.Files))
		for _, f := range e.Files {
			full := f
			if !filepath.IsAbs(full) {
				full = filepath.Join(root, f)
			}
			findings := checkMarkdownFile(root, full)
			if len(findings) > 0 {
				broken++
				for _, fi := range findings {
					fmt.Fprintf(&b, "  !! %s\n", fi)
				}
			} else if fileExists(full) {
				fmt.Fprintf(&b, "  ok %s\n", f)
			}
		}
		if e.Result != "" {
			fmt.Fprintf(&b, "  result: %s\n", e.Result)
		}
	}
	if broken == 0 {
		fmt.Fprintf(&b, "cross-file link validation: PASS\n")
	} else {
		fmt.Fprintf(&b, "cross-file link validation: FAIL (%d broken reference(s))\n", broken)
	}
	return b.String()
}

// hasMergeVerifyFailures reports whether any subagent reported status=failed.
func hasMergeVerifyFailures(entries []MergeEntry) bool {
	return countMergeFailures(entries) > 0
}

// countMergeFailures returns how many subagents reported status=failed.
func countMergeFailures(entries []MergeEntry) int {
	n := 0
	for _, e := range entries {
		if e.Status == "failed" {
			n++
		}
	}
	return n
}
