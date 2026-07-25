// Package coverage verifies TE-7 advanced/ stratification and
// Security-Sensitive markers for huaweicloud-*-ops skills.
//
// TE-7 requires deep AIOps / FinOps / SecOps / safety content to live under
// references/advanced/ so SKILL.md and references/*.md stay focused on
// agent-executable flows. This mirrors scripts/check_advanced_coverage.py,
// but discovers skills dynamically from --root instead of importing a
// hardcoded skill list (the hardcoded list is a B-class repo concern that the
// Go binary must not depend on — see Spec §2.2).
package coverage

import (
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

// advancedTopicPattern matches filenames that denote an advanced topic, e.g.
// aiops-patterns.md, cost-optimization.md, security-baseline.md.
var advancedTopicPattern = regexp.MustCompile(`(?:^|-)(?:aiops|cost|security|safety|observability|knowledge)(?:-|\.|$)`)

// securityMarkerPatterns detect Security-Sensitive markers (English or Chinese)
// so reviewers can audit which destructive operations require explicit
// operator confirmation.
var securityMarkerPatterns = []*regexp.Regexp{
	regexp.MustCompile(`Security-Sensitive`),
	regexp.MustCompile(`⚠`),
	regexp.MustCompile(`(?i)\b(?:warning|caution|danger)\b`),
	regexp.MustCompile(`(?:高危|危险|敏感|不可逆|慎用)`),
}

// ExemptAdvanced lists skills whose advanced depth is intentionally
// co-located with the runbook (meta-skill generator, etc.) and therefore are
// not required to have a references/advanced/ directory.
var ExemptAdvanced = map[string]bool{
	"huaweicloud-skill-generator": true,
}

// SkillReport is the per-skill outcome of a coverage check.
type SkillReport struct {
	Skill           string   `json:"skill"`
	AdvancedFiles   []string `json:"advanced_files"`
	AdvancedTopics  []string `json:"advanced_topics"`
	SecurityMarkers int      `json:"security_marker_count"`
	SecurityFiles   []string `json:"security_marker_files"`
	Errors          []string `json:"errors"`
	Warnings        []string `json:"warnings"`
	OK              bool     `json:"ok"`
}

// Report aggregates SkillReport values across a repository.
type Report struct {
	OK                 bool          `json:"ok"`
	SkillsChecked      int           `json:"skills_checked"`
	SkillsWithAdvanced int           `json:"skills_with_advanced"`
	Errors             []string      `json:"errors"`
	Warnings           []string      `json:"warnings"`
	Reports            []SkillReport `json:"reports"`
}

// DiscoverSkills returns sorted huaweicloud-*-ops directory names under root.
func DiscoverSkills(root string) ([]string, error) {
	entries, err := os.ReadDir(root)
	if err != nil {
		return nil, err
	}
	var skills []string
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		name := e.Name()
		if strings.HasPrefix(name, "huaweicloud-") && strings.HasSuffix(name, "-ops") {
			skills = append(skills, name)
		}
	}
	sort.Strings(skills)
	return skills, nil
}

// ValidateSkill checks one skill's advanced/ coverage and Security-Sensitive
// markers. warnOnly demotes the missing-advanced/ error to a warning.
func ValidateSkill(root, skill string, warnOnly bool) SkillReport {
	references := filepath.Join(root, skill, "references")
	advancedFiles := discoverAdvancedFiles(references)

	advancedTopics := map[string]bool{}
	for _, f := range advancedFiles {
		for _, m := range advancedTopicPattern.FindAllStringSubmatch(f, -1) {
			topic := strings.ToLower(strings.Trim(m[0], "-."))
			advancedTopics[topic] = true
		}
	}

	refFiles := collectReferenceFiles(references)
	var topics []string
	for t := range advancedTopics {
		topics = append(topics, t)
	}
	sort.Strings(topics)

	var secMarkerFiles []string
	secMarkers := 0
	for _, path := range refFiles {
		hits := countSecurityMarkers(path)
		if hits > 0 {
			secMarkers += hits
			secMarkerFiles = append(secMarkerFiles, filepath.Base(path)+"="+strconv.Itoa(hits))
		}
	}

	rep := SkillReport{
		Skill:           skill,
		AdvancedFiles:   relNames(references, advancedFiles),
		AdvancedTopics:  topics,
		SecurityMarkers: secMarkers,
		SecurityFiles:   secMarkerFiles,
	}

	if !ExemptAdvanced[skill] && len(advancedFiles) == 0 {
		msg := skill + ": missing references/advanced/*.md (TE-7 stratification)"
		if warnOnly {
			rep.Warnings = append(rep.Warnings, msg)
		} else {
			rep.Errors = append(rep.Errors, msg)
		}
	}
	if secMarkers == 0 {
		rep.Warnings = append(rep.Warnings, skill+": no Security-Sensitive markers in any references/*.md")
	}

	rep.OK = len(rep.Errors) == 0
	return rep
}

// ValidateAll checks every discovered skill under root.
func ValidateAll(root string, warnOnly bool) (Report, error) {
	skills, err := DiscoverSkills(root)
	if err != nil {
		return Report{}, err
	}
	rep := Report{SkillsChecked: len(skills)}
	for _, s := range skills {
		sr := ValidateSkill(root, s, warnOnly)
		rep.Reports = append(rep.Reports, sr)
		if len(sr.AdvancedFiles) > 0 {
			rep.SkillsWithAdvanced++
		}
		rep.Errors = append(rep.Errors, sr.Errors...)
		rep.Warnings = append(rep.Warnings, sr.Warnings...)
	}
	rep.OK = len(rep.Errors) == 0
	return rep, nil
}

func discoverAdvancedFiles(references string) []string {
	dir := filepath.Join(references, "advanced")
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}
	var out []string
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".md") {
			out = append(out, e.Name())
		}
	}
	sort.Strings(out)
	return out
}

func collectReferenceFiles(references string) []string {
	var out []string
	top, err := os.ReadDir(references)
	if err == nil {
		for _, e := range top {
			if !e.IsDir() && strings.HasSuffix(e.Name(), ".md") {
				out = append(out, filepath.Join(references, e.Name()))
			}
		}
	}
	adv, err := os.ReadDir(filepath.Join(references, "advanced"))
	if err == nil {
		for _, e := range adv {
			if !e.IsDir() && strings.HasSuffix(e.Name(), ".md") {
				out = append(out, filepath.Join(references, "advanced", e.Name()))
			}
		}
	}
	sort.Strings(out)
	return out
}

func countSecurityMarkers(path string) int {
	data, err := os.ReadFile(path)
	if err != nil {
		return 0
	}
	text := string(data)
	total := 0
	for _, p := range securityMarkerPatterns {
		total += len(p.FindAllString(text, -1))
	}
	return total
}

func relNames(references string, names []string) []string {
	out := make([]string, 0, len(names))
	for _, n := range names {
		out = append(out, filepath.Join("references", "advanced", n))
	}
	return out
}
