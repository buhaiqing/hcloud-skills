package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/buhaiqing/hcloud-skills/skillcheck/internal/embed"
	"github.com/buhaiqing/hcloud-skills/skillcheck/internal/schema"
)

// runValidateSchema handles:
//
//	skillcheck validate schema <trace|summary|alarm-plan|eval-queries> --file <path>
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
	if *file == "" {
		return fmt.Errorf("validate schema %s: --file is required", kind)
	} else if *file == "-" {
		instanceData, err = io.ReadAll(osStdin)
	} else {
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
func validateEvalQueries(instanceData, schemaData []byte) ([]string, error) {
	dec := json.NewDecoder(bytes.NewReader(instanceData))
	dec.UseNumber()
	var parsed any
	if err := dec.Decode(&parsed); err != nil {
		return nil, fmt.Errorf("parse instance: %w", err)
	}

	var defName string
	switch v := parsed.(type) {
	case []any:
		if len(v) == 0 {
			return []string{"$: expected non-empty array"}, nil
		}
		first, ok := v[0].(map[string]any)
		if !ok {
			return []string{"$: every array item must be an object"}, nil
		}
		defName = evalArrayDefFor(first)
		if defName == "" {
			return []string{"$: unrecognized array entry shape"}, nil
		}
		// Python validates each array item against the entry $def (the array
		// itself has no schema). Validate every element, collecting errors.
		var all []string
		for i, item := range v {
			itemBytes, err := json.Marshal(item)
			if err != nil {
				all = append(all, fmt.Sprintf("$[%d]: %v", i, err))
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
	case map[string]any:
		defName = evalObjectDefFor(v)
		if defName == "" {
			return []string{"$: unrecognized object shape; expected evaluation_queries, should_match, or should_trigger"}, nil
		}
	default:
		return []string{"$: expected array or object"}, nil
	}

	return schema.ValidateDef(schemaData, defName, instanceData)
}
