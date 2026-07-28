package embedder

import (
	"context"
	"fmt"
	"hash/fnv"
	"math"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

// LocalFasttext is the default Embedder. Pure-Go, no vocab, no model file.
// Produces L2-normalized vectors from char n-gram hashing trick (the fastText
// "feature hashing" technique). Quality is below transformer-grade but
// sufficient for re-ranking the 5 candidates that survived Stage-1.
type LocalFasttext struct {
	dim      int
	maxBytes int
	maxQPS   int

	qpsMu     sync.Mutex
	qpsWindow time.Time
	qpsCount  atomic.Int64

	closeOnce sync.Once
}

// MaxLocalFasttextQPS is a defense-in-depth rate limit per process. Stage-2
// latency budgets assume <<1000 calls/sec; anything more suggests upstream
// RateBudget violations or a runaway loop.
const MaxLocalFasttextQPS = 5000

func newLocalFasttext(cfg ProviderConfig) (*LocalFasttext, error) {
	if cfg.Dim == 0 {
		cfg.Dim = DefaultDim
	}
	if cfg.Dim < MinDim || cfg.Dim > MaxDim {
		return nil, fmt.Errorf("local-fasttext dim=%d is outside [%d, %d]. Fix: change embedding.dim in capability-registry.json, or remove the field (default is %d)", cfg.Dim, MinDim, MaxDim, DefaultDim)
	}
	return &LocalFasttext{
		dim:      cfg.Dim,
		maxBytes: MaxInputBytes,
		maxQPS:   MaxLocalFasttextQPS,
	}, nil
}

// Name returns the canonical provider identifier.
func (e *LocalFasttext) Name() string { return "local-fasttext" }

// Preflight validates config without allocating resources. Safe to call
// before Init; will be re-run by Init as the first step.
func (e *LocalFasttext) Preflight(cfg ProviderConfig) PreflightReport {
	r := PreflightReport{Provider: "local-fasttext"}
	if cfg.Endpoint != "" {
		r.Warnings = append(r.Warnings, Issue{
			Field:   "endpoint",
			Message: "local-fasttext is in-process; embedding.endpoint is unused",
			Fix:     "Remove embedding.endpoint from capability-registry.json (it only applies to remote providers).",
		})
	}
	if cfg.AuthEnv != "" {
		r.Warnings = append(r.Warnings, Issue{
			Field:   "auth_env",
			Message: "local-fasttext is in-process; embedding.auth_env is unused",
			Fix:     "Remove embedding.auth_env. Switching to a remote provider later is a separate config edit (provider_name + endpoint + auth_env).",
		})
	}
	if cfg.ProjectID != "" {
		r.Warnings = append(r.Warnings, Issue{
			Field:   "project_id",
			Message: "local-fasttext is in-process; embedding.project_id is unused",
			Fix:     "Remove embedding.project_id (Huawei Cloud only).",
		})
	}
	if cfg.Dim != 0 && (cfg.Dim < MinDim || cfg.Dim > MaxDim) {
		r.Errors = append(r.Errors, Issue{
			Field:   "dim",
			Message: fmt.Sprintf("dim=%d is outside allowed range [%d, %d]", cfg.Dim, MinDim, MaxDim),
			Fix:     fmt.Sprintf("Set embedding.dim to a value in [%d, %d], or remove the dim field to use the default %d.", MinDim, MaxDim, DefaultDim),
			DocURL:  "docs/deployment-guide.md#1.5-router-policy-registry",
		})
	}
	if cfg.Dim == 0 {
		r.Info = append(r.Info, fmt.Sprintf("dim not set; defaulting to %d", DefaultDim))
	}
	r.OK = len(r.Errors) == 0
	return r
}

// Init validates config and stores immutable settings. Idempotent.
func (e *LocalFasttext) Init(ctx context.Context, cfg ProviderConfig) error {
	if e.dim != 0 {
		return nil // already initialised
	}
	if report := e.Preflight(cfg); report.OK == false {
		return report
	}
	if cfg.Dim == 0 {
		cfg.Dim = DefaultDim
	}
	e.dim = cfg.Dim
	e.maxBytes = MaxInputBytes
	e.maxQPS = MaxLocalFasttextQPS
	return nil
}

// Close releases resources. Idempotent.
func (e *LocalFasttext) Close() error {
	e.closeOnce.Do(func() {})
	return nil
}

// Health always returns nil for the in-process provider.
func (e *LocalFasttext) Health(ctx context.Context) error { return nil }

// Embed returns a fresh L2-normalized vector for text. Security gates:
//  1. context cancellation checked first
//  2. panic recovered (hash bugs cannot crash the router)
//  3. input byte-length cap
//  4. lightweight per-process rate gate
//  5. L2-normalize + NaN/Inf scrub
//  6. return a copy so the caller cannot mutate internal state
func (e *LocalFasttext) Embed(ctx context.Context, text string) ([]float32, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}
	if err := e.allowQPS(); err != nil {
		return nil, err
	}
	if len(text) > e.maxBytes {
		return nil, fmt.Errorf("input length %d exceeds security cap %d. Fix: shorten the query, or raise HC_EMBED_MAX_BYTES env var (not yet implemented; default %d)", len(text), e.maxBytes, e.maxBytes)
	}
	var recovered any
	defer func() { recovered = recover() }()
	vec := e.compute(text)
	if recovered != nil {
		return nil, fmt.Errorf("local-fasttext internal panic recovered: %v", recovered)
	}
	e.l2Normalize(vec)
	out := make([]float32, len(vec))
	copy(out, vec)
	return out, nil
}

// Score returns cosine similarity between query and doc.
func (e *LocalFasttext) Score(ctx context.Context, query, doc string) (float64, error) {
	qv, err := e.Embed(ctx, query)
	if err != nil {
		return 0, err
	}
	dv, err := e.Embed(ctx, doc)
	if err != nil {
		return 0, err
	}
	return cosine(qv, dv), nil
}

// compute returns a fresh dim-length vector using the char n-gram hashing trick.
func (e *LocalFasttext) compute(text string) []float32 {
	vec := make([]float32, e.dim)
	text = strings.ToLower(text)
	runes := []rune(text)
	if len(runes) == 0 {
		return vec
	}
	// 1- to 3- char n-grams of the lower-cased text.
	for n := 1; n <= 3; n++ {
		for i := 0; i+n <= len(runes); i++ {
			gram := string(runes[i : i+n])
			idx, sign := e.bucket(gram)
			vec[idx] += float32(sign)
		}
	}
	return vec
}

func (e *LocalFasttext) bucket(gram string) (int, int) {
	h := fnv.New64a()
	_, _ = h.Write([]byte(gram))
	idx := int(h.Sum64() % uint64(e.dim))
	h2 := fnv.New64a()
	_, _ = h2.Write([]byte(gram))
	_, _ = h2.Write([]byte("/s"))
	sign := 1
	if h2.Sum64()&1 == 1 {
		sign = -1
	}
	return idx, sign
}

func (e *LocalFasttext) l2Normalize(v []float32) {
	var sum float64
	for _, x := range v {
		sum += float64(x) * float64(x)
	}
	if sum == 0 || math.IsNaN(sum) {
		return
	}
	inv := 1.0 / math.Sqrt(sum)
	for i, x := range v {
		y := float64(x) * inv
		if math.IsNaN(y) || math.IsInf(y, 0) {
			v[i] = 0
			continue
		}
		v[i] = float32(y)
	}
}

func cosine(a, b []float32) float64 {
	if len(a) != len(b) {
		return 0
	}
	var dot, an, bn float64
	for i := range a {
		dot += float64(a[i]) * float64(b[i])
		an += float64(a[i]) * float64(a[i])
		bn += float64(b[i]) * float64(b[i])
	}
	if an == 0 || bn == 0 {
		return 0
	}
	return dot / (math.Sqrt(an) * math.Sqrt(bn))
}

func (e *LocalFasttext) allowQPS() error {
	if e.maxQPS <= 0 {
		return nil
	}
	now := time.Now()
	e.qpsMu.Lock()
	defer e.qpsMu.Unlock()
	if now.Sub(e.qpsWindow) > time.Second {
		e.qpsWindow = now
		e.qpsCount.Store(0)
	}
	n := e.qpsCount.Add(1)
	if int(n) > e.maxQPS {
		return fmt.Errorf("local-fasttext rate cap exceeded (%d calls/sec). Fix: reduce router traffic or raise MaxLocalFasttextQPS in code", e.maxQPS)
	}
	return nil
}
