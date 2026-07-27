package exp_audit

import (
	"encoding/json"
	"fmt"
	"os"
	"testing"

	"github.com/buhaiqing/hcloud-skills/hwcloud-skillcheck/internal/critic"
	"github.com/buhaiqing/hcloud-skills/hwcloud-skillcheck/internal/gcl"
)

func TestExp(t *testing.T) {
	tmp := t.TempDir()

	// A: default smoke
	r := gcl.Run(gcl.RunConfig{Skill: "huaweicloud-ecs-ops", Request: "smoke", Command: "echo ok", MaxIter: 1, Timeout: 5, Root: tmp})
	fmt.Printf("A. echo ok -> ExitCode=%d\n", r.ExitCode)

	// B: false command
	rB := gcl.Run(gcl.RunConfig{Skill: "huaweicloud-ecs-ops", Request: "smoke", Command: "false", MaxIter: 1, Timeout: 5, Root: tmp})
	fmt.Printf("B. false -> ExitCode=%d\n", rB.ExitCode)

	// C: unknown skill uses default MaxIter 2
	rC := gcl.Run(gcl.RunConfig{Skill: "unknown-skill", Request: "smoke", Command: "echo ok", MaxIter: 0, Timeout: 5, Root: tmp})
	fmt.Printf("C. unknown-skill MaxIter=0 -> ExitCode=%d (should be 0=PASS)\n", rC.ExitCode)

	// D: leaked command + masked excerpt — leak is on raw command, not excerpt
	rD := gcl.Run(gcl.RunConfig{Skill: "huaweicloud-ecs-ops", Request: "smoke", Command: "echo HW_SECRET_ACCESS_KEY=LeakABCDEFGHIJKLMNOPQRSTUVWXYZ", MaxIter: 1, Timeout: 5, Root: tmp})
	fmt.Printf("D. leak-in-cmd -> ExitCode=%d (should be 3=SAFETY)\n", rD.ExitCode)

	// E: Score with empty command + exit 0 + empty excerpt
	out := critic.Score(map[string]any{"command": "", "exit_code": 0, "result_excerpt": ""})
	b, _ := json.MarshalIndent(out, "", "  ")
	fmt.Println("E. critic.Score empty cmd/output:")
	fmt.Println(string(b))

	// F: Score with extra dimension keys — Decide thresholds only check the 5
	scores := map[string]float64{
		"correctness":     1.0,
		"safety":          1.0,
		"idempotency":     0.5,
		"traceability":    0.5,
		"spec_compliance": 1.0,
		"finops_estimate": 999.0, // bogus extra dim
	}
	fmt.Printf("F. Decide with extra dim: %s\n", gcl.Decide(scores))

	// G: Score with MISSING dimension — Decide map missing entry is < threshold → RETRY
	missing := map[string]float64{
		"correctness": 1.0,
		"safety":      1.0,
		// idempotency missing
		"traceability":    0.5,
		"spec_compliance": 1.0,
	}
	fmt.Printf("G. Decide with missing dim: %s (should be RETRY)\n", gcl.Decide(missing))

	// H: persist trace on PASS, ensure file written + mask applied
	if r.TracePath != "" {
		data, _ := os.ReadFile(r.TracePath)
		fmt.Printf("H. trace path=%s, bytes=%d, contains <masked>=%v\n",
			r.TracePath, len(data),
			containsBytes(data, []byte("<masked>")))
	}
}

func containsBytes(haystack, needle []byte) bool {
	for i := 0; i+len(needle) <= len(haystack); i++ {
		match := true
		for j := 0; j < len(needle); j++ {
			if haystack[i+j] != needle[j] {
				match = false
				break
			}
		}
		if match {
			return true
		}
	}
	return false
}
