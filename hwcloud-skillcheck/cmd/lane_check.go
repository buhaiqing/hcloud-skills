package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// runCheckLanes enforces M6: no self-test (or other non-production) telemetry
// events may appear in <root>/audit-results/production/. Walks the production
// subtree, parses each .json envelope's "lane" field, and returns an error
// listing every offender. Mirrors the writer shape from
// internal/telemetry/lane.go: Write() emits {lane, kind, ts, payload} into
// audit-results/<lane>/, so any file under production/ whose lane field is
// not "production" is a lane-crossing that this gate must catch.
//
// Empty-lane and malformed-JSON handling: per the Task 2 brief, a missing
// "lane" field is treated as not-yet-a-violation (skip), not as an error.
// Any envelope that fails to parse is added to the bad list with a clear
// "(malformed json)" tag so it cannot silently slip past the gate.
func runCheckLanes(args []string) error {
	fs := newFlagSet("hwcloud-skillcheck check lanes")
	root := fs.String("root", ".", "repo root")
	if err := fs.Parse(args); err != nil {
		return err
	}
	prodDir := filepath.Join(*root, "audit-results", "production")
	var bad []string
	_ = filepath.Walk(prodDir, func(p string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() || !strings.HasSuffix(p, ".json") {
			return nil
		}
		b, _ := os.ReadFile(p)
		var doc struct {
			Lane string `json:"lane"`
		}
		// Match the brief's behavior: treat empty lane as "skip", but flag
		// any document we cannot parse — a malformed envelope in
		// production/ cannot be defended as "production" by silence.
		if err := json.Unmarshal(b, &doc); err != nil {
			bad = append(bad, fmt.Sprintf("%s (malformed json)", p))
			return nil
		}
		if doc.Lane != "" && doc.Lane != "production" {
			bad = append(bad, fmt.Sprintf("%s (lane=%s)", p, doc.Lane))
		}
		return nil
	})
	if len(bad) > 0 {
		return fmt.Errorf("cross-lane writes detected in production/:\n  %s", strings.Join(bad, "\n  "))
	}
	return nil
}
