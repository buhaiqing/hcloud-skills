// trend.go: recurrence detection across GCL trace families. The closed loop
// has three failure-handling stages (aggregate → failure_patterns → match
// pre-execution risk), but no metric answers "did the same defect recur in
// the next window?". A non-decreasing recurrence rate means the rule book
// is not absorbing Critic findings — the upstream signal the loop is meant
// to convert into learning. This package fills that gap.
//
// TrendReport is the only public entry. It reuses the same trust boundary
// as Aggregate/ScanTraces (ClassifyTrace + IsTraceFileName + TraceFilePatterns)
// so the dual-family consumption rule — gcl-trace-*.json +
// orchestrator-trace-*.json — stays in one place. ScanTraces is not used
// because it filters by `skill`; trend is cross-skill by design.
package learning

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// TrendReport is the closed-shape recurrence summary. The field set and
// types are frozen (see gcl-spec.md §trend): every field below is part of
// the contract, in this exact order. Critics compare the rendered JSON
// against this schema — do not rename, drop, or reorder.
//
// Recurrence rule: pattern_key appears in ≥2 traces within the window.
// The threshold is hardcoded at 2 because the question the metric answers
// is "did this defect recur?" — a single observation is not recurrence.
type TrendReport struct {
	WindowDays         int             `json:"window_days"`
	TotalTraces        int             `json:"total_traces"`
	TotalFindings      int             `json:"total_findings"`
	RecurringFindings  int             `json:"recurring_findings"`
	RecurrenceRate     float64         `json:"recurrence_rate"`
	ByPattern          []PatternBucket `json:"by_pattern"`
	TracesByCriticType map[string]int  `json:"traces_by_critic_type"`
}

// PatternBucket is one row of TrendReport.ByPattern. Sorted by count DESC
// then pattern_key ASC for deterministic output (callers diff snapshots).
type PatternBucket struct {
	PatternKey  string   `json:"pattern_key"`
	Count       int      `json:"count"`
	FirstSeen   string   `json:"first_seen"`
	LastSeen    string   `json:"last_seen"`
	CriticTypes []string `json:"critic_types"`
	Verdicts    []string `json:"verdicts"`
}

// TrendReport scans audit-results/ under root for both trace families,
// keeps evidence-class traces within the window, extracts one finding per
// trace × critic-suggestion (the only finding-shaped array the schema
// guarantees — see gcl-trace.schema.json → iterations.items.critic.suggestions),
// groups by normalized pattern_key, and emits the closed TrendReport shape.
//
// windowDays<=0 means "all traces". The trace mtime is the window anchor:
// L4 traces carry started_at (preferred), gcl traces only have mtime.
// started_at older than now-windowDays still counts as in-window — a 2026
// trace stored on a slow CI runner must not silently disappear.
//
// An absent audit-results/ directory is NOT an error: a fresh checkout
// returns the zero-value TrendReport (recurrence_rate=0, by_pattern=[]).
// Anything else (partial write, permission denied) is returned as a
// non-nil error so the caller can surface it.
func ComputeTrendReport(root string, windowDays int) (*TrendReport, error) {
	rep := &TrendReport{
		WindowDays:         windowDays,
		ByPattern:          []PatternBucket{}, // pre-init to non-nil so zero-finding JSON renders as [] (frozen schema contract)
		TracesByCriticType: map[string]int{},
	}

	tracesDir := filepath.Join(root, "audit-results")
	entries, err := os.ReadDir(tracesDir)
	if err != nil {
		if os.IsNotExist(err) {
			return rep, nil
		}
		return nil, fmt.Errorf("read audit-results: %w", err)
	}

	cutoff := time.Time{}
	if windowDays > 0 {
		cutoff = time.Now().UTC().Add(-time.Duration(windowDays) * 24 * time.Hour)
	}

	// bucket: pattern_key → { count, first_seen, last_seen, critic_types (set), verdicts (set) }
	type bucket struct {
		count      int
		firstSeen  time.Time
		lastSeen   time.Time
		criticSet  map[string]struct{}
		verdictSet map[string]struct{}
	}
	buckets := map[string]*bucket{}

	for _, e := range entries {
		if e.IsDir() || !IsTraceFileName(e.Name()) {
			continue
		}
		fp := filepath.Join(tracesDir, e.Name())
		raw, rErr := os.ReadFile(fp)
		if rErr != nil {
			continue
		}
		var trace map[string]any
		if uErr := json.Unmarshal(raw, &trace); uErr != nil {
			continue
		}
		// Reuse the frozen classification order (parse → schema-invalid →
		// smoke → evidence). Anything that fails validation is dropped here
		// for the same reason Aggregate drops it: an unverifiable trace is
		// untrusted input.
		class, _ := ClassifyTrace(raw, trace)
		if class != TraceEvidence {
			continue
		}

		// Window filter — prefer started_at when present, else mtime.
		traceTime, ok := traceAnchorTime(trace, fp)
		if !ok {
			continue
		}
		if !cutoff.IsZero() && traceTime.UTC().Before(cutoff) {
			continue
		}

		rep.TotalTraces++

		final, _ := trace["final"].(map[string]any)
		criticType := "unknown"
		var verdicts []string
		var category string
		if final != nil {
			if ct, ok := final["critic_type"].(string); ok && ct != "" {
				criticType = ct
			}
			if st, ok := final["status"].(string); ok && st != "" {
				verdicts = append(verdicts, st)
			}
			if fpBlock, ok := final["failure_pattern"].(map[string]any); ok {
				if c, ok := fpBlock["category"].(string); ok && c != "" {
					category = c
				}
			}
		}
		rep.TracesByCriticType[criticType]++

		// One finding per critic-suggestion across iterations. Suggestions
		// are the only finding-shaped array the canonical schema mandates
		// (see gcl-trace.schema.json iterations.items.critic.suggestions:
		// required array of strings). PASS traces carry []suggestions and
		// contribute 0 findings — exactly what we want, since PASS has
		// nothing to recur.
		suggestions := collectSuggestions(trace)
		if len(suggestions) == 0 {
			continue
		}
		for _, sug := range suggestions {
			rep.TotalFindings++
			key := normalizePatternKey(category, sug)
			b := buckets[key]
			if b == nil {
				b = &bucket{
					firstSeen:  traceTime,
					lastSeen:   traceTime,
					criticSet:  map[string]struct{}{},
					verdictSet: map[string]struct{}{},
				}
				buckets[key] = b
			}
			b.count++
			if traceTime.Before(b.firstSeen) {
				b.firstSeen = traceTime
			}
			if traceTime.After(b.lastSeen) {
				b.lastSeen = traceTime
			}
			b.criticSet[criticType] = struct{}{}
			for _, v := range verdicts {
				b.verdictSet[v] = struct{}{}
			}
		}
	}

	// Emit by_pattern (sorted DESC by count, ASC by pattern_key for
	// tie-break — deterministic JSON output).
	keys := make([]string, 0, len(buckets))
	for k := range buckets {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool {
		bi, bj := buckets[keys[i]], buckets[keys[j]]
		if bi.count != bj.count {
			return bi.count > bj.count
		}
		return keys[i] < keys[j]
	})
	for _, k := range keys {
		b := buckets[k]
		rep.ByPattern = append(rep.ByPattern, PatternBucket{
			PatternKey:  k,
			Count:       b.count,
			FirstSeen:   b.firstSeen.UTC().Format(time.RFC3339),
			LastSeen:    b.lastSeen.UTC().Format(time.RFC3339),
			CriticTypes: setToSortedSlice(b.criticSet),
			Verdicts:    setToSortedSlice(b.verdictSet),
		})
		if b.count >= 2 {
			rep.RecurringFindings++
		}
	}

	// recurrence_rate = recurring / total_findings. Zero findings → 0.0
	// (no division by zero). This is the rate at which a unique defect
	// observation in the window was preceded by another observation of the
	// same defect; a non-decreasing series means the rule book is not
	// absorbing Critic findings.
	if rep.TotalFindings > 0 {
		rep.RecurrenceRate = float64(rep.RecurringFindings) / float64(rep.TotalFindings)
	}
	return rep, nil
}

// traceAnchorTime returns the timestamp to use for window filtering: L4
// traces carry started_at; gcl traces only have mtime. Returns ok=false
// when neither is available so the caller skips the trace — refusing to
// time-bucket a trace whose age cannot be determined is safer than
// silently dropping it later.
func traceAnchorTime(trace map[string]any, fp string) (time.Time, bool) {
	if s, ok := trace["started_at"].(string); ok && s != "" {
		if t, err := time.Parse(time.RFC3339, s); err == nil {
			return t, true
		}
	}
	info, err := os.Stat(fp)
	if err != nil {
		return time.Time{}, false
	}
	return info.ModTime(), true
}

// collectSuggestions flattens iterations[].critic.suggestions across the
// whole trace. Each iteration is one Critic pass; suggestions are the
// per-pass finding list. Defensive against non-array types so a partially
// malformed trace (e.g. truncated critic block) yields an empty slice
// instead of panicking.
func collectSuggestions(trace map[string]any) []string {
	iterations, _ := trace["iterations"].([]any)
	var out []string
	for _, it := range iterations {
		im, ok := it.(map[string]any)
		if !ok {
			continue
		}
		c, ok := im["critic"].(map[string]any)
		if !ok {
			continue
		}
		sugs, ok := c["suggestions"].([]any)
		if !ok {
			continue
		}
		for _, s := range sugs {
			if str, ok := s.(string); ok && str != "" {
				out = append(out, str)
			}
		}
	}
	return out
}

// normalizePatternKey returns the "<category>|<error>" key the metric
// groups by. Category is taken from final.failure_pattern.category when
// present (the canonical vocabulary in ValidCategories); otherwise the
// fallback bucket is the empty string so un-categorized suggestions still
// group by suggestion text and produce real recurrence signal.
//
// Error is the suggestion text itself, normalized as:
//  1. lower-cased
//  2. whitespace collapsed (so "  OOMKilled  in  pod  " and "OOMKilled in pod"
//     map to the same bucket instead of splitting into near-duplicates)
//  3. trailing punctuation stripped (".;:!?)]}\"'") — so "OOMKilled in pod."
//     and "OOMKilled in pod" merge. Without this, the same defect worded with
//     a trailing period would be a different bucket and recurrence would
//     under-fire (the metric would silently under-report repeat defects).
//
// recurrence_rate is therefore a lower bound (exact-match normalization): a
// pair of true recurrences that differ only by paraphrase will not merge.
func normalizePatternKey(category, suggestion string) string {
	sug := strings.Join(strings.Fields(strings.ToLower(suggestion)), " ")
	// Strip trailing punctuation in a loop: "foo." → "foo", "foo..;" → "foo".
	sug = strings.TrimRight(sug, ".,;:!?)]}\"'")
	if category == "" {
		category = "uncategorized"
	}
	return category + "|" + sug
}

// setToSortedSlice returns a deterministic slice from a set. The output
// feeds CriticTypes/Verdicts — JSON consumers diff snapshots, so the
// rendering must not depend on map iteration order.
func setToSortedSlice(set map[string]struct{}) []string {
	out := make([]string, 0, len(set))
	for k := range set {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}
