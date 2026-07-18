package cmd

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// runLint handles `skillcheck lint go [--root <dir>] [--fix]`.
//
// Phase 1 replaces scripts/run_ruff.sh with a Go-native lint entry point. It
// runs `gofmt -l` (formatting) and `go vet` over the skillcheck module. The
// `go` toolchain is required at lint time; external users who only run the
// validation commands need not have Go installed — lint is opt-in and never
// blocks the A-class checks (see Spec §2.4).
func runLint(args []string) error {
	fs := newFlagSet("skillcheck lint go")
	root := fs.String("root", ".", "Go module root (defaults to cwd)")
	fix := fs.Bool("fix", false, "rewrite files with gofmt -w")
	quiet := fs.Bool("quiet", false, "only report failures")
	if err := fs.Parse(args); err != nil {
		return err
	}
	rootDir, err := filepath.Abs(*root)
	if err != nil {
		return err
	}

	goBin, err := exec.LookPath("go")
	if err != nil {
		return fmt.Errorf("lint go: 'go' toolchain not found on PATH (required for linting)")
	}

	var failures []string

	// gofmt: list files needing formatting (or rewrite them with -fix). Use
	// the `gofmt` binary directly on the directory tree — `go fmt` only
	// accepts package arguments, not a raw directory path.
	gofmtBin, err := exec.LookPath("gofmt")
	if err != nil {
		return fmt.Errorf("lint go: 'gofmt' not found on PATH (required for linting)")
	}
	formatArgs := []string{"-l"}
	if *fix {
		formatArgs = []string{"-w"}
	}
	formatArgs = append(formatArgs, rootDir)
	cmd := exec.Command(gofmtBin, formatArgs...)
	var fmtOut bytes.Buffer
	cmd.Stdout = &fmtOut
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("lint go: gofmt failed: %w", err)
	}
	if !*fix {
		for _, line := range strings.Split(strings.TrimSpace(fmtOut.String()), "\n") {
			if line == "" {
				continue
			}
			failures = append(failures, "gofmt: "+line)
		}
	}

	// go vet: static analysis.
	vetCmd := exec.Command(goBin, "vet", "./...")
	vetCmd.Dir = rootDir
	var vetOut bytes.Buffer
	vetCmd.Stdout = &vetOut
	vetCmd.Stderr = &vetOut
	if err := vetCmd.Run(); err != nil {
		for _, line := range strings.Split(vetOut.String(), "\n") {
			if strings.TrimSpace(line) == "" {
				continue
			}
			failures = append(failures, "go vet: "+line)
		}
	}

	if !*quiet {
		fmt.Printf("lint go: gofmt + go vet on %s\n", rootDir)
	}
	if len(failures) > 0 {
		for _, f := range failures {
			fmt.Fprintln(os.Stderr, "  "+f)
		}
		return fmt.Errorf("lint go: %d issue(s) found", len(failures))
	}
	fmt.Println("lint go: clean")
	return nil
}
