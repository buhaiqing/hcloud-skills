package l4

import "time"

// Schema and cap constants. Bumping the schema version is a breaking
// change; Loaders refuse unknown schemas.
const (
	ContextSchema      = "context-memory/v1"
	MaxRecentTasks     = 20
	MaxRecentErrors    = 20
	MaxOpenTasks       = 50
	SessionRotateAfter = 24 * time.Hour
)

// Context is the entire document held in <root>/.l4-memory/context.json.
// See docs/superpowers/specs/agent-cross-call-memory.md §5.
type Context struct {
	Schema       string            `json:"schema"`
	SessionID    string            `json:"session_id"`
	CreatedAt    string            `json:"created_at"`
	LastUpdated  string            `json:"last_updated"`
	RecentTasks  []TaskSummary     `json:"recent_tasks"`
	OpenTasks    []string          `json:"open_tasks"`
	RecentErrors []ErrorSummary    `json:"recent_errors"`
	Preferences  map[string]string `json:"preferences"`
}

// TaskSummary is a compact record of one past task.
type TaskSummary struct {
	TaskID       string `json:"task_id"`
	Fault        string `json:"fault,omitempty"`
	StartedAt    string `json:"started_at"`
	FinishedAt   string `json:"finished_at,omitempty"`
	Status       string `json:"status"`
	PrimarySkill string `json:"primary_skill,omitempty"`
}

// ErrorSummary is a compact record of one past error.
type ErrorSummary struct {
	Timestamp  string `json:"ts"`
	Skill      string `json:"skill"`
	Action     string `json:"action"`
	ErrorClass string `json:"error_class"`
	ErrorMsg   string `json:"error_msg,omitempty"`
}
