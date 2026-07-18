// Package cmd implements the skillcheck command-line interface using only
// the Go standard library (plus gopkg.in/yaml.v3 for YAML handling), keeping
// the binary free of extra dependencies per the project Spec.
package cmd

import (
	"flag"
	"fmt"
	"io"
	"os"
	"strings"
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
		return runCheck(args)
	case "scan":
		return runScan(args)
	case "aggregate":
		return runAggregate(args)
	case "lint":
		return runLint(args)
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
	fmt.Fprintln(w, "  skillcheck check example-config --root <dir>                validate assets/example-config.yaml")
	fmt.Fprintln(w, "  skillcheck check markdown-links --root <dir>                validate local Markdown links")
	fmt.Fprintln(w, "  skillcheck check references-links --root <dir>              validate references/ anchor health")
	fmt.Fprintln(w, "  skillcheck check advanced-coverage --root <dir>             validate TE-7 advanced/ coverage")
	fmt.Fprintln(w, "  skillcheck scan secret <trace|summary|alarm-plan> ...       scan artifacts for credential leaks")
	fmt.Fprintln(w, "  skillcheck aggregate trace --root <dir>                     aggregate gcl-trace-*.json into a summary")
	fmt.Fprintln(w, "  skillcheck lint go --root <dir> [--fix]                      gofmt + go vet the module")
	fmt.Fprintln(w, "  skillcheck validate --root <dir>                             run all A-class checks (total entry)")
}

func runValidate(args []string) error {
	if len(args) == 0 {
		// Total-entry: run every A-class check against the default root (cwd).
		return runValidateAll([]string{})
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
		// `validate --root <dir>` is shorthand for the total-entry.
		return runValidateAll(args)
	}
}

// runValidateAll is the `validate` total-entry (default when no subcommand is
// given). It runs the full A-class check suite against --root (default cwd),
// matching validate_local.py's A-class orchestration. A non-zero exit is
// returned if any check fails.
func runValidateAll(args []string) error {
	fs := newFlagSet("skillcheck validate")
	root := fs.String("root", ".", "skill repository root (default: current directory)")
	jsonOut := fs.Bool("json", false, "emit a combined JSON summary")
	if err := fs.Parse(args); err != nil {
		return err
	}

	steps := []struct {
		name string
		fn   func([]string) error
	}{
		{"validate frontmatter", func(a []string) error { return runValidateFrontmatter(a) }},
		{"validate eval-queries", func(a []string) error { return runValidateEvalQueries(a) }},
		{"validate product-assessment", func(a []string) error { return runValidateProductAssessment(a) }},
		{"check example-config", func(a []string) error { return runCheckExampleConfig(a) }},
		{"check markdown-links", func(a []string) error { return runCheckMarkdownLinks(a) }},
		{"check references-links", func(a []string) error { return runCheckReferencesLinks(a) }},
		{"check advanced-coverage", func(a []string) error { return runCheckAdvancedCoverage(a) }},
	}

	var failed []string
	for _, step := range steps {
		err := step.fn([]string{"--root", *root})
		if err != nil {
			failed = append(failed, step.name)
			fmt.Fprintf(os.Stderr, "FAIL  %s: %v\n", step.name, err)
		} else {
			fmt.Fprintf(os.Stdout, "PASS  %s\n", step.name)
		}
	}

	if *jsonOut {
		fmt.Printf("{\"ok\": %v, \"failed\": %d, \"steps\": %d}\n", len(failed) == 0, len(failed), len(steps))
	}
	if len(failed) > 0 {
		return fmt.Errorf("validate: %d/%d steps failed (%s)", len(failed), len(steps), strings.Join(failed, ", "))
	}
	fmt.Println("validate: all A-class checks passed")
	return nil
}

// newFlagSet builds a flag set that writes errors/usage to stderr and exits
// non-zero on parse failure — matching typical CLI ergonomics.
func newFlagSet(name string) *flag.FlagSet {
	fs := flag.NewFlagSet(name, flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	return fs
}
