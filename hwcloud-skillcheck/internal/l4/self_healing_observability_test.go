package l4

import (
	"bytes"
	"log/slog"
	"strings"
	"testing"
	"time"
)

func TestPreExecHookObservability(t *testing.T) {
	dir := t.TempDir()
	mem, err := NewOutcomeMemory(dir)
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC().Format(time.RFC3339)
	for i := 0; i < 2; i++ {
		if err := mem.Record(OutcomeRecord{
			ID: string(rune('a' + i)), Timestamp: now, Skill: "huaweicloud-ecs-ops", Action: "delete-instance", Outcome: "failure",
		}); err != nil {
			t.Fatal(err)
		}
	}

	var logs bytes.Buffer
	previousLogger := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(&logs, nil)))
	defer slog.SetDefault(previousLogger)
	previousMetrics := DefaultHealingMetrics
	DefaultHealingMetrics = &HealingMetrics{}
	defer func() { DefaultHealingMetrics = previousMetrics }()

	decision := PreExecHook(
		TaskStep{Skill: "huaweicloud-ecs-ops", Action: "delete-instance", Risk: "high"},
		mem,
		HealingPolicy{FailureRateSkipThreshold: 0.5, MinSamples: 2, LookbackWindow: time.Hour},
	)

	if decision.Action != "skip" {
		t.Fatalf("decision action = %q, want skip", decision.Action)
	}
	if got := DefaultHealingMetrics.PreExecSkip.Load(); got != 1 {
		t.Fatalf("PreExecSkip = %d, want 1", got)
	}
	for _, field := range []string{
		"msg=healing_decision",
		"skill=huaweicloud-ecs-ops",
		"action=delete-instance",
		"hook=PreExecHook",
		"decision_action=skip",
		`decision_reason="high historical failure rate"`,
		"retry_count=0",
		"risk=high",
	} {
		if !strings.Contains(logs.String(), field) {
			t.Errorf("log missing %q: %s", field, logs.String())
		}
	}
}
