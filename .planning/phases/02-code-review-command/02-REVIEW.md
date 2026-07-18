---
phase: 02-code-review-command
reviewed: 2026-07-18T22:00:00Z
depth: deep
files_reviewed: 20
files_reviewed_list:
  - skillcheck/main.go
  - skillcheck/cmd/root.go
  - skillcheck/cmd/validate.go
  - skillcheck/cmd/validate_eval.go
  - skillcheck/cmd/validate_repo.go
  - skillcheck/cmd/check.go
  - skillcheck/cmd/scan.go
  - skillcheck/cmd/aggregate.go
  - skillcheck/cmd/lint.go
  - skillcheck/internal/schema/schema.go
  - skillcheck/internal/security/security.go
  - skillcheck/internal/yaml/yaml.go
  - skillcheck/internal/coverage/coverage.go
  - skillcheck/internal/embed/embed.go
  - skillcheck/cmd/cmd_test.go
  - skillcheck/cmd/aggregate_test.go
  - skillcheck/cmd/check_test.go
  - skillcheck/cmd/scan_test.go
  - skillcheck/cmd/validate_repo_test.go
  - skillcheck/internal/schema/schema_test.go
  - skillcheck/internal/security/security_test.go
  - skillcheck/internal/yaml/yaml_test.go
  - skillcheck/internal/coverage/coverage_test.go
  - skillcheck/internal/embed/embed_test.go
  - skillcheck/testdata/equivalence_test.py
  - docs/superpowers/specs/skillcheck-cli.md
findings:
  critical: 2
  warning: 6
  info: 5
  total: 13
status: issues_found
---

# Phase 2: Code Review Report

**Reviewed:** 2026-07-18T22:00:00Z
**Depth:** deep
**Files Reviewed:** 20 source + 10 test + 2 docs
**Status:** issues_found

## Summary

The skillcheck Go CLI is a well-structured migration of ~5000 lines of Python validation scripts into a single Go binary. The codebase follows Go conventions well overall, has good test coverage for a v1, and demonstrates clear understanding of the domain. However, several issues were found: two CRITICAL bugs (incorrect `rune` indexing for multi-byte characters, and a `gofmt -w` flag that silently skips listing), six WARNING-level issues (error handling gaps, dead code, test coverage holes), and five INFO items (naming, duplication, unused code).

The most severe bug is in `runeAt()` which treats `s[i]` as a rune but only extracts a single byte — this will produce incorrect results for any multi-byte UTF-8 character in check_example_config placeholder detection. The second CRITICAL bug is in the lint command where `--fix` replaces `-l` with `-w`, losing the listing output that feeds the failure detection logic.

## Critical Issues

### CR-01: runeAt() byte indexing produces wrong rune for multi-byte characters

**File:** `skillcheck/cmd/check.go:197-202`
**Issue:** `runeAt()` converts `s[i]` (a single byte) to `rune`, but `s[i]` is only one byte of a potentially multi-byte UTF-8 sequence. In Go, indexing a string with `[]` yields the byte at that position, not a rune. When `s[i]` falls in the middle of a multi-byte character, `rune(s[i])` will produce a garbage rune (typically an invalid continuation byte like 0x80-0xBF decoded as a standalone rune).

This function is used in `checkExamplePlaceholders` (line 182-184) to detect whether a `{` is preceded by another `{` or a `}` is followed by another `}`. If the YAML file contains any multi-byte characters (Chinese comments, emoji, non-ASCII punctuation) at the wrong offset, the check will malfunction.

**Fix:**
```go
func runeAt(s string, i int) rune {
    if i < 0 || i >= len(s) {
        return 0
    }
    r, _ := utf8.DecodeRuneInString(s[i:])
    return r
}
```

Also add `"unicode/utf8"` to the import block.

### CR-02: lint --fix mode silently skips failure detection

**File:** `skillcheck/cmd/lint.go:46-65`
**Issue:** When `--fix` is set, the code replaces `-l` with `-w` (line 48), which causes `gofmt` to rewrite files in-place and produce no stdout output. The `if !*fix { ... }` block (line 58) skips parsing the output for failures. However, the code still appends `rootDir` and runs `gofmt -w <dir>` which could fail silently — if `gofmt -w` encounters errors (e.g., unparseable Go files), it prints to stderr but the exit code is checked. However, after `--fix` mode, no failures are ever added even if `gofmt` had issues.

More critically, when `--fix` is used, the `go vet` step still runs and can produce failures, but the failure collection only captures `go vet` output. The `gofmt` failures in `--fix` mode are not tracked, but the command still exits 0 from gofmt's perspective.

**Fix:**
```go
formatArgs := []string{"-l"}
if *fix {
    formatArgs = []string{"-l"} // Still use -l to list files that need formatting
    // Apply -w separately: run gofmt -w on those files
}
```

Or better, restructure to always run with `-l` first, then conditionally apply `-w`:
```go
// First, list files needing formatting
listCmd := exec.Command(gofmtBin, "-l", rootDir)
var listOut bytes.Buffer
listCmd.Stdout = &listOut
listCmd.Stderr = os.Stderr
if err := listCmd.Run(); err != nil {
    return fmt.Errorf("lint go: gofmt failed: %w", err)
}
needsFormat := strings.TrimSpace(listOut.String())
if needsFormat != "" {
    for _, line := range strings.Split(needsFormat, "\n") {
        if line != "" {
            failures = append(failures, "gofmt: "+line)
        }
    }
    if *fix {
        fixCmd := exec.Command(gofmtBin, "-w", rootDir)
        fixCmd.Stderr = os.Stderr
        if err := fixCmd.Run(); err != nil {
            return fmt.Errorf("lint go: gofmt -w failed: %w", err)
        }
    }
}
```

## Warnings

### WR-01: Unused marshalJSON helper function

**File:** `skillcheck/cmd/validate_eval.go:12-14`
**Issue:** `marshalJSON(v)` is a one-liner that simply calls `json.Marshal(v)`. It's used only once (in `validateEvalQueriesFile` at line 231) and adds no value over calling `json.Marshal` directly. In `validate.go:162`, the same pattern is handled with a direct `json.Marshal(item)` call without a wrapper. This is dead code that adds a maintenance burden.

**Fix:** Remove the function and replace its single call site with `json.Marshal(item)`.

### WR-02: detectEvalFormat function is dead code (duplicated logic)

**File:** `skillcheck/cmd/validate_eval.go:19-49`
**Issue:** `detectEvalFormat()` is a near-exact duplicate of the logic already present in `validateEvalQueries()` in `validate.go:136-186`. Both functions parse JSON with UseNumber, detect array vs object format, identify the $def name, and return errors for unrecognized shapes. `detectEvalFormat` is only called from `validateEvalQueriesFile` (validate_repo.go:222), which is itself called from `runValidateEvalQueries`. The function in `validate.go:136` (`validateEvalQueries`) is the top-level schema validation path and is never called when going through `validateEvalQueriesFile`.

This means there are TWO code paths for eval-queries validation with duplicated logic, and any fix to one must be manually synced to the other.

**Fix:** Consolidate the detection logic into a single shared function. Move the format-detection into `validate_eval.go` and have both `validateEvalQueries` and `validateEvalQueriesFile` call it.

### WR-03: Error silently swallowed in discoverSkillDirs

**File:** `skillcheck/cmd/check.go:101-114`
**Issue:** `discoverSkillDirs` silently returns nil on `os.ReadDir` error (line 103: `return nil`), which means if the root directory doesn't exist or is unreadable, the function returns an empty list with no error. This is called from `runCheckExampleConfig` (line 73), which will then process zero skills and return nil (success). The user gets no indication that the root path was invalid.

**Fix:** Either return the error from `discoverSkillDirs`, or log a warning:
```go
func discoverSkillDirs(root string) ([]string, error) {
    entries, err := os.ReadDir(root)
    if err != nil {
        return nil, err
    }
    ...
}
```
Then update the caller to handle the error.

### WR-04: Missing test for ScanJSON function

**File:** `skillcheck/internal/security/security_test.go`
**Issue:** `ScanJSON` (security.go:147-157) has no test coverage. This function is exported and designed for structured JSON scanning (used by scan.go for artifact scanning). While `ScanContent` is well-tested (13 test cases), `ScanJSON` which walks the JSON tree recursively has zero tests.

The `walkValue` function (security.go:164-188) and `scanStringField` (190-207) are only exercised through `ScanJSON`, so they are also uncovered.

**Fix:** Add table-driven tests for `ScanJSON`:
```go
func TestScanJSON(t *testing.T) {
    cases := []struct {
        name    string
        input   []byte
        want    int // expected number of findings
        wantTyp string // expected finding type
    }{
        {"clean json", []byte(`{"a":"hello"}`), 0, ""},
        {"leak in value", []byte(`{"request":"SK=abcdefghijklmnopqrstuvwxyz012345"}`), 1, "sk"},
        // nested objects, arrays, etc.
    }
    for _, tc := range cases {
        t.Run(tc.name, func(t *testing.T) {
            findings, err := ScanJSON(tc.input)
            if err != nil { t.Fatal(err) }
            if len(findings) != tc.want {
                t.Errorf("got %d findings, want %d", len(findings), tc.want)
            }
        })
    }
}
```

### WR-05: Error ignored from security.ScanContent

**File:** `skillcheck/cmd/scan.go:197`
**Issue:** `security.ScanContent(data)` returns `([]Finding, error)` but the error is discarded with `_`:
```go
findings, _ := security.ScanContent(data)
```
If the security scanner returns an error (e.g., from regex compilation or internal failure), it's silently lost. While the current implementation of `ScanContent` always returns `nil` for the error, this is an interface design issue: if a future implementation returns a non-nil error, it will be silently swallowed.

**Fix:** Check and handle the error:
```go
findings, err := security.ScanContent(data)
if err != nil {
    res.error = fmt.Sprintf("scan error: %v", err)
    results = append(results, res)
    continue
}
```

Same issue exists in `scan.go:255` for the `--self-check` path.

### WR-06: aggregate.go: splitComma is a pointless wrapper around splitOnComma

**File:** `skillcheck/cmd/aggregate.go:303-309`
**Issue:** `splitComma(s)` calls `splitOnComma(s)` and does an identity pass — iterating over the results and appending each one to a new slice. The two functions are identical in behavior but `splitComma` adds allocation and indirection for no reason. The `splitOnComma` function already does the work correctly.

**Fix:** Replace all calls to `splitComma` with `splitOnComma` and remove `splitComma`:
```go
dims := splitOnComma(rubricDims)
statuses := splitOnComma(finalStatuses)
```

## Info

### IN-01: itoa helper duplicates strconv.Itoa

**File:** `skillcheck/internal/coverage/coverage.go:217-237`
**Issue:** A custom `itoa()` function is implemented using byte buffer arithmetic. Go's standard library `strconv.Itoa()` exists for this purpose and should be preferred. The custom implementation also has a subtle issue: for `n = math.MinInt`, the negation `n = -n` would overflow (though this edge case is unlikely given the domain of security marker counts).

**Fix:**
```go
import "strconv"
// Replace itoa(n) calls with strconv.Itoa(n)
```

### IN-02: hasKey helper is only used in validate.go

**File:** `skillcheck/cmd/validate.go:127-130`
**Issue:** `hasKey(m map[string]any, key string) bool` is a one-liner that wraps `_, ok := m[key]`. It's only used in `evalArrayDefFor` and `evalObjectDefFor` (validate.go:100, 117, 118, 121). In the rest of the codebase, the same pattern is done inline. This is a trivial wrapper that adds a level of indirection without improving readability.

**Fix:** Inline the check or keep it if preferred for readability (minor style choice).

### IN-03: validate_eval.go file exists with only dead/redundant code

**File:** `skillcheck/cmd/validate_eval.go`
**Issue:** After removing `marshalJSON` and `detectEvalFormat`, this file would contain only `productAssessmentBlockRe`, `productPillars`, `productStatuses`, `requiredAssessmentTop`, `validateProductAssessment`, and `validateAssessmentObject`. However, `productAssessmentBlockRe` through `requiredAssessmentTop` are variables, not functions — they're used by `validateProductAssessment` and `validateAssessmentObject`. The naming `validate_eval.go` is misleading since most of its content is about product-assessment, not eval-queries. The eval-queries functions (`detectEvalFormat`) are dead code.

**Fix:** Rename the file to something like `validate_assessment.go` and remove the dead `detectEvalFormat` / `marshalJSON` functions.

### IN-04: Duplicated schema decode closure in schema.go

**File:** `skillcheck/internal/schema/schema.go:280-288 and 326-334`
**Issue:** Both `ValidateDef` and `ValidateFile` define an identical inner function `decode` that creates a `json.NewDecoder` with `UseNumber` and decodes a byte slice. This is a 7-line closure repeated verbatim in two places.

**Fix:** Extract to a package-level helper:
```go
func decodeJSON(b []byte) (any, error) {
    dec := json.NewDecoder(bytes.NewReader(b))
    dec.UseNumber()
    var v any
    if err := dec.Decode(&v); err != nil {
        return nil, err
    }
    return v, nil
}
```

### IN-05: numOf helper in aggregate_test.go duplicates intOf from aggregate.go

**File:** `skillcheck/cmd/aggregate_test.go:139-150`
**Issue:** `numOf(v)` in the test file is an exact copy of `intOf(v)` from aggregate.go. Tests should reuse the production function rather than duplicating it, unless there's a specific reason to test independently.

**Fix:** Use `intOf` from the production code in tests (it's in the same package, so it's accessible):
```go
if intOf(totals["total_runs"]) != 3 {
```

---

_Reviewed: 2026-07-18T22:00:00Z_
_Reviewer: Claude (gsd-code-reviewer)_
_Depth: deep_