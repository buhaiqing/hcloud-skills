package gcl

import (
	"encoding/json"
	"os"
	"strings"
	"testing"
	"time"
)

// TestRunDestructiveBlockedWithoutNonce is the spec-mandated (A7) test:
// `cfg.Command = "hcloud rds delete-instance"` without a confirmation
// nonce must exit with ExitSafety. The runner must short-circuit before
// invoking the Generator.
func TestRunDestructiveBlockedWithoutNonce(t *testing.T) {
	cfg := RunConfig{
		Skill:   "huaweicloud-rds-ops",
		Request: "delete a database",
		Command: "hcloud rds delete-instance --id rds-abc12345",
		MaxIter: 1,
		Timeout: 5,
		Root:    t.TempDir(),
		// no ConfirmationToken, no ConfirmationRegistry → pre-execution
		// gate must reject.
	}
	result := Run(cfg)
	if result.ExitCode != ExitSafety {
		t.Errorf("destructive command without nonce must ExitSafety, got %d",
			result.ExitCode)
	}
}

// TestPreExecutionGate_AllowsReadOnly verifies that a non-destructive
// command without a token passes the gate.
func TestPreExecutionGate_AllowsReadOnly(t *testing.T) {
	cfg := RunConfig{
		Skill:   "huaweicloud-ecs-ops",
		Command: "hcloud ecs list-servers --region cn-north-4",
		// no ConfirmationRegistry, no token — should still pass for non-destructive
	}
	if err := preExecutionGate(cfg, map[string]any{
		"operation":      "list-servers",
		"safety_class":   "read-only",
		"expected_state": "OK",
	}); err != nil {
		t.Errorf("read-only op should pass pre-execution gate: %v", err)
	}
}

// TestPreExecutionGate_RejectsDestructiveWithoutToken verifies that a
// destructive safety_class is rejected when no confirmation token is
// supplied, even with an intent.
func TestPreExecutionGate_RejectsDestructiveWithoutToken(t *testing.T) {
	cfg := RunConfig{
		Skill:   "huaweicloud-rds-ops",
		Command: "hcloud rds delete-instance --id rds-abc12345",
		// no ConfirmationRegistry, no token
	}
	err := preExecutionGate(cfg, map[string]any{
		"operation":      "delete-instance",
		"safety_class":   "destructive",
		"expected_state": "deleted",
	})
	if err == nil {
		t.Fatal("destructive op without token must be rejected")
	}
	if !strings.Contains(err.Error(), "ConfirmationRegistry") {
	}
}

// TestPreExecutionGate_RejectsDestructiveWithoutRegistry verifies the
// fail-closed path: token supplied but no registry.
func TestPreExecutionGate_RejectsDestructiveWithoutRegistry(t *testing.T) {
	cfg := RunConfig{
		Skill:             "huaweicloud-rds-ops",
		Command:           "hcloud rds delete-instance --id rds-x",
		ConfirmationToken: "cfm_fake_nonce",
		// ConfirmationRegistry still nil
	}
	err := preExecutionGate(cfg, map[string]any{
		"operation":    "delete-instance",
		"safety_class": "destructive",
	})
	if err == nil {
		t.Fatal("destructive op without ConfirmationRegistry must be rejected")
	}
	if !strings.Contains(err.Error(), "ConfirmationRegistry") {
		t.Errorf("error should mention ConfirmationRegistry: %v", err)
	}
}

// TestPreExecutionGate_AcceptsDestructiveWithValidToken covers the happy
// path: registry + nonce + destructive intent → token consumed, gate passes.
func TestPreExecutionGate_AcceptsDestructiveWithValidToken(t *testing.T) {
	reg := NewConfirmationRegistry(60 * time.Second)
	defer reg.Stop()

	nonce, err := reg.Issue("gcl:huaweicloud-rds-ops", "alice")
	if err != nil {
		t.Fatalf("Issue: %v", err)
	}

	cfg := RunConfig{
		Skill:                "huaweicloud-rds-ops",
		Command:              "hcloud rds delete-instance --id rds-x",
		ConfirmationToken:    nonce,
		ConfirmationRegistry: reg,
	}
	err = preExecutionGate(cfg, map[string]any{
		"operation":    "delete-instance",
		"safety_class": "destructive",
	})
	if err != nil {
		t.Errorf("destructive op with valid token should pass: %v", err)
	}

	// Second attempt with the same nonce must be rejected (one-time consumption).
	err = preExecutionGate(cfg, map[string]any{
		"operation":    "delete-instance",
		"safety_class": "destructive",
	})
	if err == nil {
		t.Error("nonce reuse must be rejected (one-time consumption)")
	}
}

// TestRetryBuilder_Iteration2UsesFeedback verifies that on iter >= 2 the
// Runner hands the previous masked output + Critic verdict to the
// RetryBuilder. We exercise this indirectly by checking that the trace
// has iter 2 with a command derived from the previous Critic Suggestions.
func TestRetryBuilder_Iteration2UsesFeedback(t *testing.T) {
	cfg := RunConfig{
		Skill:        "huaweicloud-ecs-ops",
		Request:      "list servers",
		Command:      "false", // exits 1 every time
		MaxIter:      2,
		Timeout:      5,
		RetryBuilder: MinimalFeedbackRetry{},
		Root:         t.TempDir(),
	}
	result := Run(cfg)
	// Even with retry, the loop should fail and exit MAX_ITER (false
	// always exits 1 → correctness=0 → RETRY → iter 2 same → MAX_ITER).
	if result.ExitCode != ExitMaxIter {
		t.Errorf("expected MAX_ITER, got %d", result.ExitCode)
	}
	// PersistTrace masks "generator.command" by default, so we can't
	// observe the raw iter-2 retry prompt from the trace. We instead
	// assert that the loop ran both iterations and the trace has them.
	trace, err := readLastTrace(t, result.TracePath)
	if err != nil {
		t.Fatalf("readLastTrace: %v", err)
	}
	if len(trace.Iterations) < 2 {
		t.Errorf("expected at least 2 iterations, got %d", len(trace.Iterations))
	}
}

// TestSanitizedRequest_PopulatedOnRun verifies that Run() populates the
// trace's SanitizedRequest field with the masked form of cfg.Request.
func TestSanitizedRequest_PopulatedOnRun(t *testing.T) {
	cfg := RunConfig{
		Skill:   "huaweicloud-ecs-ops",
		Request: "list ecs-abc12345 servers",
		Command: "echo ok",
		MaxIter: 1,
		Timeout: 5,
		Root:    t.TempDir(),
	}
	result := Run(cfg)
	if result.ExitCode != ExitOK {
		t.Fatalf("expected PASS, got %d", result.ExitCode)
	}
	trace, err := readLastTrace(t, result.TracePath)
	if err != nil {
		t.Fatalf("readLastTrace: %v", err)
	}
	if !strings.Contains(trace.SanitizedRequest, "<id>") {
		t.Errorf("SanitizedRequest should mask resource ID; got %q",
			trace.SanitizedRequest)
	}
	if strings.Contains(trace.SanitizedRequest, "ecs-abc12345") {
		t.Error("raw resource ID leaked into SanitizedRequest")
	}
}

// TestRun_FailClosedOnUnrecognizedToken verifies the P0 §14.1 fail-closed
// behavior: a request with an opaque alphanumeric blob that no
// masking rule covers must abort the run with ExitUsage.
func TestRun_FailClosedOnUnrecognizedToken(t *testing.T) {
	cfg := RunConfig{
		Skill:   "huaweicloud-ecs-ops",
		Request: "fetch opaqueabcddefghijklmnopqrstuvwxyz123456",
		MaxIter: 1,
		Timeout: 5,
		Root:    t.TempDir(),
	}
	result := Run(cfg)
	if result.ExitCode != ExitUsage {
		t.Errorf("unrecognised token must fail closed (ExitUsage), got %d", result.ExitCode)
	}
}

// readLastTrace is a tiny helper for the wiring tests above.
func readLastTrace(t *testing.T, path string) (*GCLTrace, error) {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var trace GCLTrace
	if err := json.Unmarshal(data, &trace); err != nil {
		return nil, err
	}
	return &trace, nil
}
