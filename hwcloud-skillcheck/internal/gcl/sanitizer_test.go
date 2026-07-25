package gcl

import (
	"testing"
)

func TestIsValidSafetyClass(t *testing.T) {
	valid := []string{"read-only", "mutating", "destructive"}
	for _, v := range valid {
		if !IsValidSafetyClass(v) {
			t.Errorf("IsValidSafetyClass(%q) = false, want true", v)
		}
	}
	invalid := []string{"", "explosive", "READ-ONLY", "mutating ", "read_only"}
	for _, v := range invalid {
		if IsValidSafetyClass(v) {
			t.Errorf("IsValidSafetyClass(%q) = true, want false", v)
		}
	}
}

func TestSanitizeOperationIntent_Valid(t *testing.T) {
	for _, sc := range []string{"read-only", "mutating", "destructive"} {
		raw := `{"operation":"delete","resource_scope":["ecs-1234"],"expected_state":"gone","safety_class":"` + sc + `"}`
		got, err := SanitizeOperationIntent(raw)
		if err != nil {
			t.Errorf("SanitizeOperationIntent(%q) error = %v, want nil", raw, err)
			continue
		}
		if got == nil {
			t.Errorf("SanitizeOperationIntent(%q) = nil, want non-nil", raw)
			continue
		}
		if got["safety_class"] != sc {
			t.Errorf("safety_class = %v, want %q", got["safety_class"], sc)
		}
	}
}

func TestSanitizeOperationIntent_InvalidSafetyClass(t *testing.T) {
	raw := `{"operation":"delete","resource_scope":["i-123"],"expected_state":"gone","safety_class":"explosive"}`
	_, err := SanitizeOperationIntent(raw)
	if err == nil {
		t.Error("SanitizeOperationIntent(invalid safety_class) = nil, want *SanitizeError")
	}
	var se *SanitizeError
	if ok := assertErrorAs(err, &se); !ok {
		t.Errorf("error type = %T, want *SanitizeError", err)
		return
	}
	if se.Field != "safety_class" {
		t.Errorf("Field = %q, want %q", se.Field, "safety_class")
	}
}

func TestSanitizeOperationIntent_Empty(t *testing.T) {
	got, err := SanitizeOperationIntent("")
	if err != nil {
		t.Errorf("SanitizeOperationIntent(\"\") error = %v, want nil", err)
	}
	if got != nil {
		t.Errorf("SanitizeOperationIntent(\"\") = %v, want nil", got)
	}
}

func TestSanitizeOperationIntent_NotJSON(t *testing.T) {
	// Input with a detectable secret pattern — should be masked.
	raw := `SECRET_ACCESS_KEY = mysecretvalue123456`
	got, err := SanitizeOperationIntent(raw)
	if err != nil {
		t.Errorf("SanitizeOperationIntent(not-json) error = %v, want nil", err)
	}
	if got == nil {
		t.Error("SanitizeOperationIntent(not-json) = nil, want partial with summary")
		return
	}
	summary, ok := got["summary"].(string)
	if !ok {
		t.Errorf("summary type = %T, want string", got["summary"])
		return
	}
	// Should contain masking sentinel, not raw secret.
	if summary == raw {
		t.Error("summary should be masked, got raw input")
	}
	if contains(summary, "mysecretvalue") {
		t.Error("summary must not contain raw secret value")
	}
}

func TestSanitizeOperationIntent_SecretFieldsMasked(t *testing.T) {
	raw := `{"operation":"list","resource_scope":[],"expected_state":"ok","safety_class":"read-only","secret_key":"hunter2"}`
	got, err := SanitizeOperationIntent(raw)
	if err != nil {
		t.Errorf("SanitizeOperationIntent(...) error = %v", err)
	}
	if got["secret_key"] != "<masked>" {
		t.Errorf("secret_key = %v, want <masked>", got["secret_key"])
	}
}

func TestMaskResourceID_AlreadyMasked(t *testing.T) {
	cases := []string{"***", "<masked>", "ecs-***", "***-***"}
	for _, id := range cases {
		got := MaskResourceID(id)
		if got != id {
			t.Errorf("MaskResourceID(%q) = %q, want unchanged", id, got)
		}
	}
}

func TestMaskResourceID_ARN(t *testing.T) {
	id := "acs:ecs:cn-north-4:123456789:/i-12345678"
	got := MaskResourceID(id)
	if got == id {
		t.Error("ARN should be masked, got unchanged")
	}
	if got == "***" {
		t.Error("ARN should preserve prefix, got fully masked")
	}
}

func TestMaskResourceID_UUID(t *testing.T) {
	id := "550e8400-e29b-41d4-a716-446655440000"
	got := MaskResourceID(id)
	if got != "***" {
		t.Errorf("UUID = %q, want ***", got)
	}
}

func TestMaskResourceID_Standard(t *testing.T) {
	cases := []struct {
		input, want string
	}{
		{"ecs-1234abc", "ecs-***"},
		{"rds-instance-xy", "rds-***"}, // prefix = segment before first hyphen
		{"single", "***"},
		{"a", "***"},
		{"ALB-123", "ALB-***"},
	}
	for _, c := range cases {
		got := MaskResourceID(c.input)
		if got != c.want {
			t.Errorf("MaskResourceID(%q) = %q, want %q", c.input, got, c.want)
		}
	}
}

func TestMaskResourceScope_List(t *testing.T) {
	scope := []any{"ecs-123", "acs:vpc:cn-north-4:uid:/vpc-456", "***"}
	got := maskResourceScope(scope)
	list, ok := got.([]any)
	if !ok {
		t.Fatalf("type = %T, want []any", got)
	}
	if len(list) != 3 {
		t.Fatalf("len = %d, want 3", len(list))
	}
	if list[0] != "ecs-***" {
		t.Errorf("[0] = %q, want ecs-***", list[0])
	}
	// ARNs preserve prefix
	arn, ok := list[1].(string)
	if !ok {
		t.Fatalf("[1] type = %T, want string", list[1])
	}
	if arn == "acs:vpc:cn-north-4:uid:/vpc-456" {
		t.Error("[1] should be masked")
	}
	if list[2] != "***" {
		t.Errorf("[2] = %q, want ***", list[2])
	}
}

// assertErrorAs is a minimal errors.As equivalent to avoid importing errors.
func assertErrorAs(err error, target **SanitizeError) bool {
	type errorAs interface{ Error() string }
	// We only need to check direct type since we control the error type here.
	se, ok := err.(*SanitizeError)
	if !ok {
		return false
	}
	*target = se
	return true
}
