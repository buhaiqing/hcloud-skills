package l4

import (
	"strings"
	"testing"
	"time"
)

func TestRealExecutor_Success(t *testing.T) {
	r := NewRealExecutor()
	exitCode, out, err := r.Run(`echo hello`, 0)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if exitCode != 0 {
		t.Fatalf("want exit 0, got %d", exitCode)
	}
	if !strings.Contains(out, "hello") {
		t.Fatalf("want output to contain hello, got %q", out)
	}
}

func TestRealExecutor_Failure(t *testing.T) {
	r := NewRealExecutor()
	exitCode, _, err := r.Run(`false`, 0)
	if err == nil && exitErrMsg(err) == "" {
		// ok: Go returns a non-nil *exec.ExitError for non-zero exits
	}
	if exitCode != 1 {
		t.Fatalf("want exit 1, got %d", exitCode)
	}
	if err == nil {
		t.Fatalf("want non-nil err for exit 1")
	}
}

func TestRealExecutor_Timeout(t *testing.T) {
	r := NewRealExecutor()
	start := time.Now()
	_, _, err := r.Run(`sleep 5`, 10*time.Millisecond)
	elapsed := time.Since(start)
	if err == nil {
		t.Fatalf("want timeout err, got nil")
	}
	if !strings.Contains(err.Error(), "deadline exceeded") {
		t.Fatalf("want deadline exceeded, got %v", err)
	}
	if elapsed > 2*time.Second {
		t.Fatalf("took too long: %v (expected ~10ms)", elapsed)
	}
}

func TestRealExecutor_OutputTruncation(t *testing.T) {
	r := NewRealExecutor()
	r.MaxBytes = 1024 // 1 KB cap for fast test
	_, out, _ := r.Run(`head -c 2000000 /dev/urandom | base64`, 30*time.Second)
	// base64 inflates; expect at most MaxBytes + base64 overhead slack.
	// We just verify the buffer never exceeded the cap + a small slack.
	if int64(len(out)) > int64(r.MaxBytes)+4096 {
		t.Fatalf("output %d bytes exceeds MaxBytes+slack %d", len(out), r.MaxBytes+4096)
	}
	if len(out) == 0 {
		t.Fatalf("expected non-empty output even when truncated")
	}
}

func TestRealExecutor_EnvPassthrough(t *testing.T) {
	r := NewRealExecutor()
	r.Env = append(r.Env, "FOO_TEST_VALUE=bar123")
	_, out, err := r.Run(`printf '%s' "$FOO_TEST_VALUE"`, 0)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if !strings.Contains(out, "bar123") {
		t.Fatalf("want output to contain bar123 (env passthrough), got %q", out)
	}
}

// exitErrMsg extracts error text without leaking the *exec.ExitError typed
// nil interface trap.
func exitErrMsg(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}
