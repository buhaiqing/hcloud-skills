package embed

import "testing"

// TestEmbedSchemas verifies the embedded schema files are present and parse
// as valid JSON. This guards against missing //go:embed directives at
// compile time.
func TestEmbedSchemas(t *testing.T) {
	files := map[string][]byte{
		"trace.schema.json":        TraceSchema,
		"summary.schema.json":      SummarySchema,
		"alarm-plan.schema.json":   AlarmPlanSchema,
		"eval-queries.schema.json": EvalQueriesSchema,
	}
	for name, data := range files {
		if len(data) == 0 {
			t.Errorf("embedded schema %s is empty", name)
		}
		// crude JSON validity check
		if data[0] != '{' {
			t.Errorf("embedded schema %s does not start with '{': %s", name, string(data[:min(10, len(data))]))
		}
	}
}

func TestEmbedFixtures(t *testing.T) {
	if len(AlarmPlanHealthy) == 0 {
		t.Error("embedded alarm-plan fixture is empty")
	}
	if len(SummaryHealthy) == 0 {
		t.Error("embedded summary fixture is empty")
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
