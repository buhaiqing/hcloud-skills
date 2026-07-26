// trace.go ports scripts/trace_learning.py — aggregate GCL traces into the
// per-skill failure_patterns.json.
package learning

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"
)

// ValidCategories is the set of failure-pattern categories (mirrors Python).
var ValidCategories = map[string]struct{}{
	"cli_parameter":    {},
	"runtime":          {},
	"cross_skill":      {},
	"permission":       {},
	"resource_state":   {},
	"network":          {},
	"token_efficiency": {},
	"skill_generation": {},
}

// SignatureKey builds the (category, error, command) dedup tuple.
// Mirrors Python's _signature_key().
func SignatureKey(category, errStr, command string) string {
	cmdKey := ""
	if command != "" {
		fields := strings.Fields(command)
		if len(fields) > 0 {
			cmdKey = fields[0]
		}
	}
	return fmt.Sprintf("%s|%s|%s", category, errStr, cmdKey)
}

// ExtractPatternFromTrace returns the failure_pattern block from a trace's
// final block, or nil if the trace is a PASS / has no learnable signal.
func ExtractPatternFromTrace(trace map[string]any) map[string]any {
	final, _ := trace["final"].(map[string]any)
	if final == nil {
		return nil
	}
	fp, _ := final["failure_pattern"].(map[string]any)
	if fp == nil {
		return nil
	}
	return fp
}

// MakePatternID returns the next sequential id (e.g. "ECS-FP004") based on
// existing patterns.
var fpIDRegex = regexp.MustCompile(`(\d+)$`)

func MakePatternID(skill string, existing []map[string]any) string {
	prefix := strings.ToUpper(strings.ReplaceAll(strings.ReplaceAll(skill, "huaweicloud-", ""), "-ops", ""))
	maxNum := 0
	for _, p := range existing {
		pid, _ := p["id"].(string)
		m := fpIDRegex.FindStringSubmatch(pid)
		if len(m) >= 2 {
			n := 0
			fmt.Sscanf(m[1], "%d", &n)
			if n > maxNum {
				maxNum = n
			}
		}
	}
	return fmt.Sprintf("%s-FP%03d", prefix, maxNum+1)
}

// CreatePatternEntry builds a new failure_patterns.json entry from a raw
// failure_pattern block.
func CreatePatternEntry(fp map[string]any, skill string, existing []map[string]any, traceFile string) map[string]any {
	category, _ := fp["category"].(string)
	if _, ok := ValidCategories[category]; !ok {
		category = "runtime"
	}
	now := NowISO()
	entry := map[string]any{
		"id":       MakePatternID(skill, existing),
		"category": category,
		"signature": map[string]any{
			"error_code":          "",
			"error_message_regex": fp["error"],
			"command_pattern":     firstToken(fp["command"]),
		},
		"root_cause": truncString(fmt.Sprintf("%v", fp["fix"]), 200, "Unknown — pending analysis"),
		"fix": map[string]any{
			"strategy":     pickStrategy(category),
			"action":       truncString(fmt.Sprintf("%v", fp["fix"]), 200, ""),
			"playbook_ref": nil,
		},
		"prevention": "",
		"stats": map[string]any{
			"occurrence_count": 1,
			"first_seen":       now,
			"last_seen":        now,
			"auto_fixed_count": 0,
			"escalated_count":  0,
			"success_rate":     0.0,
		},
		"learned_from": []string{traceFile},
	}
	return entry
}

func firstToken(v any) string {
	s, _ := v.(string)
	if s == "" {
		return ""
	}
	fields := strings.Fields(s)
	if len(fields) == 0 {
		return ""
	}
	return fields[0]
}

func truncString(s string, n int, fallback string) string {
	if s == "" {
		return fallback
	}
	if len(s) <= n {
		return s
	}
	return s[:n]
}

func pickStrategy(category string) string {
	if category == "cli_parameter" || category == "runtime" {
		return "retry"
	}
	return "halt"
}

// MergePattern merges a new observation into an existing pattern entry.
// Mirrors Python's _merge_pattern().
func MergePattern(existing map[string]any, fp map[string]any, traceFile string) {
	stats, _ := existing["stats"].(map[string]any)
	if stats == nil {
		stats = map[string]any{}
		existing["stats"] = stats
	}
	stats["occurrence_count"] = countFromStats(stats) + 1
	stats["last_seen"] = NowISO()
	if _, ok := stats["first_seen"]; !ok {
		stats["first_seen"] = NowISO()
	}
	learned, _ := existing["learned_from"].([]any)
	for _, l := range learned {
		if l == traceFile {
			return // already recorded
		}
	}
	existing["learned_from"] = append(learned, traceFile)
	// Update fix if new one is more specific.
	if newFix, ok := fp["fix"].(string); ok && newFix != "" {
		fix, _ := existing["fix"].(map[string]any)
		if fix == nil {
			fix = map[string]any{}
			existing["fix"] = fix
		}
		cur, _ := fix["action"].(string)
		if len(newFix) > len(cur) {
			if len(newFix) > 200 {
				newFix = newFix[:200]
			}
			fix["action"] = newFix
		}
	}
}

func countFromStats(s map[string]any) int {
	v, ok := s["occurrence_count"]
	if !ok {
		return 0
	}
	switch x := v.(type) {
	case int:
		return x
	case int64:
		return int(x)
	case float64:
		return int(x)
	}
	return 0
}

// ScanTraces returns all gcl-trace-*.json files for a skill, optionally
// filtered by mtime.
type TraceFile struct {
	Path string
	Data map[string]any
}

func ScanTraces(root, skill string, sinceHours *int) []TraceFile {
	tracesDir := filepath.Join(root, "audit-results")
	entries, err := os.ReadDir(tracesDir)
	if err != nil {
		return nil
	}
	var out []TraceFile
	now := time.Now().UTC()
	for _, e := range entries {
		if e.IsDir() || !strings.HasPrefix(e.Name(), "gcl-trace-") || !strings.HasSuffix(e.Name(), ".json") {
			continue
		}
		fp := filepath.Join(tracesDir, e.Name())
		raw, err := os.ReadFile(fp)
		if err != nil {
			continue
		}
		var data map[string]any
		if err := json.Unmarshal(raw, &data); err != nil {
			continue
		}
		if data["skill"] != skill {
			continue
		}
		if sinceHours != nil {
			info, err := os.Stat(fp)
			if err == nil {
				ageHours := now.Sub(info.ModTime().UTC()).Hours()
				if ageHours > float64(*sinceHours) {
					continue
				}
			}
		}
		out = append(out, TraceFile{Path: fp, Data: data})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Path < out[j].Path })
	return out
}

// AggregateResult is the structured return of Aggregate().
type AggregateResult struct {
	Scanned      int
	NewCount     int
	UpdatedCount int
	SkippedCount int
	WrittenTo    string
}

// Aggregate runs the loop: scan → extract → dedup → merge → write.
//
// Stream implementation: each gcl-trace-*.json is read, parsed, and
// consumed one at a time. The full trace map and its raw JSON go out
// of scope after we've extracted the failure_pattern block — peak
// memory is O(1 trace) rather than O(N traces × ~50 KB). For
// ScanTraces callers (test fixtures, ad-hoc tools) the full slice is
// still available; production aggregate traffic goes through this
// function only.
func Aggregate(root, skill string, sinceHours *int, dryRun bool) (*AggregateResult, error) {
	data := LoadFailurePatterns(root, skill)
	patterns, _ := data["patterns"].([]any)

	type keyT struct{ category, errStr, cmdPat string }
	existing := map[keyT]map[string]any{}
	for _, p := range patterns {
		pm, _ := p.(map[string]any)
		if pm == nil {
			continue
		}
		cat, _ := pm["category"].(string)
		sig, _ := pm["signature"].(map[string]any)
		errRegex, _ := sig["error_message_regex"].(string)
		cmdPat, _ := sig["command_pattern"].(string)
		existing[keyT{cat, errRegex, cmdPat}] = pm
	}

	res := &AggregateResult{}

	// Walk audit-results/ once. Each iteration read+parse+extract+merge
	// happens inline; the parsed map and raw bytes drop out of scope at
	// the end of each iteration, so the GC can reclaim them.
	tracesDir := filepath.Join(root, "audit-results")
	entries, dirErr := os.ReadDir(tracesDir)
	if dirErr == nil {
		now := time.Now().UTC()
		// Sorted for deterministic iteration order — matches the prior
		// ScanTraces behavior so callers relying on trace order see the
		// same sequence.
		sort.Slice(entries, func(i, j int) bool { return entries[i].Name() < entries[j].Name() })
		for _, e := range entries {
			if e.IsDir() || !strings.HasPrefix(e.Name(), "gcl-trace-") || !strings.HasSuffix(e.Name(), ".json") {
				continue
			}
			fp := filepath.Join(tracesDir, e.Name())
			if sinceHours != nil {
				info, sErr := os.Stat(fp)
				if sErr == nil {
					ageHours := now.Sub(info.ModTime().UTC()).Hours()
					if ageHours > float64(*sinceHours) {
						continue
					}
				}
			}
			raw, rErr := os.ReadFile(fp)
			if rErr != nil {
				continue
			}
			var trace map[string]any
			if uErr := json.Unmarshal(raw, &trace); uErr != nil {
				continue
			}
			// Drop the raw bytes now — they were only needed for Unmarshal.
			raw = nil
			if trace["skill"] != skill {
				continue
			}
			res.Scanned++
			traceName := e.Name()
			patternFp := ExtractPatternFromTrace(trace)
			if patternFp == nil {
				res.SkippedCount++
				continue
			}
			cat, _ := patternFp["category"].(string)
			errStr, _ := patternFp["error"].(string)
			cmd, _ := patternFp["command"].(string)
			cmdKey := firstToken(cmd)
			k := keyT{cat, errStr, cmdKey}
			if _, ok := existing[k]; ok {
				MergePattern(existing[k], patternFp, traceName)
				res.UpdatedCount++
			} else {
				entry := CreatePatternEntry(patternFp, skill, sliceFromPatterns(patterns), traceName)
				patterns = append(patterns, entry)
				existing[k] = entry
				res.NewCount++
			}
		}
	}

	meta, _ := data["meta"].(map[string]any)
	if meta == nil {
		meta = map[string]any{}
		data["meta"] = meta
	}
	if v, ok := meta["source_traces_analyzed"].(float64); ok {
		meta["source_traces_analyzed"] = v + float64(res.Scanned)
	} else {
		meta["source_traces_analyzed"] = res.Scanned
	}
	data["patterns"] = patterns

	if dryRun {
		return res, nil
	}
	if res.NewCount > 0 || res.UpdatedCount > 0 {
		out, err := SaveFailurePatterns(root, skill, data)
		if err != nil {
			return res, err
		}
		res.WrittenTo = out
	}
	return res, nil
}

func sliceFromPatterns(patterns []any) []map[string]any {
	out := make([]map[string]any, 0, len(patterns))
	for _, p := range patterns {
		if pm, ok := p.(map[string]any); ok {
			out = append(out, pm)
		}
	}
	return out
}

// LoadFailurePatterns returns the existing failure_patterns.json or a fresh
// scaffold (mirrors Python's load_failure_patterns()).
func LoadFailurePatterns(root, skill string) map[string]any {
	skillID := "huaweicloud-" + strings.ReplaceAll(skill, "huaweicloud-", "")
	skillID = strings.ReplaceAll(skillID, "-ops", "") // tolerate bare shortname
	if !strings.HasPrefix(skill, "huaweicloud-") {
		skillID = "huaweicloud-" + skill + "-ops"
	} else {
		skillID = skill
	}
	path := filepath.Join(root, skillID, "assets", "failure_patterns.json")
	raw, err := os.ReadFile(path)
	if err != nil {
		return map[string]any{
			"$schema":  "failure-patterns/v1",
			"skill_id": skillID,
			"patterns": []any{},
			"meta": map[string]any{
				"total_patterns":         0,
				"last_aggregation":       NowISO(),
				"source_traces_analyzed": 0,
			},
		}
	}
	var out map[string]any
	if err := json.Unmarshal(raw, &out); err != nil {
		return map[string]any{
			"$schema":  "failure-patterns/v1",
			"skill_id": skillID,
			"patterns": []any{},
			"meta": map[string]any{
				"total_patterns":         0,
				"last_aggregation":       NowISO(),
				"source_traces_analyzed": 0,
			},
		}
	}
	return out
}

// SaveFailurePatterns persists the in-memory failure_patterns.json back to disk.
func SaveFailurePatterns(root, skill string, data map[string]any) (string, error) {
	skillID := skill
	if !strings.HasPrefix(skill, "huaweicloud-") {
		skillID = "huaweicloud-" + skill + "-ops"
	}
	dir := filepath.Join(root, skillID, "assets")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	p := filepath.Join(dir, "failure_patterns.json")
	if err := writeJSON(p, data); err != nil {
		return "", err
	}
	return p, nil
}
