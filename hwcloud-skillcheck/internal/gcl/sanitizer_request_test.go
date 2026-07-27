package gcl

import (
	"strings"
	"testing"
)

// TestSanitizeRequest_StripsResourceIDs: input contains a bare resource ID
// (ecs-abc12345); the output must contain the placeholder "<id>" and must NOT
// contain the raw ID.
func TestSanitizeRequest_StripsResourceIDs(t *testing.T) {
	in := "delete ecs-abc12345 server"
	got, err := SanitizeRequest(in)
	if err != nil {
		t.Fatalf("SanitizeRequest error = %v, want nil", err)
	}
	if strings.Contains(got, "ecs-abc12345") {
		t.Errorf("SanitizeRequest(%q) = %q; raw resource ID leaked", in, got)
	}
	if !strings.Contains(got, "<id>") {
		t.Errorf("SanitizeRequest(%q) = %q; expected <id> placeholder", in, got)
	}
}

// TestSanitizeRequest_StripsARNs: input contains a Huawei Cloud ARN; output
// must contain "<arn>" and must NOT contain the raw ARN.
func TestSanitizeRequest_StripsARNs(t *testing.T) {
	in := "acs:cn-north-4:project:ecs:instance:abc"
	got, err := SanitizeRequest(in)
	if err != nil {
		t.Fatalf("SanitizeRequest error = %v, want nil", err)
	}
	if strings.Contains(got, "acs:cn-north-4") {
		t.Errorf("SanitizeRequest(%q) = %q; raw ARN leaked", in, got)
	}
	if !strings.Contains(got, "<arn>") {
		t.Errorf("SanitizeRequest(%q) = %q; expected <arn> placeholder", in, got)
	}
}

// TestSanitizeRequest_StripsCredentials: input contains a credential pattern
// (AK=...); output must contain "<redacted>" and must NOT contain the raw
// credential value.
func TestSanitizeRequest_StripsCredentials(t *testing.T) {
	// 20+ chars of [A-Za-z0-9/+] so it matches the SK pattern (and is
	// recognizable as a credential by the fail-closed classifier).
	in := "AK=ABCDEFGHIJ1234567890"
	got, err := SanitizeRequest(in)
	if err != nil {
		t.Fatalf("SanitizeRequest error = %v, want nil", err)
	}
	if strings.Contains(got, "ABCDEFGHIJ1234567890") {
		t.Errorf("SanitizeRequest(%q) = %q; raw credential leaked", in, got)
	}
	if !strings.Contains(got, "<redacted>") {
		t.Errorf("SanitizeRequest(%q) = %q; expected <redacted> placeholder", in, got)
	}
}

// TestSanitizeRequest_Idempotent: input already contains placeholders; output
// must be unchanged.
func TestSanitizeRequest_Idempotent(t *testing.T) {
	cases := []string{
		"already <id> here",
		"already <masked> here",
		"already *** here",
	}
	for _, in := range cases {
		got, err := SanitizeRequest(in)
		if err != nil {
			t.Fatalf("SanitizeRequest(%q) error = %v, want nil", in, err)
		}
		if got != in {
			t.Errorf("SanitizeRequest(%q) = %q; expected unchanged (idempotent)", in, got)
		}
	}
}

// TestSanitizeRequest_FailClosedOnUnrecognized: input contains a long base64
// blob that does NOT match any known pattern (resource ID, ARN, credential).
// SanitizeRequest must return an error (fail-closed) rather than pass through
// the unrecognized token.
func TestSanitizeRequest_FailClosedOnUnrecognized(t *testing.T) {
	// 40+ chars of unrecognized content that matches no pattern.
	// Avoid digits-with-hyphen (would match resource ID) and "acs:" (ARN),
	// and avoid any trailing "=<value>" or ":<value>" credential pattern.
	in := "ZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZ this is opaque"
	_, err := SanitizeRequest(in)
	if err == nil {
		t.Fatalf("SanitizeRequest(%q) error = nil, want error (fail-closed)", in)
	}
}

// TestSanitizeRequest_FailClosedOnUnderscoreBypass closes a bypass
// where an attacker could defeat the fail-closed check by inserting
// an underscore into a 16+ char token. The requestOpaqueTokenRe
// alphabet now includes `_` and `-`, so a 20-char token with
// underscores must also be rejected.
func TestSanitizeRequest_FailClosedOnUnderscoreBypass(t *testing.T) {
	in := "fetch abcd_efgh_ijkl_mnop_qrst_uvwx please"
	_, err := SanitizeRequest(in)
	if err == nil {
		t.Fatalf("SanitizeRequest(%q) error = nil, want error (fail-closed for underscore-bypass)", in)
	}
}

// TestSanitizeRequest_FailClosedOnHyphenBypass covers the same path
// for tokens that use `-` as the only separator.
func TestSanitizeRequest_FailClosedOnHyphenBypass(t *testing.T) {
	in := "fetch abcd-efgh-ijkl-mnop-qrst-uvwx please"
	_, err := SanitizeRequest(in)
	if err == nil {
		t.Fatalf("SanitizeRequest(%q) error = nil, want error (fail-closed for hyphen-bypass)", in)
	}
}
