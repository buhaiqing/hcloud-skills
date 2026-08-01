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

// MakePatternID returns the next sequential id (e.g. "ECS-FP004") for
// the given nextNum. Callers compute nextNum once via MaxPatternID and
// increment per new pattern, avoiding the previous O(N·P) hotspot where
// the whole patterns slice was re-scanned on every insertion.
func MakePatternID(skill string, nextNum int) string {
	prefix := strings.ToUpper(strings.ReplaceAll(strings.ReplaceAll(skill, "huaweicloud-", ""), "-ops", ""))
	return fmt.Sprintf("%s-FP%03d", prefix, nextNum)
}

// MaxPatternID scans patterns and returns the highest numeric pattern id
// found in their "id" field, or 0 when none. Used by callers that
// previously passed the full pattern list to MakePatternID.
//
// Pattern id convention: <PREFIX>-FP<NNN>. The numeric suffix is parsed
// here; anything that does not match the convention falls back to 0.
var fpIDRegex = regexp.MustCompile(`(\d+)$`)

func MaxPatternID(patterns []any) int {
	maxNum := 0
	for _, p := range patterns {
		pm, ok := p.(map[string]any)
		if !ok {
			continue
		}
		pid, _ := pm["id"].(string)
		m := fpIDRegex.FindStringSubmatch(pid)
		if len(m) >= 2 {
			n := 0
			fmt.Sscanf(m[1], "%d", &n)
			if n > maxNum {
				maxNum = n
			}
		}
	}
	return maxNum
}

// CreatePatternEntry builds a new failure_patterns.json entry from a raw
// failure_pattern block. nextNum is the integer to embed in the new
// pattern's id; see MakePatternID + MaxPatternID.
func CreatePatternEntry(fp map[string]any, skill string, nextNum int, traceFile string) map[string]any {
	category, _ := fp["category"].(string)
	if _, ok := ValidCategories[category]; !ok {
		category = "runtime"
	}
	now := NowISO()
	entry := map[string]any{
		"id":       MakePatternID(skill, nextNum),
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

// truncString returns at most n leading bytes of s (or fallback when s is
// empty). When truncating, it returns a freshly allocated string via
// strings.Clone so the caller's storage does NOT hold a (ptr, len)
// header pointing into the source backing array — otherwise the original
// megabyte-scale fix text would stay pinned even after we only intended
// to keep 200 bytes of it.
func truncString(s string, n int, fallback string) string {
	if s == "" {
		return fallback
	}
	if len(s) <= n {
		return s
	}
	return strings.Clone(s[:n])
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
				// strings.Clone so we don't pin the entire fp["fix"]
				// backing array (the (ptr, len) header would otherwise
				// hold a 1 MB source live for a 200-byte field).
				newFix = strings.Clone(newFix[:200])
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

// RecordFixOutcome updates a failure pattern's stats after an autonomous fix
// attempt (L4 self-evolution closed loop). On success it bumps
// auto_fixed_count; on failure it bumps escalated_count. It then recomputes
// success_rate = auto_fixed_count / (auto_fixed_count + escalated_count). A
// failed fix lowers success_rate, which de-ranks the pattern below its
// auto_execute_threshold so the autofix executor blocks it next time.
func RecordFixOutcome(root, skill, patternID string, success bool) error {
	data := LoadFailurePatterns(root, skill)
	patterns, _ := data["patterns"].([]any)
	for _, p := range patterns {
		pm, _ := p.(map[string]any)
		if pm == nil {
			continue
		}
		if id, _ := pm["id"].(string); id != patternID {
			continue
		}
		stats, _ := pm["stats"].(map[string]any)
		if stats == nil {
			stats = map[string]any{}
			pm["stats"] = stats
		}
		// auto_fixed_count / escalated_count are float64 in JSON.
		autoFixed := toFloat(stats["auto_fixed_count"])
		escalated := toFloat(stats["escalated_count"])
		if success {
			autoFixed++
			stats["auto_fixed_count"] = autoFixed
		} else {
			escalated++
			stats["escalated_count"] = escalated
		}
		total := autoFixed + escalated
		var rate float64
		if total > 0 {
			rate = autoFixed / total
		}
		stats["success_rate"] = rate
		stats["last_seen"] = NowISO()
		_, err := SaveFailurePatterns(root, skill, data)
		return err
	}
	return nil
}

// toFloat reads a numeric value from an any (JSON float64 / int / int64).
func toFloat(v any) float64 {
	switch x := v.(type) {
	case float64:
		return x
	case int:
		return float64(x)
	case int64:
		return float64(x)
	case float32:
		return float64(x)
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

	// Compute the next available id once. Aggregate previously called
	// sliceFromPatterns(patterns) for every new pattern to feed
	// MakePatternID, scanning the whole list (O(P)) on every miss →
	// O(N·P) total. Track nextNum locally and increment after each
	// insert.
	nextNum := MaxPatternID(patterns) + 1

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
				entry := CreatePatternEntry(patternFp, skill, nextNum, traceName)
				nextNum++
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
