// Command hwcloud-skillcheck is a cross-platform single-binary CLI for validating
// hcloud-skills repositories. It replaces the Python scripts under
// scripts/ with a zero-dependency (no interpreter required) Go binary.
//
// Usage:
//
//	hwcloud-skillcheck validate --root ./my-skills
//	hwcloud-skillcheck validate schema trace --file trace.json
//	hwcloud-skillcheck scan secret trace --self-check
package main

import (
	"fmt"
	"os"

	"github.com/buhaiqing/hcloud-skills/hwcloud-skillcheck/cmd"
)

func main() {
	if err := cmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}
