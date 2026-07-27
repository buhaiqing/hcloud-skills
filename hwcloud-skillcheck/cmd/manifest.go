package cmd

import (
	"flag"
	"fmt"

	"github.com/buhaiqing/hcloud-skills/hwcloud-skillcheck/internal/manifest"
)

func runManifest(args []string) error {
	fs := flag.NewFlagSet("manifest", flag.ContinueOnError)
	fs.Usage = func() {
		fmt.Fprintln(flag.CommandLine.Output(), "hwcloud-skillcheck manifest gen --root <dir> --out <dir>")
	}
	cmd := fs.String("cmd", "gen", "subcommand (gen)")
	root := fs.String("root", ".", "skill repository root")
	out := fs.String("out", "audit-results/sandbox/manifests", "output directory")
	if err := fs.Parse(args); err != nil {
		return err
	}
	switch *cmd {
	case "gen":
		return manifest.Generate(*root, *out)
	default:
		return fmt.Errorf("unknown manifest command %q", *cmd)
	}
}
