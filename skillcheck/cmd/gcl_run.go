package cmd

import (
	"bytes"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"

	"github.com/buhaiqing/hcloud-skills/skillcheck/internal/gcl"
	"github.com/buhaiqing/hcloud-skills/skillcheck/internal/yaml"
)

// runGCL dispatches `skillcheck gcl` subcommands.
func runGCL(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("gcl: missing subcommand (use 'run' or 'alarm-wire')")
	}
	switch args[0] {
	case "run":
		return runGCLRun(args[1:])
	case "alarm-wire":
		return runGCLAlarmWire(args[1:])
	case "-h", "--help", "help":
		fmt.Fprintln(os.Stdout, "skillcheck gcl run --root <dir> [--json] [--quiet]")
		fmt.Fprintln(os.Stdout, "skillcheck gcl alarm-wire --root <dir> [--json] [--quiet] [--plan-file <path>]")
		return nil
	default:
		return fmt.Errorf("gcl: unknown subcommand %q", args[0])
	}
}

// runGCLRun implements `skillcheck gcl run`.
// It runs the GCL structural critic loop against a skill directory.
func runGCLRun(args []string) error {
	fs := newFlagSet("skillcheck gcl run")
	root := fs.String("root", ".", "skill directory (e.g., huaweicloud-ecs-ops/)")
	jsonOut := fs.Bool("json", false, "emit JSON report")
	quiet := fs.Bool("quiet", false, "suppress stdout except final result")
	model := fs.String("model", "", "LLM model name for the Generator (e.g. 'anthropic/claude-3-5-sonnet'). Stored in trace. If empty, 'unknown' is recorded.")
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

	// Run GCL with a smoke command (echo ok) to test the structural critic path.
	// Root is set to skillDir so audit-results/ is created there.
	cfg := gcl.RunConfig{
		Skill:   skillName,
		Request: "smoke test",
		Command: "echo ok",
		Root:    skillDir,
		Model:   *model,
	}

	// Suppress gcl.Run's printf output when --json or --quiet.
	var result gcl.RunResult
	if *jsonOut || *quiet {
		// Redirect stdout/stderr to suppress gcl.Run's messages.
		// Use goroutines with WaitGroup to avoid pipe deadlock when
		// gcl.Run output exceeds the 64KB pipe buffer.
		var wg sync.WaitGroup
		var stdoutBuf, stderrBuf bytes.Buffer
		oldStdout := os.Stdout
		oldStderr := os.Stderr
		rStdout, wStdout, _ := os.Pipe()
		rStderr, wStderr, _ := os.Pipe()
		os.Stdout = wStdout
		os.Stderr = wStderr
		wg.Add(2)
		go func() {
			defer wg.Done()
			io.Copy(&stdoutBuf, rStdout)
			rStdout.Close()
		}()
		go func() {
			defer wg.Done()
			io.Copy(&stderrBuf, rStderr)
			rStderr.Close()
		}()
		result = gcl.Run(cfg)
		os.Stdout = oldStdout
		os.Stderr = oldStderr
		wStdout.Close()
		wStderr.Close()
		wg.Wait()                                     // drain both pipes before continuing
		_, _ = stdoutBuf.String(), stderrBuf.String() // captured but not used
	} else {
		result = gcl.Run(cfg)
	}

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
