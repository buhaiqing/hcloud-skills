package cmd

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/buhaiqing/hcloud-skills/hwcloud-skillcheck/internal/confusion"
)

func runTelemetry(args []string) error {
	if len(args) == 0 || args[0] == "-h" || args[0] == "--help" {
		return fmt.Errorf("telemetry: missing subcommand (use 'confusion')")
	}
	if args[0] != "confusion" {
		return fmt.Errorf("telemetry: unknown subcommand %q", args[0])
	}
	fs := newFlagSet("telemetry confusion")
	root := fs.String("root", ".", "skill repository root")
	if err := fs.Parse(args[1:]); err != nil {
		return err
	}
	matrix, err := confusion.Derive(*root)
	if err != nil {
		return err
	}
	return json.NewEncoder(os.Stdout).Encode(matrix)
}
