package cmd

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

// AGENTS.md mechanical invariants enforced by `validate agents`.
//
// These limits come from CADL: AGENTS.md is read on every task, so it must
// stay lean and the active curated-asset (CA) section must stay within budget.
// The mechanical gate catches regression without reading the prose; the
// archive pointer in AGENTS.md keeps the full Why/How-to-apply for retired CAs.
const (
	agentsMaxLines     = 500
	agentsMaxCAEntries = 12
)

// caHeadingRE matches the start of an active CA entry: `### CA-N. <title>`.
// Archived entries (already pointing at ca-archive.md) are not counted.
var caHeadingRE = regexp.MustCompile(`^### CA-(\d+)\.\s`)

// runValidateAgents handles:
//
//	hwcloud-skillcheck validate agents --root <dir>
//
// It enforces three mechanical invariants on <dir>/AGENTS.md:
//  1. total line count <= agentsMaxLines
//  2. active `### CA-N.` entry count <= agentsMaxCAEntries
//  3. CA numbers form a contiguous, gap-free sequence with no duplicates
//
// Exit 0 = all checks pass, 1 = any check fails (missing AGENTS.md is also a
// hard error — the file is mandatory at the repo root).
func runValidateAgents(args []string) error {
	fs := newFlagSet("validate agents")
	root := fs.String("root", ".", "skill repository root (default: current directory)")
	if err := fs.Parse(args); err != nil {
		return err
	}
	rootDir, err := filepath.Abs(*root)
	if err != nil {
		return err
	}

	path := filepath.Join(rootDir, "AGENTS.md")
	// Track errors and warnings separately so a failure still reports the
	// full picture to the operator; one missing file shouldn't hide a CA
	// numbering bug detected during a partial read.
	var errs []string

	f, openErr := os.Open(path)
	if openErr != nil {
		errs = append(errs, fmt.Sprintf("open %s: %v", path, openErr))
		fmt.Fprintln(os.Stderr, "FAIL:", errs[0])
		return fmt.Errorf("validate agents: %d error(s)", len(errs))
	}

	var (
		lineCount int
		caNumbers []int
	)
	scanner := bufio.NewScanner(f)
	// A single CA entry can be a few hundred chars (Rule + Why + How to apply);
	// default 64KiB tokens are too tight for the long ones we have seen.
	scanner.Buffer(make([]byte, 64*1024), 1024*1024)
	for scanner.Scan() {
		lineCount++
		if m := caHeadingRE.FindStringSubmatch(scanner.Text()); m != nil {
			n, perr := strconv.Atoi(m[1])
			if perr != nil {
				// Unreachable: regex guarantees \d+ matches.
				continue
			}
			caNumbers = append(caNumbers, n)
		}
	}
	if scanErr := scanner.Err(); scanErr != nil {
		errs = append(errs, fmt.Sprintf("scan %s: %v", path, scanErr))
	}
	_ = f.Close()

	// Check 1: line budget.
	if lineCount > agentsMaxLines {
		errs = append(errs, fmt.Sprintf(
			"%s: line count %d exceeds limit %d (CADL: AGENTS.md must stay lean)",
			path, lineCount, agentsMaxLines,
		))
	}

	// Check 2: CA entry count budget.
	if len(caNumbers) > agentsMaxCAEntries {
		entries := make([]string, 0, len(caNumbers))
		for _, n := range caNumbers {
			entries = append(entries, fmt.Sprintf("CA-%d", n))
		}
		errs = append(errs, fmt.Sprintf(
			"%s: %d CA entries exceeds limit %d (entries: %s). Prune weakest entries or move them to references/ca-archive.md.",
			path, len(caNumbers), agentsMaxCAEntries, strings.Join(entries, ", "),
		))
	}

	// Check 3: contiguous, gap-free, duplicate-free sequence.
	// Sorted copy preserves the original insertion order in error messages.
	sorted := append([]int(nil), caNumbers...)
	// Avoid mutating caller-visible slices; sort a working copy.
	sort.Ints(sorted)
	var dupNums []int
	seen := make(map[int]int, len(sorted)) // num -> first occurrence index
	for i, n := range sorted {
		if _, exists := seen[n]; exists {
			dupNums = append(dupNums, n)
			continue
		}
		seen[n] = i
	}
	// After dedup tracking, walk sorted unique numbers to detect gaps.
	unique := make([]int, 0, len(seen))
	for _, i := range sorted {
		if len(unique) == 0 || unique[len(unique)-1] != i {
			unique = append(unique, i)
		}
	}
	var gaps [][2]int
	for i := 1; i < len(unique); i++ {
		if unique[i] != unique[i-1]+1 {
			gaps = append(gaps, [2]int{unique[i-1], unique[i]})
		}
	}
	if len(dupNums) > 0 {
		errs = append(errs, fmt.Sprintf(
			"%s: duplicate CA numbers: %v",
			path, dupNums,
		))
	}
	if len(gaps) > 0 {
		parts := make([]string, 0, len(gaps))
		for _, g := range gaps {
			parts = append(parts, fmt.Sprintf("CA-%d..CA-%d", g[0], g[1]))
		}
		errs = append(errs, fmt.Sprintf(
			"%s: non-contiguous CA numbers (gaps: %s). Renumber to keep CA-N continuous.",
			path, strings.Join(parts, ", "),
		))
	}

	if len(errs) > 0 {
		for _, e := range errs {
			fmt.Fprintln(os.Stderr, "FAIL:", e)
		}
		return fmt.Errorf("validate agents: %d error(s)", len(errs))
	}
	fmt.Printf(
		"OK: AGENTS.md passed agents gate (lines=%d/%d, ca_entries=%d/%d)\n",
		lineCount, agentsMaxLines, len(caNumbers), agentsMaxCAEntries,
	)
	return nil
}
