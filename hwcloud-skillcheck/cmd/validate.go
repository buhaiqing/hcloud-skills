package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/buhaiqing/hcloud-skills/hwcloud-skillcheck/internal/embed"
	"github.com/buhaiqing/hcloud-skills/hwcloud-skillcheck/internal/schema"
)

// runValidateSchema handles:
//
//	hwcloud-skillcheck validate schema <trace|summary|alarm-plan|eval-queries> --file <path>
//
// The instance is read from --file (or stdin when "-" is given) and validated
// against the embedded schema. Exit code 0 = valid, 1 = invalid.
func runValidateSchema(args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("validate schema: missing kind (trace|summary|alarm-plan|eval-queries)")
	}
	kind := args[0]

	fs := newFlagSet("validate schema " + kind)
	file := fs.String("file", "", "instance JSON file path ('-' for stdin)")
	if err := fs.Parse(args[1:]); err != nil {
		return err
	}

	schemaData, err := schemaForKind(kind)
	if err != nil {
		return err
	}

	var instanceData []byte
	switch *file {
	case "":
		return fmt.Errorf("validate schema %s: --file is required", kind)
	case "-":
		instanceData, err = io.ReadAll(osStdin)
	default:
		instanceData, err = os.ReadFile(*file)
	}
	if err != nil {
		return fmt.Errorf("read instance: %w", err)
	}

	// eval-queries is a $defs-only union contract: the top-level schema has no
	// required/properties, so ValidateFile would trivially accept anything.
	// Dispatch to format-specific $def validation instead.
	if kind == "eval-queries" {
		errors, err := validateEvalQueries(instanceData, schemaData)
		if err != nil {
			return err
		}
		if len(errors) > 0 {
			for _, e := range errors {
				fmt.Fprintln(os.Stderr, "FAIL:", e)
			}
			return fmt.Errorf("validate schema eval-queries: %d error(s)", len(errors))
		}
		fmt.Printf("OK: eval-queries instance valid against eval-queries schema\n")
		return nil
	}

	errors, err := schema.ValidateFile(instanceData, schemaData)
	if err != nil {
		return err
	}
	if len(errors) > 0 {
		for _, e := range errors {
			fmt.Fprintln(os.Stderr, "FAIL:", e)
		}
		return fmt.Errorf("validate schema %s: %d error(s)", kind, len(errors))
	}
	fmt.Printf("OK: %s instance valid against %s schema\n", kind, kind)
	return nil
}

func schemaForKind(kind string) ([]byte, error) {
	switch kind {
	case "trace":
		return embed.TraceSchema, nil
	case "summary":
		return embed.SummarySchema, nil
	case "alarm-plan":
		return embed.AlarmPlanSchema, nil
	case "eval-queries":
		return embed.EvalQueriesSchema, nil
	default:
		return nil, fmt.Errorf("unknown schema kind %q (want trace|summary|alarm-plan|eval-queries)", kind)
	}
}

// evalArrayDefFor detects which $def an eval-queries array entry belongs to,
// mirroring validate_eval_queries_schema._detect_array_format.
func evalArrayDefFor(item map[string]any) string {
	switch {
	case hasKey(item, "should_activate"):
		return "activateArrayEntry"
	case hasKey(item, "should_match"):
		return "matchArrayEntry"
	case hasKey(item, "should_trigger"):
		return "triggerArrayEntry"
	case hasKey(item, "description"):
		return "smokeArrayEntry"
	default:
		return ""
	}
}

// evalObjectDefFor detects which $def an eval-queries object document uses.
func evalObjectDefFor(data map[string]any) string {
	switch {
	case hasKey(data, "evaluation_queries"):
		return "structuredObject"
	case hasKey(data, "should_match"):
		return "matchObject"
	case hasKey(data, "should_trigger"):
		return "triggerObject"
	default:
		return ""
	}
}

func hasKey(m map[string]any, key string) bool {
	_, ok := m[key]
	return ok
}

// validateEvalQueries validates an eval_queries.json instance against the
// embedded eval-queries union schema, auto-detecting its format (array vs
// object) and dispatching to the matching $def. Mirrors
// scripts/validate_eval_queries_schema.py:validate_eval_document.
//
// Format detection is shared with validateEvalQueriesFile via detectEvalFormat
// (WR-02): parse failures are hard errors; shape mismatches are soft messages.
func validateEvalQueries(instanceData, schemaData []byte) ([]string, error) {
	defName, parsed, err := detectEvalFormat(instanceData)
	if err != nil {
		msg := err.Error()
		if strings.HasPrefix(msg, "parse instance:") {
			return nil, err
		}
		return []string{msg}, nil
	}

	if arr, ok := parsed.([]any); ok {
		// Python validates each array item against the entry $def (the array
		// itself has no schema). Validate every element, collecting errors.
		var all []string
		for i, item := range arr {
			itemBytes, mErr := json.Marshal(item)
			if mErr != nil {
				all = append(all, fmt.Sprintf("$[%d]: %v", i, mErr))
				continue
			}
			errs, verr := schema.ValidateDef(schemaData, defName, itemBytes)
			if verr != nil {
				return nil, verr
			}
			for _, e := range errs {
				all = append(all, fmt.Sprintf("$[%d]%s", i, e))
			}
		}
		return all, nil
	}

	return schema.ValidateDef(schemaData, defName, instanceData)
}
