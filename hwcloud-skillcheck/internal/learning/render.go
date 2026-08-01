package learning

import (
	"fmt"
	"os"
	"regexp"
)

// placeholderRe matches {{output.key}} and {{env.KEY}} placeholders.
var placeholderRe = regexp.MustCompile(`\{\{(output|env)\.([^}]+)\}\}`)

// RenderOutput substitutes {{output.key}} and {{env.KEY}} placeholders in a
// playbook command string.
//
//   - {{output.key}} is resolved from the provided captured outputs map.
//     A missing key → returns (_, false, error) so the caller blocks execution.
//   - {{env.KEY}} is resolved from the process environment; a missing env var
//     also yields ok=false + error.
//
// Returns ok=true only when every placeholder was substituted.
func RenderOutput(tmpl string, outputs map[string]string) (string, bool, error) {
	var firstErr error
	rendered := placeholderRe.ReplaceAllStringFunc(tmpl, func(m string) string {
		sub := placeholderRe.FindStringSubmatch(m)
		if sub == nil {
			return m
		}
		kind, key := sub[1], sub[2]
		switch kind {
		case "output":
			v, ok := outputs[key]
			if !ok {
				if firstErr == nil {
					firstErr = fmt.Errorf("missing output placeholder {{output.%s}}", key)
				}
				return m
			}
			return v
		case "env":
			v, ok := os.LookupEnv(key)
			if !ok {
				if firstErr == nil {
					firstErr = fmt.Errorf("missing env placeholder {{env.%s}}", key)
				}
				return m
			}
			return v
		}
		return m
	})
	if firstErr != nil {
		return rendered, false, firstErr
	}
	if placeholderRe.MatchString(rendered) {
		return rendered, false, fmt.Errorf("unresolved placeholder remains: %q", rendered)
	}
	return rendered, true, nil
}

// EvalPreconditions runs each precondition command via runFn (which executes a
// shell command and returns exit code / output / error). A precondition passes
// iff it exits 0. Returns ok=true only when all preconditions pass; failed lists
// the commands that did not.
func EvalPreconditions(preconds []string, outputs map[string]string, run func(string) (int, string, error)) (bool, []string) {
	var failed []string
	for _, pre := range preconds {
		rendered, ok, err := RenderOutput(pre, outputs)
		if err != nil || !ok {
			failed = append(failed, pre)
			continue
		}
		code, _, err := run(rendered)
		if err != nil || code != 0 {
			failed = append(failed, pre)
		}
	}
	return len(failed) == 0, failed
}
