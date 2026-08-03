// Package gcl provides the Generator-Critic-Loop runtime components.
package gcl

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// HallucinationResult is the combined output of L1+L2+L3 hallucination checks.
type HallucinationResult struct {
	Blocked bool      `json:"blocked"`
	L1      *L1Result `json:"l1,omitempty"`
	L2      *L2Result `json:"l2,omitempty"`
	L3      *L3Result `json:"l3,omitempty"`
	Summary string    `json:"summary"`
}

// L1Result — CLI parameter existence check.
type L1Result struct {
	Blocked      bool     `json:"blocked"`
	FlagsChecked int      `json:"flags_checked"`
	InvalidFlags []string `json:"invalid_flags,omitempty"`
	Details      string   `json:"details,omitempty"`
}

// L2Result — JSON structure compliance check.
type L2Result struct {
	Blocked bool    `json:"blocked"`
	Errors  []L2Err `json:"errors,omitempty"`
	Details string  `json:"details,omitempty"`
}

type L2Err struct {
	Path     string `json:"path"`
	Message  string `json:"message"`
	Expected string `json:"expected,omitempty"`
	Actual   string `json:"actual,omitempty"`
}

// L3Result — WAF compliance check.
type L3Result struct {
	Blocked    bool          `json:"blocked"`
	Violations []L3Violation `json:"violations,omitempty"`
	Details    string        `json:"details,omitempty"`
}

type L3Violation struct {
	Dimension string `json:"dimension"` // security | cost | stability
	Rule      string `json:"rule"`
	Detail    string `json:"detail"`
}

// BlockedBySafety returns true if the result should trigger SAFETY_FAIL.
func (h *HallucinationResult) BlockedBySafety() bool {
	if h.L1 != nil && h.L1.Blocked {
		return true
	}
	if h.L2 != nil && h.L2.Blocked {
		return true
	}
	// L3 violations do not auto-block; they are surfaced to the Critic.
	return false
}

// ---- Interface -----------------------------------------------------------

// HallucinationDetector orchestrates L1+L2+L3 checks.
type HallucinationDetector interface {
	Run(ctx context.Context, gen GeneratorOutput, trace *GCLTrace) (*HallucinationResult, error)
}

// DefaultHallucinationDetector is the production implementation.
type DefaultHallucinationDetector struct {
	SkillRoot string // path to the skill directory (e.g. /repo/huaweicloud-ecs-ops)
}

// NewHallucinationDetector returns a detector for the given skill root.
func NewHallucinationDetector(skillRoot string) HallucinationDetector {
	return &DefaultHallucinationDetector{SkillRoot: skillRoot}
}

// Run executes all three layers and returns the combined result.
func (d *DefaultHallucinationDetector) Run(ctx context.Context, gen GeneratorOutput, trace *GCLTrace) (*HallucinationResult, error) {
	result := &HallucinationResult{}

	// L1: CLI parameter existence
	l1, err := d.checkCLIParams(gen.Command)
	if err != nil {
		return nil, fmt.Errorf("L1 check failed: %w", err)
	}
	result.L1 = l1

	// L2: JSON structure compliance
	l2, err := d.checkJSONStructure(gen.ResultExcerpt)
	if err != nil {
		return nil, fmt.Errorf("L2 check failed: %w", err)
	}
	result.L2 = l2

	// L3: WAF compliance
	l3, err := d.checkWAF(gen, trace)
	if err != nil {
		return nil, fmt.Errorf("L3 check failed: %w", err)
	}
	result.L3 = l3

	// Determine blocked status
	result.Blocked = result.BlockedBySafety()

	// Summary
	var parts []string
	if result.L1.Blocked {
		parts = append(parts, fmt.Sprintf("L1: CLI flags %v blocked", result.L1.InvalidFlags))
	}
	if result.L2.Blocked {
		parts = append(parts, fmt.Sprintf("L2: %d JSON errors blocked", len(result.L2.Errors)))
	}
	if len(result.L3.Violations) > 0 {
		parts = append(parts, fmt.Sprintf("L3: %d WAF violations", len(result.L3.Violations)))
	}
	if result.Blocked {
		result.Summary = "BLOCKED: " + strings.Join(parts, "; ")
	} else if len(parts) == 0 {
		result.Summary = "3/3 layers passed"
	} else {
		result.Summary = "3/3 layers passed; warnings: " + strings.Join(parts, "; ")
	}

	return result, nil
}

// LoadSkillFlags reads the CLI flags declared in references/cli-usage.md.
// Returns a map of flag name → true if the flag is valid.
func (d *DefaultHallucinationDetector) loadCLIUsage() (map[string]bool, error) {
	cliPath := filepath.Join(d.SkillRoot, "references", "cli-usage.md")
	data, err := readFileBytes(cliPath)
	if err != nil {
		return nil, err
	}
	flags := extractFlagsFromCLIUsage(string(data))
	return flags, nil
}

// extractFlagsFromCLIUsage parses --flag patterns from cli-usage.md content.
func extractFlagsFromCLIUsage(content string) map[string]bool {
	flags := make(map[string]bool)
	// Match --flag or --flag=value patterns in code blocks or tables.
	lines := strings.Split(content, "\n")
	for _, line := range lines {
		// Match --<name> patterns (strip trailing punctuation).
		for _, token := range strings.Fields(line) {
			if strings.HasPrefix(token, "--") && len(token) > 2 {
				name := strings.TrimSuffix(strings.TrimPrefix(token, "--"), ",")
				name = strings.Split(name, "=")[0]
				if name != "" {
					flags[name] = true
				}
			}
		}
	}
	return flags
}

// readFileBytes reads a file, returning nil bytes on missing file (not an error).
func readFileBytes(path string) ([]byte, error) {
	data, err := osReadFile(path)
	if err != nil && osIsNotExist(err) {
		return nil, nil
	}
	return data, err
}

// osReadFile wraps os.ReadFile for testability.
var _ = osReadFile // silence unused warning; real impl uses os.ReadFile
var osReadFile = func(name string) ([]byte, error) { return os.ReadFile(name) }

// osIsNotExist is os.IsNotExist for testability.
var osIsNotExist = func(err error) bool { return os.IsNotExist(err) }

// osOpenFile is os.Open for testability.
var osOpenFile = func(name string) (ioReadCloser, error) { return os.Open(name) }

// ioReadCloser is os.File for testability.
type ioReadCloser interface {
	Read([]byte) (int, error)
	Close() error
}
