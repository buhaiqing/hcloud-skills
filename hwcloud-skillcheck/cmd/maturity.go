package cmd

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/buhaiqing/hcloud-skills/hwcloud-skillcheck/internal/maturity"
)

func runMaturity(args []string) error {
	fs := flag.NewFlagSet("maturity", flag.ContinueOnError)
	fs.Usage = func() {
		fmt.Fprintln(os.Stderr, "Usage: hwcloud-skillcheck maturity report --root <dir>")
	}
	cmd := fs.String("cmd", "report", "sub-command (report)")
	root := fs.String("root", ".", "root of skill repository")
	if err := fs.Parse(args); err != nil {
		return err
	}
	switch *cmd {
	case "report":
		manifestRoot := filepath.Join(*root, "audit-results", "sandbox", "manifests")
		r, err := maturity.Rollup(manifestRoot)
		if err != nil {
			return fmt.Errorf("rollup: %w", err)
		}
		fmt.Println("Maturity Report")
		fmt.Println("===============")
		for skill, score := range r.PerSkill {
			fmt.Printf("%-30s %.2f\n", skill, score)
		}
		return nil
	default:
		return fmt.Errorf("unknown maturity sub-command %q", *cmd)
	}
}
