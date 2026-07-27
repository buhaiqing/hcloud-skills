package cmd

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/buhaiqing/hcloud-skills/hwcloud-skillcheck/internal/ab"
)

func runAB(args []string) error {
	fs := newFlagSet("hwcloud-skillcheck ab")
	cmd := fs.String("cmd", "compare", "subcommand")
	root := fs.String("root", ".", "repo root")
	oldRef := fs.String("old", "HEAD~1", "git ref for old run")
	if err := fs.Parse(args); err != nil {
		return err
	}

	switch *cmd {
	case "compare":
		return runABCompare(*root, *oldRef)
	default:
		return fmt.Errorf("unknown ab subcommand: %s", *cmd)
	}
}

func runABCompare(root, oldRef string) error {
	oldPath := root + "/.ab/old.json"
	curPath := root + "/.ab/cur.json"
	allowPath := root + "/.ab/allowlist.json"

	oldData, err := os.ReadFile(oldPath)
	if err != nil {
		return fmt.Errorf("read old result: %w", err)
	}
	var oldResult ab.Result
	if err := json.Unmarshal(oldData, &oldResult); err != nil {
		return fmt.Errorf("decode old result: %w", err)
	}

	curData, err := os.ReadFile(curPath)
	if err != nil {
		return fmt.Errorf("read cur result: %w", err)
	}
	var curResult ab.Result
	if err := json.Unmarshal(curData, &curResult); err != nil {
		return fmt.Errorf("decode cur result: %w", err)
	}

	var allow map[string]bool
	allowData, err := os.ReadFile(allowPath)
	if err == nil {
		if err := json.Unmarshal(allowData, &allow); err != nil {
			return fmt.Errorf("decode allowlist: %w", err)
		}
	}

	diff := ab.CompareWith(oldResult, curResult, allow)
	if len(diff.Drift) > 0 {
		fmt.Fprintf(os.Stderr, "DRIFT detected (%d scenario(s)):\n", len(diff.Drift))
		for scenario, entry := range diff.Drift {
			fmt.Fprintf(os.Stderr, "  Scenario: %s\n", scenario)
			fmt.Fprintf(os.Stderr, "    OLD: %q\n", entry.Old)
			fmt.Fprintf(os.Stderr, "    NEW: %q\n", entry.New)
		}
		return fmt.Errorf("non-allowlisted drift detected")
	}
	fmt.Println("OK: no drift")
	return nil
}
