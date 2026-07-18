package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// runValidateContract dispatches `skillcheck validate generator-contract`,
// `skillcheck validate safety-class`, and `skillcheck validate resource-scope`.
func runValidateContract(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("validate: missing subcommand (generator-contract|safety-class|resource-scope)")
	}
	switch args[0] {
	case "generator-contract":
		return runValidateGeneratorContract(args[1:])
	case "safety-class":
		return runValidateSafetyClass(args[1:])
	case "resource-scope":
		return runValidateResourceScope(args[1:])
	case "-h", "--help", "help":
		fmt.Fprintln(os.Stdout, "skillcheck validate <generator-contract|safety-class|resource-scope> --root <dir>")
		return nil
	default:
		return fmt.Errorf("validate: unknown subcommand %q", args[0])
	}
}

// ---------------------------------------------------------------------------
// validate generator-contract (L2-B)
// ---------------------------------------------------------------------------

// contractCheckItem represents one contract check.
type contractCheckItem struct {
	Scope string `json:"scope"`
	Item  string `json:"item"`
	OK    bool   `json:"ok"`
}

// generatorContractReport is the full report.
type generatorContractReport struct {
	OK       bool                `json:"ok"`
	Summary  contractSummary     `json:"summary"`
	Checks   []contractCheckItem `json:"checks"`
	Failures []contractFailure   `json:"failures"`
}

type contractSummary struct {
	Total   int `json:"total"`
	Passing int `json:"passing"`
	Failing int `json:"failing"`
}

type contractFailure struct {
	Scope  string `json:"scope"`
	Item   string `json:"item"`
	Reason string `json:"reason"`
}

// contractItems defines all the regex patterns to check.
var contractItems = []struct {
	scope, item, pattern string
}{
	{"template", "metadata.gcl.required", `(?m)^  gcl:\n    required: true`},
	{"template", "metadata.gcl.default_max_iter", `(?m)^    default_max_iter: 2`},
	{"template", "metadata.gcl.rubric_version", `(?m)^    rubric_version: "v1"`},
	{"template", "metadata.gcl.trace_path", `audit-results/gcl-trace-YYYYMMDD-HHMMSS\.json`},
	{"template", "quality_gate_heading", `(?m)^## Quality Gate \(GCL\)$`},
	{"template", "rubric_artifact", `references/rubric\.md`},
	{"template", "prompt_templates_artifact", `references/prompt-templates\.md`},
	{"template", "operation_intent", `operation_intent`},
	{"template", "shared_backbone_reference", `gcl-prompt-backbone\.md`},
	{"generator_skill", "compat_backbone", `references/gcl-prompt-backbone\.md`},
	{"generator_skill", "output_rubric", "\x60references/rubric\\.md\x60"},
	{"generator_skill", "output_prompt_templates", "\x60references/prompt-templates\\.md\x60"},
	{"generator_skill", "metadata_gcl_instruction", "\x60metadata\\.gcl\x60"},
	{"backbone", "generator_section", `(?m)^## 1\. Generator prompt template$`},
	{"backbone", "critic_section", `(?m)^## 2\. Critic prompt template$`},
	{"backbone", "orchestrator_section", `(?m)^## 3\. Orchestrator prompt template$`},
	{"backbone", "hcloud_primary", `PRIMARY: hcloud`},
	{"backbone", "go_sdk_fallback", `huaweicloud-sdk-go-v3`},
	{"backbone", "operation_intent", `\{\{output\.operation_intent\}\}`},
	{"backbone", "critic_no_raw_request", `Do NOT consider the original user request`},
	{"backbone", "critic_read_only", `read-only`},
	{"backbone", "trace_persistence", `audit-results/gcl-trace-YYYYMMDD-HHMMSS\.json`},
}

var requiredContractFiles = map[string]string{
	"template":        "huaweicloud-skill-generator/references/huaweicloud-skill-template.md",
	"generator_skill": "huaweicloud-skill-generator/SKILL.md",
	"backbone":        "huaweicloud-skill-generator/references/gcl-prompt-backbone.md",
}

func readContractFile(root, scope string) string {
	rel, ok := requiredContractFiles[scope]
	if !ok {
		return ""
	}
	path := filepath.Join(root, rel)
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	return string(data)
}

func hasBarePlaceholdersContract(text string) bool {
	text = stripComments(text)
	allowed := regexp.MustCompile(`\{\{\s*(env|user|output)\.[^{}]+\}\}`).ReplaceAllString(text, "")
	allowed = regexp.MustCompile(`\$\{[A-Z_][A-Z0-9_]*\}`).ReplaceAllString(allowed, "")
	// Strip escaped {{...}} and then check for bare {placeholders}
	allowed = regexp.MustCompile(`\{\{[^}]+\}\}`).ReplaceAllString(allowed, "")
	return regexp.MustCompile(`\{[a-zA-Z_][a-zA-Z0-9_.-]*\}`).MatchString(allowed)
}

func stripComments(text string) string {
	var lines []string
	for _, line := range strings.Split(text, "\n") {
		if !strings.HasPrefix(strings.TrimSpace(line), "#") {
			lines = append(lines, line)
		}
	}
	return strings.Join(lines, "\n")
}

func checkGeneratorContract(root string) generatorContractReport {
	texts := map[string]string{}
	var failures []contractFailure

	// Check file existence
	for scope, rel := range requiredContractFiles {
		path := filepath.Join(root, rel)
		if !fileExists(path) {
			failures = append(failures, contractFailure{
				Scope:  scope,
				Item:   "file_exists",
				Reason: "missing file",
			})
			texts[scope] = ""
		} else {
			texts[scope] = readFileString(path)
		}
	}

	var checks []contractCheckItem

	// Pattern checks
	for _, ci := range contractItems {
		text := texts[ci.scope]
		ok := regexp.MustCompile(ci.pattern).MatchString(text)
		checks = append(checks, contractCheckItem{Scope: ci.scope, Item: ci.item, OK: ok})
		if !ok {
			failures = append(failures, contractFailure{
				Scope:  ci.scope,
				Item:   ci.item,
				Reason: fmt.Sprintf("pattern not found: %s", ci.pattern),
			})
		}
	}

	// Bare placeholder checks for template and backbone
	for _, scope := range []string{"template", "backbone"} {
		text := texts[scope]
		if text != "" {
			ok := !hasBarePlaceholdersContract(text)
			checks = append(checks, contractCheckItem{Scope: scope, Item: "no_bare_placeholders", OK: ok})
			if !ok {
				failures = append(failures, contractFailure{
					Scope:  scope,
					Item:   "no_bare_placeholders",
					Reason: "bare {placeholder} detected",
				})
			}
		}
	}

	passing := 0
	for _, c := range checks {
		if c.OK {
			passing++
		}
	}

	return generatorContractReport{
		OK:       len(failures) == 0,
		Summary:  contractSummary{Total: len(checks), Passing: passing, Failing: len(checks) - passing},
		Checks:   checks,
		Failures: failures,
	}
}

func mapGet(m map[string]string, key string) string {
	if v, ok := m[key]; ok {
		return v
	}
	return ""
}

func runValidateGeneratorContract(args []string) error {
	fs := newFlagSet("skillcheck validate generator-contract")
	root := fs.String("root", ".", "skill repository root")
	jsonOut := fs.Bool("json", false, "emit JSON report")
	if err := fs.Parse(args); err != nil {
		return err
	}
	rootDir, err := filepath.Abs(*root)
	if err != nil {
		return err
	}

	report := checkGeneratorContract(rootDir)

	if *jsonOut {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		enc.Encode(report)
	} else {
		fmt.Printf("Generator GCL contract: %d/%d checks pass.\n", report.Summary.Passing, report.Summary.Total)
		for _, f := range report.Failures {
			fmt.Printf("  FAIL %s.%s: %s\n", f.Scope, f.Item, f.Reason)
		}
	}

	if !report.OK {
		return fmt.Errorf("generator-contract: %d check(s) failed", report.Summary.Failing)
	}
	return nil
}

// ---------------------------------------------------------------------------
// validate safety-class (L3-A)
// ---------------------------------------------------------------------------

// safetyClassResult holds the result of safety_class enum validation.
type safetyClassResult struct {
	SchemaOK     bool     `json:"schema_ok"`
	CodeOK       bool     `json:"code_ok"`
	DocsOK       bool     `json:"docs_ok"`
	TracesOK     bool     `json:"traces_ok"`
	SchemaErrors []string `json:"schema_errors"`
	CodeErrors   []string `json:"code_errors"`
	DocsErrors   []string `json:"docs_errors"`
	TracesErrors []string `json:"traces_errors"`
	OK           bool     `json:"ok"`
}

var expectedSafetyClassValues = []string{"read-only", "mutating", "destructive"}

func runValidateSafetyClass(args []string) error {
	fs := newFlagSet("skillcheck validate safety-class")
	root := fs.String("root", ".", "skill repository root")
	jsonOut := fs.Bool("json", false, "emit JSON report")
	if err := fs.Parse(args); err != nil {
		return err
	}
	rootDir, err := filepath.Abs(*root)
	if err != nil {
		return err
	}

	// skillcheckRoot = where the skillcheck module lives.
	// Tests can override via SKILLCHECK_ROOT env var.
	skillcheckRoot := os.Getenv("SKILLCHECK_ROOT")
	if skillcheckRoot == "" {
		skillcheckRoot = filepath.Dir(filepath.Dir(os.Args[0]))
		if filepath.Base(os.Args[0]) == "skillcheck" {
			skillcheckRoot = filepath.Dir(os.Args[0])
		}
	}

	result := checkSafetyClassEnum(skillcheckRoot, rootDir)

	if *jsonOut {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		enc.Encode(result)
	} else {
		for _, e := range result.SchemaErrors {
			fmt.Printf("  FAIL schema: %s\n", e)
		}
		for _, e := range result.CodeErrors {
			fmt.Printf("  FAIL code: %s\n", e)
		}
		for _, e := range result.DocsErrors {
			fmt.Printf("  FAIL docs: %s\n", e)
		}
		for _, e := range result.TracesErrors {
			fmt.Printf("  FAIL traces: %s\n", e)
		}
		if result.OK {
			fmt.Println("[safety_class enum] OK")
		} else {
			fmt.Printf("[safety_class enum] FAIL: %d issue(s)\n",
				len(result.SchemaErrors)+len(result.CodeErrors)+len(result.DocsErrors)+len(result.TracesErrors))
		}
	}

	if !result.OK {
		return fmt.Errorf("safety-class: validation failed")
	}
	return nil
}

func checkSafetyClassEnum(skillcheckRoot, root string) safetyClassResult {
	var result safetyClassResult

	result.SchemaErrors = checkSafetyClassSchema(root)
	result.CodeErrors = checkSafetyClassCode(skillcheckRoot)
	result.DocsErrors = checkSafetyClassDocs(root)
	result.TracesErrors = checkSafetyClassTraces(root)

	result.SchemaOK = len(result.SchemaErrors) == 0
	result.CodeOK = len(result.CodeErrors) == 0
	result.DocsOK = len(result.DocsErrors) == 0
	result.TracesOK = len(result.TracesErrors) == 0
	result.OK = result.SchemaOK && result.CodeOK && result.DocsOK && result.TracesOK

	return result
}

func checkSafetyClassSchema(root string) []string {
	var errors []string
	schemaPath := filepath.Join(root, "huaweicloud-ces-ops/assets/gcl-trace.schema.json")
	data, err := os.ReadFile(schemaPath)
	if err != nil {
		return []string{fmt.Sprintf("%s: missing canonical GCL trace schema", schemaPath)}
	}

	var schema map[string]any
	if err := json.Unmarshal(data, &schema); err != nil {
		return []string{fmt.Sprintf("%s: invalid JSON: %v", schemaPath, err)}
	}

	intent, ok := schema["properties"].(map[string]any)["operation_intent"].(map[string]any)
	if !ok {
		return []string{fmt.Sprintf("%s: operation_intent missing from schema", schemaPath)}
	}
	sc, ok := intent["properties"].(map[string]any)["safety_class"].(map[string]any)
	if !ok {
		return []string{fmt.Sprintf("%s: operation_intent.safety_class missing", schemaPath)}
	}
	declared, ok := sc["enum"].([]any)
	if !ok {
		return []string{fmt.Sprintf("%s: operation_intent.safety_class.enum missing or not array", schemaPath)}
	}
	var declaredStrs []string
	for _, v := range declared {
		if s, ok := v.(string); ok {
			declaredStrs = append(declaredStrs, s)
		}
	}
	if len(declaredStrs) != 3 || declaredStrs[0] != "read-only" || declaredStrs[1] != "mutating" || declaredStrs[2] != "destructive" {
		errors = append(errors, fmt.Sprintf("%s: enum=%v != [read-only mutating destructive]", schemaPath, declaredStrs))
	}

	required, _ := intent["required"].([]any)
	requiredSet := map[string]bool{}
	for _, v := range required {
		if s, ok := v.(string); ok {
			requiredSet[s] = true
		}
	}
	for _, key := range []string{"safety_class", "operation", "resource_scope", "expected_state"} {
		if !requiredSet[key] {
			errors = append(errors, fmt.Sprintf("%s: operation_intent.%s must be in required", schemaPath, key))
		}
	}
	return errors
}

func checkSafetyClassCode(skillcheckRoot string) []string {
	var errors []string
	// Check that the skillcheck binary's embedded sanitizer has the right values.
	// skillcheckRoot is the root of the skillcheck module (where skillcheck cmd lives).
	// We read the source directly rather than importing the gcl package (circular).
	srcPath := filepath.Join(skillcheckRoot, "internal", "gcl", "sanitizer.go")
	data, err := os.ReadFile(srcPath)
	if err != nil {
		return []string{fmt.Sprintf("%s: not found", srcPath)}
	}
	src := string(data)

	// Check SAFETY_CLASS_VALUES
	for _, val := range expectedSafetyClassValues {
		if !strings.Contains(src, fmt.Sprintf("%q", val)) &&
			!strings.Contains(src, fmt.Sprintf("\"%s\"", val)) {
			errors = append(errors, fmt.Sprintf("sanitizer.go: missing safety_class value %q", val))
		}
	}
	return errors
}

func checkSafetyClassDocs(root string) []string {
	var errors []string
	for _, rel := range []string{
		"docs/gcl-spec.md",
		"huaweicloud-skill-generator/references/gcl-prompt-backbone.md",
	} {
		path := filepath.Join(root, rel)
		if !fileExists(path) {
			errors = append(errors, fmt.Sprintf("%s: missing", rel))
			continue
		}
		text := readFileString(path)
		for _, val := range expectedSafetyClassValues {
			if !regexp.MustCompile(`\b` + regexp.QuoteMeta(val) + `\b`).MatchString(text) {
				errors = append(errors, fmt.Sprintf("%s: enum value %q not documented", rel, val))
			}
		}
	}
	return errors
}

func checkSafetyClassTraces(root string) []string {
	var errors []string
	auditDir := filepath.Join(root, "audit-results")
	if !dirExists(auditDir) {
		return nil
	}
	entries, _ := os.ReadDir(auditDir)
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasPrefix(entry.Name(), "gcl-trace-") || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}
		data, err := os.ReadFile(filepath.Join(auditDir, entry.Name()))
		if err != nil {
			continue
		}
		var trace map[string]any
		if json.Unmarshal(data, &trace) != nil {
			continue
		}
		intent, ok := trace["operation_intent"].(map[string]any)
		if !ok {
			continue
		}
		sc, ok := intent["safety_class"].(string)
		if !ok || sc == "" {
			continue
		}
		valid := false
		for _, v := range expectedSafetyClassValues {
			if sc == v {
				valid = true
				break
			}
		}
		if !valid {
			errors = append(errors, fmt.Sprintf("%s: operation_intent.safety_class=%q is not in the canonical enum", entry.Name(), sc))
		}
	}
	return errors
}

// ---------------------------------------------------------------------------
// validate resource-scope (L3-B)
// ---------------------------------------------------------------------------

var allowedResourceScopePatterns = []string{
	`^\*+$`,                       // pure ****
	`^<masked>$`,                  // explicit placeholder
	`^[A-Za-z][A-Za-z0-9-]*-\*+$`, // prefix-***
}

var rawIDPattern = regexp.MustCompile(`[A-Za-z0-9]{4,}`)

func runValidateResourceScope(args []string) error {
	fs := newFlagSet("skillcheck validate resource-scope")
	root := fs.String("root", ".", "skill repository root")
	jsonOut := fs.Bool("json", false, "emit JSON report")
	if err := fs.Parse(args); err != nil {
		return err
	}
	rootDir, err := filepath.Abs(*root)
	if err != nil {
		return err
	}

	// skillcheckRoot = where the skillcheck module lives.
	// Tests can override via SKILLCHECK_ROOT env var.
	skillcheckRoot := os.Getenv("SKILLCHECK_ROOT")
	if skillcheckRoot == "" {
		skillcheckRoot = filepath.Dir(filepath.Dir(os.Args[0]))
		if filepath.Base(os.Args[0]) == "skillcheck" {
			skillcheckRoot = filepath.Dir(os.Args[0])
		}
	}

	result := checkResourceScopeContract(skillcheckRoot, rootDir)

	if *jsonOut {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		enc.Encode(result)
	} else {
		for _, e := range result.SchemaErrors {
			fmt.Printf("  FAIL schema: %s\n", e)
		}
		for _, e := range result.CodeErrors {
			fmt.Printf("  FAIL code: %s\n", e)
		}
		for _, e := range result.DocsErrors {
			fmt.Printf("  FAIL docs: %s\n", e)
		}
		for _, e := range result.TracesErrors {
			fmt.Printf("  FAIL traces: %s\n", e)
		}
		if result.OK {
			fmt.Println("[resource_scope PII contract] OK")
		} else {
			fmt.Printf("[resource_scope PII contract] FAIL: %d issue(s)\n",
				len(result.SchemaErrors)+len(result.CodeErrors)+len(result.DocsErrors)+len(result.TracesErrors))
		}
	}

	if !result.OK {
		return fmt.Errorf("resource-scope: validation failed")
	}
	return nil
}

type resourceScopeResult struct {
	SchemaOK       bool     `json:"schema_ok"`
	CodeOK         bool     `json:"code_ok"`
	RunnerMaskedOK bool     `json:"runner_masked_fields_ok"`
	DocsOK         bool     `json:"docs_ok"`
	TracesOK       bool     `json:"traces_ok"`
	SchemaErrors   []string `json:"schema_errors"`
	CodeErrors     []string `json:"code_errors"`
	DocsErrors     []string `json:"docs_errors"`
	TracesErrors   []string `json:"traces_errors"`
	OK             bool     `json:"ok"`
}

func checkResourceScopeContract(skillcheckRoot, root string) resourceScopeResult {
	var r resourceScopeResult
	r.SchemaErrors = checkResourceScopeSchema(root)
	r.CodeErrors = checkResourceScopeCode(skillcheckRoot)
	r.DocsErrors = checkResourceScopeDocs(root)
	r.TracesErrors = checkResourceScopeTraces(root)

	r.SchemaOK = len(r.SchemaErrors) == 0
	r.CodeOK = len(r.CodeErrors) == 0
	r.RunnerMaskedOK = checkResourceScopeMaskedFields(skillcheckRoot)
	r.DocsOK = len(r.DocsErrors) == 0
	r.TracesOK = len(r.TracesErrors) == 0
	r.OK = r.SchemaOK && r.CodeOK && r.RunnerMaskedOK && r.DocsOK && r.TracesOK
	return r
}

func checkResourceScopeSchema(root string) []string {
	var errors []string
	schemaPath := filepath.Join(root, "huaweicloud-ces-ops/assets/gcl-trace.schema.json")
	data, err := os.ReadFile(schemaPath)
	if err != nil {
		return []string{fmt.Sprintf("%s: missing canonical GCL trace schema", schemaPath)}
	}

	var schema map[string]any
	if err := json.Unmarshal(data, &schema); err != nil {
		return []string{fmt.Sprintf("%s: invalid JSON: %v", schemaPath, err)}
	}

	intent, ok := schema["properties"].(map[string]any)["operation_intent"].(map[string]any)
	if !ok {
		return []string{fmt.Sprintf("%s: operation_intent missing from schema", schemaPath)}
	}
	rs, ok := intent["properties"].(map[string]any)["resource_scope"].(map[string]any)
	if !ok {
		return []string{fmt.Sprintf("%s: operation_intent.resource_scope missing", schemaPath)}
	}
	if rs["type"] != "array" {
		errors = append(errors, fmt.Sprintf("%s: resource_scope.type != 'array'", schemaPath))
	}
	items, ok := rs["items"].(map[string]any)
	if !ok {
		errors = append(errors, fmt.Sprintf("%s: resource_scope.items missing", schemaPath))
		return errors
	}
	if items["type"] != "string" {
		// Not an error per Python — just note
	}
	anyOf, ok := items["anyOf"].([]any)
	if !ok {
		return []string{fmt.Sprintf("%s: resource_scope.items.anyOf missing", schemaPath)}
	}
	var actualPatterns []string
	for _, p := range anyOf {
		if pm, ok := p.(map[string]any); ok {
			if pat, ok := pm["pattern"].(string); ok {
				actualPatterns = append(actualPatterns, pat)
			}
		}
	}
	if len(actualPatterns) != 3 {
		errors = append(errors, fmt.Sprintf("%s: resource_scope anyOf patterns count=%d, want 3", schemaPath, len(actualPatterns)))
	}
	// Compare as strings (order matters)
	for i, want := range allowedResourceScopePatterns {
		if i >= len(actualPatterns) || actualPatterns[i] != want {
			errors = append(errors, fmt.Sprintf("%s: resource_scope anyOf pattern[%d]=%q != want %q", schemaPath, i, safeStr(actualPatterns, i), want))
		}
	}
	return errors
}

func safeStr(s []string, i int) string {
	if i < len(s) {
		return s[i]
	}
	return "<missing>"
}

func checkResourceScopeCode(skillcheckRoot string) []string {
	var errors []string
	sanitizerPath := filepath.Join(skillcheckRoot, "internal", "gcl", "sanitizer.go")
	data, err := os.ReadFile(sanitizerPath)
	if err != nil {
		return []string{fmt.Sprintf("%s: not found", sanitizerPath)}
	}
	src := string(data)
	if !strings.Contains(src, "MaskResourceID") {
		errors = append(errors, "sanitizer.go: MaskResourceID function missing")
	}
	// Check idempotency: *** pass-through
	if !strings.Contains(src, "alreadyMaskedRe") && !strings.Contains(src, "***") {
		errors = append(errors, "sanitizer.go: idempotent masking check missing")
	}
	return errors
}

func checkResourceScopeMaskedFields(skillcheckRoot string) bool {
	path := filepath.Join(skillcheckRoot, "internal", "gcl", "runner.go")
	data, err := os.ReadFile(path)
	if err != nil {
		return false
	}
	src := string(data)
	return strings.Contains(src, `"operation_intent"`) || strings.Contains(src, "'operation_intent'")
}

func checkResourceScopeDocs(root string) []string {
	var errors []string
	for _, rel := range []string{
		"docs/gcl-spec.md",
		"huaweicloud-skill-generator/references/gcl-prompt-backbone.md",
	} {
		path := filepath.Join(root, rel)
		if !fileExists(path) {
			errors = append(errors, fmt.Sprintf("%s: missing", rel))
			continue
		}
		text := readFileString(path)
		if !strings.Contains(text, "resource_scope") {
			errors = append(errors, fmt.Sprintf("%s: missing documented `resource_scope`", rel))
		}
		if !regexp.MustCompile(`(?i)mask`).MatchString(text) {
			errors = append(errors, fmt.Sprintf("%s: missing any reference to masking", rel))
		}
	}
	return errors
}

func checkResourceScopeTraces(root string) []string {
	var errors []string
	auditDir := filepath.Join(root, "audit-results")
	if !dirExists(auditDir) {
		return nil
	}
	entries, _ := os.ReadDir(auditDir)
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasPrefix(entry.Name(), "gcl-trace-") || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}
		data, err := os.ReadFile(filepath.Join(auditDir, entry.Name()))
		if err != nil {
			continue
		}
		var trace map[string]any
		if json.Unmarshal(data, &trace) != nil {
			continue
		}
		intent, ok := trace["operation_intent"].(map[string]any)
		if !ok {
			continue
		}
		rs, ok := intent["resource_scope"].([]any)
		if !ok {
			continue
		}
		for i, item := range rs {
			itemStr, ok := item.(string)
			if !ok {
				continue
			}
			if looksLikeRawID(itemStr) {
				errors = append(errors, fmt.Sprintf("%s: operation_intent.resource_scope[%d]=%q is not masked; rerun gcl_runner.py to apply mask_resource_id", entry.Name(), i, itemStr))
			}
		}
	}
	return errors
}

func looksLikeRawID(value string) bool {
	if value == "" {
		return false
	}
	// Already masked?
	for _, pat := range allowedResourceScopePatterns {
		if regexp.MustCompile(pat).MatchString(value) {
			return false
		}
	}
	// Has significant identifier content?
	return rawIDPattern.MatchString(value)
}
