// Command mockhcloud is a hermetic fake of the `hcloud` CLI used for E2E
// tests. It reads a script file (a list of {match, response, exit_code}
// records), dispatches each invocation to the longest-prefix match, and
// prints the scripted response to stdout. Every call is logged as JSON to
// stderr so tests can audit what was exercised.
//
// mockhcloud performs NO network I/O. It only reads the script file from
// disk and writes to stdout/stderr. The runtime dep graph is asserted in
// main_test.go:TestMockhcloudNoNetwork.
package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/buhaiqing/hcloud-skills/hwcloud-skillcheck/internal/scriptlang"
)

func main() {
	var scriptPath string
	args := os.Args[1:]
	for i := 0; i < len(args); i++ {
		if args[i] == "--script" && i+1 < len(args) {
			scriptPath = args[i+1]
			args = append(args[:i], args[i+2:]...)
			break
		}
	}
	if scriptPath == "" {
		fmt.Fprintln(os.Stderr, "usage: mockhcloud --script <path> <verb> [args...]")
		os.Exit(2)
	}
	data, err := os.ReadFile(scriptPath)
	if err != nil {
		fmt.Fprintln(os.Stderr, "read script:", err)
		os.Exit(2)
	}
	var spec scriptlang.Spec
	if err := json.Unmarshal(data, &spec); err != nil {
		fmt.Fprintln(os.Stderr, "parse script:", err)
		os.Exit(2)
	}
	cmd := strings.Join(args, " ")
	entry, ok := spec.Match(cmd)
	if !ok {
		fmt.Fprintln(os.Stderr, "no match:", cmd)
		os.Exit(1)
	}
	// Log the call to stderr (so it doesn't pollute stdout).
	log := map[string]any{"cmd": cmd, "matched": entry.Match, "exit": entry.ExitCode}
	lb, _ := json.Marshal(log)
	fmt.Fprintln(os.Stderr, string(lb))
	fmt.Print(entry.Response)
	os.Exit(entry.ExitCode)
}
