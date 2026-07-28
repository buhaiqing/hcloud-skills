// Package embedder implements the Strategy interface for Stage-2 embedding
// under the P2 Router Provider Strategy (see docs/superpowers/specs/
// 2026-07-27-harness-runtime-p1p2-design.md §4.2.4).
//
// Default sandbox mode (per spec §4.2.4): local-fasttext, no setup required.
// Cloud sandbox mode requires three env vars (endpoint, auth_env containing
// an AK/SK pair or token, optional project_id) and is reached only when the
// capability-registry.json embedding block OR HC_EMBED_PROVIDER env var
// explicitly selects it. There is no implicit egress.
//
// Lifecycle:
//
//	New(cfg) → Preflight(cfg) (validates config, returns multi-issue report)
//	  → Init(ctx, cfg) (allocates real resources)
//	  → Embed(ctx, text) []float32 (the hot path)
//	  → Close()
//
// The Preflight stage is the user-facing contract: configuration errors are
// surfaced with concrete remediation strings BEFORE any network or native call
// happens. PreflightReport carries Errors + Warnings + per-Issue.Fix, so
// users see every problem at once.
package embedder

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"sync"
)

// DefaultDim is the canonical embedding dimension; matches all-MiniLM-L6-v2
// when an ONNX impl is wired. LocalFasttext is dimension-agnostic and accepts
// any value in [MinDim, MaxDim].
const (
	DefaultDim    = 384
	MinDim        = 64
	MaxDim        = 4096
	MaxInputBytes = 64 * 1024 // 64 KiB per text — far above any realistic skill description
)

// Embedder is the Strategy interface. The router reads the same shape
// regardless of which provider is configured; the provider name is recorded
// in trace metadata for audit.
type Embedder interface {
	Name() string
	Init(ctx context.Context, cfg ProviderConfig) error
	Embed(ctx context.Context, text string) ([]float32, error)
	Score(ctx context.Context, query, doc string) (float64, error)
	Health(ctx context.Context) error
	Close() error
	Preflight(cfg ProviderConfig) PreflightReport
}

// ProviderConfig is the JSON-serializable configuration loaded from
// capability-registry.json's `embedding` block. All fields are validated by
// Preflight before Init runs.
//
// On the wire (capability-registry.json):
//
//	"embedding": {
//	  "mode":          "local" | "cloud",                   // UX shorthand
//	  "provider_name": "local-fasttext" | "huaweicloud-modelarts" | "onnx-runtime",
//	  "endpoint":      "<url>",                              // remote only; HTTPS required
//	  "auth_env":      "<env-var-name>",                     // mandatory for cloud
//	  "project_id":    "<hw-project-id>",                    // Huawei Cloud only
//	  "dim":           384,                                  // [MinDim, MaxDim]
//	  "timeout_ms":    500,                                  // per-Embed-call hard cap
//	  "fallback_chain": ["local-fasttext"],                  // offline degradation
//	  "extra":         { ... }                               // provider-specific knobs
//	}
type ProviderConfig struct {
	Mode          string            `json:"mode,omitempty"`
	ProviderName  string            `json:"provider_name,omitempty"`
	Endpoint      string            `json:"endpoint,omitempty"`
	AuthEnv       string            `json:"auth_env,omitempty"`
	ProjectID     string            `json:"project_id,omitempty"`
	Dim           int               `json:"dim,omitempty"`
	TimeoutMs     int               `json:"timeout_ms,omitempty"`
	FallbackChain []string          `json:"fallback_chain,omitempty"`
	Extra         map[string]string `json:"extra,omitempty"`
}

// PreflightReport is the structured output of an embedder's config validation.
type PreflightReport struct {
	Provider string   `json:"provider"`
	OK       bool     `json:"ok"`
	Errors   []Issue  `json:"errors,omitempty"`
	Warnings []Issue  `json:"warnings,omitempty"`
	Info     []string `json:"info,omitempty"`
}

// Issue is a single config-level concern. Fix is required for every Error,
// advisory only for Warning. DocURL references the user-manual section.
type Issue struct {
	Field   string `json:"field"`
	Message string `json:"message"`
	Fix     string `json:"fix"`
	DocURL  string `json:"doc_url,omitempty"`
}

// Error renders the report as a human-friendly multi-line message suitable
// for CLI stderr.
func (r PreflightReport) Error() string {
	if r.OK {
		return ""
	}
	out := fmt.Sprintf("sandbox preflight failed: provider=%s", r.Provider)
	for _, e := range r.Errors {
		out += "\n  - " + e.Field + ": " + e.Message
		if e.Fix != "" {
			out += "\n    Fix: " + e.Fix
		}
		if e.DocURL != "" {
			out += "\n    See: " + e.DocURL
		}
	}
	for _, w := range r.Warnings {
		out += "\n  ! " + w.Field + ": " + w.Message
		if w.Fix != "" {
			out += "\n    Fix: " + w.Fix
		}
	}
	return out
}

// Validate returns nil when OK, otherwise r itself.
func (r PreflightReport) Validate() error {
	if r.OK {
		return nil
	}
	return r
}

// resolveProviderName collapses (Mode, ProviderName) into the canonical name.
func resolveProviderName(cfg ProviderConfig) (string, error) {
	if cfg.Mode == "" && cfg.ProviderName == "" {
		return "local-fasttext", nil // default per §4.2.4
	}
	if cfg.Mode == "local" && cfg.ProviderName == "" {
		return "local-fasttext", nil
	}
	if cfg.Mode == "cloud" && cfg.ProviderName == "" {
		return "", errors.New(`mode="cloud" requires explicit provider_name (e.g. "huaweicloud-modelarts"). Fix: set embedding.provider_name OR remove embedding.mode.`)
	}
	allowMode := map[string]bool{"local": true, "cloud": true, "off": true}
	if cfg.ProviderName != "" && cfg.Mode != "" && !allowMode[cfg.Mode] {
		return "", fmt.Errorf(`mode=%q is unknown; allowed: "local", "cloud", "off", or omit. Fix: remove the mode field or change to one of the allowed values.`, cfg.Mode)
	}
	if cfg.ProviderName != "" && cfg.Mode != "" {
		want := map[string]string{"local": "local-fasttext", "cloud": "huaweicloud-modelarts", "off": "none"}[cfg.Mode]
		if want != "" && cfg.ProviderName != want {
			return "", fmt.Errorf(`mode=%q conflicts with provider_name=%q. Fix: either remove mode (provider_name alone is sufficient) or set them to agree.`, cfg.Mode, cfg.ProviderName)
		}
	}
	if cfg.ProviderName != "" {
		return cfg.ProviderName, nil
	}
	return "local-fasttext", nil
}

// ProviderStatus is what `router info` / `router embed-test` surfaces.
type ProviderStatus struct {
	Active        string          `json:"active"`
	Mode          string          `json:"mode"`
	FallbackChain []string        `json:"fallback_chain"`
	Dim           int             `json:"dim"`
	TimeoutMs     int             `json:"timeout_ms,omitempty"`
	LastPreflight PreflightReport `json:"last_preflight"`
}

var (
	defaultOnce   sync.Once
	defaultEmbed  Embedder
	defaultStatus ProviderStatus
	defaultErr    error
)

// Reset clears the package-level default. Test-only.
func Reset() {
	defaultOnce = sync.Once{}
	defaultEmbed = nil
	defaultStatus = ProviderStatus{}
	defaultErr = nil
}

// Default returns the package-level embedder, lazily initialized.
func Default(ctx context.Context) (Embedder, error) {
	defaultOnce.Do(func() {
		cfg, lerr := LoadConfig()
		if lerr != nil {
			defaultErr = fmt.Errorf("load embedding config: %w", lerr)
			return
		}
		emb, perr := PreflightAndInit(ctx, cfg)
		defaultStatus = buildStatus(cfg, emb, perr)
		defaultEmbed = emb
		defaultErr = perr
	})
	return defaultEmbed, defaultErr
}

// Status returns the most recent Default() initialization status.
func Status() ProviderStatus { return defaultStatus }

// PreflightAndInit runs the canonical ordering: Preflight → Init.
func PreflightAndInit(ctx context.Context, cfg ProviderConfig) (Embedder, error) {
	providerName, err := resolveProviderName(cfg)
	if err != nil {
		return nil, fmt.Errorf("embedding config: %w", err)
	}
	cfg.ProviderName = providerName

	probe, err := NewUninitialized(providerName)
	if err != nil {
		return nil, err
	}
	report := probe.Preflight(cfg)
	if vErr := report.Validate(); vErr != nil {
		return nil, vErr
	}
	if err := probe.Init(ctx, cfg); err != nil {
		return nil, fmt.Errorf("init %q failed: %w", providerName, err)
	}
	return probe, nil
}

// NewUninitialized returns an unconfigured Embedder for inspection.
func NewUninitialized(name string) (Embedder, error) {
	switch name {
	case "local-fasttext":
		return &LocalFasttext{}, nil
	case "huaweicloud-modelarts":
		return &HuaweiCloud{}, nil
	case "none":
		return &None{}, nil
	case "onnx-runtime":
		return nil, fmt.Errorf("provider %q requires vendor provisioning (libonnxruntime + header); see audit-results/p2-blockers.md", name)
	default:
		return nil, fmt.Errorf("unknown embedding provider %q. Available: local-fasttext, huaweicloud-modelarts, none, onnx-runtime. Fix: set embedding.provider_name to one of these.", name)
	}
}

// NewWithFallback tries the primary first; if preflight or init fails, each
// fallback is tried in order.
func NewWithFallback(ctx context.Context, cfg ProviderConfig) (Embedder, error) {
	primary, err := PreflightAndInit(ctx, cfg)
	if err == nil {
		return primary, nil
	}
	for _, fbName := range cfg.FallbackChain {
		fbCfg := cfg
		fbCfg.ProviderName = fbName
		fbCfg.Mode = ""
		emb, ferr := PreflightAndInit(ctx, fbCfg)
		if ferr == nil {
			return emb, nil
		}
	}
	return nil, fmt.Errorf("primary provider %q init failed and no fallback succeeded: %w", cfg.ProviderName, err)
}

// LoadConfig reads embedding config from (in priority order):
//  1. HC_EMBED_PROVIDER env var
//  2. HC_CAPABILITY_REGISTRY env var (path to capability-registry.json)
//  3. ./capability-registry.json relative to CWD
func LoadConfig() (ProviderConfig, error) {
	cfg := ProviderConfig{
		Mode:          "local",
		ProviderName:  "local-fasttext",
		Dim:           DefaultDim,
		TimeoutMs:     500,
		FallbackChain: []string{"local-fasttext"},
	}
	if v := os.Getenv("HC_EMBED_PROVIDER"); v != "" {
		cfg.ProviderName = v
		if v == "local-fasttext" {
			cfg.Mode = "local"
		} else {
			cfg.Mode = "cloud"
		}
	}
	registryPath := os.Getenv("HC_CAPABILITY_REGISTRY")
	if registryPath == "" {
		registryPath = "capability-registry.json"
	}
	data, err := os.ReadFile(registryPath)
	if err != nil {
		return cfg, nil
	}
	var raw struct {
		Embedding ProviderConfig `json:"embedding"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return cfg, fmt.Errorf("parse %s: %w", registryPath, err)
	}
	if raw.Embedding.ProviderName != "" || raw.Embedding.Mode != "" {
		cfg = raw.Embedding
		if v := os.Getenv("HC_EMBED_PROVIDER"); v != "" {
			cfg.ProviderName = v
			switch v {
			case "local-fasttext":
				cfg.Mode = "local"
			case "none":
				cfg.Mode = "off"
			default:
				cfg.Mode = "cloud"
			}
		}
	}
	if cfg.Dim == 0 {
		cfg.Dim = DefaultDim
	}
	return cfg, nil
}

func buildStatus(cfg ProviderConfig, emb Embedder, perr error) ProviderStatus {
	mode := "local"
	if cfg.ProviderName != "" && cfg.ProviderName != "local-fasttext" {
		mode = "cloud"
	}
	st := ProviderStatus{
		Active:        cfg.ProviderName,
		Mode:          mode,
		FallbackChain: cfg.FallbackChain,
		Dim:           cfg.Dim,
		TimeoutMs:     cfg.TimeoutMs,
	}
	if emb != nil {
		st.LastPreflight = PreflightReport{Provider: emb.Name(), OK: true, Info: []string{"active"}}
	}
	if perr != nil {
		st.LastPreflight = PreflightReport{Provider: cfg.ProviderName, OK: false, Errors: []Issue{{Field: "init", Message: perr.Error()}}}
	}
	return st
}
