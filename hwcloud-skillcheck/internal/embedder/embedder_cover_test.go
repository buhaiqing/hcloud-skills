package embedder

import (
	"context"
	"errors"
	"testing"
)

// ─── buildStatus tests ───────────────────────────────────────────────────────

func TestBuildStatus_LocalMode(t *testing.T) {
	st := buildStatus(ProviderConfig{ProviderName: "local-fasttext", Dim: 128}, &LocalFasttext{}, nil)
	if st.Mode != "local" {
		t.Fatalf("want local, got %s", st.Mode)
	}
	if st.Active != "local-fasttext" {
		t.Fatalf("want local-fasttext, got %s", st.Active)
	}
	if st.Dim != 128 {
		t.Fatalf("want dim=128, got %d", st.Dim)
	}
	if !st.LastPreflight.OK {
		t.Fatal("want preflight OK")
	}
}

func TestBuildStatus_CloudMode(t *testing.T) {
	st := buildStatus(ProviderConfig{ProviderName: "huaweicloud-modelarts", Dim: 256}, nil, errors.New("net unreachable"))
	if st.Mode != "cloud" {
		t.Fatalf("want cloud, got %s", st.Mode)
	}
	if st.LastPreflight.OK {
		t.Fatal("want preflight FAIL")
	}
	if len(st.LastPreflight.Errors) == 0 {
		t.Fatal("want error in preflight report")
	}
}

func TestBuildStatus_FallbackChain(t *testing.T) {
	cfg := ProviderConfig{ProviderName: "cloud", FallbackChain: []string{"local-fasttext", "none"}}
	st := buildStatus(cfg, nil, nil)
	if len(st.FallbackChain) != 2 {
		t.Fatalf("want 2 fallbacks, got %d", len(st.FallbackChain))
	}
}

// ─── Reset / Default / Status tests ─────────────────────────────────────────

func TestReset_ClearsDefaultState(t *testing.T) {
	// Seed default via Default()
	emb, err := Default(context.Background())
	if err != nil {
		t.Fatalf("Default() seed failed: %v", err)
	}
	if emb == nil {
		t.Fatal("Default() returned nil embedder")
	}

	// Reset
	Reset()

	// After reset, calling Default() should re-initialize
	emb2, err2 := Default(context.Background())
	if err2 != nil {
		t.Fatalf("Default() after Reset failed: %v", err2)
	}
	if emb2 == nil {
		t.Fatal("Default() after Reset returned nil")
	}
}

func TestStatus_AfterDefaultInit(t *testing.T) {
	Reset()
	_, err := Default(context.Background())
	if err != nil {
		t.Fatalf("Default() failed: %v", err)
	}
	st := Status()
	if st.Active == "" {
		t.Fatal("Status().Active is empty after Default()")
	}
	if st.Dim == 0 {
		t.Fatal("Status().Dim is 0 after Default()")
	}
}

// ─── NewWithFallback tests ───────────────────────────────────────────────────

func TestNewWithFallback_PrimarySucceeds(t *testing.T) {
	Reset()
	cfg := ProviderConfig{ProviderName: "local-fasttext"}
	emb, err := NewWithFallback(context.Background(), cfg)
	if err != nil {
		t.Fatalf("NewWithFallback failed: %v", err)
	}
	if emb == nil {
		t.Fatal("expected embedder, got nil")
	}
	if emb.Name() != "local-fasttext" {
		t.Fatalf("want local-fasttext, got %s", emb.Name())
	}
}

func TestNewWithFallback_PrimaryFailsFallbackSucceeds(t *testing.T) {
	Reset()
	cfg := ProviderConfig{
		ProviderName:  "onnx-runtime", // unsupported → will fail
		FallbackChain: []string{"local-fasttext"},
	}
	emb, err := NewWithFallback(context.Background(), cfg)
	if err != nil {
		t.Fatalf("NewWithFallback fallback should succeed: %v", err)
	}
	if emb == nil {
		t.Fatal("expected fallback embedder, got nil")
	}
	if emb.Name() != "local-fasttext" {
		t.Fatalf("want fallback local-fasttext, got %s", emb.Name())
	}
}

func TestNewWithFallback_BothFail(t *testing.T) {
	Reset()
	cfg := ProviderConfig{
		ProviderName:  "onnx-runtime",
		FallbackChain: []string{"onnx-runtime"}, // also unsupported
	}
	_, err := NewWithFallback(context.Background(), cfg)
	if err == nil {
		t.Fatal("expected error when both primary and fallback fail")
	}
}

// ─── None provider 0% methods ────────────────────────────────────────────────

func TestNone_CloseAlwaysNil(t *testing.T) {
	n := &None{}
	if err := n.Close(); err != nil {
		t.Fatalf("None.Close() should return nil: %v", err)
	}
}

func TestNone_HealthAlwaysNil(t *testing.T) {
	n := &None{}
	if err := n.Health(context.Background()); err != nil {
		t.Fatalf("None.Health() should return nil: %v", err)
	}
}

func TestNone_Name(t *testing.T) {
	n := &None{}
	if n.Name() != "none" {
		t.Fatalf("want none, got %s", n.Name())
	}
}

// ─── LocalFasttext 0% methods ────────────────────────────────────────────────

func TestLocalFasttext_Health(t *testing.T) {
	n := &LocalFasttext{}
	if err := n.Health(context.Background()); err != nil {
		t.Fatalf("LocalFasttext.Health() should return nil: %v", err)
	}
}

func TestLocalFasttext_Score_NotImplemented(t *testing.T) {
	n := &LocalFasttext{}
	_, err := n.Score(context.Background(), "query", "doc")
	// Score is stage-2 rerank; LocalFasttext only does Embed
	if err == nil {
		t.Fatal("expected error for Score (not implemented in local-fasttext)")
	}
}

// ─── HuaweiCloud 0% methods ──────────────────────────────────────────────────

func TestHuaweiCloud_Name(t *testing.T) {
	h := &HuaweiCloud{}
	if h.Name() != "huaweicloud-modelarts" {
		t.Fatalf("want huaweicloud-modelarts, got %s", h.Name())
	}
}

func TestHuaweiCloud_Close(t *testing.T) {
	// HuaweiCloud.Close should return nil (no-op or cleanup)
	h := &HuaweiCloud{}
	if err := h.Close(); err != nil {
		t.Fatalf("HuaweiCloud.Close() returned error: %v", err)
	}
}
