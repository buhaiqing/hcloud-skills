package registry

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	internalyaml "github.com/buhaiqing/hcloud-skills/hwcloud-skillcheck/internal/yaml"
)

type Entry struct {
	Skill               string
	Name                string
	Description         string
	Version             string
	CLIApplicability    string
	SideEffectClass     string
	RequiredPermissions []string
	Inputs              []string
	Outputs             []string
	Path                string
}

type Registry struct {
	entries map[string]Entry
}

func Boot(root string) (*Registry, error) {
	dirs, err := os.ReadDir(root)
	if err != nil {
		return nil, fmt.Errorf("read registry root: %w", err)
	}
	result := &Registry{entries: make(map[string]Entry)}
	for _, dir := range dirs {
		name := dir.Name()
		if !dir.IsDir() || !strings.HasPrefix(name, "huaweicloud-") || !strings.HasSuffix(name, "-ops") || name == "huaweicloud-skill-generator" {
			continue
		}
		path := filepath.Join(root, name, "SKILL.md")
		frontmatter, err := readFrontmatter(path)
		if err != nil {
			return nil, fmt.Errorf("index %s: %w", name, err)
		}
		metadata, _ := frontmatter["metadata"].(map[string]any)
		entry := Entry{
			Skill:            name,
			Name:             stringValue(frontmatter["name"]),
			Description:      stringValue(frontmatter["description"]),
			Version:          stringValue(metadata["version"]),
			CLIApplicability: stringValue(metadata["cli_applicability"]),
			SideEffectClass:  stringValue(frontmatter["side_effect_class_max"]),
			Path:             path,
		}
		result.entries[name] = entry
	}
	return result, nil
}

func (r *Registry) Lookup(skill string) (Entry, bool) {
	entry, ok := r.entries[skill]
	return entry, ok
}

func (r *Registry) Entries() []Entry {
	entries := make([]Entry, 0, len(r.entries))
	for _, entry := range r.entries {
		entries = append(entries, entry)
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].Skill < entries[j].Skill })
	return entries
}

func readFrontmatter(path string) (map[string]any, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	lines := make([]string, 0, 32)
	for scanner.Scan() {
		line := scanner.Text()
		lines = append(lines, line)
		if len(lines) > 1 && line == "---" {
			return internalyaml.ExtractFrontmatter([]byte(strings.Join(lines, "\n")))
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return nil, fmt.Errorf("missing closing frontmatter fence")
}

func stringValue(value any) string {
	switch typed := value.(type) {
	case string:
		return typed
	case fmt.Stringer:
		return typed.String()
	case int:
		return fmt.Sprint(typed)
	case float64:
		return fmt.Sprint(typed)
	default:
		return ""
	}
}
