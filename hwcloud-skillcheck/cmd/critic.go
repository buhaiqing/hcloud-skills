package cmd

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/buhaiqing/hcloud-skills/hwcloud-skillcheck/internal/critic"
)

// runCritic dispatches `hwcloud-skillcheck critic score` (the rule-based 5-dimension
// scorer ported from scripts/critic_v1.py).
func runCritic(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("critic: missing subcommand (use 'score')")
	}
	switch args[0] {
	case "score":
		return runCriticScore(args[1:])
	case "-h", "--help", "help":
		fmt.Fprintln(os.Stdout, "hwcloud-skillcheck critic score --generator <path> [--emit --critic-out <path>]")
		return nil
	default:
		return fmt.Errorf("critic: unknown subcommand %q", args[0])
	}
}

func runCriticScore(args []string) error {
	fs := newFlagSet("hwcloud-skillcheck critic score")
	gen := fs.String("generator", "", "path to generator trace JSON (required)")
	emit := fs.Bool("emit", false, "write critic JSON to --critic-out in addition to stdout")
	out := fs.String("critic-out", "critic.json", "output path when --emit is set")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *gen == "" {
		return fmt.Errorf("--generator is required")
	}
	result, err := critic.ScoreFile(*gen)
	if err != nil {
		return err
	}
	encoded, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return err
	}
	if *emit {
		if wErr := os.WriteFile(*out, append(encoded, '\n'), 0o644); wErr != nil {
			return wErr
		}
	}
	os.Stdout.Write(append(encoded, '\n'))
	return nil
}
