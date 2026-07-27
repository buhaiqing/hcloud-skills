// Package drift guards against silent drift between the canonical
// huaweicloud-skill-generator copy and the agent runtime copy under
// .agents/skills/. It mirrors scripts/check_skill_generator_drift.py 1:1 so
// callers can swap the Python invocation for the Go binary with no behaviour
// change.
//
// Two roots are tracked:
//
//	<root>/huaweicloud-skill-generator/         — canonical, git-tracked
//	<root>/.agents/skills/huaweicloud-skill-generator/  — runtime, gitignored
//
// `Check` requires both copies exist and reports any drift. `Sync` reconciles
// the runtime copy from the canonical root (creates the runtime root on
// demand when missing, e.g. on fresh CI checkouts).
package drift

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

// CanonicalRel is the canonical skill-generator root, relative to repo root.
const CanonicalRel = "huaweicloud-skill-generator"

// RuntimeRel is the agent runtime copy, relative to repo root (gitignored).
const RuntimeRel = ".agents/skills/huaweicloud-skill-generator"

// skipNames are files we never compare (macOS noise).
var skipNames = map[string]bool{".DS_Store": true}

// Drift captures the three drift classes reported by `Check`.
type Drift struct {
	OnlyCanonical []string `json:"only_canonical"`
	OnlyRuntime   []string `json:"only_runtime"`
	Differing     []string `json:"differing"`
}

// Report is the result of Check or Sync.
type Report struct {
	OK      bool     `json:"ok"`
	Errors  []string `json:"errors,omitempty"`
	Actions []string `json:"actions,omitempty"`
	Drift   Drift    `json:"drift"`
	DryRun  bool     `json:"dry_run,omitempty"`
}

// Check verifies both copies exist and are byte-for-byte equal. Any drift
// produces a non-OK Report with the offending files classified.
func Check(root string) (*Report, error) {
	canonical := filepath.Join(root, CanonicalRel)
	runtime := filepath.Join(root, RuntimeRel)
	r := &Report{}
	if !isDir(canonical) {
		r.Errors = append(r.Errors, fmt.Sprintf("%s: canonical skill root missing", canonical))
	}
	if !isDir(runtime) {
		r.Errors = append(r.Errors, fmt.Sprintf("%s: runtime skill root missing (agent runtime will fail to load)", runtime))
	}
	if len(r.Errors) > 0 {
		return r, nil
	}
	d, err := collectDrift(canonical, runtime)
	if err != nil {
		return nil, err
	}
	r.Drift = d
	r.appendDriftErrors(runtime)
	r.OK = len(r.Errors) == 0
	return r, nil
}

// Sync reconciles the runtime copy from the canonical root. When the runtime
// directory is missing it is created on demand so Sync is self-healing on
// fresh CI checkouts. When dryRun is true no filesystem mutations occur; the
// report still lists what would have been done.
func Sync(root string, dryRun bool) (*Report, error) {
	canonical := filepath.Join(root, CanonicalRel)
	runtime := filepath.Join(root, RuntimeRel)
	r := &Report{DryRun: dryRun}
	if !isDir(canonical) {
		r.Errors = append(r.Errors, fmt.Sprintf("%s: missing", canonical))
		return r, nil
	}
	if !isDir(runtime) {
		r.Actions = append(r.Actions, fmt.Sprintf("create %s", runtime))
		if !dryRun {
			if err := os.MkdirAll(runtime, 0o755); err != nil {
				return r, err
			}
		}
	}
	r.appendFileActions(canonical, runtime, dryRun)
	// appendFileActions records per-file failures in r.Errors but does not
	// raise to the caller. Honor that: any reconcile error fails the sync.
	r.OK = len(r.Errors) == 0
	return r, nil
}

// collectDrift walks both trees and classifies each relative path.
func collectDrift(canonical, runtime string) (Drift, error) {
	canon := indexFiles(canonical)
	runt := indexFiles(runtime)
	var onlyCan, onlyRun, diff []string
	for rel := range canon {
		if _, ok := runt[rel]; !ok {
			onlyCan = append(onlyCan, rel)
			continue
		}
		equal, err := sameContent(canon[rel], runt[rel])
		if err != nil {
			return Drift{}, err
		}
		if !equal {
			diff = append(diff, rel)
		}
	}
	for rel := range runt {
		if _, ok := canon[rel]; !ok {
			onlyRun = append(onlyRun, rel)
		}
	}
	sort.Strings(onlyCan)
	sort.Strings(onlyRun)
	sort.Strings(diff)
	return Drift{OnlyCanonical: onlyCan, OnlyRuntime: onlyRun, Differing: diff}, nil
}

func (r *Report) appendDriftErrors(runtime string) {
	if len(r.Drift.OnlyCanonical) > 0 {
		r.Errors = append(r.Errors,
			fmt.Sprintf("%s: missing files: %s%s",
				runtime,
				joinNames(r.Drift.OnlyCanonical, 5),
				ellipsis(r.Drift.OnlyCanonical, 5)))
	}
	if len(r.Drift.OnlyRuntime) > 0 {
		r.Errors = append(r.Errors,
			fmt.Sprintf("%s: extra files not in canonical: %s%s",
				runtime,
				joinNames(r.Drift.OnlyRuntime, 5),
				ellipsis(r.Drift.OnlyRuntime, 5)))
	}
	if len(r.Drift.Differing) > 0 {
		r.Errors = append(r.Errors,
			fmt.Sprintf("%s: %d file(s) drifted from canonical: %s%s",
				runtime, len(r.Drift.Differing),
				joinNames(r.Drift.Differing, 5),
				ellipsis(r.Drift.Differing, 5)))
	}
}

// appendFileActions emits the copy/overwrite/remove actions for drift
// reconciliation and applies them when !dryRun.
func (r *Report) appendFileActions(canonical, runtime string, dryRun bool) {
	d, err := collectDrift(canonical, runtime)
	if err != nil {
		r.Errors = append(r.Errors, err.Error())
		return
	}
	for _, rel := range d.OnlyCanonical {
		src := filepath.Join(canonical, rel)
		dst := filepath.Join(runtime, rel)
		r.Actions = append(r.Actions, fmt.Sprintf("copy %s -> %s", src, dst))
		if !dryRun {
			if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
				r.Errors = append(r.Errors, err.Error())
				continue
			}
			if err := copyFile(src, dst); err != nil {
				r.Errors = append(r.Errors, err.Error())
			}
		}
	}
	for _, rel := range d.Differing {
		src := filepath.Join(canonical, rel)
		dst := filepath.Join(runtime, rel)
		r.Actions = append(r.Actions, fmt.Sprintf("overwrite %s from %s", dst, src))
		if !dryRun {
			if err := copyFile(src, dst); err != nil {
				r.Errors = append(r.Errors, err.Error())
			}
		}
	}
	for _, rel := range d.OnlyRuntime {
		dst := filepath.Join(runtime, rel)
		r.Actions = append(r.Actions, fmt.Sprintf("remove %s (not in canonical)", dst))
		if !dryRun {
			if err := os.Remove(dst); err != nil && !os.IsNotExist(err) {
				r.Errors = append(r.Errors, err.Error())
			}
		}
	}
}

// indexFiles returns a map of relative path → absolute path for every regular
// file under root (skipping skipNames).
func indexFiles(root string) map[string]string {
	out := map[string]string{}
	_ = filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			return nil
		}
		if skipNames[d.Name()] {
			return nil
		}
		rel, rErr := filepath.Rel(root, path)
		if rErr != nil {
			return nil
		}
		out[filepath.ToSlash(rel)] = path
		return nil
	})
	return out
}
func sameBytes(a, b string) (bool, error) {
	ha, errA := hashFile(a)
	hb, errB := hashFile(b)
	if errA != nil || errB != nil {
		return false, fmt.Errorf("hash compare %s vs %s: %w / %w", a, b, errA, errB)
	}
	return ha == hb, nil
}

// sameContent reports whether two files are semantically equal, ignoring
// noise that does not affect agent execution.  For .md files it strips HTML
// comments and normalises whitespace; for all other files it falls back to
// sameBytes.
func sameContent(a, b string) (bool, error) {
	ext := strings.ToLower(filepath.Ext(a))
	if ext != ".md" && ext != ".markdown" {
		return sameBytes(a, b)
	}
	dataA, errA := os.ReadFile(a)
	dataB, errB := os.ReadFile(b)
	if errA != nil || errB != nil {
		return sameBytes(a, b)
	}
	normA := normalizeMarkdown(dataA)
	normB := normalizeMarkdown(dataB)
	return bytes.Equal(normA, normB), nil
}

// normalizeMarkdown removes HTML/markdown comments, collapses blank lines,
// trims trailing whitespace, and strips leading/trailing blank lines.
// This makes comment-only and formatting-only changes invisible to drift detection.
var commentRe = regexp.MustCompile(`<!--[\s\S]*?-->`)

func normalizeMarkdown(data []byte) []byte {
	// Remove HTML comments
	out := commentRe.ReplaceAll(data, nil)
	// Remove blank lines (collapse 2+ to 1)
	out = blankLineRe.ReplaceAll(out, []byte{'\n'})
	// Trim trailing whitespace per line
	lines := bytes.Split(out, []byte{'\n'})
	for i, line := range lines {
		lines[i] = bytes.TrimRight(line, " \t")
	}
	out = bytes.Join(lines, []byte{'\n'})
	// Strip leading/trailing blank lines
	out = bytes.Trim(out, "\n")
	return out
}

// blankLineRe matches one or more consecutive blank lines (line containing
// only zero or more space/tab characters).
var blankLineRe = regexp.MustCompile(`(?m)(^[ \t]*\n){2,}`)

func hashFile(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:]), nil
}

func copyFile(src, dst string) error {
	data, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	info, err := os.Stat(src)
	if err != nil {
		return err
	}
	return os.WriteFile(dst, data, info.Mode().Perm())
}

func isDir(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}

func joinNames(names []string, limit int) string {
	n := len(names)
	if n > limit {
		n = limit
	}
	return strings.Join(names[:n], ", ")
}

func ellipsis(names []string, limit int) string {
	if len(names) > limit {
		return " ..."
	}
	return ""
}
