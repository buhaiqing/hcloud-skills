// Command skillcheck is a cross-platform single-binary CLI for validating
// hcloud-skills repositories. It replaces the Python scripts under
// scripts/ with a zero-dependency (no interpreter required) Go binary.
//
// Usage:
//
//	skillcheck validate --root ./my-skills
//	skillcheck validate schema trace --file trace.json
//	skillcheck scan secret trace --self-check
package main

import (
	"fmt"
	"os"

	"github.com/buhaiqing/hcloud-skills/skillcheck/cmd"
)

func main() {
	if err := cmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}
