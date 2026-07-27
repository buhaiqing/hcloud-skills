package telemetry

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// Lane partitions telemetry events by trust boundary. Each lane writes to its
// own subdirectory under <repo>/audit-results/, so CI gates (see Task 2) can
// prove no self-test event ever crossed into a production artifact.
type Lane string

const (
	LaneSelfTest   Lane = "self-test"
	LaneSandbox    Lane = "sandbox"
	LaneProduction Lane = "production"
)

// laneRoot is the audit-results subdirectory relative to the repo root.
func laneRoot() string { return "audit-results" }

// Write drops a JSON envelope {lane, kind, ts, payload} at
// <repoRoot>/audit-results/<lane>/<UTC>-<rand>.json. The directory is created
// mode 0700 and the file is written mode 0600 so concurrent CI runs in
// different lanes never collide on disk and unprivileged users cannot read
// events outside their lane.
func Write(lane Lane, kind string, payload map[string]any) error {
	root := findRepoRoot(mustGetwd())
	dir := filepath.Join(root, laneRoot(), string(lane))
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return fmt.Errorf("telemetry mkdir: %w", err)
	}
	ts := time.Now().UTC().Format("20060102-150405")
	suf := make([]byte, 4)
	_, _ = rand.Read(suf)
	fp := filepath.Join(dir, fmt.Sprintf("%s-%s.json", ts, hex.EncodeToString(suf)))
	doc := map[string]any{
		"lane":    string(lane),
		"kind":    kind,
		"ts":      time.Now().UTC().Format(time.RFC3339),
		"payload": payload,
	}
	b, _ := json.MarshalIndent(doc, "", "  ")
	return os.WriteFile(fp, append(b, '\n'), 0o600)
}

func mustGetwd() string {
	cwd, err := os.Getwd()
	if err != nil {
		return "."
	}
	return cwd
}

// findRepoRoot walks up from start until it finds a directory containing
// AGENTS.md (the repo marker) and returns that directory. Bounded at 6 hops
// so a runaway symlink chain cannot loop forever; if no marker is found,
// start is returned as the best-effort fallback.
func findRepoRoot(start string) string {
	cur := start
	for range 6 {
		if _, err := os.Stat(filepath.Join(cur, "AGENTS.md")); err == nil {
			return cur
		}
		parent := filepath.Dir(cur)
		if parent == cur {
			return start
		}
		cur = parent
	}
	return start
}
