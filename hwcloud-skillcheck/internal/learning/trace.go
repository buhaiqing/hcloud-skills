// trace.go ports scripts/trace_learning.py — aggregate GCL traces into the
// per-skill failure_patterns.json.
package learning

import (
	"encoding/json"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/buhaiqing/hcloud-skills/hwcloud-skillcheck/internal/embed"
	"github.com/buhaiqing/hcloud-skills/hwcloud-skillcheck/internal/schema"
)

// Trace sources. Every trace reader (this package's Aggregate/ScanTraces and
// cmd/aggregate.go) must accept both writers' output; an absent top-level
// "source" field means TraceSourceGCL (the ~100 legacy gcl traces written
// before the field existed).
const (
	TraceSourceGCL = "gcl"
	TraceSourceL4  = "l4"
)

// TraceFilePatterns are the audit-results/ filenames that carry traces:
//   - gcl-trace-<timestamp>-<hex>.json      (internal/gcl.PersistTrace)
//   - orchestrator-trace-<faultID>.json     (internal/l4.HandleFault)
var TraceFilePatterns = []string{"gcl-trace-*.json", "orchestrator-trace-*.json"}

// IsTraceFileName reports whether an audit-results/ entry name is a trace
// file produced by either writer.
func IsTraceFileName(name string) bool {
	for _, pat := range TraceFilePatterns {
		if ok, err := path.Match(pat, name); err == nil && ok {
			return true
		}
	}
	return false
}

// TraceSource returns the writer that produced a trace: TraceSourceL4 for
// orchestrator traces, TraceSourceGCL otherwise (including legacy gcl traces
// that predate the field).
func TraceSource(trace map[string]any) string {
	if s, ok := trace["source"].(string); ok && s != "" {
		return s
	}
	return TraceSourceGCL
}

// TraceClass is how a trace file is consumed. The classification order is
// frozen and MUST be applied in exactly this sequence:
//
//  1. parse           — unparseable JSON never becomes a trace at all: the
//     reader WARNs and drops it (cmd/aggregate.go).
//  2. schema-invalid  — the trace does not conform to the canonical trace
//     schema (embed.TraceSchema, the same schema
//     `hwcloud-skillcheck validate schema trace` enforces).
//     An audit-results/ file is written by a runtime run, so
//     it is untrusted input: a non-conforming trace is
//     attacker-shaped, not merely quirky. Counted in
//     invalid_trace and excluded from every metric AND from
//     failure-pattern merging.
//  3. smoke           — conforming, but carries no verification signal (see
//     IsSmokeTrace): counted in skipped_smoke, excluded from
//     every metric and from merging.
//  4. evidence        — the contributing runs. evidence_runs counts only
//     these; pass_rate's denominator is their number, and
//     only they feed by_source / by_skill / by_critic_type /
//     rubric averages / failure-pattern merge.
type TraceClass int

const (
	// TraceInvalid is the zero value on purpose: a trace nobody classified
	// must not be able to move a metric (fail closed).
	TraceInvalid TraceClass = iota
	// TraceSmoke: schema-valid, no verification signal.
	TraceSmoke
	// TraceEvidence: schema-valid and non-smoke — the only traces allowed to
	// move a metric.
	TraceEvidence
)

// ValidateTrace returns the canonical-schema violations of a raw trace file
// (empty slice = conforming). It is the trust boundary for the read path: the
// schema is the embedded copy of huaweicloud-ces-ops/assets/gcl-trace.schema.json
// (internal/embed), so validation needs no repo-relative file and matches what
// `validate schema trace` enforces. A validator-level failure (unparseable
// schema/instance) is reported as a violation too: an unverifiable trace must
// not be trusted.
func ValidateTrace(raw []byte) []string {
	errs, err := schema.ValidateFile(raw, embed.TraceSchema)
	if err != nil {
		return []string{"schema validator error: " + err.Error()}
	}
	return errs
}

// ClassifyTrace returns the consumption class of one parsed trace and, when it
// is TraceInvalid, the schema errors that disqualified it (first error is
// reported to the operator). Both readers — cmd/aggregate.go and this package's
// Aggregate/ScanTraces — call this so the frozen order (schema-invalid before
// smoke, both before evidence) cannot drift between them.
func ClassifyTrace(raw []byte, trace map[string]any) (TraceClass, []string) {
	if errs := ValidateTrace(raw); len(errs) > 0 {
		return TraceInvalid, errs
	}
	if IsSmokeTrace(trace) {
		return TraceSmoke, nil
	}
	return TraceEvidence, nil
}

// traceWarn writes one attribution line for a trace the reader refused.
func traceWarn(name, msg string) {
	fmt.Fprintf(os.Stderr, "WARN: %s: %s\n", name, msg)
}

// IsSmokeTrace reports whether a trace carries no verifiable signal and must
// therefore stay out of pass-rate math, failure-pattern merging, and knowledge
// extraction. A trace is smoke when any of these hold:
//
//   - it has no `final` block — nothing terminal was recorded. This is how the
//     ~100 orchestrator traces written before the P0 fix are classified; they
//     were never consumable at all.
//   - its request or fault is the literal "smoke" (the pre-commit smoke gate
//     runs `l4 handle --fault smoke`).
//   - its recorded step count is 0: no step executed, so nothing was verified.
//     L4 traces always carry orchestration.step_count; gcl traces carry no step
//     count at all, and "absent" is not "zero" — an empty-iteration gcl trace
//     is a budget/SAFETY_FAIL run, which is real signal that must keep counting
//     against the pass rate.
//
// A trace whose final status is exactly SAFETY_FAIL is NEVER smoke, whatever
// the request/fault/step count says. A safety failure is the strongest signal a
// trace can carry — the run reached the terminal, safety-critical verdict — so
// classifying it as "nothing was verified" deleted a genuine failure from
// pass_rate and from failure-pattern learning at the same time (the smoke gate
// fixture and the SAFETY_FAIL verdict collide in exactly this shape). The
// verdict wins: it must stay in the metric set and in the learning corpus.
func IsSmokeTrace(trace map[string]any) bool {
	if trace == nil {
		return true
	}
	final, ok := trace["final"].(map[string]any)
	if !ok {
		return true
	}
	if status, _ := final["status"].(string); status == "SAFETY_FAIL" {
		return false
	}
	if isSmokeToken(trace["request"]) || isSmokeToken(trace["fault"]) {
		return true
	}
	if n, ok := traceStepCount(trace); ok && n == 0 {
		return true
	}
	return false
}

// isSmokeToken reports whether a request/fault value is the smoke marker.
// Values are matched case-insensitively on the trimmed whole string so a real
// fault that merely mentions smoke ("smoke detected in rack 3") is not
// misclassified.
func isSmokeToken(v any) bool {
	s, ok := v.(string)
	if !ok {
		return false
	}
	return strings.EqualFold(strings.TrimSpace(s), "smoke")
}

// traceStepCount returns the step count explicitly recorded on a trace and
// whether the trace carries one at all. L4 traces use
// orchestration.step_count; a top-level step_count is honored too. JSON
// numbers decode as float64; a string count is tolerated for hand-written
// traces. gcl traces record neither (they record iterations instead, which is
// not a step count — see IsSmokeTrace).
func traceStepCount(trace map[string]any) (int, bool) {
	if n, ok := asInt(trace["step_count"]); ok {
		return n, true
	}
	if orch, ok := trace["orchestration"].(map[string]any); ok {
		if n, ok := asInt(orch["step_count"]); ok {
			return n, true
		}
	}
	return 0, false
}

// asInt normalizes a JSON number (or numeric string) to int.
func asInt(v any) (int, bool) {
	switch x := v.(type) {
	case float64:
		return int(x), true
	case int:
		return x, true
	case int64:
		return int(x), true
	case json.Number:
		if n, err := x.Int64(); err == nil {
			return int(n), true
		}
	case string:
		if n, err := strconv.Atoi(strings.TrimSpace(x)); err == nil {
			return n, true
		}
	}
	return 0, false
}

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

// maxPatternStringLen caps every trace-derived string written into
// failure_patterns.json. The values are produced by an unbounded runtime run
// (a critic suggestion, a CLI stderr line); 200 bytes is the long-standing cap
// for these fields (see CreatePatternEntry / MergePattern).
const maxPatternStringLen = 200

// controlCharReason reports the first control character in s ("" = none).
// A raw trace string can carry \x00-\x1f or \x7f, which would let a crafted
// trace forge log lines and JSON layout inside the knowledge base.
func controlCharReason(s string) string {
	for i := range len(s) {
		if c := s[i]; c < 0x20 || c == 0x7f {
			return fmt.Sprintf("contains control character 0x%02x at offset %d", c, i)
		}
	}
	return ""
}

// patternString returns v as the string a pattern field would store.
func patternString(v any) string {
	if s, ok := v.(string); ok {
		return s
	}
	if v == nil {
		return ""
	}
	return fmt.Sprintf("%v", v)
}

// skillIdentityTokens returns the tokens that appear in (nearly) every command
// a skill's steps are built from: the CLI binary and the product short name.
//
// matchPreExecutionRisk (internal/l4) folds a pattern's error_message_regex to
// its first whitespace-separated field and tests it with
// strings.Contains(command, field). A pattern anchored on one of these tokens
// therefore matches EVERY planned step, and a match is not a warning: the
// executor records SKIPPED_BY_PATTERN_RISK and does not run the step. Such an
// entry is not knowledge, it is a step-disable switch.
func skillIdentityTokens(skill string) []string {
	short := strings.TrimSuffix(strings.TrimPrefix(skill, "huaweicloud-"), "-ops")
	tokens := []string{"hcloud"}
	if short != "" && !strings.EqualFold(short, "hcloud") {
		tokens = append(tokens, short)
	}
	return tokens
}

// ValidateTracePattern reports why a failure_pattern block extracted from a
// trace must NOT enter failure_patterns.json, or nil when it is safe to merge.
//
// This is the trust boundary of the write path. A trace under audit-results/ is
// produced by a runtime run, and `learning trace aggregate` merges whatever it
// finds there into the knowledge base; the knowledge base then drives
// matchPreExecutionRisk, whose match SKIPS the planned step
// (SKIPPED_BY_PATTERN_RISK) and whose fix.action is fed to the autofix
// executor. An unvalidated merge therefore lets a crafted trace disable steps
// and inject hostile text into the remediation path. The checks:
//
//   - category must be one this knowledge base knows (ValidCategories — the
//     vocabulary mirrored from internal/learning/knowledge.go; an unknown
//     category yields a pattern no consumer can reason about).
//   - error_message_regex and command_pattern must compile and must not match
//     the empty string: a pattern that matches everything makes every step
//     skippable.
//   - the error pattern's anchor (its first field — the literal the live
//     matcher substrings into every command) must not be one of the skill's
//     identity tokens, and the pattern must not match one either: anchoring on
//     "hcloud" or the product name matches every command of the skill.
//   - every string written must be control-character free and within
//     maxPatternStringLen.
//
// Rejection is the only safe outcome: a rejected pattern is dropped (counted in
// AggregateResult.RejectedPatterns and WARNed), never repaired into something
// the validator would accept.
func ValidateTracePattern(fp map[string]any, skill string) error {
	if fp == nil {
		return fmt.Errorf("no failure_pattern block")
	}
	category, categoryIsString := fp["category"].(string)
	if _, known := ValidCategories[category]; !known || !categoryIsString {
		return fmt.Errorf("category %q is not in the known failure-pattern vocabulary", patternString(fp["category"]))
	}
	rawError, errIsString := fp["error"].(string)
	if !errIsString || rawError == "" {
		return fmt.Errorf("error is not a non-empty string (%T)", fp["error"])
	}
	if _, cmdIsString := fp["command"].(string); !cmdIsString && fp["command"] != nil {
		return fmt.Errorf("command is not a string (%T)", fp["command"])
	}
	cmdPat := firstToken(fp["command"])

	for _, f := range []struct{ name, value string }{
		{"skill", skill},
		{"category", category},
		{"error", rawError},
		{"command", cmdPat},
		{"fix", patternString(fp["fix"])},
	} {
		if len(f.value) > maxPatternStringLen {
			return fmt.Errorf("%s is %d bytes, over the %d-byte cap", f.name, len(f.value), maxPatternStringLen)
		}
		if reason := controlCharReason(f.value); reason != "" {
			return fmt.Errorf("%s %s", f.name, reason)
		}
	}

	// The two signature patterns are compiled by consumers; validate them as
	// the regexes they will be.
	identity := skillIdentityTokens(skill)
	for _, f := range []struct{ name, pattern string }{
		{"error_message_regex", rawError},
		{"command_pattern", cmdPat},
	} {
		if f.pattern == "" {
			continue
		}
		re, err := regexp.Compile(f.pattern)
		if err != nil {
			return fmt.Errorf("%s %q does not compile: %v", f.name, f.pattern, err)
		}
		if re.MatchString("") {
			return fmt.Errorf("%s %q matches the empty string", f.name, f.pattern)
		}
		if f.name != "error_message_regex" {
			continue
		}
		for _, tok := range identity {
			// The anchor is what the live matcher substrings into every
			// planned command; the whole-pattern match catches a universal
			// regex whose first field looks harmless ("hcloud.*").
			if strings.EqualFold(firstToken(f.pattern), tok) || strings.EqualFold(re.String(), tok) || re.MatchString(tok) {
				return fmt.Errorf("%s %q is anchored on the skill's own invocation token %q, so it matches every command", f.name, f.pattern, tok)
			}
		}
	}
	return nil
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
		// provenance/verified let a reader tell a machine-learned entry from a
		// curated one: trace-derived entries are written by
		// `learning trace aggregate` from runtime traces (untrusted until
		// ValidateTracePattern accepted them) and carry verified=false, while
		// the curated knowledge base (internal/learning/knowledge.go) has
		// neither field.
		"provenance": "trace",
		"verified":   false,
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

// TraceFile is one scanned trace: its path plus the decoded payload.
type TraceFile struct {
	Path string
	Data map[string]any
}

// ScanTraces returns all valid trace files for a skill (gcl-trace-*.json from
// the GCL runner and orchestrator-trace-*.json from the L4 orchestrator),
// optionally filtered by mtime.
//
// Every candidate is classified (see ClassifyTrace) before it is returned: a
// trace that fails canonical-schema validation is dropped with one WARN naming
// the file and the first violation, because an unverifiable trace must not be
// allowed to move a metric in any caller. Smoke traces are returned — they are
// a classification callers make (IsSmokeTrace), not a read error.
func ScanTraces(root, skill string, sinceHours *int) []TraceFile {
	tracesDir := filepath.Join(root, "audit-results")
	entries, err := os.ReadDir(tracesDir)
	if err != nil {
		return nil
	}
	var out []TraceFile
	now := time.Now().UTC()
	for _, e := range entries {
		if e.IsDir() || !IsTraceFileName(e.Name()) {
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
			// ScanTraces has no AggregateResult to bump — the caller learns
			// the miss from the empty slice. The CLI summary path (cmd/
			// aggregate.go aggregateTraces) counts these directly off the
			// raw file path, so the operator still sees them.
			continue
		}
		if class, errs := ClassifyTrace(raw, data); class == TraceInvalid {
			traceWarn(e.Name(), fmt.Sprintf("invalid trace (excluded from every metric): %s", errs[0]))
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
	// SkippedSmoke counts traces rejected as smoke (no `final` block, a smoke
	// request/fault, or zero executed steps). They are excluded from Scanned
	// and therefore from meta.source_traces_analyzed — a smoke trace proves
	// nothing about the skill, so counting it would fake loop health.
	SkippedSmoke int
	// SkippedSkillMismatch counts traces dropped because their top-level
	// `skill` field did not match the --skill filter. The previous behavior
	// was a silent `continue` ahead of res.Scanned++, which produced
	// source_traces_analyzed=0 for every skill when the writer's attribution
	// was wrong (the L4 writer used the literal "unknown" whenever a fault
	// matched no keyword rule). Counting it makes a misconfigured writer
	// visible in the CLI summary instead of hiding behind "everything is 0".
	SkippedSkillMismatch int
	// InvalidTraces counts traces dropped by canonical-schema validation (see
	// ClassifyTrace). They are excluded from Scanned, from every metric, and
	// from failure-pattern merging: a trace-shaped file that does not conform
	// to the schema is untrusted input, not a low-quality observation.
	InvalidTraces int
	// RejectedPatterns counts failure_pattern blocks that ValidateTracePattern
	// refused to merge. Each rejection is one WARN; the count makes a burst of
	// crafted traces visible in the CLI output instead of only in the log.
	RejectedPatterns int
	WrittenTo        string
	// EmptyLoop is set when this Aggregate call did not increment
	// source_traces_analyzed at all (res.Scanned == 0) AND there was at
	// least one trace file under audit-results/ that the loop considered.
	// The trigger for an operator-visible WARN: a non-empty trace set that
	// produced zero learnable signal means the learning loop is broken —
	// most commonly because every trace carried skill="unknown" and
	// SkippedSkillMismatch swallowed them all. The flag is independent of
	// dry-run so a preview of an empty loop is also loud.
	EmptyLoop bool
}

// Aggregate runs the loop: scan → classify → extract → dedup → merge → write.
//
// Classification order per trace file is frozen (see ClassifyTrace): parse →
// schema-invalid (InvalidTraces) → smoke (SkippedSmoke) → evidence (Scanned).
// Only evidence traces reach the failure-pattern merge, and even then the
// extracted block must pass ValidateTracePattern — an audit-results/ file is
// untrusted input, and a merged pattern can skip planned steps.
//
// Stream implementation: each trace file (gcl-trace-*.json or
// orchestrator-trace-*.json) is read, parsed, and consumed one at a time. The
// full trace map and its raw JSON go out of scope after we've extracted the
// failure_pattern block — peak memory is O(1 trace) rather than
// O(N traces × ~50 KB). For ScanTraces callers (test fixtures, ad-hoc tools)
// the full slice is still available; production aggregate traffic goes through
// this function only.
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
	// traceFilesConsidered counts every file the loop examined — used by
	// the EmptyLoop alarm to distinguish "no trace files at all" (the
	// audit-results/ directory was empty or absent; nothing to do, no
	// alarm) from "trace files exist but none contributed" (the silent
	// attribute bug — this is the case that produced the
	// source_traces_analyzed=0 deadlock).
	traceFilesConsidered := 0
	if dirErr == nil {
		now := time.Now().UTC()
		// Sorted for deterministic iteration order — matches the prior
		// ScanTraces behavior so callers relying on trace order see the
		// same sequence.
		sort.Slice(entries, func(i, j int) bool { return entries[i].Name() < entries[j].Name() })
		for _, e := range entries {
			if e.IsDir() || !IsTraceFileName(e.Name()) {
				continue
			}
			traceFilesConsidered++
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
			if trace["skill"] != skill {
				// Writer-side attribution (l4.HandleFault resolveTraceSkill)
				// or a typo in --skill made this trace invisible to the
				// learner. Count it so a burst of misattributed traces is
				// visible in the CLI summary, and the operator can see
				// "loop health = 0 because writer output the wrong skill"
				// instead of just "loop health = 0".
				res.SkippedSkillMismatch++
				continue
			}
			// Classification order is frozen: parse (done above) →
			// schema-invalid → smoke → evidence. Validation runs on the raw
			// bytes, before `raw` is dropped, so the schema sees exactly what
			// was written above.
			class, schemaErrs := ClassifyTrace(raw, trace)
			// Drop the raw bytes now — they were only needed for Unmarshal and
			// classification.
			raw = nil
			switch class {
			case TraceInvalid:
				// A non-conforming trace is untrusted input: it must not reach
				// the failure-pattern merge (see ValidateTracePattern).
				res.InvalidTraces++
				traceWarn(e.Name(), fmt.Sprintf("invalid trace (excluded from every metric and from learning): %s", schemaErrs[0]))
				continue
			case TraceSmoke:
				// Smoke traces prove nothing (no `final`, smoke request/fault,
				// or zero steps): keep them out of source_traces_analyzed and
				// out of failure_patterns entirely.
				res.SkippedSmoke++
				continue
			}
			res.Scanned++
			traceName := e.Name()
			patternFp := ExtractPatternFromTrace(trace)
			if patternFp == nil {
				res.SkippedCount++
				continue
			}
			if vErr := ValidateTracePattern(patternFp, skill); vErr != nil {
				res.RejectedPatterns++
				traceWarn(traceName, fmt.Sprintf("rejected untrusted failure pattern (not merged): %v", vErr))
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

	// EmptyLoop alarm: a non-empty trace set that consumed zero signal is
	// the exact shape the silent-attribute bug produced — and the L4 writer
	// emitted skill="unknown" on every fault that matched no keyword rule,
	// so SkippedSkillMismatch would have absorbed them all. Firing this in
	// dry-run too means a preview of an empty loop is loud before any
	// failure_patterns.json is touched. The message names the four common
	// causes (zero trace files, all skill-mismatch, all smoke, all
	// schema-invalid) so the operator does not have to guess.
	if traceFilesConsidered > 0 && res.Scanned == 0 {
		res.EmptyLoop = true
		fmt.Fprintf(os.Stderr,
			"WARN: learning loop consumed 0 traces for skill %q (considered %d file(s) under %s; skipped_skill_mismatch=%d, skipped_smoke=%d, invalid_trace=%d). failure_patterns.json will NOT grow this run. Possible causes: writer emits skill=\"unknown\"; --skill typo; every trace was smoke/safety-irrelevant; every trace is schema-invalid. Inspect with `hwcloud-skillcheck aggregate trace --root %s`.\n",
			skill, traceFilesConsidered, tracesDir,
			res.SkippedSkillMismatch, res.SkippedSmoke, res.InvalidTraces, root)
	}

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

// LoadFailurePatterns returns the merged view of the seed definitions
// (failure_patterns.seed.json) and the runtime overlay (failure_patterns.json),
// or a fresh scaffold when neither exists (seed/runtime KB split, spec #T3):
// per ID the definitions come from the seed, stats/learned_from from the
// overlay; overlay-only (trace-derived) entries are appended whole. With no
// seed file the overlay alone is the document — legacy single-file skills
// keep their exact pre-split behavior.
func LoadFailurePatterns(root, skill string) map[string]any {
	skillID := "huaweicloud-" + strings.ReplaceAll(skill, "huaweicloud-", "")
	skillID = strings.ReplaceAll(skillID, "-ops", "") // tolerate bare shortname
	if !strings.HasPrefix(skill, "huaweicloud-") {
		skillID = "huaweicloud-" + skill + "-ops"
	} else {
		skillID = skill
	}
	dir := filepath.Join(root, skillID, "assets")
	scaffold := func() map[string]any {
		return map[string]any{
			"$schema":  "failure-patterns/v1",
			"skill_id": skillID,
			"patterns": []any{},
			"meta": map[string]any{
				// float64 keeps scaffold counters type-identical to the
				// JSON-decoded ones (encoding/json yields float64), so
				// consumers can assert one numeric type in both worlds.
				"total_patterns":         float64(0),
				"last_aggregation":       NowISO(),
				"source_traces_analyzed": float64(0),
			},
		}
	}
	readDoc := func(name string) map[string]any {
		raw, err := os.ReadFile(filepath.Join(dir, name))
		if err != nil {
			return nil
		}
		var doc map[string]any
		if json.Unmarshal(raw, &doc) != nil {
			return nil
		}
		return doc
	}
	overlay := readDoc("failure_patterns.json")
	seed := readDoc("failure_patterns.seed.json")
	if seed == nil {
		if overlay != nil {
			return overlay
		}
		return scaffold()
	}
	seedPats, _ := seed["patterns"].([]any)
	ovPats := []any{}
	if overlay != nil {
		overlay["patterns"], _ = overlay["patterns"].([]any)
		ovPats, _ = overlay["patterns"].([]any)
	}
	ovByID := map[string]map[string]any{}
	for _, p := range ovPats {
		if pm, ok := p.(map[string]any); ok {
			if id, _ := pm["id"].(string); id != "" {
				ovByID[id] = pm
			}
		}
	}
	merged := make([]any, 0, len(seedPats)+len(ovPats))
	seedIDs := map[string]bool{}
	for _, p := range seedPats {
		pm, ok := p.(map[string]any)
		if !ok {
			continue
		}
		id, _ := pm["id"].(string)
		seedIDs[id] = true
		if ov, ok := ovByID[id]; ok {
			// Authority split: definitions from seed, runtime stats from overlay.
			if st, ok := ov["stats"]; ok {
				pm["stats"] = st
			}
			if lf, ok := ov["learned_from"]; ok {
				pm["learned_from"] = lf
			}
		}
		merged = append(merged, pm)
	}
	for _, p := range ovPats {
		pm, ok := p.(map[string]any)
		if !ok {
			continue
		}
		if id, _ := pm["id"].(string); seedIDs[id] {
			continue
		}
		merged = append(merged, p) // trace-derived entry, whole
	}
	var meta map[string]any
	if overlay != nil {
		meta, _ = overlay["meta"].(map[string]any)
	}
	if meta == nil {
		meta = scaffold()["meta"].(map[string]any)
	}
	meta["total_patterns"] = float64(len(merged))
	return map[string]any{
		"$schema":  "failure-patterns/v1",
		"skill_id": skillID,
		"patterns": merged,
		"meta":     meta,
	}
}

// SaveFailurePatterns persists the merged view back to the runtime overlay
// (failure_patterns.json). Seed-owned IDs are reduced to their runtime fields
// so curated definitions are never duplicated into the overlay (#T4);
// non-seed entries are written whole. The seed file is never touched.
func SaveFailurePatterns(root, skill string, data map[string]any) (string, error) {
	skillID := skill
	if !strings.HasPrefix(skill, "huaweicloud-") {
		skillID = "huaweicloud-" + skill + "-ops"
	}
	dir := filepath.Join(root, skillID, "assets")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	seedIDs := map[string]bool{}
	if raw, err := os.ReadFile(filepath.Join(dir, "failure_patterns.seed.json")); err == nil {
		var seed struct {
			Patterns []struct {
				ID string `json:"id"`
			} `json:"patterns"`
		}
		if json.Unmarshal(raw, &seed) == nil {
			for _, sp := range seed.Patterns {
				seedIDs[sp.ID] = true
			}
		}
	}
	pats, _ := data["patterns"].([]any)
	out := make([]any, 0, len(pats))
	for _, p := range pats {
		pm, ok := p.(map[string]any)
		if !ok {
			out = append(out, p)
			continue
		}
		id, _ := pm["id"].(string)
		if !seedIDs[id] {
			out = append(out, p)
			continue
		}
		partial := map[string]any{"id": id}
		if st, ok := pm["stats"]; ok {
			partial["stats"] = st
		}
		if lf, ok := pm["learned_from"]; ok {
			switch v := lf.(type) {
			case []any:
				if len(v) > 0 {
					partial["learned_from"] = v
				}
			case []string:
				if len(v) > 0 {
					partial["learned_from"] = v
				}
			}
		}
		out = append(out, partial)
	}
	meta, _ := data["meta"].(map[string]any)
	if meta == nil {
		meta = map[string]any{}
	}
	meta["last_aggregation"] = NowISO()
	meta["total_patterns"] = float64(len(pats))
	data["patterns"] = out
	data["meta"] = meta
	p := filepath.Join(dir, "failure_patterns.json")
	if err := writeJSON(p, data); err != nil {
		return "", err
	}
	return p, nil
}
