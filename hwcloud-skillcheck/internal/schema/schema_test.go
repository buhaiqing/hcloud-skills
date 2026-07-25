package schema

import "testing"

// TestJSONType verifies the json_type mapping matches the Python reference
// (scripts/json_schema_subset.py: json_type). Booleans are "boolean", not
// "integer"; integers are "integer" even when isinstance(int) — Go has no
// bool/int overlap, but we must keep the bool check first.
func TestJSONType(t *testing.T) {
	cases := []struct {
		val  any
		want string
	}{
		{nil, "null"},
		{true, "boolean"},
		{false, "boolean"},
		{42, "integer"},
		{3.14, "number"},
		{"s", "string"},
		{[]any{}, "array"},
		{map[string]any{}, "object"},
	}
	for _, c := range cases {
		if got := jsonType(c.val); got != c.want {
			t.Errorf("jsonType(%v) = %q, want %q", c.val, got, c.want)
		}
	}
}

func TestTypeMatches(t *testing.T) {
	// integer matches "number" (Python rule: integer is subset of number).
	if !typeMatches(5, "number") {
		t.Error("integer should match type 'number'")
	}
	if typeMatches(5, "string") {
		t.Error("integer should NOT match type 'string'")
	}
	// list of expected types
	if !typeMatches("x", []string{"string", "null"}) {
		t.Error("string should match one of [string,null]")
	}
}

func TestValidateValueScalars(t *testing.T) {
	// const
	errs := ValidateValue(1, map[string]any{"const": 2}, "$")
	if len(errs) != 1 {
		t.Errorf("const mismatch should yield 1 error, got %d: %v", len(errs), errs)
	}
	// enum
	errs = ValidateValue("b", map[string]any{"enum": []any{"a", "c"}}, "$")
	if len(errs) != 1 {
		t.Errorf("enum mismatch should yield 1 error, got %d: %v", len(errs), errs)
	}
	// minimum / maximum
	errs = ValidateValue(3, map[string]any{"minimum": 5}, "$")
	if len(errs) != 1 {
		t.Errorf("below minimum should yield 1 error, got %v", errs)
	}
	errs = ValidateValue(10, map[string]any{"maximum": 5}, "$")
	if len(errs) != 1 {
		t.Errorf("above maximum should yield 1 error, got %v", errs)
	}
	// minLength
	errs = ValidateValue("ab", map[string]any{"minLength": 5}, "$")
	if len(errs) != 1 {
		t.Errorf("short string should yield 1 error, got %v", errs)
	}
}

func TestValidateValueDateTime(t *testing.T) {
	// valid RFC3339
	if errs := ValidateValue("2026-07-18T10:00:00Z", map[string]any{"format": "date-time"}, "$"); len(errs) != 0 {
		t.Errorf("valid date-time should pass, got %v", errs)
	}
	// invalid
	if errs := ValidateValue("not-a-date", map[string]any{"format": "date-time"}, "$"); len(errs) != 1 {
		t.Errorf("invalid date-time should yield 1 error, got %v", errs)
	}
}

func TestValidateValueObject(t *testing.T) {
	schema := map[string]any{
		"type":       "object",
		"required":   []any{"name"},
		"properties": map[string]any{"name": map[string]any{"type": "string"}},
	}
	// missing required
	errs := ValidateValue(map[string]any{}, schema, "$")
	if len(errs) != 1 || errs[0] == "" {
		t.Errorf("missing required should yield 1 error, got %v", errs)
	}
	// additionalProperties false
	schemaAP := map[string]any{
		"type":                 "object",
		"properties":           map[string]any{"name": map[string]any{"type": "string"}},
		"additionalProperties": false,
	}
	errs = ValidateValue(map[string]any{"name": "ok", "extra": 1}, schemaAP, "$")
	if len(errs) != 1 {
		t.Errorf("additional property should yield 1 error, got %v", errs)
	}
	// nested property error
	errs = ValidateValue(map[string]any{"name": 123}, schema, "$")
	if len(errs) != 1 {
		t.Errorf("nested wrong type should yield 1 error, got %v", errs)
	}
}

func TestValidateValueArray(t *testing.T) {
	schema := map[string]any{
		"type":  "array",
		"items": map[string]any{"type": "integer"},
	}
	errs := ValidateValue([]any{1, "bad", 3}, schema, "$")
	if len(errs) != 1 {
		t.Errorf("array item type error should yield 1 error, got %v", errs)
	}
	// minItems
	errs = ValidateValue([]any{1}, map[string]any{"type": "array", "minItems": 3}, "$")
	if len(errs) != 1 {
		t.Errorf("below minItems should yield 1 error, got %v", errs)
	}
}

func TestResolveSchemaRefs(t *testing.T) {
	schema := map[string]any{
		"$defs": map[string]any{
			"Name": map[string]any{"type": "string"},
		},
		"properties": map[string]any{
			"name": map[string]any{"$ref": "#/$defs/Name"},
		},
	}
	resolved, err := ResolveSchemaRefs(schema)
	if err != nil {
		t.Fatalf("resolveSchemaRefs error: %v", err)
	}
	props, ok := resolved["properties"].(map[string]any)
	if !ok {
		t.Fatal("resolved properties missing")
	}
	name, ok := props["name"].(map[string]any)
	if !ok || name["type"] != "string" {
		t.Errorf("ref not resolved to {type:string}, got %v", name)
	}
	// unknown ref
	_, err = ResolveSchemaRefs(map[string]any{"$ref": "#/$defs/Missing"})
	if err == nil {
		t.Error("unknown $ref should error")
	}
}

// TestValidateDef verifies ValidateDef validates an instance against a single
// named $def within a schema that has no top-level required/properties (the
// eval-queries "union contract" shape). Mirrors
// scripts/validate_eval_queries_schema.py:_schema_def + validate_value.
func TestValidateDef(t *testing.T) {
	schemaData := []byte(`{
	  "$defs": {
	    "nonEmptyString": {"type": "string", "minLength": 1},
	    "matchArrayEntry": {
	      "type": "object",
	      "required": ["query", "should_match", "skill"],
	      "properties": {
	        "query": {"$ref": "#/$defs/nonEmptyString"},
	        "should_match": {"type": "boolean"},
	        "skill": {"$ref": "#/$defs/nonEmptyString"},
	        "reason": {"type": "string"}
	      },
	      "additionalProperties": false
	    }
	  }
	}`)

	good := []byte(`{"query":"list ecs","should_match":true,"skill":"huaweicloud-ecs-ops"}`)
	if errs, err := ValidateDef(schemaData, "matchArrayEntry", good); err != nil {
		t.Fatalf("unexpected error: %v", err)
	} else if len(errs) != 0 {
		t.Errorf("valid matchArrayEntry should pass, got %v", errs)
	}

	bad := []byte(`{"query":"","should_match":true}`)
	if errs, err := ValidateDef(schemaData, "matchArrayEntry", bad); err != nil {
		t.Fatalf("unexpected error: %v", err)
	} else if len(errs) == 0 {
		t.Error("missing required fields / empty query should fail")
	}

	// unknown def name must surface an error.
	if _, err := ValidateDef(schemaData, "nope", good); err == nil {
		t.Error("unknown $def name should error")
	}
}
