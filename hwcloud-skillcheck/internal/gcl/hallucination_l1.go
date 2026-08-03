package gcl

import (
	"regexp"
	"strings"
)

// checkCLIParams validates that all --flags in the Generator command exist
// in the skill's references/cli-usage.md.
func (d *DefaultHallucinationDetector) checkCLIParams(command string) (*L1Result, error) {
	result := &L1Result{}

	if command == "" {
		result.Details = "no command to check"
		return result, nil
	}

	// Extract all --flags from the command.
	flags, err := extractFlagsFromCommand(command)
	if err != nil {
		result.Details = err.Error()
		return result, nil
	}
	result.FlagsChecked = len(flags)

	if len(flags) == 0 {
		result.Details = "no flags to check"
		return result, nil
	}

	// Load the skill's known flags.
	knownFlags, err := d.loadCLIUsage()
	if err != nil {
		result.Details = "could not load cli-usage.md: " + err.Error()
		return result, nil
	}

	// Check each flag.
	var invalid []string
	for _, flag := range flags {
		// Strip leading -- and split on = to get the flag name.
		name := strings.TrimPrefix(flag, "--")
		name = strings.Split(name, "=")[0]

		if !knownFlags[name] {
			invalid = append(invalid, flag)
		}
	}

	result.InvalidFlags = invalid
	if len(invalid) > 0 {
		result.Blocked = true
		result.Details = "invalid flags: " + strings.Join(invalid, ", ")
	} else {
		result.Details = "all flags valid"
	}

	return result, nil
}

// extractFlagsFromCommand returns all --flag or --flag=value tokens from a command string.
func extractFlagsFromCommand(command string) ([]string, error) {
	var flags []string
	// Match --word or --word=value patterns.
	re := regexp.MustCompile(`--[a-zA-Z][-a-zA-Z0-9_]*(?:=[^\s]+)?`)
	matches := re.FindAllString(command, -1)
	return append(flags, matches...), nil
}
