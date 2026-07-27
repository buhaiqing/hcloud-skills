package l4

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// TestGCLHighRiskVerbsInSync asserts that internal/gcl/runner.go
// mirrors internal/l4.HighRiskVerbs. The gcl package cannot import
// l4 (import cycle: l4 → gcl), so this test lives in l4 and reads
// the gcl source file at test time.
//
// The two regex tables MUST stay byte-identical. If a developer adds
// a verb to one and not the other, the gcl Run() will silently miss
// destructive commands and the trust boundary will leak.
func TestGCLHighRiskVerbsInSync(t *testing.T) {
	gcl := extractGCLRegex(t, "var gclHighRiskVerbs")
	l4v := extractAlternation(HighRiskVerbs)
	if gcl != l4v {
		t.Errorf("gclHighRiskVerbs and l4.HighRiskVerbs are out of sync.\n"+
			"  gcl: %q\n"+
			"  l4:  %q\n"+
			"Update both internal/gcl/runner.go (gclHighRiskVerbs) and "+
			"internal/l4/rbac.go (HighRiskVerbs) in the same commit.",
			gcl, l4v)
	}
}

// TestGCLHighRiskCommandsInSync mirrors TestGCLHighRiskVerbsInSync
// for the force/delete/destroy/purge flag table.
func TestGCLHighRiskCommandsInSync(t *testing.T) {
	gcl := extractGCLRegex(t, "var gclHighRiskCommands")
	l4c := extractAlternation(HighRiskCommands)
	if gcl != l4c {
		t.Errorf("gclHighRiskCommands and l4.HighRiskCommands are out of sync.\n"+
			"  gcl: %q\n"+
			"  l4:  %q\n"+
			"Update both internal/gcl/runner.go (gclHighRiskCommands) and "+
			"internal/l4/rbac.go (HighRiskCommands) in the same commit.",
			gcl, l4c)
	}
}

// extractGCLRegex reads internal/gcl/runner.go, locates the var block
// named by `header`, and returns the alternation source of the first
// `regexp.MustCompile(`...`)` call inside that block. The convention
// in gcl/runner.go is one regex per var; if the file ever needs more,
// this test should be extended to compare the whole slice (sorted).
func extractGCLRegex(t *testing.T, header string) string {
	t.Helper()
	path := gclRunnerPath(t)
	block := extractVarBlock(t, path, header)
	// Backticks inside a raw string literal are not escaped. Pattern
	// is the source form `regexp.MustCompile(`X`)`; we capture X.
	re := regexp.MustCompile("regexp\\.MustCompile\\(`([^`]*)`\\)")
	matches := re.FindAllStringSubmatch(block, -1)
	if len(matches) == 0 {
		t.Fatalf("no regexp.MustCompile in %q; block:\n%s", header, block)
		return ""
	}
	first := matches[0][1]
	open := strings.Index(first, "(")
	if open < 0 {
		t.Fatalf("no ( in regex %q from %q", first, header)
		return ""
	}
	depth := 0
	for i := open; i < len(first); i++ {
		switch first[i] {
		case '(':
			depth++
		case ')':
			depth--
			if depth == 0 {
				return first[open+1 : i]
			}
		}
	}
	t.Fatalf("unbalanced parens in regex %q from %q", first, header)
	return ""
}

// gclRunnerPath returns the absolute path of internal/gcl/runner.go
// relative to the l4 package's working directory.
func gclRunnerPath(t *testing.T) string {
	t.Helper()
	p, err := filepath.Abs(filepath.Join("..", "gcl", "runner.go"))
	if err != nil {
		t.Fatalf("abs path: %v", err)
	}
	if _, err := os.Stat(p); err != nil {
		t.Fatalf("runner.go not found at %s: %v", p, err)
	}
	return p
}

// extractVarBlock reads source at path and returns the substring
// starting at the first occurrence of `header` and ending at the
// matching `}` at brace-depth 0. A deliberately tiny parser — only
// fits the gcl/runner.go convention.
func extractVarBlock(t *testing.T, path, header string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	src := string(data)
	start := strings.Index(src, header)
	if start < 0 {
		t.Fatalf("header %q not found in %s", header, path)
		return ""
	}
	depth := 0
	for i := start; i < len(src); i++ {
		switch src[i] {
		case '{':
			depth++
		case '}':
			depth--
			if depth == 0 {
				return src[start : i+1]
			}
		}
	}
	t.Fatalf("unbalanced braces in block at offset %d in %s", start, path)
	return ""
}

// extractAlternation reduces a `(?i)\b(foo|bar|baz)\b`-style regex
// to its alternation source. Returns the substring between the
// first `(` and the matching `)` at depth 0.
func extractAlternation(patterns []*regexp.Regexp) string {
	if len(patterns) == 0 {
		return ""
	}
	src := patterns[0].String()
	open := strings.Index(src, "(")
	if open < 0 {
		return src
	}
	depth := 0
	for i := open; i < len(src); i++ {
		switch src[i] {
		case '(':
			depth++
		case ')':
			depth--
			if depth == 0 {
				return src[open+1 : i]
			}
		}
	}
	return src
}
