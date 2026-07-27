package clifixtures

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type Fixture struct {
	Args          []string `json:"args"`
	StdoutExcerpt string   `json:"stdout_excerpt"`
	StderrExcerpt string   `json:"stderr_excerpt"`
	ExitCode      int      `json:"exit_code"`
	CapturedAt    string   `json:"captured_at,omitempty"`
}

// Check replays `args` against the recorded fixture and returns
// (match, err). match is false on any drift. err is non-nil on
// internal failure.
func Check(root string, args []string, stdout, stderr string, exit int) (bool, error) {
	fix, err := load(root, args)
	if err != nil {
		return false, err
	}
	if fix.StdoutExcerpt != stdout || fix.StderrExcerpt != stderr || fix.ExitCode != exit {
		return false, nil
	}
	return true, nil
}

func load(root string, args []string) (Fixture, error) {
	name := strings.Join(args, "__") + ".json"
	path := filepath.Join(root, "internal", "clifixtures", "fixtures", name)
	b, err := os.ReadFile(path)
	if err != nil {
		return Fixture{}, fmt.Errorf("fixture %s: %w", name, err)
	}
	var f Fixture
	if err := json.Unmarshal(b, &f); err != nil {
		return Fixture{}, fmt.Errorf("parse %s: %w", name, err)
	}
	return f, nil
}
