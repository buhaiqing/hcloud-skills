package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// caHeadingMultilineRE is caHeadingRE applied to a whole-file buffer. validate
// agents scans line by line, so its bare `^` already means "line start"; here
// the pattern must be told explicitly that every line start counts, otherwise a
// bare `### CA-N.` heading below the first line is invisible.
var caHeadingMultilineRE = regexp.MustCompile(`(?m)` + caHeadingRE.String())

// docContract is one group of literal-anchor invariants over a reference doc:
// every anchor must appear verbatim, byte for byte, in the named file.
//
// Literal anchors instead of semantic regex are deliberate. On 2026-09-24 an
// external process restored references/gcl-runtime.md and
// references/skill-update-rule.md to HEAD while go test, ruff, validate and
// validate agents were all green — doc invariants had no executable gate, so a
// silent rollback was invisible. A semantic regex would let a reworded-but-
// equivalent threshold table pass again; the cost of full-line anchors is that
// reformatting the doc requires an explicit anchor update, which is the point.
// docContractsMarker identifies the repository that owns these contracts.
// The pinned docs are hcloud-skills internals, so a foreign skill repo running
// `validate --root .` is skipped instead of failed. The marker is deliberately
// NOT one of the four pinned docs: deleting all four must still fail loudly,
// otherwise "remove the contracts" would become the cheapest bypass.
const docContractsMarker = "docs/gcl-spec.md"

type docContract struct {
	file    string
	anchors []string
	// bareCAForbidden fails when any `### CA-N.` heading is present, reusing
	// caHeadingRE from validate_agents.go. Archived entries must keep the
	// CA-A prefix so they can never collide with the active CA sequence.
	bareCAForbidden bool
}

var docContracts = []docContract{
	{
		// GCL thresholds are frozen: values may be re-justified or replaced,
		// but only together with the gate anchor in the same commit, per
		// references/gcl-runtime.md §Threshold Calibration.
		file: "references/gcl-runtime.md",
		anchors: []string{
			"## Threshold Calibration",
			"| Correctness (GCL pass bar) | ≥ 0.5 |",
			"| Safety | = 1.0 |",
			"| Idempotency | ≥ 0.5 |",
			"| Traceability | ≥ 0.5 |",
			"| Spec Compliance | ≥ 0.5 |",
			"| confidence (low / mid / high) | 0.70 / 0.85 / 0.95 |",
			"| auto_execute (low / mid / high) | 0.70 / 0.85 / 0.95 |",
		},
	},
	{
		// Round 3 closes the reflection loop; the command literals are the
		// executable half of that contract.
		file: "references/skill-update-rule.md",
		anchors: []string{
			"## Round 3 — Rubric Auto-Proposal (Closed Loop)",
			"reflection-findings.jsonl",
			"propose --file",
			"rubric_item == null",
		},
	},
	{
		// Anchors keep the trailing period so `### CA-A11 ` or `### CA-A11foo`
		// cannot satisfy them by prefix alone.
		file: "references/ca-archive.md",
		anchors: []string{
			"### CA-A11.",
			"### CA-A12.",
			"### CA-A13.",
			"### CA-A14.",
		},
		bareCAForbidden: true,
	},
	{
		file: "references/self-healing-spec.md",
		anchors: []string{
			"### 5.1 触发条件",
			"### 5.2 学习流程",
			"### 5.3 Loop Health Observability",
			"### 5.4 GCL Runner 集成",
			"source_traces_analyzed",
		},
	},
}

// runValidateDocContracts handles:
//
//	hwcloud-skillcheck validate doc-contracts --root <dir>
//
// Every reference doc in docContracts is mandatory once the root carries
// docContractsMarker; a missing file is then a hard error, matching validate
// agents' treatment of AGENTS.md. Roots without the marker are skipped (exit 0)
// because the pinned contracts are hcloud-skills internals. Exit 1 = any anchor
// missing, a bare archived CA heading found, or a pinned doc missing.
func runValidateDocContracts(args []string) error {
	fs := newFlagSet("validate doc-contracts")
	root := fs.String("root", ".", "skill repository root (default: current directory)")
	if err := fs.Parse(args); err != nil {
		return err
	}
	rootDir, err := filepath.Abs(*root)
	if err != nil {
		return err
	}

	if _, statErr := os.Stat(filepath.Join(rootDir, filepath.FromSlash(docContractsMarker))); statErr != nil {
		fmt.Printf("SKIP: doc-contracts not applicable (no %s under %s)\n", docContractsMarker, rootDir)
		fmt.Println("WARN: if this repo owns the reference contracts, the marker is missing — " +
			"treat that as a gate removal, not a skip")
		return nil
	}

	anchorCount := 0
	var errs []string
	for _, contract := range docContracts {
		body, readErr := os.ReadFile(filepath.Join(rootDir, contract.file))
		if readErr != nil {
			errs = append(errs, fmt.Sprintf("open %s: %v", contract.file, readErr))
			continue
		}
		text := string(body)
		for _, anchor := range contract.anchors {
			anchorCount++
			if !strings.Contains(text, anchor) {
				errs = append(errs, fmt.Sprintf("%s: missing anchor %q", contract.file, anchor))
			}
		}
		if contract.bareCAForbidden {
			// \s also matches \n here, so a heading with an empty title is
			// caught too — a deliberate superset of validate agents' per-line
			// scan, and the safe direction for a collision check.
			if bare := caHeadingMultilineRE.FindAllString(text, -1); len(bare) > 0 {
				errs = append(errs, fmt.Sprintf(
					"%s: bare %q heading collides with AGENTS.md active numbering: %s",
					contract.file, "### CA-N.", strings.Join(bare, ", ")))
			}
		}
	}

	if len(errs) > 0 {
		for _, e := range errs {
			fmt.Fprintln(os.Stderr, "FAIL:", e)
		}
		return fmt.Errorf("validate doc-contracts: %d error(s)", len(errs))
	}
	fmt.Printf("OK: reference docs passed doc-contracts gate (%d files, %d anchors)\n",
		len(docContracts), anchorCount)
	return nil
}
