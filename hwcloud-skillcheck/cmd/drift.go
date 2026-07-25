package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/buhaiqing/hcloud-skills/hwcloud-skillcheck/internal/drift"
)

// runDrift dispatches `hwcloud-skillcheck drift check|sync`.
func runDrift(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("drift: missing subcommand (use 'check' or 'sync')")
	}
	switch args[0] {
	case "check":
		return runDriftCheck(args[1:])
	case "sync":
		return runDriftSync(args[1:])
	case "-h", "--help", "help":
		fmt.Fprintln(os.Stdout, "hwcloud-skillcheck drift check --root <dir> [--json]")
		fmt.Fprintln(os.Stdout, "hwcloud-skillcheck drift sync  --root <dir> [--apply] [--dry-run]")
		return nil
	default:
		return fmt.Errorf("drift: unknown subcommand %q", args[0])
	}
}

func runDriftCheck(args []string) error {
	fs := newFlagSet("hwcloud-skillcheck drift check")
	root := fs.String("root", ".", "skill repository root")
	jsonOut := fs.Bool("json", false, "emit JSON report")
	if err := fs.Parse(args); err != nil {
		return err
	}
	rootDir, err := filepath.Abs(*root)
	if err != nil {
		return err
	}
	report, err := drift.Check(rootDir)
	if err != nil {
		return err
	}
	if *jsonOut {
		out, jErr := json.MarshalIndent(report, "", "  ")
		if jErr != nil {
			return jErr
		}
		os.Stdout.Write(append(out, '\n'))
		return nil
	}
	if report.OK {
		fmt.Println("[skill_generator drift] OK: runtime copy matches canonical")
		return nil
	}
	for _, e := range report.Errors {
		fmt.Fprintf(os.Stderr, "FAIL: %s\n", e)
	}
	fmt.Fprintf(os.Stderr,
		"\n[skill_generator drift] FAIL: %d issue(s); "+
			"run `hwcloud-skillcheck drift sync --apply`\n",
		len(report.Errors))
	return fmt.Errorf("drift check: %d issue(s)", len(report.Errors))
}

func runDriftSync(args []string) error {
	fs := newFlagSet("hwcloud-skillcheck drift sync")
	root := fs.String("root", ".", "skill repository root")
	dryRun := fs.Bool("dry-run", true, "default: dry-run (no writes); pass --apply to write")
	apply := fs.Bool("apply", false, "apply changes (overrides --dry-run)")
	if err := fs.Parse(args); err != nil {
		return err
	}
	rootDir, err := filepath.Abs(*root)
	if err != nil {
		return err
	}
	dry := !*apply && *dryRun
	report, err := drift.Sync(rootDir, dry)
	if err != nil {
		return err
	}
	for _, a := range report.Actions {
		if dry {
			fmt.Println("DRY-RUN: " + a)
		} else {
			fmt.Println(a)
		}
	}
	if len(report.Actions) == 0 {
		fmt.Println("no drift; nothing to do")
	} else if !dry {
		fmt.Println("synced")
	}
	if !report.OK {
		return fmt.Errorf("drift sync: %d issue(s)", len(report.Errors))
	}
	return nil
}
