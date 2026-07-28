package confusion

import (
	"os"
	"path/filepath"
	"testing"
)

func TestConfusionMatrixDerivable(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, "audit-results")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	trace := `{"skill":"huaweicloud-ecs-ops","operation_intent":{"intended_skill":"huaweicloud-ecs-ops"},"router_decision":{"chosen":"huaweicloud-rds-ops"}}`
	if err := os.WriteFile(filepath.Join(dir, "gcl-trace-one.json"), []byte(trace), 0o600); err != nil {
		t.Fatal(err)
	}
	got, err := Derive(root)
	if err != nil {
		t.Fatal(err)
	}
	if got.Top1VsIntended["huaweicloud-ecs-ops"]["huaweicloud-rds-ops"] != 1 {
		t.Fatalf("unexpected matrix: %+v", got)
	}
}
