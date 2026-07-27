package golden

import (
	"encoding/json"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
)

const MinScenariosPerSkill = 5

var argSplitRe = regexp.MustCompile(`\s+`)

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

func splitArgs(cmd string) []string { return argSplitRe.Split(cmd, -1) }

func runScenario(root, script string, sc Scenario) (bool, error) {
	bin := filepath.Join(root, "bin", "mockhcloud")
	cmd := exec.Command(bin, append([]string{"--script", script}, splitArgs(sc.Command)...)...)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return false, err
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return false, err
	}
	if err := cmd.Start(); err != nil {
		return false, err
	}
	var out []byte
	done := make(chan struct{})
	go func() {
		out, _ = io.ReadAll(stdout)
		_, _ = io.ReadAll(stderr)
		close(done)
	}()
	<-done
	exit := 0
	if err := cmd.Wait(); err != nil {
		if ee, ok := err.(*exec.ExitError); ok {
			exit = ee.ExitCode()
		} else {
			return false, err
		}
	}
	if string(out) != sc.ExpectedStdout || exit != sc.ExpectedExitCode {
		return false, nil
	}
	return true, nil
}

func Run(root string) (*Report, error) {
	r := &Report{PerSkill: map[string]int{}}
	base := filepath.Join(root, "internal", "golden", "testdata")
	err := filepath.Walk(base, func(p string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() || !strings.HasSuffix(p, ".json") {
			return nil
		}
		rel, _ := filepath.Rel(base, p)
		parts := strings.SplitN(rel, string(filepath.Separator), 2)
		if len(parts) < 2 {
			return nil
		}
		skill := parts[0]
		skillDir := skill
		if skill == "cross-product" {
			skillDir = "cross-product"
		}
		r.PerSkill[skill]++
		b, _ := os.ReadFile(p)
		var sc Scenario
		if err := json.Unmarshal(b, &sc); err != nil {
			r.Errors = append(r.Errors, p+": "+err.Error())
			return nil
		}
		scriptPath := filepath.Join(root, "testdata", skillDir, sc.Name+".script.json")
		ok, _ := runScenario(root, scriptPath, sc)
		if ok {
			r.Passed++
		} else {
			r.Failed++
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return r, nil
}
