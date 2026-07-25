// Package yaml provides YAML parsing helpers for hcloud-skills validation,
// ported from scripts/validate_skills_frontmatter.py and
// scripts/check_example_config.py. It uses gopkg.in/yaml.v3 for robust
// parsing and supplements it with anchor/placeholder scanning needed by the
// example-config check.
package yaml

import (
	"fmt"
	"regexp"
	"strings"

	"gopkg.in/yaml.v3"
)

var (
	frontmatterRe = regexp.MustCompile(`(?s)^---\s*\n(.*?)\n---`)
	yamlBlockRe   = regexp.MustCompile("(?s)```yaml\\s*\\n(.*?)\\n```")
	anchorDefRe   = regexp.MustCompile(`&([A-Za-z0-9_-]+)`)
	anchorRefRe   = regexp.MustCompile(`\*([A-Za-z0-9_-]+)`)
)

// ExtractFrontmatter extracts and parses the leading --- fenced YAML block
// from a SKILL.md document. Returns the parsed mapping or an error when the
// frontmatter is absent or not valid YAML.
func ExtractFrontmatter(content []byte) (map[string]any, error) {
	m := frontmatterRe.FindSubmatch(content)
	if m == nil {
		return nil, fmt.Errorf("missing YAML frontmatter")
	}
	return parseMapping(m[1])
}

// ExtractYAMLBlock extracts the content of the first ```yaml fenced block.
func ExtractYAMLBlock(content []byte) (string, error) {
	m := yamlBlockRe.FindSubmatch(content)
	if m == nil {
		return "", fmt.Errorf("missing ```yaml block")
	}
	return string(m[1]), nil
}

// DetectAnchors scans YAML source lines for anchor definitions (&name) and
// references (*name / <<: *name). It returns the set of defined anchors, the
// set of referenced anchors, and errors for any reference without a prior
// definition. Mirrors check_example_config.py's detect_anchors.
func DetectAnchors(lines []string) (defined map[string]bool, referenced map[string]bool, errors []string) {
	defined = map[string]bool{}
	referenced = map[string]bool{}
	for i, line := range lines {
		// Skip JSON-ish / comment lines so anchor scans don't trip on
		// unrelated literals.
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}
		for _, name := range anchorDefRe.FindAllStringSubmatch(line, -1) {
			defined[name[1]] = true
		}
		for _, name := range anchorRefRe.FindAllStringSubmatch(line, -1) {
			referenced[name[1]] = true
			if !defined[name[1]] {
				errors = append(errors, fmt.Sprintf("line %d: anchor %q referenced before defined", i+1, name[1]))
			}
		}
	}
	return defined, referenced, errors
}

func parseMapping(raw []byte) (map[string]any, error) {
	var out map[string]any
	if err := yaml.Unmarshal(raw, &out); err != nil {
		return nil, err
	}
	if out == nil {
		return nil, fmt.Errorf("frontmatter is not a mapping")
	}
	return out, nil
}
