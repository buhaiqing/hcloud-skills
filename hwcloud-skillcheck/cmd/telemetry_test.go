package cmd

import "testing"

func TestTelemetryConfusionRunsFromTraceRoot(t *testing.T) {
	if err := runTelemetry([]string{"confusion", "--root", t.TempDir()}); err != nil {
		t.Fatalf("runTelemetry: %v", err)
	}
}
