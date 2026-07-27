package main

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestMockHcloud_EmitsScriptedResponse(t *testing.T) {
	// Build the binary from this package directory.
	tmp := t.TempDir()
	bin := filepath.Join(tmp, "mockhcloud")
	build := exec.Command("go", "build", "-o", bin, ".")
	if out, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build: %v\n%s", err, out)
	}
	// Script with one entry.
	script := filepath.Join(tmp, "script.json")
	if err := os.WriteFile(script, []byte(`{"entries":[{"match":"ecs list","response":"[]","exit_code":0}]}`), 0o600); err != nil {
		t.Fatalf("write script: %v", err)
	}
	// stdout is the scripted response; stderr is the audit log.
	// CombinedOutput would conflate them, so we check stdout only.
	cmd := exec.Command(bin, "--script", script, "ecs", "list", "--region", "cn-north-4")
	stdout := new(bytes.Buffer)
	cmd.Stdout = stdout
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		t.Fatalf("run: %v\nstderr: %s", err, stderr.String())
	}
	if stdout.String() != "[]" {
		t.Errorf("stdout = %q, want \"[]\"", stdout.String())
	}
	// stderr must be a JSON line with the call audit.
	if !strings.Contains(stderr.String(), `"matched":"ecs list"`) {
		t.Errorf("stderr audit missing: %q", stderr.String())
	}
}

// TestMockhcloudNoNetwork is the A1.4 contract test: mockhcloud must not
// perform real network I/O. The runtime half proves the binary still returns
// a scripted response when the environment is hostile (no proxy reachable,
// no cloud credentials present). The static half proves the runtime path
// does not import net/http, net/url, or net — so there is no code path
// that could open a socket even by accident.
func TestMockhcloudNoNetwork(t *testing.T) {
	tmp := t.TempDir()
	bin := filepath.Join(tmp, "mockhcloud")
	build := exec.Command("go", "build", "-o", bin, ".")
	if out, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build: %v\n%s", err, out)
	}
	script := filepath.Join(tmp, "script.json")
	if err := os.WriteFile(script, []byte(`{"entries":[{"match":"ecs list","response":"[]","exit_code":0}]}`), 0o600); err != nil {
		t.Fatalf("write script: %v", err)
	}

	// Runtime half: run under offline-hostile env (unset any cloud creds,
	// point HTTP env at an unroutable address so any accidental net dial
	// would fail fast rather than hang).
	cmd := exec.Command(bin, "--script", script, "ecs", "list", "--region", "cn-north-4")
	cmd.Env = append(os.Environ(),
		"HC_ACCESS_KEY=",
		"HC_SECRET_KEY=",
		"HW_ACCESS_KEY_ID=",
		"HW_SECRET_ACCESS_KEY=",
		// 198.51.100.0/24 is TEST-NET-2 (RFC 5737) — guaranteed unroutable.
		"HTTP_PROXY=http://198.51.100.1:1",
		"HTTPS_PROXY=http://198.51.100.1:1",
		"NO_PROXY=*",
	)
	stdout := new(bytes.Buffer)
	cmd.Stdout = stdout
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		t.Fatalf("run under offline env: %v\nstderr: %s", err, stderr.String())
	}
	if stdout.String() != "[]" {
		t.Errorf("offline run stdout = %q, want \"[]\"", stdout.String())
	}

	// Static half: runtime dep graph contains no net packages.
	list := exec.Command("go", "list", "-deps", ".")
	depsOut, err := list.CombinedOutput()
	if err != nil {
		t.Fatalf("go list -deps: %v\n%s", err, depsOut)
	}
	for _, banned := range []string{"net", "net/http", "net/url", "crypto/tls"} {
		for _, line := range strings.Split(string(depsOut), "\n") {
			if strings.TrimSpace(line) == banned {
				t.Errorf("runtime dependency %q is forbidden: mockhcloud must not link the net stack", banned)
			}
		}
	}
}
