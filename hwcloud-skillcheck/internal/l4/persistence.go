// Package l4 provides L4 closed-loop orchestrator: task persistence,
// RBAC permission control, trust scoring, topology, and predictive fault handling.
package l4

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	"golang.org/x/sync/errgroup"
)

// TaskState represents a multi-step execution checkpoint.
// It is persisted to disk after each completed step so that a crash can
// be recovered without re-running completed steps.
type TaskState struct {
	// ID is the unique task identifier (16 hex chars, no dashes).
	ID string `json:"id"`
	// Fault is the original user request.
	Fault string `json:"fault"`
	// Root is the repository root at task creation time.
	Root string `json:"root"`
	// CreatedAt is the task creation timestamp (ISO-8601 UTC).
	CreatedAt string `json:"created_at"`
	// UpdatedAt is the last checkpoint timestamp.
	UpdatedAt string `json:"updated_at"`
	// Status is one of: running | paused | completed | failed | aborted.
	Status TaskStatus `json:"status"`
	// CurrentStep is the 0-based index of the next step to execute.
	// Completed steps are in Steps[0:CurrentStep].
	CurrentStep int `json:"current_step"`
	// Steps is the ordered list of planned steps.
	Steps []TaskStep `json:"steps"`
	// Results captures the outcome of each completed step.
	Results []StepResult `json:"results,omitempty"`
	// RecoveryCount is the number of times this task was resumed after a crash.
	RecoveryCount int `json:"recovery_count"`
	// ParentTaskID links resumptions to their origin task.
	ParentTaskID string `json:"parent_task_id,omitempty"`
}

// TaskStatus is the set of valid task states.
type TaskStatus string

const (
	TaskStatusRunning   TaskStatus = "running"
	TaskStatusPaused    TaskStatus = "paused"
	TaskStatusCompleted TaskStatus = "completed"
	TaskStatusFailed    TaskStatus = "failed"
	TaskStatusAborted   TaskStatus = "aborted"
)

// TaskStep mirrors ExecutionStep for serialization.
type TaskStep struct {
	Step         int    `json:"step"`
	Skill        string `json:"skill"`
	SkillShort   string `json:"skill_short,omitempty"`
	Action       string `json:"action"`
	Verb         string `json:"verb,omitempty"`
	Risk         string `json:"risk"`
	RequiresRBAC bool   `json:"requires_rbac"`
	Command      string `json:"command,omitempty"`
}

// StepResult captures the outcome of one completed step.
type StepResult struct {
	Step         int                `json:"step"`
	Skill        string             `json:"skill,omitempty"`
	Command      string             `json:"command"`
	StartedAt    string             `json:"started_at"`
	FinishedAt   string             `json:"finished_at"`
	ExitCode     int                `json:"exit_code"`
	Success      bool               `json:"success"`
	Error        string             `json:"error,omitempty"`
	Output       string             `json:"output,omitempty"`
	RBACApproved bool               `json:"rbac_approved"`
	RBACReason   string             `json:"rbac_reason,omitempty"`
	GCLDecision  string             `json:"gcl_decision"`
	GCLScores    map[string]float64 `json:"gcl_scores,omitempty"`
}

// newTaskID returns a 16-char hex string for TaskState.ID.
func newTaskID() string {
	var b [8]byte
	_ = mustReadRandom(b[:])
	return hex.EncodeToString(b[:])
}

func mustReadRandom(b []byte) error {
	_, err := rand.Read(b)
	return err
}

// EnsureMemoryDir creates <root>/.l4-memory with mode 0700 if missing.
// Returns the absolute-or-joined directory path.
func EnsureMemoryDir(root string) (string, error) {
	dir := filepath.Join(root, ".l4-memory")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return "", err
	}
	return dir, nil
}

// PersistTask writes task state to <root>/.l4-tasks/<id>.json.
// The directory is created with mode 0700 (owner-only).
func PersistTask(root, id string, state *TaskState) error {
	state.UpdatedAt = NowISO()
	taskDir := filepath.Join(root, ".l4-tasks")
	if err := os.MkdirAll(taskDir, 0o700); err != nil {
		return fmt.Errorf("persist task: mkdir: %w", err)
	}
	path := filepath.Join(taskDir, id+".json")
	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return fmt.Errorf("persist task: marshal: %w", err)
	}
	if err := os.WriteFile(path, append(data, '\n'), 0o600); err != nil {
		return fmt.Errorf("persist task: write: %w", err)
	}
	return nil
}

// LoadTask reads a persisted task state from disk.
// Returns nil, nil if the task does not exist.
func LoadTask(root, id string) (*TaskState, error) {
	path := filepath.Join(root, ".l4-tasks", id+".json")
	raw, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("load task: read: %w", err)
	}
	var state TaskState
	if err := json.Unmarshal(raw, &state); err != nil {
		return nil, fmt.Errorf("load task: unmarshal: %w", err)
	}
	return &state, nil
}

// ListTasks returns all persisted task IDs under root.
func ListTasks(root string) ([]string, error) {
	taskDir := filepath.Join(root, ".l4-tasks")
	entries, err := os.ReadDir(taskDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("list tasks: readdir: %w", err)
	}
	var ids []string
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".json") {
			continue
		}
		id := strings.TrimSuffix(e.Name(), ".json")
		ids = append(ids, id)
	}
	return ids, nil
}

// ResumeTask increments RecoveryCount and sets Status to running.
// If the task is already completed or aborted, ResumeTask returns an error.
func ResumeTask(state *TaskState) error {
	switch state.Status {
	case TaskStatusCompleted, TaskStatusAborted:
		return fmt.Errorf("cannot resume task in status %q", state.Status)
	}
	if state.CurrentStep >= len(state.Steps) {
		state.Status = TaskStatusCompleted
		return nil
	}
	state.RecoveryCount++
	state.Status = TaskStatusRunning
	state.UpdatedAt = NowISO()
	return nil
}

// PauseTask sets Status to paused. A paused task can be resumed later.
func PauseTask(state *TaskState) error {
	if state.Status != TaskStatusRunning {
		return fmt.Errorf("cannot pause task in status %q", state.Status)
	}
	state.Status = TaskStatusPaused
	state.UpdatedAt = NowISO()
	return nil
}

// CompleteTask marks the task as completed and clears current_step.
func CompleteTask(state *TaskState) {
	state.Status = TaskStatusCompleted
	state.CurrentStep = len(state.Steps)
	state.UpdatedAt = NowISO()
}

// FailTask marks the task as failed with an error message.
func FailTask(state *TaskState, errMsg string) {
	state.Status = TaskStatusFailed
	state.UpdatedAt = NowISO()
	if len(state.Results) > 0 {
		state.Results[len(state.Results)-1].Error = errMsg
	}
}

// AbortTask marks the task as aborted (user-requested cancellation).
func AbortTask(state *TaskState) {
	state.Status = TaskStatusAborted
	state.UpdatedAt = NowISO()
}

// NextStep returns the next step to execute, or nil if all steps are done.
func (s *TaskState) NextStep() *TaskStep {
	if s.CurrentStep >= len(s.Steps) {
		return nil
	}
	return &s.Steps[s.CurrentStep]
}

// IsResumable returns true if the task can be resumed.
func (s *TaskState) IsResumable() bool {
	return s.Status == TaskStatusPaused || s.Status == TaskStatusFailed
}

// CompletedCount returns the number of steps that have finished.
func (s *TaskState) CompletedCount() int {
	return len(s.Results)
}

// RemainingCount returns the number of steps not yet started.
func (s *TaskState) RemainingCount() int {
	return len(s.Steps) - s.CurrentStep
}

// Progress returns a human-readable progress string like "3/5 steps completed".
func (s *TaskState) Progress() string {
	return fmt.Sprintf("%d/%d steps completed", len(s.Results), len(s.Steps))
}

// DefaultTaskTTL is the age after which a stale "running" task is considered
// orphaned and eligible for recovery.
const DefaultTaskTTL = 24 * time.Hour

// IsStale returns true if a running task has not been updated for longer
// than ttl.
func (s *TaskState) IsStale(ttl time.Duration) bool {
	if s.Status != TaskStatusRunning {
		return false
	}
	t, err := parseISO(s.UpdatedAt)
	if err != nil {
		return false
	}
	return time.Since(t) > ttl
}

// ParseTaskID extracts task ID from a trace path or raw ID.
// Handles formats: "<root>/.l4-tasks/<id>.json", "<root>/audit-results/...",
// or a bare <id>.
func ParseTaskID(raw string) string {
	// Strip path prefix and extension.
	id := filepath.Base(raw)
	id = strings.TrimSuffix(id, ".json")
	// Strip common suffixes.
	id = strings.TrimPrefix(id, "orchestrator-trace-")
	id = strings.TrimPrefix(id, "gcl-trace-")
	id = strings.TrimSuffix(id, "-trace")
	return id
}

// preFetchFailurePatterns loads failure_patterns.json for each skill
// concurrently (capped at NumCPU). Best-effort: missing/unreadable skills
// are omitted from the returned map. Shared by HandleFault and
// RunExecutionLoop (Eng-M2 / T-7).
func preFetchFailurePatterns(root string, skills []string) map[string][]map[string]any {
	cache := map[string][]map[string]any{}
	if len(skills) == 0 {
		return cache
	}
	var mu sync.Mutex
	g := new(errgroup.Group)
	g.SetLimit(runtime.NumCPU())
	for _, skill := range skills {
		skill := skill
		g.Go(func() error {
			patterns, err := readFailurePatternsForSkill(root, skill)
			if err == nil {
				mu.Lock()
				cache[skill] = patterns
				mu.Unlock()
			}
			return nil
		})
	}
	_ = g.Wait()
	return cache
}

// readFailurePatternsForSkill loads <root>/<skill>/assets/failure_patterns.json.
func readFailurePatternsForSkill(root, skill string) ([]map[string]any, error) {
	if skill == "" || strings.Contains(skill, "..") || strings.ContainsAny(skill, `/\`) {
		return nil, fmt.Errorf("invalid skill id %q", skill)
	}
	skillID := skill
	if !strings.HasPrefix(skill, "huaweicloud-") {
		skillID = "huaweicloud-" + skill + "-ops"
	}
	path := filepath.Join(root, skillID, "assets", "failure_patterns.json")
	clean := filepath.Clean(path)
	rootClean := filepath.Clean(root)
	if !strings.HasPrefix(clean, rootClean+string(os.PathSeparator)) && clean != rootClean {
		return nil, fmt.Errorf("skill path escapes root: %s", skill)
	}
	raw, err := os.ReadFile(clean)
	if err != nil {
		return nil, err
	}
	var data struct {
		Patterns []map[string]any `json:"patterns"`
	}
	if err := json.Unmarshal(raw, &data); err != nil {
		return nil, err
	}
	return data.Patterns, nil
}
