package cmd

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/buhaiqing/hcloud-skills/hwcloud-skillcheck/internal/l4"
)

func runMemoryInspect(args []string) error {
	if len(args) == 0 || args[0] != "inspect" {
		return fmt.Errorf("memory: missing subcommand (use 'inspect')")
	}
	fs := newFlagSet("memory inspect")
	root := fs.String("root", ".", "workspace root")
	if err := fs.Parse(args[1:]); err != nil {
		return err
	}
	rootDir, err := filepath.Abs(*root)
	if err != nil {
		return err
	}

	if _, err := l4.NewOutcomeMemory(rootDir); err != nil {
		return err
	}
	outcomePath := filepath.Join(rootDir, ".l4-memory", "outcomes.jsonl")
	outcomes, err := loadOutcomeRecords(outcomePath)
	if err != nil {
		return err
	}
	sort.SliceStable(outcomes, func(i, j int) bool { return outcomes[i].Timestamp > outcomes[j].Timestamp })

	fmt.Fprintf(os.Stdout, "outcome_memory_path: %s\n", outcomePath)
	fmt.Fprintf(os.Stdout, "outcome_records:    %d  (last 90 days, 90-day retention)\n", len(outcomes))
	recentCount := min(5, len(outcomes))
	fmt.Fprintf(os.Stdout, "outcome_recent:     last %d records\n", recentCount)
	for _, record := range outcomes[:recentCount] {
		fmt.Fprintf(os.Stdout, "  [%s] %s / %-19s outcome=%s", record.Timestamp, record.Skill, record.Action, record.Outcome)
		if record.ErrorClass != "" {
			fmt.Fprintf(os.Stdout, "  error_class=%s", record.ErrorClass)
		}
		if record.RetryCount > 0 {
			fmt.Fprintf(os.Stdout, "  retry=%d", record.RetryCount)
		}
		fmt.Fprintln(os.Stdout)
	}

	contextPath := filepath.Join(rootDir, ".l4-memory", "context.json")
	contextMemory, err := l4.NewContextMemory(rootDir)
	if err != nil {
		return err
	}
	context, err := contextMemory.Load()
	if err != nil {
		return err
	}
	fmt.Fprintf(os.Stdout, "\ncontext_memory_path: %s\n", contextPath)
	fmt.Fprintf(os.Stdout, "session_id:          %s\n", context.SessionID)
	fmt.Fprintf(os.Stdout, "created_at:          %s  (age %s)\n", context.CreatedAt, contextAge(context.CreatedAt))
	fmt.Fprintf(os.Stdout, "recent_tasks:        %d of %d\n", len(context.RecentTasks), l4.MaxRecentTasks)
	for _, task := range context.RecentTasks {
		timestamp := task.FinishedAt
		if timestamp == "" {
			timestamp = task.StartedAt
		}
		fmt.Fprintf(os.Stdout, "  [%s] task=%s  status=%s  skill=%s\n", timestamp, task.TaskID, task.Status, task.PrimarySkill)
	}
	fmt.Fprintf(os.Stdout, "recent_errors:       %d\n", len(context.RecentErrors))
	for _, recentError := range context.RecentErrors {
		fmt.Fprintf(os.Stdout, "  [%s] %s / %s  error_class=%s\n", recentError.Timestamp, recentError.Skill, recentError.Action, recentError.ErrorClass)
	}
	fmt.Fprintf(os.Stdout, "open_tasks:          %d\n", len(context.OpenTasks))
	preferences, err := json.Marshal(context.Preferences)
	if err != nil {
		return err
	}
	fmt.Fprintf(os.Stdout, "preferences:         %s\n", preferences)
	return nil
}

func loadOutcomeRecords(path string) ([]l4.OutcomeRecord, error) {
	file, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	defer file.Close()

	var records []l4.OutcomeRecord
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		var record l4.OutcomeRecord
		if json.Unmarshal(scanner.Bytes(), &record) == nil {
			records = append(records, record)
		}
	}
	return records, scanner.Err()
}

func contextAge(createdAt string) string {
	created, err := time.Parse(time.RFC3339, createdAt)
	if err != nil {
		return "unknown"
	}
	age := time.Since(created).Truncate(time.Hour)
	if age < 0 {
		age = 0
	}
	return age.String()
}
