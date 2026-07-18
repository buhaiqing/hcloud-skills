// Install, check, or uninstall the repository-managed git pre-commit hook.
//
// The hook source lives at .githooks/pre-commit (tracked by git) and is copied
// into .git/hooks/pre-commit by this program. The destination is owned by the
// user, so re-running is idempotent.
//
// Usage:
//   go run scripts/install_hook.go           # install
//   go run scripts/install_hook.go --check   # print status
//   go run scripts/install_hook.go --uninstall # remove
package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const hookName = "pre-commit"

func main() {
	check := flag.Bool("check", false, "print install status and exit")
	uninstall := flag.Bool("uninstall", false, "remove the installed hook")
	flag.Parse()

	root, err := findRepoRoot()
	if err != nil {
		fail("not a git worktree: %v", err)
	}

	if *check {
		os.Exit(checkHook(root))
	}
	if *uninstall {
		os.Exit(uninstallHook(root))
	}
	os.Exit(installHook(root))
}

func findRepoRoot() (string, error) {
	wd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	// Walk up looking for .git (directory or gitdir-file for worktrees).
	for {
		gitDir := filepath.Join(wd, ".git")
		info, err := os.Stat(gitDir)
		if err == nil && info.IsDir() {
			return wd, nil
		}
		// Worktree: .git is a file containing "gitdir: /path/to/repo/.git/worktrees/<name>"
		if err == nil && !info.IsDir() {
			data, rerr := os.ReadFile(gitDir)
			if rerr == nil && len(data) > 8 && strings.HasPrefix(string(data), "gitdir: ") {
				// realPath e.g. "/repo/.git/worktrees/name"
				// Dir(worktrees/name) = .git/worktrees  →  Dir(.git/worktrees) = .git  →  Dir(.git) = repo root
				realPath := strings.TrimSpace(string(data[8:]))
				return filepath.Dir(filepath.Dir(filepath.Dir(realPath))), nil
			}
		}
		parent := filepath.Dir(wd)
		if parent == wd {
			break
		}
		wd = parent
	}
	return "", fmt.Errorf(".git directory not found")
}

func installHook(root string) int {
	source := filepath.Join(root, ".githooks", hookName)
	dest := filepath.Join(root, ".git", "hooks", hookName)

	if _, err := os.Stat(source); os.IsNotExist(err) {
		fail("source hook not found: %s", source)
	} else if err != nil {
		fail("stat source: %v", err)
	}

	if err := os.MkdirAll(filepath.Dir(dest), 0755); err != nil {
		fail("create hooks dir: %v", err)
	}

	if err := copyFile(source, dest); err != nil {
		fail("copy: %v", err)
	}

	if err := os.Chmod(dest, 0755); err != nil {
		fail("chmod: %v", err)
	}

	fmt.Printf("installed: %s -> %s\n", source, dest)
	return 0
}

func uninstallHook(root string) int {
	dest := filepath.Join(root, ".git", "hooks", hookName)
	if _, err := os.Stat(dest); os.IsNotExist(err) {
		fmt.Printf("not installed: %s does not exist\n", dest)
		return 0
	}
	if err := os.Remove(dest); err != nil {
		fail("remove: %v", err)
	}
	fmt.Printf("removed: %s\n", dest)
	return 0
}

func checkHook(root string) int {
	dest := filepath.Join(root, ".git", "hooks", hookName)
	if _, err := os.Stat(dest); os.IsNotExist(err) {
		fmt.Printf("MISSING: %s not installed; run `go run scripts/install_hook.go`\n", dest)
		return 1
	}
	fmt.Printf("OK: %s is installed\n", dest)
	return 0
}

func copyFile(src, dst string) error {
	data, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	return os.WriteFile(dst, data, 0644)
}

func fail(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "ERROR: "+format+"\n", args...)
	os.Exit(1)
}
