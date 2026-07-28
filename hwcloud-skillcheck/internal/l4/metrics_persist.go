package l4

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
)

// CounterSnapshot is the on-disk shape under .l4-memory/metrics.json so a
// separate `metrics` scrape process can observe counters produced by
// `l4 handle` (code-review HIGH: in-process atomics alone are silent-wrong).
type CounterSnapshot struct {
	PreExecSkip         uint64 `json:"pre_exec_skip"`
	PostFailureRetry    uint64 `json:"post_failure_retry"`
	PostFailureEscalate uint64 `json:"post_failure_escalate"`
	TrustFromOutcome    uint64 `json:"trust_from_outcome"`
}

var (
	metricsPersistMu   sync.Mutex
	metricsPersistRoot string
)

// SetMetricsPersistRoot enables durable counter updates under
// <root>/.l4-memory/metrics.json. Empty root disables persistence.
func SetMetricsPersistRoot(root string) {
	metricsPersistMu.Lock()
	defer metricsPersistMu.Unlock()
	metricsPersistRoot = root
}

func metricsPersistPath(root string) string {
	return filepath.Join(root, ".l4-memory", "metrics.json")
}

// LoadCounterSnapshot reads persisted counters. Missing file → zero snapshot.
func LoadCounterSnapshot(root string) (CounterSnapshot, error) {
	var snap CounterSnapshot
	if root == "" {
		return snap, nil
	}
	raw, err := os.ReadFile(metricsPersistPath(root))
	if err != nil {
		if os.IsNotExist(err) {
			return snap, nil
		}
		return snap, err
	}
	if err := json.Unmarshal(raw, &snap); err != nil {
		return snap, err
	}
	return snap, nil
}

func bumpPersisted(mutate func(*CounterSnapshot)) {
	metricsPersistMu.Lock()
	root := metricsPersistRoot
	metricsPersistMu.Unlock()
	if root == "" {
		return
	}
	if _, err := EnsureMemoryDir(root); err != nil {
		return
	}
	metricsPersistMu.Lock()
	defer metricsPersistMu.Unlock()
	snap, _ := loadCounterSnapshotUnlocked(root)
	mutate(&snap)
	_ = saveCounterSnapshotUnlocked(root, snap)
}

func loadCounterSnapshotUnlocked(root string) (CounterSnapshot, error) {
	var snap CounterSnapshot
	raw, err := os.ReadFile(metricsPersistPath(root))
	if err != nil {
		if os.IsNotExist(err) {
			return snap, nil
		}
		return snap, err
	}
	if err := json.Unmarshal(raw, &snap); err != nil {
		return snap, err
	}
	return snap, nil
}

func saveCounterSnapshotUnlocked(root string, snap CounterSnapshot) error {
	raw, err := json.MarshalIndent(snap, "", "  ")
	if err != nil {
		return err
	}
	tmp := metricsPersistPath(root) + ".tmp"
	if err := os.WriteFile(tmp, append(raw, '\n'), 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, metricsPersistPath(root))
}

// WritePrometheusFromRoot emits counters from the persisted snapshot under
// root (preferred for the metrics CLI). Falls back to in-process atomics
// when the file is absent.
func WritePrometheusFromRoot(w io.Writer, root string) error {
	snap, err := LoadCounterSnapshot(root)
	if err != nil {
		return err
	}
	// Merge: take max(persisted, in-process) so a same-process scrape still
	// sees live increments before the next disk flush completes.
	if DefaultHealingMetrics != nil {
		if v := DefaultHealingMetrics.PreExecSkip.Load(); v > snap.PreExecSkip {
			snap.PreExecSkip = v
		}
		if v := DefaultHealingMetrics.PostFailureRetry.Load(); v > snap.PostFailureRetry {
			snap.PostFailureRetry = v
		}
		if v := DefaultHealingMetrics.PostFailureEscalate.Load(); v > snap.PostFailureEscalate {
			snap.PostFailureEscalate = v
		}
	}
	if DefaultTrustSource != nil {
		if v := DefaultTrustSource.FromOutcomeMemory.Load(); v > snap.TrustFromOutcome {
			snap.TrustFromOutcome = v
		}
	}
	if snap == (CounterSnapshot{}) {
		// No disk + no live → still emit zero series via in-process path.
		return WritePrometheus(w)
	}
	return writeSnapshotPrometheus(w, snap)
}

func writeSnapshotPrometheus(w io.Writer, snap CounterSnapshot) error {
	metrics := []struct {
		name, help string
		value      uint64
	}{
		{"l4_healing_pre_exec_skip_total", "PreExecHook skip decisions", snap.PreExecSkip},
		{"l4_healing_post_failure_retry_total", "PostFailureHook retry decisions", snap.PostFailureRetry},
		{"l4_healing_post_failure_escalate_total", "PostFailureHook escalate decisions", snap.PostFailureEscalate},
	}
	for _, metric := range metrics {
		if _, err := fmt.Fprintf(w, "# HELP %s %s\n# TYPE %s counter\n%s %d\n",
			metric.name, metric.help, metric.name, metric.name, metric.value); err != nil {
			return err
		}
	}
	_, err := fmt.Fprintf(w,
		"# HELP l4_trust_source_lookups_total Trust lookup count by source\n"+
			"# TYPE l4_trust_source_lookups_total counter\n"+
			`l4_trust_source_lookups_total{from="outcome_memory"} %d`+"\n",
		snap.TrustFromOutcome)
	return err
}
