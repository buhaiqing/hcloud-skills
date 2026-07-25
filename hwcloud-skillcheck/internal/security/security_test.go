package security

import (
	"strings"
	"testing"
)

// find returns the single finding of the given type, or fails the test.
func find(t *testing.T, findings []Finding, typ string) Finding {
	t.Helper()
	for _, f := range findings {
		if f.Type == typ {
			return f
		}
	}
	t.Fatalf("expected finding of type %q, got: %v", typ, findings)
	return Finding{}
}

func TestScanContentNoLeak(t *testing.T) {
	// Clean text must not produce any findings.
	data := []byte("This is a normal operation on the ECS instance.\nNothing secret here.")
	findings, err := ScanContent(data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(findings) != 0 {
		t.Fatalf("expected no findings, got: %v", findings)
	}
}

func TestScanContentHWSecretAccessKey(t *testing.T) {
	data := []byte("export HW_SECRET_ACCESS_KEY=abc123DEF456ghi789JKL==")
	findings, err := ScanContent(data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	f := find(t, findings, "hw_secret_access_key")
	if f.Line != 1 {
		t.Errorf("expected line 1, got %d", f.Line)
	}
	if !strings.Contains(f.Snippet, "<masked>") {
		t.Errorf("snippet should be masked, got %q", f.Snippet)
	}
	if strings.Contains(f.Snippet, "abc123DEF456") {
		t.Errorf("snippet must not contain raw secret, got %q", f.Snippet)
	}
}

func TestScanContentSecretAccessKey(t *testing.T) {
	data := []byte("SECRET_ACCESS_KEY = ZmFrZXNlY3JldGtleWZha2VzZWNyZXRrZXk=")
	findings, err := ScanContent(data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	f := find(t, findings, "secret_access_key")
	if !strings.Contains(f.Snippet, "<masked>") {
		t.Errorf("snippet should be masked, got %q", f.Snippet)
	}
}

func TestScanContentSecretAccessKeyCamel(t *testing.T) {
	// Note: the Python pattern requires an unquoted value, so the value must
	// not be wrapped in quotes (a quoted JSON value is out of scope).
	data := []byte("SecretAccessKey=Vlongsecretvalue0123456789abcdef")
	findings, err := ScanContent(data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	f := find(t, findings, "secret_access_key_camel")
	if !strings.Contains(f.Snippet, "<masked>") {
		t.Errorf("snippet should be masked, got %q", f.Snippet)
	}
}

func TestScanContentSK(t *testing.T) {
	data := []byte("session token SK=abcdefghijklmnopqrstuvwxyz012345")
	findings, err := ScanContent(data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	f := find(t, findings, "sk")
	if !strings.Contains(f.Snippet, "<masked>") {
		t.Errorf("snippet should be masked, got %q", f.Snippet)
	}
}

func TestScanContentBearerToken(t *testing.T) {
	data := []byte("Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9")
	findings, err := ScanContent(data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	f := find(t, findings, "bearer_token")
	if !strings.Contains(f.Snippet, "<masked>") {
		t.Errorf("snippet should be masked, got %q", f.Snippet)
	}
}

func TestScanContentPasswordAssignment(t *testing.T) {
	data := []byte("password=Sup3rS3cretValue")
	findings, err := ScanContent(data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	f := find(t, findings, "password_assignment")
	if !strings.Contains(f.Snippet, "<masked>") {
		t.Errorf("snippet should be masked, got %q", f.Snippet)
	}
}

func TestScanContentAPIKeyAssignment(t *testing.T) {
	data := []byte("api_key=abcdefghijklmnopqrstuvwxyz012345")
	findings, err := ScanContent(data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	f := find(t, findings, "api_key_assignment")
	if !strings.Contains(f.Snippet, "<masked>") {
		t.Errorf("snippet should be masked, got %q", f.Snippet)
	}
}

func TestScanContentPrivateKeyBlock(t *testing.T) {
	data := []byte("-----BEGIN RSA PRIVATE KEY-----\nMIIEogIBAAKCAQEA\n-----END RSA PRIVATE KEY-----")
	findings, err := ScanContent(data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	f := find(t, findings, "private_key_block")
	if !strings.Contains(f.Snippet, "PRIVATE KEY") {
		t.Errorf("snippet should retain the block marker, got %q", f.Snippet)
	}
}

func TestScanContentMultiPattern(t *testing.T) {
	// Both an SK and a password in the same content => two findings.
	data := []byte("SK=abcdefghijklmnopqrstuvwxyz012345\nsomething\npassword=qwertyuiopasdfgh")
	findings, err := ScanContent(data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(findings) < 2 {
		t.Fatalf("expected >=2 findings, got %d: %v", len(findings), findings)
	}
	// SK is on line 1, password on line 3.
	var gotSK, gotPw bool
	for _, f := range findings {
		if f.Type == "sk" && f.Line == 1 {
			gotSK = true
		}
		if f.Type == "password_assignment" && f.Line == 3 {
			gotPw = true
		}
	}
	if !gotSK || !gotPw {
		t.Fatalf("expected SK on line 1 and password on line 3, got %v", findings)
	}
}

func TestScanContentAlreadyMasked(t *testing.T) {
	// Content already containing <masked> must be skipped entirely,
	// mirroring the Python early-return in scan_text.
	data := []byte("HW_SECRET_ACCESS_KEY=<masked> and also password=stillvisible")
	findings, err := ScanContent(data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(findings) != 0 {
		t.Fatalf("expected no findings when <masked> present, got %v", findings)
	}
}

func TestMaskSecrets(t *testing.T) {
	in := []byte("HW_SECRET_ACCESS_KEY=abc123DEF456 and SK=abcdefghijklmnopqrstuvwxyz012345")
	out := MaskSecrets(in)
	s := string(out)
	if strings.Contains(s, "abc123DEF456") {
		t.Errorf("masked output must not contain raw HW secret: %q", s)
	}
	if strings.Contains(s, "abcdefghijklmnopqrstuvwxyz012345") {
		t.Errorf("masked output must not contain raw SK: %q", s)
	}
	if !strings.Contains(s, "HW_SECRET_ACCESS_KEY=<masked>") {
		t.Errorf("expected masked HW secret, got %q", s)
	}
	if !strings.Contains(s, "SK=<masked>") {
		t.Errorf("expected masked SK, got %q", s)
	}
}
