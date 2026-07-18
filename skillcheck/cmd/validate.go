package cmd

import (
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
