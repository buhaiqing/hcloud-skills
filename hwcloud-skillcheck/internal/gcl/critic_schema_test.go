package gcl

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// helperBinaryPath builds the critictest/test_helper Go binary the
// first time it is called (per test run) and returns its absolute path.
// The binary is placed under t.TempDir() so each test gets a clean
// PATH and the artifact is GC'd at test end.
func helperBinaryPath(t *testing.T) string {
	t.Helper()
	tmp := t.TempDir()
	bin := filepath.Join(tmp, "critic-helper")
	// Build from the helper's own source file. The test runs from
	// the gcl package directory so we use a relative path.
	pkg := "./critictest/test_helper"
	cmd := exec.Command("go", "build", "-o", bin, pkg)
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		t.Fatalf("build critictest helper: %v", err)
	}
	return bin
}

// writePayloadFile writes a JSON file in t.TempDir() and returns its
// path. PAYLOAD_FILE points the helper at it.
func writePayloadFile(t *testing.T, payload string) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), "payload.json")
	if err := os.WriteFile(p, []byte(payload), 0o600); err != nil {
		t.Fatalf("write payload: %v", err)
	}
	return p
}

// TestExternalCritic_ValidWireIsAccepted covers the happy path: a
// subprocess that emits JSON conforming to critic_output.schema.json
// must produce a CriticResult with all 5 dimensions populated, Mode
// carried through, and Blocking honored.
func TestExternalCritic_ValidWireIsAccepted(t *testing.T) {
	bin := helperBinaryPath(t)
	payload := `{"scores":{"correctness":0.9,"safety":1.0,"idempotency":0.8,"traceability":0.7,"spec_compliance":0.85},"suggestions":["ok"],"blocking":false,"mode":"isolated-critic"}`
	t.Setenv("PAYLOAD_FILE", writePayloadFile(t, payload))
	c := NewExternalCritic(bin)
	got := c.Score(context.Background(), GeneratorOutput{Command: "echo hi"})
	if got.Mode != "isolated-critic" {
		t.Errorf("Mode: want isolated-critic, got %q", got.Mode)
	}
	if got.Blocking {
		t.Errorf("Blocking should be false on valid payload, got true")
	}
	for _, dim := range []string{"correctness", "safety", "idempotency", "traceability", "spec_compliance"} {
		v, ok := got.Scores[dim]
		if !ok {
			t.Errorf("Score missing %q (have %v)", dim, got.Scores)
		}
		if v < 0 || v > 1 {
			t.Errorf("%s score out of [0,1]: %v", dim, v)
		}
	}
}

// TestExternalCritic_SchemaInvalid covers A6 of the P0 spec: an
// external Critic whose JSON fails schema validation must yield
// Mode starting with "schema-invalid:" so the rubric thresholds
// reject all scores and the Runner goes RETRY/HALT.
func TestExternalCritic_SchemaInvalid(t *testing.T) {
	bin := helperBinaryPath(t)
	// Valid JSON object but missing the required "scores" property.
	payload := `{"suggestions":["missing scores field"]}`
	t.Setenv("PAYLOAD_FILE", writePayloadFile(t, payload))
	c := NewExternalCritic(bin)
	got := c.Score(context.Background(), GeneratorOutput{Command: "echo hi"})
	if !strings.HasPrefix(got.Mode, "schema-invalid") {
		t.Errorf("Mode must start with schema-invalid; got %q", got.Mode)
	}
	if len(got.Suggestions) == 0 {
		t.Error("Suggestions must include at least one diagnostic on schema failure")
	}
}

// TestExternalCritic_SchemaInvalidOutOfRange: safety=1.5 violates
// maximum=1 → schema-invalid path.
func TestExternalCritic_SchemaInvalidOutOfRange(t *testing.T) {
	bin := helperBinaryPath(t)
	payload := `{"scores":{"correctness":0.9,"safety":1.5,"idempotency":0.8,"traceability":0.7,"spec_compliance":0.85}}`
	t.Setenv("PAYLOAD_FILE", writePayloadFile(t, payload))
	c := NewExternalCritic(bin)
	got := c.Score(context.Background(), GeneratorOutput{Command: "echo hi"})
	if !strings.HasPrefix(got.Mode, "schema-invalid") {
		t.Errorf("Mode must start with schema-invalid; got %q", got.Mode)
	}
}

// TestExternalCritic_DecodeError: output is not even valid JSON.
func TestExternalCritic_DecodeError(t *testing.T) {
	bin := helperBinaryPath(t)
	payload := `not json at all`
	t.Setenv("PAYLOAD_FILE", writePayloadFile(t, payload))
	c := NewExternalCritic(bin)
	got := c.Score(context.Background(), GeneratorOutput{Command: "echo hi"})
	if got.Mode != "decode-error" {
		t.Errorf("Mode: want decode-error, got %q", got.Mode)
	}
}

// TestExternalCritic_TimeoutContext: a tight context deadline aborts
// the subprocess without panicking.
func TestExternalCritic_TimeoutContext(t *testing.T) {
	bin := helperBinaryPath(t)
	// Helper reads from stdin and then writes payload — if we cancel
	// the ctx before the parent finishes writing, the subprocess is
	// killed. Use a tiny ctx to force the path.
	payload := `{"scores":{"correctness":0.5,"safety":1.0,"idempotency":0.5,"traceability":0.5,"spec_compliance":0.5}}`
	t.Setenv("PAYLOAD_FILE", writePayloadFile(t, payload))
	c := NewExternalCritic(bin)
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	got := c.Score(ctx, GeneratorOutput{Command: "echo hi"})
	// Either exec-error or schema-invalid is acceptable; the test
	// just asserts the call returns synchronously with a populated Mode.
	if got.Mode == "" {
		t.Error("Mode must be populated even on timeout")
	}
}
