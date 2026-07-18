package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"unicode/utf8"

	"github.com/buhaiqing/hcloud-skills/skillcheck/internal/coverage"
	"github.com/buhaiqing/hcloud-skills/skillcheck/internal/yaml"
)

// runCheck dispatches the `skillcheck check` subcommands.
func runCheck(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("check: missing subcommand (use 'example-config'|'markdown-links'|'references-links'|'audit-results')")
	}
	switch args[0] {
	case "example-config":
		return runCheckExampleConfig(args[1:])
	case "markdown-links":
		return runCheckMarkdownLinks(args[1:])
	case "references-links":
		return runCheckReferencesLinks(args[1:])
	case "advanced-coverage":
		return runCheckAdvancedCoverage(args[1:])
	case "audit-results":
		return runCheckAuditResults(args[1:])
	case "-h", "--help", "help":
		fmt.Fprintln(os.Stdout, "skillcheck check <example-config|markdown-links|references-links|audit-results> --root <dir>")
		return nil
	default:
		return fmt.Errorf("check: unknown subcommand %q", args[0])
	}
}

// ---------------------------------------------------------------------------
// check example-config
// ---------------------------------------------------------------------------

var (
	examplePlaceholderRe   = regexp.MustCompile(`\{\{\s*(env|user|output)\.[^{}\s]+\}\}`)
	exampleBarePlaceholder = regexp.MustCompile(`\{[a-zA-Z_][a-zA-Z0-9_.-]*\}`)
	exampleSecretLiteral   = regexp.MustCompile(`(?i)(?:secret\s*[:=]\s*['"][^'"\s]+|sk\s*[:=]\s*['"]?[A-Za-z0-9+/]{16,})`)
)

// checkExampleConfigResult holds per-file outcomes.
type checkExampleConfigResult struct {
	file       string
	ok         bool
	errors     []string
	warnings   []string
	anchors    int
	repeatKeys int
}

// runCheckExampleConfig validates every huaweicloud-*-ops/*/assets/example-config.yaml.
// It mirrors scripts/check_example_config.py: no plaintext secrets, valid
// {{placeholder}} usage, basic YAML structure, and YAML anchors referenced only
// after definition.
func runCheckExampleConfig(args []string) error {
	fs := newFlagSet("skillcheck check example-config")
	root := fs.String("root", ".", "skill repository root")
	warnOnly := fs.Bool("warn-only", false, "treat failures as warnings (exit 0)")
	jsonOut := fs.Bool("json", false, "emit JSON report")
	if err := fs.Parse(args); err != nil {
		return err
	}
	rootDir, err := filepath.Abs(*root)
	if err != nil {
		return err
	}
	skills, err := discoverSkillDirs(rootDir)
	if err != nil {
		return fmt.Errorf("example-config: %w", err)
	}

	results := make([]checkExampleConfigResult, 0, len(skills))
	for _, skill := range skills {
		results = append(results, validateExampleConfig(rootDir, skill))
	}

	hasErrors := false
	for _, r := range results {
		if !r.ok {
			hasErrors = true
		}
	}

	if *jsonOut {
		printExampleConfigJSON(results)
	} else {
		printExampleConfigHuman(results)
	}

	if hasErrors && !*warnOnly {
		return fmt.Errorf("example-config check failed: %d file(s) with errors", countExampleConfigErrors(results))
	}
	return nil
}

// discoverSkillDirs returns sorted skill directory names (huaweicloud-*-ops)
// directly under root.
func discoverSkillDirs(root string) ([]string, error) {
	entries, err := os.ReadDir(root)
	if err != nil {
		return nil, err
	}
	var skills []string
	for _, e := range entries {
		if e.IsDir() && strings.HasPrefix(e.Name(), "huaweicloud-") && strings.HasSuffix(e.Name(), "-ops") {
			skills = append(skills, e.Name())
		}
	}
	sort.Strings(skills)
	return skills, nil
}

// validateExampleConfig validates one skill's example-config.yaml.
func validateExampleConfig(root, skill string) checkExampleConfigResult {
	rel := filepath.Join(skill, "assets", "example-config.yaml")
	path := filepath.Join(root, rel)
	res := checkExampleConfigResult{file: rel}

	data, err := os.ReadFile(path)
	if err != nil {
		res.ok = false
		res.errors = append(res.errors, fmt.Sprintf("%s: missing example-config.yaml", rel))
		return res
	}

	text := string(data)
	// Extract the YAML block: prefer ```yaml fence, else treat whole file as raw.
	block := text
	if m, e := yaml.ExtractYAMLBlock(data); e == nil {
		block = m
	}

	// No plaintext secret literals.
	res.errors = append(res.errors, checkExampleSecrets(text)...)
	// Placeholders must be well-formed {{env.*}} etc., not bare {x}.
	res.errors = append(res.errors, checkExamplePlaceholders(text)...)
	// Basic YAML structure: every non-blank line is key:value or nested.
	res.errors = append(res.errors, checkExampleYAMLBasic(block, rel)...)
	// Anchor references must follow definitions.
	defined, _, anchorErrs := yaml.DetectAnchors(strings.Split(block, "\n"))
	res.errors = append(res.errors, anchorErrs...)
	res.anchors = len(defined)

	// TE-5 soft warning: a key repeated 3+ times without any anchors.
	repeats := detectRepeatedKeys(block)
	res.repeatKeys = len(repeats)
	if len(repeats) > 0 && len(defined) == 0 {
		res.warnings = append(res.warnings,
			fmt.Sprintf("%s: %d key(s) repeated 3+ times without YAML anchors (%s)", rel, len(repeats), strings.Join(take(repeats, 5), ", ")))
	}

	res.ok = len(res.errors) == 0
	return res
}

func checkExampleSecrets(text string) []string {
	var errs []string
	for i, line := range strings.Split(text, "\n") {
		if strings.HasPrefix(strings.TrimSpace(line), "#") {
			continue
		}
		if exampleSecretLiteral.MatchString(line) {
			errs = append(errs, fmt.Sprintf("line %d: looks like plaintext secret — use <masked> or {{env.*}}", i+1))
		}
	}
	return errs
}

func checkExamplePlaceholders(text string) []string {
	var errs []string
	for i, line := range strings.Split(text, "\n") {
		if strings.HasPrefix(strings.TrimSpace(line), "#") {
			continue
		}
		for _, m := range exampleBarePlaceholder.FindAllString(line, -1) {
			idx := strings.Index(line, m)
			// Skip {{...}} (env/user/output) placeholders: the brace must not
			// be immediately preceded or followed by another brace.
			prev := runeAt(line, idx-1)
			next := runeAt(line, idx+len(m))
			if prev == '{' || next == '}' {
				continue
			}
			if examplePlaceholderRe.MatchString(line) {
				continue
			}
			errs = append(errs, fmt.Sprintf("line %d: bare placeholder in %q", i+1, truncate(line, 80)))
			break
		}
	}
	return errs
}

func runeAt(s string, i int) rune {
	if i < 0 || i >= len(s) {
		return 0
	}
	r, _ := utf8.DecodeRuneInString(s[i:])
	return r
}

func checkExampleYAMLBasic(block, rel string) []string {
	var errs []string
	lineNo := 0
	for _, raw := range strings.Split(block, "\n") {
		lineNo++
		stripped := strings.TrimSpace(raw)
		// Comments, blanks, separators and JSON braces tolerated.
		if stripped == "" || strings.HasPrefix(stripped, "#") || stripped == "---" {
			continue
		}
		if stripped == "{" || stripped == "}" || stripped == "[" || stripped == "]" {
			continue
		}
		// A top-level (non-indented) line with no ':' is malformed.
		if !strings.Contains(stripped, ":") && !strings.HasPrefix(raw, " ") {
			errs = append(errs, fmt.Sprintf("%s:%d: expected key:value, got %q", rel, lineNo, stripped))
		}
	}
	return errs
}

// detectRepeatedKeys counts keys (indent <= 2) that appear 3+ times in block.
func detectRepeatedKeys(block string) []string {
	counts := map[string]int{}
	keyRe := regexp.MustCompile(`^(\s+)([A-Za-z_][\w-]*):\s`)
	for _, raw := range strings.Split(block, "\n") {
		stripped := strings.TrimSpace(raw)
		if stripped == "" || strings.HasPrefix(stripped, "#") {
			continue
		}
		m := keyRe.FindStringSubmatch(raw)
		if m == nil {
			continue
		}
		indent := len(raw) - len(strings.TrimLeft(raw, " "))
		if indent > 2 {
			continue
		}
		counts[m[2]]++
	}
	var out []string
	for k, c := range counts {
		if c >= 3 {
			out = append(out, k)
		}
	}
	sort.Strings(out)
	return out
}

func countExampleConfigErrors(results []checkExampleConfigResult) int {
	n := 0
	for _, r := range results {
		n += len(r.errors)
	}
	return n
}

func printExampleConfigHuman(results []checkExampleConfigResult) {
	for _, r := range results {
		status := "OK"
		if !r.ok {
			status = "FAIL"
		}
		fmt.Printf("%s: %s  anchors=%d  repeats=%d\n", status, r.file, r.anchors, r.repeatKeys)
		for _, e := range r.errors {
			fmt.Printf("  - %s\n", e)
		}
		for _, w := range r.warnings {
			fmt.Printf("  ~ %s\n", w)
		}
	}
	fmt.Printf("\nChecked %d example-config.yaml file(s); errors=%d, warnings=%d\n",
		len(results), countExampleConfigErrors(results), countWarnings(results))
}

func countWarnings(results []checkExampleConfigResult) int {
	n := 0
	for _, r := range results {
		n += len(r.warnings)
	}
	return n
}

func printExampleConfigJSON(results []checkExampleConfigResult) {
	fmt.Println("{")
	fmt.Printf("  \"files_checked\": %d,\n", len(results))
	fmt.Println("  \"reports\": [")
	for i, r := range results {
		fmt.Printf("    {\"file\": %q, \"ok\": %v, \"anchors_defined\": %d, \"repeat_keys\": %d, \"errors\": %d, \"warnings\": %d}",
			r.file, r.ok, r.anchors, r.repeatKeys, len(r.errors), len(r.warnings))
		if i < len(results)-1 {
			fmt.Println(",")
		} else {
			fmt.Println()
		}
	}
	fmt.Println("  ]")
	fmt.Println("}")
}

// ---------------------------------------------------------------------------
// check markdown-links
// ---------------------------------------------------------------------------

var (
	mdLinkRe     = regexp.MustCompile(`\[[^\]]+\]\(([^)\s]+)(?:\s+"[^"]*")?\)`)
	mdBacktickRe = regexp.MustCompile("`([^`]+)`")
)

var mdIgnoredDirParts = map[string]bool{
	".git": true, ".github": true, ".omc": true, ".omo": true,
	".codebuddy": true, ".claude": true, ".agents": true, "audit-results": true,
}

var mdPathPrefixes = []string{
	"AGENTS.md", "CLAUDE.md", "README.md", "README_CN.md", "LICENSE",
	"docs/", "scripts/", "huaweicloud-", ".github/",
}

// runCheckMarkdownLinks validates local markdown links and explicit repo path
// references across always-loaded docs (AGENTS.md, README*.md, docs/*.md).
// Mirrors scripts/check_markdown_links.py.
func runCheckMarkdownLinks(args []string) error {
	fs := newFlagSet("skillcheck check markdown-links")
	root := fs.String("root", ".", "skill repository root")
	if err := fs.Parse(args); err != nil {
		return err
	}
	rootDir, err := filepath.Abs(*root)
	if err != nil {
		return err
	}

	var findings []string
	for _, path := range iterMarkdownFiles(rootDir) {
		findings = append(findings, checkMarkdownFile(rootDir, path)...)
	}

	if len(findings) > 0 {
		for _, f := range findings {
			fmt.Fprintln(os.Stderr, f)
		}
		fmt.Fprintf(os.Stderr, "ERROR: %d broken Markdown path reference(s)\n", len(findings))
		return fmt.Errorf("markdown-links check failed")
	}
	fmt.Println("OK: Markdown local links and repository path references validated")
	return nil
}

func iterMarkdownFiles(root string) []string {
	candidates := []string{
		filepath.Join(root, "AGENTS.md"),
		filepath.Join(root, "CLAUDE.md"),
		filepath.Join(root, "README.md"),
		filepath.Join(root, "README_CN.md"),
	}
	docsDir := filepath.Join(root, "docs")
	if entries, err := os.ReadDir(docsDir); err == nil {
		names := make([]string, 0, len(entries))
		for _, e := range entries {
			if !e.IsDir() && strings.HasSuffix(e.Name(), ".md") {
				names = append(names, e.Name())
			}
		}
		sort.Strings(names)
		for _, n := range names {
			candidates = append(candidates, filepath.Join(docsDir, n))
		}
	}
	var files []string
	for _, c := range candidates {
		info, err := os.Stat(c)
		if err != nil || info.IsDir() {
			continue
		}
		rel := relParts(root, c)
		if hasIgnoredPart(rel) {
			continue
		}
		files = append(files, c)
	}
	sort.Strings(files)
	return files
}

func checkMarkdownFile(root, path string) []string {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	var findings []string
	for i, line := range strings.Split(string(data), "\n") {
		for _, m := range mdLinkRe.FindAllStringSubmatch(line, -1) {
			// Skip image links (![alt](url)) — same as Python's (?<!!).
			if strings.HasPrefix(m[0], "!") {
				continue
			}
			if target := normalizeMDTarget(m[1]); target != "" {
				if !mdTargetExists(root, path, target) {
					findings = append(findings, fmt.Sprintf("%s:%d: missing markdown link target: %s",
						relDisplay(root, path), i+1, target))
				}
			}
		}
		for _, m := range mdBacktickRe.FindAllStringSubmatch(line, -1) {
			target := strings.TrimSpace(m[1])
			if nt := normalizeMDTarget(target); nt != "" {
				if looksLikeRepoPath(target) && !mdTargetExists(root, path, nt) {
					findings = append(findings, fmt.Sprintf("%s:%d: missing backtick path target: %s",
						relDisplay(root, path), i+1, nt))
				}
			}
		}
	}
	return findings
}

func normalizeMDTarget(raw string) string {
	target := strings.TrimSpace(raw)
	if target == "" || strings.HasPrefix(target, "http://") || strings.HasPrefix(target, "https://") ||
		strings.HasPrefix(target, "mailto:") || strings.HasPrefix(target, "#") ||
		strings.HasPrefix(target, "{{") || strings.HasPrefix(target, "<") {
		return ""
	}
	if i := strings.Index(target, "#"); i >= 0 {
		target = target[:i]
	}
	if i := strings.Index(target, "?"); i >= 0 {
		target = target[:i]
	}
	return target
}

func looksLikeRepoPath(text string) bool {
	if strings.ContainsAny(text, " \t") {
		return false
	}
	if strings.HasPrefix(text, "http://") || strings.HasPrefix(text, "https://") ||
		strings.HasPrefix(text, "mailto:") || strings.HasPrefix(text, "#") ||
		strings.HasPrefix(text, "{{") || strings.HasPrefix(text, "<") {
		return false
	}
	if strings.ContainsAny(text, "<>*|=") || strings.Contains(text, "--") {
		return false
	}
	if strings.HasPrefix(text, "huaweicloud-") && !strings.Contains(text, "/") {
		return false
	}
	for _, p := range mdPathPrefixes {
		if strings.HasPrefix(text, p) {
			return true
		}
	}
	return false
}

func mdTargetExists(root, source, target string) bool {
	candidate := target
	if filepath.IsAbs(candidate) {
		return fileExists(candidate)
	}
	var resolve string
	if hasPrefixAny(target, mdPathPrefixes) {
		resolve = filepath.Join(root, candidate)
	} else {
		resolve = filepath.Join(filepath.Dir(source), candidate)
	}
	for _, part := range strings.Split(resolve, string(os.PathSeparator)) {
		if part == "*" || part == "..." {
			return true
		}
	}
	return fileExists(resolve)
}

// ---------------------------------------------------------------------------
// check references-links
// ---------------------------------------------------------------------------

var refLinkRe = regexp.MustCompile(`\[[^\]]+\]\(([^)\s]+)(?:\s+"[^"]*")?\)`)
var refHeadingRe = regexp.MustCompile(`^(#{1,6})\s+(.+?)\s*#*\s*$`)

// runCheckAdvancedCoverage validates TE-7 advanced/ stratification and
// Security-Sensitive markers across every huaweicloud-*-ops skill. It
// discovers skills dynamically from --root (no hardcoded skill list), so the
// binary stays reusable on external repositories. Mirrors
// scripts/check_advanced_coverage.py.
func runCheckAdvancedCoverage(args []string) error {
	fs := newFlagSet("skillcheck check advanced-coverage")
	root := fs.String("root", ".", "skill repository root")
	jsonOut := fs.Bool("json", false, "emit JSON report")
	warnOnly := fs.Bool("warn-only", false, "demote missing advanced/ to warnings (gradual rollout)")
	if err := fs.Parse(args); err != nil {
		return err
	}
	rootDir, err := filepath.Abs(*root)
	if err != nil {
		return err
	}

	report, err := coverage.ValidateAll(rootDir, *warnOnly)
	if err != nil {
		return err
	}

	if *jsonOut {
		printAdvancedJSON(rootDir, report)
	} else {
		for _, item := range report.Reports {
			status := "OK"
			if !item.OK {
				status = "FAIL"
			}
			adv := len(item.AdvancedFiles)
			topics := strings.Join(item.AdvancedTopics, ",")
			if topics == "" {
				topics = "-"
			}
			fmt.Printf("%s: %-35s  advanced=%d  topics=%-20s  sec_markers=%d\n",
				status, item.Skill, adv, topics, item.SecurityMarkers)
			for _, e := range item.Errors {
				fmt.Printf("  - %s\n", e)
			}
			for _, w := range item.Warnings {
				fmt.Printf("  ~ %s\n", w)
			}
		}
		fmt.Printf("\nChecked %d skills; with_advanced=%d; errors=%d; warnings=%d\n",
			report.SkillsChecked, report.SkillsWithAdvanced, len(report.Errors), len(report.Warnings))
	}

	if !report.OK {
		return fmt.Errorf("advanced-coverage check failed: %d error(s)", len(report.Errors))
	}
	return nil
}

func printAdvancedJSON(root string, report coverage.Report) {
	fmt.Println("{")
	fmt.Printf("  \"ok\": %v,\n", report.OK)
	fmt.Printf("  \"skills_checked\": %d,\n", report.SkillsChecked)
	fmt.Printf("  \"skills_with_advanced\": %d,\n", report.SkillsWithAdvanced)
	fmt.Printf("  \"errors\": %d,\n", len(report.Errors))
	fmt.Printf("  \"warnings\": %d,\n", len(report.Warnings))
	fmt.Println("  \"reports\": [")
	for i, item := range report.Reports {
		fmt.Printf("    {\"skill\": %q, \"advanced_files\": %d, \"security_marker_count\": %d, \"ok\": %v}",
			item.Skill, len(item.AdvancedFiles), item.SecurityMarkers, item.OK)
		if i < len(report.Reports)-1 {
			fmt.Println(",")
		} else {
			fmt.Println()
		}
	}
	fmt.Println("  ]")
	fmt.Println("}")
}

// runCheckReferencesLinks validates deep-link (anchor) health of every
// huaweicloud-*-ops/references/*.md file. Mirrors scripts/check_references_link_health.py.
func runCheckReferencesLinks(args []string) error {
	fs := newFlagSet("skillcheck check references-links")
	root := fs.String("root", ".", "skill repository root")
	jsonOut := fs.Bool("json", false, "emit JSON report")
	warnOnly := fs.Bool("warnings-only", false, "treat warnings as non-fatal")
	if err := fs.Parse(args); err != nil {
		return err
	}
	rootDir, err := filepath.Abs(*root)
	if err != nil {
		return err
	}

	files := iterReferencesFiles(rootDir)
	cache := map[string][]string{}
	var findings []refFinding
	for _, path := range files {
		findings = append(findings, checkReferencesFile(rootDir, path, cache)...)
	}

	errors := 0
	warnings := 0
	for _, f := range findings {
		if f.severity == "error" {
			errors++
		} else {
			warnings++
		}
	}

	if *jsonOut {
		printReferencesJSON(rootDir, files, findings, errors, warnings)
	} else {
		fmt.Printf("references/ link health: scanned %d files, errors=%d, warnings=%d\n", len(files), errors, warnings)
		for _, f := range findings {
			fmt.Printf("  %-7s %s:%d: %s -> %s\n", strings.ToUpper(f.severity), relDisplay(rootDir, f.file), f.line, f.reason, f.target)
		}
	}

	if errors > 0 {
		return fmt.Errorf("references-links check failed: %d error(s)", errors)
	}
	if !*warnOnly && warnings > 0 {
		return fmt.Errorf("references-links check finished with %d warning(s)", warnings)
	}
	return nil
}

type refFinding struct {
	file     string
	line     int
	target   string
	severity string
	reason   string
}

func iterReferencesFiles(root string) []string {
	var files []string
	base := filepath.Join(root, "huaweicloud-*-ops", "references")
	matches, _ := filepath.Glob(base + "/*.md")
	for _, m := range matches {
		rel := relParts(root, m)
		if hasIgnoredPart(rel) {
			continue
		}
		files = append(files, m)
	}
	sort.Strings(files)
	return files
}

func inventoryHeadings(path string) []string {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	var anchors []string
	for _, m := range refHeadingRe.FindAllStringSubmatch(string(data), -1) {
		anchors = append(anchors, slugifyAnchor(m[2]))
	}
	return anchors
}

// slugifyAnchor implements GitHub's anchor slugifier: lowercase, whitespace ->
// '-', drop non [\w-]. Leading digits are intentionally kept (GitHub keeps
// them), matching check_references_link_health.py.
func slugifyAnchor(text string) string {
	text = strings.ReplaceAll(text, "`", "")
	text = strings.ToLower(text)
	var b strings.Builder
	for _, r := range text {
		if r == ' ' || r == '\t' || r == '\n' {
			b.WriteByte('-')
			continue
		}
		if r == '_' || r == '-' || (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
		}
	}
	return b.String()
}

func checkReferencesFile(root, path string, cache map[string][]string) []refFinding {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	inventory := funcCache(cache, path)
	var findings []refFinding
	lines := strings.Split(string(data), "\n")
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "```") || strings.HasPrefix(trimmed, "~~~") {
			continue
		}
		for _, m := range refLinkRe.FindAllStringSubmatch(line, -1) {
			// Skip image links (![alt](url)) — same as Python's (?<!!).
			if strings.HasPrefix(m[0], "!") {
				continue
			}
			raw := strings.TrimSpace(m[1])
			if raw == "" {
				continue
			}
			if strings.HasPrefix(raw, "http://") || strings.HasPrefix(raw, "https://") ||
				strings.HasPrefix(raw, "mailto:") || strings.HasPrefix(raw, "<") ||
				strings.HasPrefix(raw, "{{") {
				continue
			}
			linkPath, anchor := splitRefTarget(raw)
			if linkPath == "" {
				if anchor != "" && !containsStrInSlice(inventory, anchor) {
					findings = append(findings, refFinding{path, i + 1, "#" + anchor, "error", "missing in-page anchor"})
				}
				continue
			}
			resolved := resolveRefRelative(root, path, linkPath)
			if resolved == "" || !fileExists(resolved) {
				if strings.HasPrefix(linkPath, "huaweicloud-") && strings.Contains(linkPath, "/") {
					skillPart := strings.SplitN(linkPath, "/", 2)[0]
					if !dirExists(filepath.Join(root, skillPart)) {
						findings = append(findings, refFinding{path, i + 1, raw, "error", "missing cross-skill reference: " + skillPart})
						continue
					}
				}
				if suggestion := siblingMDSuggestion(filepath.Dir(path), linkPath); suggestion != "" {
					findings = append(findings, refFinding{path, i + 1, raw, "warning", "bare sibling link '" + linkPath + "' should reference '" + suggestion + "'"})
					continue
				}
				findings = append(findings, refFinding{path, i + 1, raw, "error", "missing link target: " + linkPath})
				continue
			}
			if anchor == "" || !isRegularFile(resolved) {
				continue
			}
			targetInv := funcCache(cache, resolved)
			if !containsStrInSlice(targetInv, anchor) {
				findings = append(findings, refFinding{path, i + 1, raw, "error", fmt.Sprintf("missing anchor %q in %s", anchor, filepath.Base(resolved))})
			}
		}
	}
	return findings
}

func funcCache(cache map[string][]string, path string) []string {
	abs, err := filepath.Abs(path)
	if err != nil {
		abs = path
	}
	if v, ok := cache[abs]; ok {
		return v
	}
	v := inventoryHeadings(path)
	cache[abs] = v
	return v
}

func splitRefTarget(raw string) (string, string) {
	if i := strings.Index(raw, "#"); i >= 0 {
		return raw[:i], raw[i+1:]
	}
	return raw, ""
}

func resolveRefRelative(root, source, linkPath string) string {
	if linkPath == "" {
		return ""
	}
	candidate := linkPath
	if filepath.IsAbs(candidate) {
		return candidate
	}
	base := source
	if fi, err := os.Stat(source); err == nil && fi.IsDir() {
		base = source
	} else {
		base = filepath.Dir(source)
	}
	return filepath.Clean(filepath.Join(base, candidate))
}

func siblingMDSuggestion(refDir, linkPath string) string {
	if strings.ContainsAny(linkPath, "/\\") {
		return ""
	}
	candidate := filepath.Join(refDir, linkPath+".md")
	if fileExists(candidate) {
		return linkPath + ".md"
	}
	return ""
}

// --- shared path helpers ---

func relParts(root, path string) []string {
	rel, err := filepath.Rel(root, path)
	if err != nil {
		return strings.Split(path, string(os.PathSeparator))
	}
	return strings.Split(rel, string(os.PathSeparator))
}

func hasIgnoredPart(parts []string) bool {
	for _, p := range parts {
		if mdIgnoredDirParts[p] {
			return true
		}
	}
	return false
}

func relDisplay(root, path string) string {
	rel, err := filepath.Rel(root, path)
	if err != nil {
		return path
	}
	return rel
}

func fileExists(p string) bool {
	info, err := os.Stat(p)
	return err == nil && !info.IsDir()
}

func dirExists(p string) bool {
	info, err := os.Stat(p)
	return err == nil && info.IsDir()
}

func isRegularFile(p string) bool {
	info, err := os.Stat(p)
	return err == nil && !info.IsDir()
}

func hasPrefixAny(s string, prefixes []string) bool {
	for _, p := range prefixes {
		if strings.HasPrefix(s, p) {
			return true
		}
	}
	return false
}

func take(s []string, n int) []string {
	if len(s) <= n {
		return s
	}
	return s[:n]
}

func containsStrInSlice(slice []string, s string) bool {
	for _, item := range slice {
		if item == s {
			return true
		}
	}
	return false
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n]
}

func printReferencesJSON(root string, files []string, findings []refFinding, errors, warnings int) {
	fmt.Println("{")
	fmt.Printf("  \"ok\": %v,\n", errors == 0)
	fmt.Printf("  \"files_scanned\": %d,\n", len(files))
	fmt.Printf("  \"errors\": %d,\n", errors)
	fmt.Printf("  \"warnings\": %d,\n", warnings)
	fmt.Println("  \"findings\": [")
	for i, f := range findings {
		fmt.Printf("    {\"file\": %q, \"line\": %d, \"target\": %q, \"severity\": %q, \"reason\": %q}",
			relDisplay(root, f.file), f.line, f.target, f.severity, f.reason)
		if i < len(findings)-1 {
			fmt.Println(",")
		} else {
			fmt.Println()
		}
	}
	fmt.Println("  ]")
	fmt.Println("}")
}

// ---------------------------------------------------------------------------
// check audit-results (L2-C)
// ---------------------------------------------------------------------------

// gitignoreRequiredPatterns are the required patterns in .gitignore for audit-results.
var gitignoreRequiredPatterns = []string{
	`^audit-results/?\s*$`,
	`^\*\*/audit-results/?\s*$`,
	`^gcl-trace-\*\.json\s*$`,
	`^\*\*/gcl-trace-\*\.json\s*$`,
	`^gcl-quality-summary-\*\.json\s*$`,
	`^\*\*/gcl-quality-summary-\*\.json\s*$`,
	`^gcl-alarm-plan-\*\.json\s*$`,
	`^\*\*/gcl-alarm-plan-\*\.json\s*$`,
}

// gclDocRequiredFragments are required documentation fragments in docs/gcl-spec.md.
var gclDocRequiredFragments = []string{"audit-results/", "GCL", "gitignore"}

func runCheckAuditResults(args []string) error {
	fs := newFlagSet("skillcheck check audit-results")
	root := fs.String("root", ".", "skill repository root")
	jsonOut := fs.Bool("json", false, "emit JSON report")
	if err := fs.Parse(args); err != nil {
		return err
	}
	rootDir, err := filepath.Abs(*root)
	if err != nil {
		return err
	}

	errors := checkAuditResultsAll(rootDir)

	if *jsonOut {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		enc.Encode(map[string]any{"ok": len(errors) == 0, "errors": errors})
	} else {
		for _, e := range errors {
			fmt.Printf("  FAIL: %s\n", e)
		}
		if len(errors) == 0 {
			fmt.Println("[audit-results guard] OK")
		} else {
			fmt.Printf("[audit-results guard] FAIL: %d issue(s)\n", len(errors))
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("audit-results guard: %d issue(s)", len(errors))
	}
	return nil
}

func checkAuditResultsAll(root string) []string {
	var allErrors []string

	gitErrors := checkGitignore(root)
	for _, e := range gitErrors {
		allErrors = append(allErrors, "gitignore: "+e)
	}

	dirErrors := checkAuditDirMode(root)
	for _, e := range dirErrors {
		allErrors = append(allErrors, "directory: "+e)
	}

	trackedErrors := checkAuditTrackedFiles(root)
	for _, e := range trackedErrors {
		allErrors = append(allErrors, "tracked_files: "+e)
	}

	docErrors := checkGCLSpecDocs(root)
	for _, e := range docErrors {
		allErrors = append(allErrors, "documents: "+e)
	}

	return allErrors
}

func checkGitignore(root string) []string {
	var errors []string
	path := filepath.Join(root, ".gitignore")
	if !fileExists(path) {
		return []string{".gitignore missing"}
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return []string{fmt.Sprintf(".gitignore: read error: %v", err)}
	}
	lines := strings.Split(string(data), "\n")
	for _, pattern := range gitignoreRequiredPatterns {
		found := false
		re := regexp.MustCompile(pattern)
		for _, line := range lines {
			if re.MatchString(strings.TrimSpace(line)) {
				found = true
				break
			}
		}
		if !found {
			errors = append(errors, fmt.Sprintf("missing pattern: %s", pattern))
		}
	}
	return errors
}

func checkAuditDirMode(root string) []string {
	auditDir := filepath.Join(root, "audit-results")
	if !dirExists(auditDir) {
		return nil
	}
	info, err := os.Stat(auditDir)
	if err != nil {
		return nil
	}
	mode := info.Mode().Perm()
	if mode&0o077 != 0 {
		return []string{fmt.Sprintf("%s: mode %s too permissive; GCL traces should be owner-only (chmod 700)", auditDir, mode.String())}
	}
	return nil
}

func checkAuditTrackedFiles(root string) []string {
	cmd := exec.Command("git", "ls-files", "audit-results/")
	cmd.Dir = root
	var out bytes.Buffer
	cmd.Stdout = &out
	if err := cmd.Run(); err != nil {
		return nil
	}
	tracked := []string{}
	for _, line := range strings.Split(strings.TrimSpace(out.String()), "\n") {
		line = strings.TrimSpace(line)
		if line != "" {
			tracked = append(tracked, line)
		}
	}
	if len(tracked) > 0 {
		msg := fmt.Sprintf("audit-results/ contains %d tracked file(s); remove from git history", len(tracked))
		if len(tracked) <= 3 {
			msg += fmt.Sprintf(" (%v)", tracked)
		}
		return []string{msg}
	}
	return nil
}

func checkGCLSpecDocs(root string) []string {
	docPath := filepath.Join(root, "docs", "gcl-spec.md")
	if !fileExists(docPath) {
		return []string{"docs/gcl-spec.md: missing — audit persistence contract undocumented"}
	}
	text, err := os.ReadFile(docPath)
	if err != nil {
		return []string{fmt.Sprintf("docs/gcl-spec.md: read error: %v", err)}
	}
	lower := strings.ToLower(string(text))
	for _, fragment := range gclDocRequiredFragments {
		if !strings.Contains(lower, strings.ToLower(fragment)) {
			return []string{fmt.Sprintf("docs/gcl-spec.md: missing fragment %q", fragment)}
		}
	}
	return nil
}
