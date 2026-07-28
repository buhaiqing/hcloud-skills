package embedder

import (
	"context"
	"fmt"
)

// None is the "no sandbox" Embedder. Stage-2 rerank is disabled; the Router
// proceeds with Stage-1 manifest scoring only. The provider records itself
// in trace metadata as "none" so audits and confusion matrices can still
// reason about which decisions were made without an embedding model.
//
// The provider is the explicit no-op opt-out for environments that:
//   - cannot run a local model (CI sandboxes, air-gapped runners)
//   - have no egress to a cloud model (locked-down production)
//   - are validating Stage-1 determinism in isolation
//
// Selecting "none" via capability-registry.json or HC_EMBED_PROVIDER=none
// is the documented mechanism; the runtime never silently downgrades to it
// (no implicit network-or-model failure fallback to none).
type None struct {
	dim int
}

// Name returns the canonical provider identifier.
func (n *None) Name() string { return "none" }

// Preflight validates config for the no-op provider. All remote fields are
// surfaced as warnings because they are unused; selecting none plus cloud
// fields is permitted (the user may be migrating off cloud) but is flagged.
func (n *None) Preflight(cfg ProviderConfig) PreflightReport {
	r := PreflightReport{Provider: "none", OK: true}
	if cfg.Endpoint != "" {
		r.Warnings = append(r.Warnings, Issue{
			Field:   "endpoint",
			Message: "embedding=none skips the embed call; endpoint is unused",
			Fix:     "Remove embedding.endpoint or switch provider_name back to \"huaweicloud-modelarts\".",
		})
	}
	if cfg.AuthEnv != "" {
		r.Warnings = append(r.Warnings, Issue{
			Field:   "auth_env",
			Message: "embedding=none skips the embed call; auth_env is unused",
			Fix:     "Remove embedding.auth_env (or unset the env var after rotating any exposed credential).",
		})
	}
	if cfg.FallbackChain != nil && len(cfg.FallbackChain) > 0 {
		r.Warnings = append(r.Warnings, Issue{
			Field:   "fallback_chain",
			Message: "fallback_chain is ignored when the primary provider is \"none\"; remove it to avoid confusion",
			Fix:     "Drop fallback_chain OR set a non-none primary provider.",
		})
	}
	r.Info = append(r.Info, "Stage-2 rerank disabled; trace will report rerank_mode=\"skipped\"")
	return r
}

// Init is a no-op; the provider has no resource surface.
func (n *None) Init(ctx context.Context, cfg ProviderConfig) error {
	if cfg.Dim == 0 {
		n.dim = DefaultDim
	} else {
		n.dim = cfg.Dim
	}
	return nil
}

// Close is a no-op (kept for the interface contract).
func (n *None) Close() error { return nil }

// Health is always nil; the provider carries no failure surface.
func (n *None) Health(ctx context.Context) error { return nil }

// Embed returns an empty vector. The Router recognises this provider by name
// and records `rerank_mode=skipped` instead of running a Stage-2 score. The
// returned nil slice is the explicit "no embedding" signal.
func (n *None) Embed(ctx context.Context, text string) ([]float32, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return nil, nil
}

// Score returns 0.0 for every pair; callers must treat the provider as a
// Stage-2 skip when they see it. The function exists to satisfy the
// Embedder interface and is intentionally not used by the Router.
func (n *None) Score(ctx context.Context, query, doc string) (float64, error) {
	return 0, fmt.Errorf("embedding=none: Stage-2 scoring is disabled; rerank_mode=skipped. Fix: remove provider_name=none or add a non-none primary provider")
}
