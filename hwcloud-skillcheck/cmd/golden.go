package cmd

import (
	"fmt"
	"os"

	"github.com/buhaiqing/hcloud-skills/hwcloud-skillcheck/internal/golden"
)

func runGolden(args []string) error {
	fs := newFlagSet("hwcloud-skillcheck golden")
	cmd := fs.String("cmd", "run", "subcommand")
	root := fs.String("root", ".", "repo root")
	if err := fs.Parse(args); err != nil {
		return err
	}
	switch *cmd {
	case "run":
		r, err := golden.Run(*root)
		if err != nil {
			return err
		}
		failed := 0
		for skill, n := range r.PerSkill {
			if n < golden.MinScenariosPerSkill {
				fmt.Fprintf(os.Stderr, "BELOW THRESHOLD: %s has %d scenarios (need %d)\n", skill, n, golden.MinScenariosPerSkill)
				failed++
			}
		}
		if failed > 0 {
			return fmt.Errorf("%d skills below threshold", failed)
		}
		fmt.Printf("OK: %d skills, %d total scenarios\n", len(r.PerSkill), r.Passed)
		return nil
	default:
		return fmt.Errorf("unknown golden subcommand: %s", *cmd)
	}
}
