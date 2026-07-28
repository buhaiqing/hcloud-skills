package cmd

import (
	"io"
	"os"
	"strings"
	"testing"

	"github.com/buhaiqing/hcloud-skills/hwcloud-skillcheck/internal/l4"
)

// TestRunTrustStats seeds the default counter and verifies the
// subcommand formats the outcome-memory source (Prometheus-style).
// Phase 4 removed the op_history line — trust is single-source from
// outcome memory now. Uses no root memory file.
func TestRunTrustStats(t *testing.T) {
	prev := l4.DefaultTrustSource
	l4.DefaultTrustSource = &l4.TrustSourceCounter{}
	l4.DefaultTrustSource.Record("outcome_memory")
	l4.DefaultTrustSource.Record("outcome_memory")
	l4.DefaultTrustSource.Record("op_history") // unknown source ignored post-Phase-4
	defer func() { l4.DefaultTrustSource = prev }()

	readEnd, writeEnd, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	prevOut := os.Stdout
	os.Stdout = writeEnd
	defer func() { os.Stdout = prevOut }()

	if err := runTrustStats([]string{"stats", "--root", "."}); err != nil {
		t.Fatal(err)
	}
	if err := writeEnd.Close(); err != nil {
		t.Fatal(err)
	}
	out, err := io.ReadAll(readEnd)
	if err != nil {
		t.Fatal(err)
	}
	got := string(out)
	for _, want := range []string{
		`trust_source{from="outcome_memory"}: 2`,
		"last_outcome_lookup:",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("missing %q in output:\n%s", want, got)
		}
	}
	// Phase 4 invariant: the legacy op_history line must NOT appear.
	if strings.Contains(got, "op_history") {
		t.Errorf("op_history line should be gone post-Phase-4, got:\n%s", got)
	}
}
