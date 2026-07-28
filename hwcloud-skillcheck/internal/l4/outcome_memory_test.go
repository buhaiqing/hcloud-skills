package l4

import (
	"encoding/json"
	"testing"
)

func TestOutcomeRecord_RoundTrip(t *testing.T) {
	in := OutcomeRecord{
		ID:           "fixed-uuid-for-test",
		Timestamp:    "2026-07-28T08:30:00Z",
		TaskID:       "task-1",
		Skill:        "huaweicloud-ecs-ops",
		Action:       "delete-instances",
		ContextHash:  "deadbeef",
		Outcome:      "failure",
		ErrorClass:   "transient",
		ErrorMsg:     "connection reset",
		RetryCount:   1,
		DurationMS:   4321,
		Risk:         "high",
		RBACDecision: "allowed",
		GCLDecision:  "PASS",
	}
	raw, err := json.Marshal(in)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var out OutcomeRecord
	if err := json.Unmarshal(raw, &out); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if out != in {
		t.Fatalf("round-trip mismatch:\n got=%+v\nwant=%+v", out, in)
	}
}