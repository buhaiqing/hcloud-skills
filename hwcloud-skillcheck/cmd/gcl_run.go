package cmd

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/buhaiqing/hcloud-skills/hwcloud-skillcheck/internal/gcl"
	"github.com/buhaiqing/hcloud-skills/hwcloud-skillcheck/internal/yaml"
)

// runGCL dispatches `hwcloud-skillcheck gcl` subcommands.
func runGCL(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("gcl: missing subcommand (use 'run', 'alarm-wire', or 'criticeval')")
	}
	switch args[0] {
	case "run":
		return runGCLRun(args[1:])
	case "alarm-wire":
		return runGCLAlarmWire(args[1:])
	case "criticeval":
		return runGCLCriticEval(args[1:])
	case "-h", "--help", "help":
		fmt.Fprintln(os.Stdout, "hwcloud-skillcheck gcl run --root <dir> [--json] [--quiet] [--command <shell>] [--request <text>] [--critic-cmd <executable>]")
		fmt.Fprintln(os.Stdout, "hwcloud-skillcheck gcl alarm-wire --root <dir> [--json] [--quiet] [--plan-file <path>] [--apply] [--yes] [--apply-target-region <id>]")
		fmt.Fprintln(os.Stdout, "hwcloud-skillcheck gcl criticeval                       # reads GeneratorOutput JSON on stdin, prints CriticResult JSON on stdout")
		return nil
	default:
		return fmt.Errorf("gcl: unknown subcommand %q", args[0])
	}
}

// runGCLRun implements `hwcloud-skillcheck gcl run`.
// It runs the GCL structural critic loop against a skill directory.
func runGCLRun(args []string) error {
	fs := newFlagSet("hwcloud-skillcheck gcl run")
	root := fs.String("root", ".", "skill directory (e.g., huaweicloud-ecs-ops/)")
	jsonOut := fs.Bool("json", false, "emit JSON report")
	quiet := fs.Bool("quiet", false, "suppress stdout except final result")
	model := fs.String("model", "", "LLM model name for the Generator (e.g. 'anthropic/claude-3-5-sonnet'). Stored in trace. If empty, 'unknown' is recorded.")
	command := fs.String("command", "", "shell command for the Generator to run (e.g. 'hcloud ecs list-servers --region cn-north-4'). When empty, a smoke 'echo ok' is run so the structural critic path can still be exercised.")
	request := fs.String("request", "smoke test", "natural-language request the Generator is responding to; recorded in trace.iterations[*].request.")
	criticCmd := fs.String("critic-cmd", "", "path to an external Critic executable. The Runner pipes GeneratorOutput JSON to its stdin and reads CriticResult JSON from stdout. When empty, the in-process Structural critic is used. Pass repeated --critic-arg to forward arguments.")
	confirmNonce := fs.String("confirm-nonce", "", "P0 trust boundary: confirmation nonce issued by a ConfirmationRegistry. Required when the command declares a destructive safety_class. Mutually exclusive with --confirm-issue.")
	confirmIssue := fs.Bool("confirm-issue", false, "P0 trust boundary: instead of consuming a nonce, issue a fresh one and print it (then exit). Used by human review flows to get the nonce they will paste back in.")
	var criticArgs []string
	fs.Var(&criticArgsValue{slice: &criticArgs}, "critic-arg", "argument forwarded to --critic-cmd (repeatable).")
	_ = criticArgs
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil // help was shown; exit cleanly
		}
		return err
	}

	skillDir, err := filepath.Abs(*root)
	if err != nil {
		return err
	}

	// Load SKILL.md to get skill name.
	skillPath := filepath.Join(skillDir, "SKILL.md")
	skillData, err := os.ReadFile(skillPath)
	if err != nil {
		return fmt.Errorf("read SKILL.md: %w", err)
	}
	skillName := filepath.Base(skillDir) // default to directory name
	fm, err := yaml.ExtractFrontmatter(skillData)
	if err == nil && fm != nil {
		if name, ok := fm["name"].(string); ok && name != "" {
			skillName = name
		}
	}

	// When --command is empty, fall back to the historical smoke default
	// ('echo ok') so callers running `gcl run` purely to validate the
	// structural critic loop continue to work unchanged.
	resolvedCommand := *command
	if resolvedCommand == "" {
		resolvedCommand = "echo ok"
	}

	cfg := gcl.RunConfig{
		Skill:   skillName,
		Request: *request,
		Command: resolvedCommand,
		Root:    skillDir,
		Model:   *model,
	}
	if *confirmNonce != "" {
		cfg.ConfirmationToken = *confirmNonce
		cfg.ConfirmationRegistry = gcl.NewConfirmationRegistry(gcl.DefaultConfirmationTTL)
		defer cfg.ConfirmationRegistry.Stop()
	} else if *confirmIssue {
		// Issue-only path: print a nonce, exit 0. No Run() invocation.
		reg := gcl.NewConfirmationRegistry(gcl.DefaultConfirmationTTL)
		defer reg.Stop()
		nonce, err := reg.Issue("gcl:"+skillName, "cli-user")
		if err != nil {
			return fmt.Errorf("issue nonce: %w", err)
		}
		fmt.Println(nonce)
		return nil
	}
	if *criticCmd != "" {
		cfg.Critic = gcl.NewExternalCritic(*criticCmd, criticArgs...)
	}

	// Suppress gcl.Run's output when --json or --quiet.

	// Suppress gcl.Run's output when --json or --quiet.
	if *jsonOut || *quiet {
		cfg.Stdout = io.Discard
		cfg.Stderr = io.Discard
	}
	result := gcl.Run(cfg)

	if *quiet {
		// Only print trace path or final status.
		if result.TracePath != "" {
			fmt.Println(result.TracePath)
		}
	} else if *jsonOut {
		printGCLRunJSON(skillName, result)
	} else {
		printGCLRunHuman(skillName, result)
	}

	// Map GCL exit codes to CLI exit codes:
	// gcl.ExitOK(0) -> 0, gcl.ExitMaxIter(1) -> 1, gcl.ExitUsage(2) -> 1,
	// gcl.ExitSafety(3) -> 2, gcl.ExitTimeout(124) -> 1.
	switch result.ExitCode {
	case gcl.ExitOK:
		return nil
	case gcl.ExitSafety:
		os.Exit(2) // SAFETY_VIOLATION
	default:
		os.Exit(1)
	}
	return nil // unreachable
}

func printGCLRunHuman(skillName string, result gcl.RunResult) {
	switch result.ExitCode {
	case gcl.ExitOK:
		fmt.Printf("PASS  %s — trace: %s\n", skillName, result.TracePath)
	case gcl.ExitSafety:
		fmt.Printf("SAFETY_VIOLATION  %s — trace: %s\n", skillName, result.TracePath)
	case gcl.ExitTimeout:
		fmt.Printf("TIMEOUT  %s — trace: %s\n", skillName, result.TracePath)
	case gcl.ExitMaxIter:
		fmt.Printf("MAX_ITER  %s — trace: %s\n", skillName, result.TracePath)
	default:
		fmt.Printf("ERROR  %s (exit %d) — trace: %s\n", skillName, result.ExitCode, result.TracePath)
	}
}

func printGCLRunJSON(skillName string, result gcl.RunResult) {
	var status string
	switch result.ExitCode {
	case gcl.ExitOK:
		status = "PASS"
	case gcl.ExitSafety:
		status = "SAFETY_VIOLATION"
	case gcl.ExitTimeout:
		status = "TIMEOUT"
	case gcl.ExitMaxIter:
		status = "MAX_ITER"
	default:
		status = "ERROR"
	}
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	enc.Encode(map[string]any{
		"skill":     skillName,
		"status":    status,
		"exit_code": result.ExitCode,
		"trace":     result.TracePath,
	})
}

// runGCLCriticEval implements `hwcloud-skillcheck gcl criticeval`.
//
// Reads a GeneratorOutput JSON document from stdin, runs the
// in-process Structural Critic on it, and prints the corresponding
// CriticResult JSON on stdout. This subcommand exists so callers can
// prove their --critic-cmd pipe works end-to-end by pointing it at
// `hwcloud-skillcheck gcl criticeval` — the binary itself plays the
// role of the out-of-process Critic.
func runGCLCriticEval(args []string) error {
	fs := newFlagSet("hwcloud-skillcheck gcl criticeval")
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}
	data, err := io.ReadAll(os.Stdin)
	if err != nil {
		return fmt.Errorf("read stdin: %w", err)
	}
	var gen gcl.GeneratorOutput
	if err := json.Unmarshal(data, &gen); err != nil {
		return fmt.Errorf("decode GeneratorOutput: %w", err)
	}
	res := gcl.StructuralCritic(gen)
	out, err := json.MarshalIndent(res, "", "  ")
	if err != nil {
		return err
	}
	_, err = os.Stdout.Write(append(out, '\n'))
	return err
}

// criticArgsValue is the flag.Value adapter for a repeatable --critic-arg.
// Each occurrence appends one string to the slice. Use with
// flag.Var(&criticArgsValue{slice: &dst}, "critic-arg", ...).
type criticArgsValue struct {
	slice *[]string
}

func (c *criticArgsValue) String() string {
	if c.slice == nil {
		return ""
	}
	return fmt.Sprintf("%v", *c.slice)
}

func (c *criticArgsValue) Set(v string) error {
	*c.slice = append(*c.slice, v)
	return nil
}
