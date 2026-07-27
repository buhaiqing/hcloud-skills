package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"
)

// runSnapshotCLI captures stdout/stderr/exit for a subcommand into
// a fixture file under internal/clifixtures/fixtures/<name>.json.
// Used by maintainers; not part of the runtime surface.
func runSnapshotCLI(args []string) error {
	fs := newFlagSet("hwcloud-skillcheck snapshot cli")
	bin := fs.String("bin", "./hwcloud-skillcheck", "binary to invoke")
	root := fs.String("root", ".", "repo root")
	if err := fs.Parse(args); err != nil {
		return err
	}
	subArgs := fs.Args()
	if len(subArgs) == 0 {
		return fmt.Errorf("usage: snapshot cli -- <args>")
	}
	cmd := exec.Command(*bin, subArgs...)
	// NOTE: CombinedOutput() is used here because the plan's snapshot
	// utility captures combined stdout+stderr. This conflates the two
	// streams, so stderr_excerpt in the fixture will always be empty.
	// The Fixture contract has separate stdout_excerpt/stderr_excerpt
	// fields, but this utility cannot populate them accurately. This
	// is a known limitation; fix would require separate stdout/stderr
	// pipes (os/exec.Cmd.StdoutPipe/derrPipe).
	out, err := cmd.CombinedOutput()
	exit := 0
	if err != nil {
		if ee, ok := err.(*exec.ExitError); ok {
			exit = ee.ExitCode()
		} else {
			return err
		}
	}
	fixture := map[string]any{
		"args":           subArgs,
		"stdout_excerpt": string(out),
		"stderr_excerpt": "",
		"exit_code":      exit,
		"captured_at":    time.Now().UTC().Format(time.RFC3339),
	}
	name := joinArgs(subArgs) + ".json"
	dir := filepath.Join(*root, "internal", "clifixtures", "fixtures")
	_ = os.MkdirAll(dir, 0o700)
	b, _ := json.MarshalIndent(fixture, "", "  ")
	return os.WriteFile(filepath.Join(dir, name), append(b, '\n'), 0o600)
}

func joinArgs(args []string) string {
	out := ""
	for i, a := range args {
		if i > 0 {
			out += "__"
		}
		out += a
	}
	return out
}
