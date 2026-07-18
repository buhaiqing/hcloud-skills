// Package cmd implements the skillcheck command-line interface using only
// the Go standard library (plus gopkg.in/yaml.v3 for YAML handling), keeping
// the binary free of extra dependencies per the project Spec.
package cmd

import (
	"flag"
	"fmt"
	"io"
	"os"
)

// osStdin is overridable in tests to inject instance data.
var osStdin io.Reader = os.Stdin

// Execute is the CLI entry point. It dispatches the first argument to a
// subcommand and returns a non-nil error for usage/validation failures.
func Execute() error {
	if len(os.Args) < 2 {
		printRootHelp(os.Stderr)
		return fmt.Errorf("missing subcommand")
	}
	sub := os.Args[1]
	args := os.Args[2:]
	switch sub {
	case "validate":
		return runValidate(args)
	case "check":
		return fmt.Errorf("check: not yet implemented (see Plan batch B3)")
	case "scan":
		return fmt.Errorf("scan: not yet implemented (see Plan batch B3)")
	case "aggregate":
		return fmt.Errorf("aggregate: not yet implemented (see Plan batch B3)")
	case "lint":
		return fmt.Errorf("lint: not yet implemented (see Plan batch B4)")
	case "-h", "--help", "help":
		printRootHelp(os.Stdout)
		return nil
	default:
		printRootHelp(os.Stderr)
		return fmt.Errorf("unknown subcommand %q", sub)
	}
}

func printRootHelp(w io.Writer) {
	fmt.Fprintln(w, "skillcheck — cross-platform hcloud-skills validator")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Usage:")
	fmt.Fprintln(w, "  skillcheck validate --root <dir>                             validate a skill repository")
	fmt.Fprintln(w, "  skillcheck validate schema <kind> --file <path>             validate a JSON instance against an embedded schema")
	fmt.Fprintln(w, "  skillcheck validate frontmatter --root <dir>                validate SKILL.md frontmatter")
	fmt.Fprintln(w, "  skillcheck validate eval-queries --root <dir>               validate assets/eval_queries.json")
	fmt.Fprintln(w, "  skillcheck validate product-assessment --root <dir>         validate well-architected-assessment.md examples")
	fmt.Fprintln(w, "  skillcheck check ...    (planned, batch B3)")
	fmt.Fprintln(w, "  skillcheck scan ...     (planned, batch B3)")
	fmt.Fprintln(w, "  skillcheck aggregate ... (planned, batch B3)")
	fmt.Fprintln(w, "  skillcheck lint ...     (planned, batch B4)")
}

func runValidate(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("validate: missing subcommand (use 'schema')")
	}
	switch args[0] {
	case "schema":
		return runValidateSchema(args[1:])
	case "frontmatter":
		return runValidateFrontmatter(args[1:])
	case "eval-queries":
		return runValidateEvalQueries(args[1:])
	case "product-assessment":
		return runValidateProductAssessment(args[1:])
	case "-h", "--help", "help":
		printRootHelp(os.Stdout)
		return nil
	default:
		return fmt.Errorf("validate: unknown subcommand %q", args[0])
	}
}

// newFlagSet builds a flag set that writes errors/usage to stderr and exits
// non-zero on parse failure — matching typical CLI ergonomics.
func newFlagSet(name string) *flag.FlagSet {
	fs := flag.NewFlagSet(name, flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	return fs
}
