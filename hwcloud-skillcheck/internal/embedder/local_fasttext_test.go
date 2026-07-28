package embedder

import (
	"context"
	"errors"
	"math"
	"reflect"
	"strings"
	"testing"
)

func initializedLocal(t *testing.T, dim int) *LocalFasttext {
	t.Helper()
	emb, err := PreflightAndInit(context.Background(), ProviderConfig{ProviderName: "local-fasttext", Dim: dim})
	if err != nil {
		t.Fatal(err)
	}
	return emb.(*LocalFasttext)
}

func TestLocalFasttextDeterministicNormalizedAndCopied(t *testing.T) {
	emb := initializedLocal(t, 128)
	first, err := emb.Embed(context.Background(), "list ECS servers")
	if err != nil {
		t.Fatal(err)
	}
	second, err := emb.Embed(context.Background(), "list ECS servers")
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(first, second) {
		t.Fatal("same input produced different vectors")
	}
	var norm float64
	for _, value := range first {
		if math.IsNaN(float64(value)) || math.IsInf(float64(value), 0) {
			t.Fatalf("non-finite value: %v", value)
		}
		norm += float64(value * value)
	}
	if math.Abs(math.Sqrt(norm)-1) > 1e-5 {
		t.Fatalf("vector is not L2-normalized: %f", math.Sqrt(norm))
	}
	first[0] = 999
	third, _ := emb.Embed(context.Background(), "list ECS servers")
	if third[0] == 999 {
		t.Fatal("caller mutation leaked into later result")
	}
}

func TestLocalFasttextSecurityGates(t *testing.T) {
	emb := initializedLocal(t, 128)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := emb.Embed(ctx, "x"); !errors.Is(err, context.Canceled) {
		t.Fatalf("cancel error=%v", err)
	}
	if _, err := emb.Embed(context.Background(), strings.Repeat("x", MaxInputBytes+1)); err == nil || !strings.Contains(err.Error(), "Fix:") {
		t.Fatalf("want friendly size error, got %v", err)
	}
	emb.maxQPS = 1
	emb.qpsWindow = emb.qpsWindow.Add(-2)
	emb.qpsCount.Store(0)
	if _, err := emb.Embed(context.Background(), "one"); err != nil {
		t.Fatal(err)
	}
	if _, err := emb.Embed(context.Background(), "two"); err == nil || !strings.Contains(err.Error(), "rate cap") {
		t.Fatalf("want rate cap error, got %v", err)
	}
	if err := emb.Close(); err != nil {
		t.Fatal(err)
	}
	if err := emb.Close(); err != nil {
		t.Fatalf("second close must be idempotent: %v", err)
	}
}

func TestLocalFasttextPreflightAggregatesWarningsAndErrors(t *testing.T) {
	report := (&LocalFasttext{}).Preflight(ProviderConfig{Dim: 1, Endpoint: "https://unused", AuthEnv: "UNUSED", ProjectID: "unused"})
	if report.OK || len(report.Errors) != 1 || len(report.Warnings) != 3 {
		t.Fatalf("unexpected report: %+v", report)
	}
	if !strings.Contains(report.Error(), "Fix:") {
		t.Fatalf("report lacks remediation: %s", report.Error())
	}
}
