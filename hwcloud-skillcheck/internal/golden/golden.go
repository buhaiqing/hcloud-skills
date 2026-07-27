package golden

import (
	"os"
	"path/filepath"
	"strings"
)

const MinScenariosPerSkill = 5

type Scenario struct {
	Name             string   `json:"name"`
	Command          string   `json:"command"`
	ExpectedStdout   string   `json:"expected_stdout_excerpt"`
	ExpectedStderr   string   `json:"expected_stderr_excerpt"`
	ExpectedExitCode int      `json:"expected_exit_code"`
	Tags             []string `json:"tags"`
}

type Report struct {
	PerSkill map[string]int
	Passed   int
	Failed   int
	Errors   []string
}

func (r *Report) BelowThreshold(skill string, threshold int) bool {
	return r.PerSkill[skill] < threshold
}

func Run(root string) (*Report, error) {
	r := &Report{PerSkill: map[string]int{}}
	base := filepath.Join(root, "internal", "golden", "testdata")
	_ = filepath.Walk(base, func(p string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() || !strings.HasSuffix(p, ".json") {
			return nil
		}
		rel, _ := filepath.Rel(base, p)
		parts := strings.SplitN(rel, string(filepath.Separator), 2)
		if len(parts) < 2 {
			return nil
		}
		skill := parts[0]
		if skill == "cross-product" {
			return nil
		}
		r.PerSkill[skill]++
		// Per-scenario run is implemented in Task 6; here we only count.
		return nil
	})
	return r, nil
}
