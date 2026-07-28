// Package cmd — `trust stats` subcommand.
//
// Exposes the outcome-memory trust-lookup counter as a Prometheus-style
// scrape. Read-only, no flag besides --root (for symmetry with sibling
// subcommands).
package cmd

import (
	"fmt"
	"os"

	"github.com/buhaiqing/hcloud-skills/hwcloud-skillcheck/internal/l4"
)

// runTrustStats prints the TrustSourceCounter snapshot for the current
// process. --root is accepted (unused) so the help line matches siblings.
func runTrustStats(args []string) error {
	fs := newFlagSet("trust stats")
	_ = fs.String("root", ".", "workspace root (unused; accepted for symmetry)")
	if err := fs.Parse(args); err != nil {
		return err
	}
	c := l4.DefaultTrustSource
	var mem uint64
	if c != nil {
		mem = c.FromOutcomeMemory.Load()
	}
	last := l4.SnapshotLastOutcomeLookup()
	if last == "" {
		last = "(never)"
	}
	fmt.Fprintf(os.Stdout, "trust_source{from=\"outcome_memory\"}: %d\n", mem)
	fmt.Fprintf(os.Stdout, "last_outcome_lookup:                  %s\n", last)
	return nil
}