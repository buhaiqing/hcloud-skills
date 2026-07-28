package confusion

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type Matrix struct {
	Top1VsIntended map[string]map[string]int `json:"top1_vs_intended"`
}

func Derive(root string) (Matrix, error) {
	matrix := Matrix{Top1VsIntended: map[string]map[string]int{}}
	base := filepath.Join(root, "audit-results")
	err := filepath.Walk(base, func(path string, info os.FileInfo, walkErr error) error {
		if walkErr != nil {
			if os.IsNotExist(walkErr) {
				return nil
			}
			return walkErr
		}
		if info.IsDir() || !strings.HasPrefix(filepath.Base(path), "gcl-trace-") || !strings.HasSuffix(path, ".json") {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		var trace struct {
			Skill           string         `json:"skill"`
			OperationIntent map[string]any `json:"operation_intent"`
			RouterDecision  struct {
				Chosen string `json:"chosen"`
			} `json:"router_decision"`
		}
		if err := json.Unmarshal(data, &trace); err != nil {
			return fmt.Errorf("decode %s: %w", path, err)
		}
		if trace.RouterDecision.Chosen == "" {
			return nil
		}
		intended := trace.Skill
		if value, ok := trace.OperationIntent["intended_skill"].(string); ok && value != "" {
			intended = value
		}
		if matrix.Top1VsIntended[intended] == nil {
			matrix.Top1VsIntended[intended] = map[string]int{}
		}
		matrix.Top1VsIntended[intended][trace.RouterDecision.Chosen]++
		return nil
	})
	if err != nil {
		return Matrix{}, err
	}
	return matrix, nil
}
