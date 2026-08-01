package critic

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestSafety(t *testing.T) {
	cases := []struct {
		name    string
		command string
		want    float64
	}{
		{"safe hcloud", "hcloud ecs list-servers", 1.0},
		{"destructive without dry-run", "hcloud ecs delete-server --region cn-north-4", 0.0},
		{"destructive with dry-run", "hcloud ecs delete-server --dry-run", 1.0},
		{"non-destructive write", "hcloud ecs create-server --name foo", 1.0},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			s, _ := scoreSafety(c.command)
			if s != c.want {
				t.Fatalf("got %v, want %v", s, c.want)
			}
		})
	}
}

func TestCorrectness(t *testing.T) {
	if s, _ := scoreCorrectness(""); s != 0.0 {
		t.Fatalf("empty command should be 0, got %v", s)
	}
	if s, _ := scoreCorrectness("hcloud ecs list-servers"); s != 1.0 {
		t.Fatalf("hcloud should be 1, got %v", s)
	}
	if s, _ := scoreCorrectness("go run main.go"); s != 1.0 {
		t.Fatalf("go run should be 1, got %v", s)
	}
	if s, _ := scoreCorrectness("rm -rf /"); s != 0.5 {
		t.Fatalf("unknown should be 0.5, got %v", s)
	}
}

func TestIdempotency(t *testing.T) {
	// Python original matches the FULL extracted token (`list`, `show`,
	// `create-server`, etc.) against READ_VERBS / WRITE_VERBS. Only exact
	// matches count; anything else falls through to 0.5.
	cases := []struct {
		command string
		want    float64
	}{
		{"hcloud foo list", 1.0},         // exact "list" in READ_VERBS
		{"hcloud foo show", 1.0},         // exact "show"
		{"hcloud foo describe", 1.0},     // exact "describe"
		{"hcloud foo get", 1.0},          // exact "get"
		{"hcloud ecs list-servers", 0.5}, // full token is "list-servers", not in set
		{"hcloud ecs create-server --name x", 0.5},
		{"hcloud ecs delete-server --id x", 0.5},
		{"hcloud ecs unknown-thing", 0.5},
	}
	for _, c := range cases {
		got := scoreIdempotency(c.command)
		if got != c.want {
			t.Errorf("%q: got %v, want %v", c.command, got, c.want)
		}
	}
}

func TestTraceability(t *testing.T) {
	if s, _ := scoreTraceability(map[string]any{"command": "x", "result_excerpt": "y"}); s != 1.0 {
		t.Fatalf("both set should be 1, got %v", s)
	}
	if s, _ := scoreTraceability(map[string]any{"command": "", "result_excerpt": "y"}); s != 0.0 {
		t.Fatalf("empty command should be 0, got %v", s)
	}
	if s, _ := scoreTraceability(map[string]any{"command": "x", "result_excerpt": ""}); s != 0.0 {
		t.Fatalf("empty excerpt should be 0, got %v", s)
	}
}

func TestSpecCompliance(t *testing.T) {
	good := map[string]any{
		"command":        "hcloud ecs list-servers",
		"result_excerpt": "ok",
		"exit_code":      0,
	}
	if s, _ := scoreSpecCompliance("hcloud ecs list-servers", good); s != 1.0 {
		t.Fatalf("clean run should be 1, got %v", s)
	}

	badExit := map[string]any{
		"command":        "hcloud ecs list-servers",
		"result_excerpt": "fail",
		"exit_code":      1,
	}
	if s, _ := scoreSpecCompliance("hcloud ecs list-servers", badExit); s != 0.5 {
		t.Fatalf("non-zero exit should be 0.5, got %v", s)
	}

	leaky := map[string]any{
		"command":        "hcloud ecs list-servers",
		"result_excerpt": "HW_SECRET_ACCESS_KEY=abc123def456ghi789jkl",
		"exit_code":      0,
	}
	if s, _ := scoreSpecCompliance("hcloud ecs list-servers", leaky); s != 0.0 {
		t.Fatalf("credential leak should be 0, got %v", s)
	}
}

func TestHasCredentialLeak(t *testing.T) {
	cases := []struct {
		text string
		want bool
	}{
		{"", false},
		{"HW_SECRET_ACCESS_KEY=abc123def456ghi789jkl", true},
		{"SECRET_ACCESS_KEY=foo bar", true},
		{"SecretAccessKey:xxxxx", true},
		{"SK=ABCDEFGHIJKLMNOPQRSTUVWXYZ123456", true},
		{"hw_secret_access_key=abc123def456ghi789jkl", true}, // 小写，(?i) 应匹配
		{"sk=abcdefghijklmnopqrstuvwxyz012345", true},        // 小写，(?i) 应匹配
		{"HW_SECRET_ACCESS_KEY=<masked>", false},
		{"plain text without secrets", false},
	}
	for _, c := range cases {
		if got := HasCredentialLeak(c.text); got != c.want {
			t.Errorf("%q: got %v, want %v", c.text, got, c.want)
		}
	}
}

func TestExtractSkillShort(t *testing.T) {
	cases := map[string]string{
		"hcloud ecs list-servers":    "ecs",
		"hcloud rds describe --id 1": "rds",
		"random command":             "",
		"":                           "",
	}
	for in, want := range cases {
		if got := ExtractSkillShort(in); got != want {
			t.Errorf("%q: got %q, want %q", in, got, want)
		}
	}
}

func TestExtractOperation(t *testing.T) {
	cases := map[string]string{
		"hcloud ecs list-servers":         "list-servers",
		"hcloud ecs delete-server --id x": "delete-server",
		"hcloud rds Create-Instance --x":  "create-instance", // lower-cased
		"":                                "",
	}
	for in, want := range cases {
		if got := ExtractOperation(in); got != want {
			t.Errorf("%q: got %q, want %q", in, got, want)
		}
	}
}

func TestEstimateCost(t *testing.T) {
	if c, _ := EstimateCost("hcloud ecs create-server --name x"); c != 0.05 {
		t.Errorf("ecs create-server: got %v, want 0.05", c)
	}
	if c, _ := EstimateCost("hcloud rds create-instance --name x"); c != 0.12 {
		t.Errorf("rds create-instance: got %v, want 0.12", c)
	}
	if c, _ := EstimateCost("hcloud ecs list-servers"); c != 0.001 {
		t.Errorf("default cost: got %v, want 0.001", c)
	}
}

func TestScoreBlocking(t *testing.T) {
	// Destructive without dry-run -> safety=0 -> blocking=true.
	r := Score(map[string]any{
		"command":        "hcloud ecs delete-server --id x",
		"result_excerpt": "ok",
		"exit_code":      0,
	})
	if !r["blocking"].(bool) {
		t.Fatal("expected blocking=true for destructive without dry-run")
	}

	// Safe read -> blocking=false.
	r2 := Score(map[string]any{
		"command":        "hcloud ecs list-servers",
		"result_excerpt": "ok",
		"exit_code":      0,
	})
	if r2["blocking"].(bool) {
		t.Fatal("expected blocking=false for read-only op")
	}
}

func TestScoreFinopsSuggestion(t *testing.T) {
	// 0.12 >= 0.1 -> should add a FinOps suggestion.
	r := Score(map[string]any{
		"command":        "hcloud rds create-instance --name x",
		"result_excerpt": "ok",
		"exit_code":      0,
	})
	suggs := r["suggestions"].([]string)
	found := false
	for _, s := range suggs {
		if len(s) >= 6 && s[:6] == "FinOps" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected FinOps suggestion, got %v", suggs)
	}
	if r["finops_estimate_usd"].(float64) != 0.12 {
		t.Fatalf("expected finops_estimate_usd=0.12, got %v", r["finops_estimate_usd"])
	}
}

func TestScoreLimitSuggestions(t *testing.T) {
	// Force many suggestions: credential leak + bad exit + missing excerpt + bad cost.
	r := Score(map[string]any{
		"command":        "hcloud rds delete-instance HW_SECRET_ACCESS_KEY=abc123def456ghi789jkl",
		"result_excerpt": "HW_SECRET_ACCESS_KEY=abc123def456ghi789jkl",
		"exit_code":      1,
	})
	suggs := r["suggestions"].([]string)
	if len(suggs) > 5 {
		t.Fatalf("expected <=5 suggestions, got %d", len(suggs))
	}
}

func TestScoreJSONShape(t *testing.T) {
	r := Score(map[string]any{
		"command":        "hcloud ecs list-servers",
		"result_excerpt": "ok",
		"exit_code":      0,
	})
	out, err := json.Marshal(r)
	if err != nil {
		t.Fatal(err)
	}
	var roundTrip map[string]any
	if err := json.Unmarshal(out, &roundTrip); err != nil {
		t.Fatal(err)
	}
	for _, k := range []string{"scores", "suggestions", "blocking", "_mode"} {
		if _, ok := roundTrip[k]; !ok {
			t.Errorf("missing %q in output", k)
		}
	}
	scores := roundTrip["scores"].(map[string]any)
	for _, dim := range []string{"correctness", "safety", "idempotency", "traceability", "spec_compliance"} {
		if _, ok := scores[dim]; !ok {
			t.Errorf("missing dim %q", dim)
		}
	}
}

func TestScoreFileRoundTrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "gen.json")
	payload := map[string]any{
		"command":        "hcloud ecs list-servers",
		"result_excerpt": "ok",
		"exit_code":      0,
	}
	data, _ := json.Marshal(payload)
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatal(err)
	}
	r, err := ScoreFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if r["blocking"].(bool) {
		t.Fatal("read-only op should not block")
	}
}
