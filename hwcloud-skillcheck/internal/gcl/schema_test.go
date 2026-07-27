package gcl

import (
	"strings"
	"testing"

	"github.com/buhaiqing/hcloud-skills/hwcloud-skillcheck/internal/embed"
	"github.com/buhaiqing/hcloud-skills/hwcloud-skillcheck/internal/schema"
)

// TestSchema_OperationIntentValid: a complete, valid operation_intent instance
// must validate against operation_intent.schema.json with zero errors.
func TestSchema_OperationIntentValid(t *testing.T) {
	instance := []byte(`{
		"operation": "delete",
		"resource_scope": ["ecs-abc12345"],
		"expected_state": "gone",
		"safety_class": "destructive"
	}`)
	errs, err := schema.ValidateFile(instance, embed.OperationIntentSchema)
	if err != nil {
		t.Fatalf("ValidateFile error: %v", err)
	}
	if len(errs) != 0 {
		t.Errorf("valid operation_intent returned errors: %v", errs)
	}
}

// TestSchema_OperationIntentMissingSafetyClass: instance without safety_class
// must fail validation with at least one error mentioning safety_class.
func TestSchema_OperationIntentMissingSafetyClass(t *testing.T) {
	instance := []byte(`{
		"operation": "delete",
		"resource_scope": ["ecs-abc12345"],
		"expected_state": "gone"
	}`)
	errs, err := schema.ValidateFile(instance, embed.OperationIntentSchema)
	if err != nil {
		t.Fatalf("ValidateFile error: %v", err)
	}
	if len(errs) == 0 {
		t.Fatal("expected at least 1 validation error for missing safety_class")
	}
	found := false
	for _, e := range errs {
		if strings.Contains(e, "safety_class") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("errors do not mention safety_class: %v", errs)
	}
}

// TestSchema_OperationIntentInvalidSafetyClass: instance with safety_class not
// in the enum must produce exactly one error.
func TestSchema_OperationIntentInvalidSafetyClass(t *testing.T) {
	instance := []byte(`{
		"operation": "delete",
		"resource_scope": ["ecs-abc12345"],
		"expected_state": "gone",
		"safety_class": "explosive"
	}`)
	errs, err := schema.ValidateFile(instance, embed.OperationIntentSchema)
	if err != nil {
		t.Fatalf("ValidateFile error: %v", err)
	}
	if len(errs) != 1 {
		t.Errorf("got %d errors, want 1: %v", len(errs), errs)
	}
}

// TestSchema_CriticOutputValid: a minimal valid critic output must validate
// against critic_output.schema.json with zero errors.
func TestSchema_CriticOutputValid(t *testing.T) {
	instance := []byte(`{
		"scores": {
			"correctness": 1.0,
			"safety": 1.0,
			"idempotency": 0.5,
			"traceability": 0.5,
			"spec_compliance": 1.0
		}
	}`)
	errs, err := schema.ValidateFile(instance, embed.CriticOutputSchema)
	if err != nil {
		t.Fatalf("ValidateFile error: %v", err)
	}
	if len(errs) != 0 {
		t.Errorf("valid critic_output returned errors: %v", errs)
	}
}

// TestSchema_CriticOutputMissingScores: empty object must fail validation with
// at least one error mentioning "scores".
func TestSchema_CriticOutputMissingScores(t *testing.T) {
	instance := []byte(`{}`)
	errs, err := schema.ValidateFile(instance, embed.CriticOutputSchema)
	if err != nil {
		t.Fatalf("ValidateFile error: %v", err)
	}
	if len(errs) == 0 {
		t.Fatal("expected at least 1 validation error for missing scores")
	}
	found := false
	for _, e := range errs {
		if strings.Contains(e, "scores") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("errors do not mention scores: %v", errs)
	}
}
