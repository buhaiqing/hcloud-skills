package l4

import (
	"crypto/rand"
	"encoding/hex"
	"time"
)

// NowISO returns the current UTC timestamp in ISO-8601 (Z) form.
// Mirrors Python's _now_iso() in scripts/{dynamic_orchestration,topology_graph,predictive_ops,progressive_trust,runtime_orchestrator}.py.
func NowISO() string {
	return time.Now().UTC().Format("2006-01-02T15:04:05Z")
}

func randomHex(n int) string {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		// Fallback: time-based hex (less random but never fails).
		ts := time.Now().UTC().UnixNano()
		for i := range b {
			b[i] = byte(ts >> (i % 8 * 8))
		}
	}
	return hex.EncodeToString(b)
}
