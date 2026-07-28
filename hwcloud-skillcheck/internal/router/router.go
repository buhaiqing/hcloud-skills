package router

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/buhaiqing/hcloud-skills/hwcloud-skillcheck/internal/embedder"
	"github.com/buhaiqing/hcloud-skills/hwcloud-skillcheck/internal/registry"
)

const maxCandidates = 5

// Intent is the optional-safety hint paired with a request.
type Intent struct {
	SafetyClass string `json:"safety_class"`
}

// Candidate is one ranked choice in the trace.
type Candidate struct {
	Skill         string  `json:"skill"`
	ManifestScore float64 `json:"manifest_score"`
	ONNXCosine    float64 `json:"onnx_cosine"`
	Rank          int     `json:"rank"`
}

// Decision is the trace block emitted by Route.
type Decision struct {
	Request             string         `json:"request"`
	Intent              Intent         `json:"op_intent"`
	Candidates          []Candidate    `json:"candidates"`
	Chosen              string         `json:"chosen"`
	FallbackUsed        bool           `json:"fallback_used"`
	DurationMs          int            `json:"duration_ms"`
	RouterPolicyVersion string         `json:"router_policy_version"`
	ConfidenceGate      ConfidenceGate `json:"confidence_gate"`
	// RerankMode is "skipped", "embedding", or "fallback_embedding".
	RerankMode string `json:"rerank_mode"`
	// RerankSource is the active embedder name (audit-only).
	RerankSource string `json:"rerank_source"`
	// EmbeddingProviderMeta is the structured provider-state for the trace.
	EmbeddingProviderMeta EmbeddingMeta `json:"embedding_provider_meta"`
	// RouterDecisionShadow is advisory; it MUST NOT mutate Chosen or any
	// caller-visible field (rubric A2.12 + §6.1).
	RouterDecisionShadow DecisionShadow `json:"router_decision_shadow,omitempty"`
}

// EmbeddingMeta is the per-dispatch provider observability payload.
type EmbeddingMeta struct {
	Primary        string `json:"primary"`
	ActiveProvider string `json:"active_provider"`
	FallbackUsed   bool   `json:"fallback_used"`
	Dim            int    `json:"dim"`
	DurationMs     int64  `json:"embedding_duration_ms"`
	InputBytes     int    `json:"input_bytes"`
	InputSha256Pfx string `json:"input_sha256_prefix"`
}

// DecisionShadow reports what the router WOULD have decided under the
// candidate policy version. Advisory only; never mutates main decision.
type DecisionShadow struct {
	RouterPolicyCandidate string `json:"router_policy_candidate"`
	Chosen                string `json:"chosen"`
	ScoreDelta            int    `json:"score_delta"`
	MarginDelta           int    `json:"margin_delta"`
	WouldHaveChanged      bool   `json:"would_have_changed"`
}

// ConfidenceGate is the structured trace signal recorded for every dispatch
// (see docs/superpowers/specs/2026-07-27-harness-runtime-p1p2-design.md
// §4.2.3). All six fields are mandatory in the trace; populate via
// computeGate() which derives them from the canonical GateThresholds in
// capability-registry.json (no runtime setter — rubric A2.14, S3).
type ConfidenceGate struct {
	Top1Score    int    `json:"top1_score"`
	Margin       int    `json:"margin"`
	EntityMatch  string `json:"entity_match"`
	HardFiltered bool   `json:"hard_filtered"`
	Decision     string `json:"decision"`
	Rationale    string `json:"rationale"`
}

// loadEmbedder resolves the active embedder with optional fallback. The
// primary is taken from capability-registry.json (policy.Embedding). When
// the primary fails to initialise (network down, bad credential, vendor
// unreachable), the call walks cfg.FallbackChain in order. Returning the
// error from the last attempt makes the failure visible to the caller;
// callers MUST handle it.
func loadEmbedder(ctx context.Context, policy *Policy) (embedder.Embedder, bool, error) {
	if policy == nil {
		return nil, false, errors.New("router policy unavailable; cannot load embedder")
	}
	primary := policy.Embedding
	primary.ProviderName = ensureProviderName(primary)
	if primary.ProviderName == "none" {
		emb, err := embedder.PreflightAndInit(ctx, primary)
		return emb, false, err
	}
	if emb, err := embedder.PreflightAndInit(ctx, primary); err == nil {
		return emb, false, nil
	}
	for _, name := range primary.FallbackChain {
		if name == "" || name == primary.ProviderName {
			continue
		}
		cfg := primary
		cfg.ProviderName = name
		cfg.Mode = ""
		if emb, err := embedder.PreflightAndInit(ctx, cfg); err == nil {
			return emb, true, nil
		}
	}
	return nil, false, fmt.Errorf("primary embedder %q and %d fallback(s) all failed", primary.ProviderName, len(primary.FallbackChain))
}

// ensureProviderName fills in the canonical default when neither the registry
// nor the env var named a provider. Mirrors embedder.resolveProviderName.
func ensureProviderName(cfg embedder.ProviderConfig) string {
	if cfg.ProviderName != "" {
		return cfg.ProviderName
	}
	return "local-fasttext"
}

// RerankerScore is the bridge from embedder.Score to per-candidate Stage-2
// scoring. LocalReranker + Reranker interface were removed in spec v0.5.0;
// lexical Jaccard is no longer a legal Stage-2 source.
func RerankerScore(ctx context.Context, emb embedder.Embedder, request string, entry registry.Entry) (float64, error) {
	doc := entry.Name + " " + entry.Description + " " + strings.Join(entry.Inputs, " ")
	return emb.Score(ctx, request, doc)
}

// Route is the Stage-1 + Stage-2 routing entry point. Stage-1 manifest filter
// runs in-process; Stage-2 embedding is delegated to the swappable provider.
// Pass emb=nil to use the package default (configured via capability-registry.json
// or HC_EMBED_PROVIDER env var).
func Route(ctx context.Context, entries []registry.Entry, request string, intent Intent, emb embedder.Embedder) Decision {
	started := time.Now()
	if emb == nil {
		emb = mustDefaultEmbedder(ctx)
	}
	policy := activePolicy()
	first := ManifestFilter(entries, request, intent)
	bySkill := make(map[string]registry.Entry, len(entries))
	for _, entry := range entries {
		bySkill[entry.Skill] = entry
	}

	rerankMode := "embedding"
	rerankSource := emb.Name()
	dim := policy.Embedding.Dim
	embStarted := time.Now()
	if emb.Name() == "none" {
		rerankMode = "skipped"
	} else {
		for index := range first {
			score, err := RerankerScore(ctx, emb, request, bySkill[first[index].Skill])
			if err == nil {
				first[index].ONNXCosine = score
			}
		}
		sort.Slice(first, func(i, j int) bool {
			if first[i].ONNXCosine == first[j].ONNXCosine {
				if first[i].ManifestScore == first[j].ManifestScore {
					return first[i].Skill < first[j].Skill
				}
				return first[i].ManifestScore > first[j].ManifestScore
			}
			return first[i].ONNXCosine > first[j].ONNXCosine
		})
	}
	for index := range first {
		first[index].Rank = index + 1
	}
	embDuration := time.Since(embStarted).Microseconds() / 1000

	decision := Decision{
		Request:             request,
		Intent:              intent,
		Candidates:          first,
		FallbackUsed:        false,
		DurationMs:          int(time.Since(started).Microseconds() / 1000),
		RouterPolicyVersion: policy.RouterPolicyVersion,
		ConfidenceGate:      computeGate(first, policy.ConfidenceGate),
		RerankMode:          rerankMode,
		RerankSource:        rerankSource,
		EmbeddingProviderMeta: EmbeddingMeta{
			Primary:        rerankSource,
			ActiveProvider: rerankSource,
			FallbackUsed:   false,
			Dim:            dim,
			DurationMs:     embDuration,
			InputBytes:     len(request),
			InputSha256Pfx: sha256Prefix(request, 8),
		},
	}
	if len(first) > 0 {
		decision.Chosen = first[0].Skill
	}
	decision.RouterDecisionShadow = computeShadow(first, decision.Chosen, policy)
	return decision
}

// RouteWithPolicy lets tests and CLI helpers drive the Router with an explicit
// capability-registry.json snapshot. It honours the same fallback-chain rules
// as the production hot path and records the outcome (primary vs. fallback)
// in EmbeddingProviderMeta.
func RouteWithPolicy(ctx context.Context, entries []registry.Entry, request string, intent Intent, policy *Policy) (Decision, error) {
	started := time.Now()
	if policy == nil {
		return Decision{}, errors.New("router policy unavailable")
	}
	emb, fellBack, err := loadEmbedder(ctx, policy)
	if err != nil {
		return Decision{}, err
	}
	first := ManifestFilter(entries, request, intent)
	bySkill := make(map[string]registry.Entry, len(entries))
	for _, entry := range entries {
		bySkill[entry.Skill] = entry
	}
	rerankMode := "embedding"
	rerankSource := emb.Name()
	dim := policy.Embedding.Dim
	embStarted := time.Now()
	if emb.Name() == "none" {
		rerankMode = "skipped"
	} else {
		for index := range first {
			score, err := RerankerScore(ctx, emb, request, bySkill[first[index].Skill])
			if err == nil {
				first[index].ONNXCosine = score
			}
		}
		sort.Slice(first, func(i, j int) bool {
			if first[i].ONNXCosine == first[j].ONNXCosine {
				if first[i].ManifestScore == first[j].ManifestScore {
					return first[i].Skill < first[j].Skill
				}
				return first[i].ManifestScore > first[j].ManifestScore
			}
			return first[i].ONNXCosine > first[j].ONNXCosine
		})
	}
	for index := range first {
		first[index].Rank = index + 1
	}
	embDuration := time.Since(embStarted).Microseconds() / 1000

	decision := Decision{
		Request:             request,
		Intent:              intent,
		Candidates:          first,
		FallbackUsed:        fellBack,
		DurationMs:          int(time.Since(started).Microseconds() / 1000),
		RouterPolicyVersion: policy.RouterPolicyVersion,
		ConfidenceGate:      computeGate(first, policy.ConfidenceGate),
		RerankMode:          rerankMode,
		RerankSource:        rerankSource,
		EmbeddingProviderMeta: EmbeddingMeta{
			Primary:        policy.Embedding.ProviderName,
			ActiveProvider: rerankSource,
			FallbackUsed:   fellBack,
			Dim:            dim,
			DurationMs:     embDuration,
			InputBytes:     len(request),
			InputSha256Pfx: sha256Prefix(request, 8),
		},
	}
	if len(first) > 0 {
		decision.Chosen = first[0].Skill
	}
	decision.RouterDecisionShadow = computeShadow(first, decision.Chosen, policy)
	return decision, nil
}

// sha256Prefix returns the first n hex chars of SHA256(s) for trace audit.
func sha256Prefix(s string, n int) string {
	sum := sha256.Sum256([]byte(s))
	const hexDigits = "0123456789abcdef"
	if n < 0 {
		n = 0
	}
	if n > len(sum)*2 {
		n = len(sum) * 2
	}
	out := make([]byte, n)
	for i := 0; i < n; i++ {
		b := sum[i/2]
		if i%2 == 0 {
			out[i] = hexDigits[b>>4]
		} else {
			out[i] = hexDigits[b&0x0f]
		}
	}
	return string(out)
}

// mustDefaultEmbedder returns the package-default embedder or panics. The
// failure path is documented in §4.2.4: a missing default means the operator
// forgot to set HC_CAPABILITY_REGISTRY / HC_EMBED_PROVIDER, which is a
// config-time (preflight) concern; reaching here means the runtime was
// started with an invalid config — fail closed.
func mustDefaultEmbedder(ctx context.Context) embedder.Embedder {
	emb, err := embedder.Default(ctx)
	if err != nil {
		panic(fmt.Errorf("default embedder not initialised: %w", err))
	}
	return emb
}

// computeShadow evaluates the candidate policy against the same candidates and
// reports the deltas. It MUST NEVER mutate the main decision (rubric A2.12,
// §6.1). When no candidate policy is configured, returns zero-value with
// WouldHaveChanged=false; the JSON tag uses omitempty so callers can
// distinguish "no shadow in flight" from "shadow ran".
func computeShadow(candidates []Candidate, mainChosen string, p *Policy) DecisionShadow {
	shadowChosen := mainChosen
	if len(candidates) > 0 {
		shadowChosen = candidates[0].Skill
	}
	shadowTop1 := 0
	shadowMargin := 0
	if len(candidates) > 0 {
		shadowTop1 = scaleToFixed(candidates[0].ManifestScore)
		if len(candidates) > 1 {
			m := scaleToFixed(candidates[0].ManifestScore) - scaleToFixed(candidates[1].ManifestScore)
			if m < 0 {
				m = 0
			}
			shadowMargin = m
		}
	}
	return DecisionShadow{
		RouterPolicyCandidate: p.RouterPolicyCandidate,
		Chosen:                shadowChosen,
		ScoreDelta:            shadowTop1 - scaleToFixedOrZero(candidates),
		MarginDelta:           shadowMargin,
		WouldHaveChanged:      shadowChosen != mainChosen,
	}
}

func scaleToFixedOrZero(candidates []Candidate) int {
	if len(candidates) == 0 {
		return 0
	}
	return scaleToFixed(candidates[0].ManifestScore)
}

func computeGate(candidates []Candidate, t GateThresholds) ConfidenceGate {
	gate := ConfidenceGate{
		EntityMatch:  "absent",
		HardFiltered: false,
	}
	if len(candidates) == 0 {
		gate.Decision = "invoke_onnx"
		gate.Rationale = "no_candidates"
		return gate
	}
	top1 := scaleToFixed(candidates[0].ManifestScore)
	gate.Top1Score = top1
	if len(candidates) > 1 {
		m := scaleToFixed(candidates[0].ManifestScore) - scaleToFixed(candidates[1].ManifestScore)
		if m < 0 {
			m = 0
		}
		gate.Margin = m
	}
	switch {
	case candidates[0].ManifestScore >= 0.8:
		gate.EntityMatch = "strong"
	case candidates[0].ManifestScore >= 0.4:
		gate.EntityMatch = "weak"
	default:
		gate.EntityMatch = "absent"
	}
	if top1 < t.Top1ScoreMin || gate.Margin < t.MarginMin || !containsString(t.EntityMatch, gate.EntityMatch) {
		gate.Decision = "invoke_onnx"
	} else {
		gate.Decision = "skip_onnx"
	}
	gate.Rationale = fmt.Sprintf("top1_score=%d(>=%d) margin=%d(>=%d) entity_match=%s",
		top1, t.Top1ScoreMin, gate.Margin, t.MarginMin, gate.EntityMatch)
	return gate
}

func scaleToFixed(score float64) int {
	if score <= 0 {
		return 0
	}
	if score >= 1 {
		return 10000
	}
	return int(score * 10000)
}

func containsString(list []string, target string) bool {
	for _, v := range list {
		if v == target {
			return true
		}
	}
	return false
}

// ManifestFilter is the Stage-1 deterministic filter (no lexical scoring).
func ManifestFilter(entries []registry.Entry, request string, intent Intent) []Candidate {
	requestTokens := tokens(request)
	candidates := make([]Candidate, 0, len(entries))
	for _, entry := range entries {
		if safetyRank(entry.SideEffectClass) > safetyRank(intent.SafetyClass) {
			continue
		}
		candidateTokens := tokens(entry.Name + " " + entry.Description + " " + strings.Join(entry.Inputs, " "))
		matches := 0
		for token := range requestTokens {
			if candidateTokens[token] {
				matches++
			}
		}
		score := 0.0
		if len(requestTokens) > 0 {
			score = float64(matches) / float64(len(requestTokens))
		}
		candidates = append(candidates, Candidate{Skill: entry.Skill, ManifestScore: score})
	}
	sort.Slice(candidates, func(i, j int) bool {
		if candidates[i].ManifestScore == candidates[j].ManifestScore {
			return candidates[i].Skill < candidates[j].Skill
		}
		return candidates[i].ManifestScore > candidates[j].ManifestScore
	})
	if len(candidates) > maxCandidates {
		candidates = candidates[:maxCandidates]
	}
	for index := range candidates {
		candidates[index].Rank = index + 1
	}
	return candidates
}

func tokens(value string) map[string]bool {
	result := make(map[string]bool)
	for _, token := range strings.FieldsFunc(strings.ToLower(value), func(r rune) bool {
		return r < 'a' || r > 'z'
	}) {
		if token != "" {
			result[token] = true
		}
	}
	return result
}

func safetyRank(class string) int {
	switch class {
	case "read-only", "":
		return 0
	case "mutating":
		return 1
	case "destructive":
		return 2
	default:
		return 3
	}
}
