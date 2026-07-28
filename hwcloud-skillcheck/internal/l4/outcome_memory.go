package l4

import "encoding/json"

// OutcomeRecord is one row in outcomes.jsonl.
// See docs/superpowers/specs/outcome-memory-self-healing.md §5.
type OutcomeRecord struct {
	ID           string `json:"id"`
	Timestamp    string `json:"ts"`
	TaskID       string `json:"task_id"`
	Skill        string `json:"skill"`
	Action       string `json:"action"`
	ContextHash  string `json:"context_hash"`
	Outcome      string `json:"outcome"`
	ErrorClass   string `json:"error_class"`
	ErrorMsg     string `json:"error_msg,omitempty"`
	RetryCount   int    `json:"retry_count"`
	DurationMS   int64  `json:"duration_ms"`
	Risk         string `json:"risk"`
	RBACDecision string `json:"rbac_decision"`
	GCLDecision  string `json:"gcl_decision"`
}

// ensure json is "used" — Marshal/Unmarshal via the package's other tests will reference it.
// In Go, types themselves do not require imports; we keep the json package ready for
// future Record methods in this file.
var _ = json.Marshal
