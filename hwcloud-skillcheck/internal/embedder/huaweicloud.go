package embedder

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"
)

// HuaweiCloud is the cloud-sandbox Embedder that calls Huawei Cloud
// ModelArts online inference endpoints. Default mode is local-fasttext;
// this provider is selected explicitly via embedding.provider_name or
// HC_EMBED_PROVIDER=huaweicloud-modelarts.
//
// Authentication: Huawei Cloud's standard HWS Signature algorithm. The
// AuthEnv env var holds either:
//
//	(a) a single token (AK=token),           — for IAM-token-only deployments
//	(b) AK|SK strings (AK=ak; SK=sk),        — for AK/SK signing
//
// The env var VALUE is read once at Init and held in memory; never written
// to disk, capability-registry.json, or trace files.
//
// Network behavior: every network call has a timeout derived from
// cfg.TimeoutMs (default 500ms). A failed call does NOT auto-retry from
// this provider; the caller's NewWithFallback handles degradation by
// switching to the configured fallback_chain entry.
//
// Trace hygiene: input text is sent over HTTPS only; the response is parsed
// but the returned vectors are NEVER written to trace logs.
type HuaweiCloud struct {
	endpoint   string
	authKind   authKind
	authToken  string
	ak         string
	sk         string
	projectID  string
	modelID    string
	dim        int
	timeout    time.Duration
	httpClient *http.Client

	closeOnce sync.Once
}

type authKind int

const (
	authToken authKind = iota
	authAKSK
)

func newHuaweiCloud(cfg ProviderConfig) (*HuaweiCloud, error) {
	if cfg.Dim == 0 {
		cfg.Dim = DefaultDim
	}
	if cfg.Dim < MinDim || cfg.Dim > MaxDim {
		return nil, fmt.Errorf("huaweicloud-modelarts dim=%d is outside [%d, %d]. Fix: set embedding.dim to a value in [%d, %d]", cfg.Dim, MinDim, MaxDim, MinDim, MaxDim)
	}
	return &HuaweiCloud{
		endpoint:  cfg.Endpoint,
		projectID: cfg.ProjectID,
		modelID:   cfg.Extra["model_id"],
		dim:       cfg.Dim,
		timeout:   time.Duration(cfg.TimeoutMs) * time.Millisecond,
	}, nil
}

// Name returns the canonical provider identifier.
func (h *HuaweiCloud) Name() string { return "huaweicloud-modelarts" }

// Preflight validates config without network. AuthEnv is the most common
// failure source for cloud deployments; we read it explicitly and report
// every concrete problem with a Fix.
func (h *HuaweiCloud) Preflight(cfg ProviderConfig) PreflightReport {
	r := PreflightReport{Provider: "huaweicloud-modelarts"}
	if cfg.Endpoint == "" {
		r.Errors = append(r.Errors, Issue{
			Field:   "endpoint",
			Message: "endpoint is required for huaweicloud-modelarts",
			Fix:     "Set embedding.endpoint to your ModelArts inference URL, e.g. https://modelarts.cn-north-4.myhuaweicloud.com/v1/infers/<id>.",
			DocURL:  "docs/deployment-guide.md#1.6-cloud-sandbox-setup",
		})
	} else {
		u, err := url.Parse(cfg.Endpoint)
		if err != nil {
			r.Errors = append(r.Errors, Issue{
				Field:   "endpoint",
				Message: fmt.Sprintf("endpoint is not a valid URL: %v", err),
				Fix:     "Fix the URL syntax; the API expects https://modelarts.<region>.myhuaweicloud.com/v1/infers/<inference_id>.",
			})
		} else if u.Scheme != "https" {
			r.Errors = append(r.Errors, Issue{
				Field:   "endpoint",
				Message: fmt.Sprintf("endpoint scheme=%q; HTTPS is required for cloud sandbox", u.Scheme),
				Fix:     "Change the URL to start with https://.",
			})
		}
		if u != nil && (strings.Contains(u.Host, "127.0.0.1") || strings.Contains(u.Host, "localhost")) {
			r.Warnings = append(r.Warnings, Issue{
				Field:   "endpoint",
				Message: "endpoint points to a loopback address; live ModelArts is reachable only from inside your VPC",
				Fix:     "If this is intended (private cloud), proceed; otherwise point to the public endpoint.",
			})
		}
	}
	if cfg.AuthEnv == "" {
		r.Errors = append(r.Errors, Issue{
			Field:   "auth_env",
			Message: "auth_env is required for huaweicloud-modelarts",
			Fix:     "Set embedding.auth_env to the NAME of an env var that holds the credentials. NEVER put the credentials themselves in capability-registry.json.",
			DocURL:  "docs/deployment-guide.md#1.6-cloud-sandbox-setup",
		})
	} else {
		val := os.Getenv(cfg.AuthEnv)
		if val == "" {
			r.Errors = append(r.Errors, Issue{
				Field:   "auth_env",
				Message: fmt.Sprintf("env var %s is unset or empty", cfg.AuthEnv),
				Fix:     fmt.Sprintf("Export the credential before launching the binary, e.g. `export %s=...`. The router will refuse to start if this is empty at preflight.", cfg.AuthEnv),
			})
		} else if strings.ContainsAny(val, "\n\t\r") {
			r.Errors = append(r.Errors, Issue{
				Field:   "auth_env",
				Message: fmt.Sprintf("env var %s contains whitespace control characters", cfg.AuthEnv),
				Fix:     "Re-export the credential without newlines or tabs (paste artefacts are a common cause).",
			})
		} else if len(val) < 8 {
			r.Warnings = append(r.Warnings, Issue{
				Field:   "auth_env",
				Message: fmt.Sprintf("env var %s value is suspiciously short (%d chars)", cfg.AuthEnv, len(val)),
				Fix:     "Verify you are not exporting a placeholder like 'changeme' or a partial key.",
			})
		}
	}
	if cfg.Dim != 0 && (cfg.Dim < MinDim || cfg.Dim > MaxDim) {
		r.Errors = append(r.Errors, Issue{
			Field:   "dim",
			Message: fmt.Sprintf("dim=%d is outside allowed range [%d, %d]", cfg.Dim, MinDim, MaxDim),
			Fix:     fmt.Sprintf("Set embedding.dim to a value in [%d, %d].", MinDim, MaxDim),
		})
	}
	if cfg.ProjectID == "" {
		r.Info = append(r.Info, "project_id not set; the cloud SDK will read it from env (HUAWEICLOUD_PROJECT_ID) or IAM token claims")
	}
	if cfg.Extra["model_id"] == "" {
		r.Warnings = append(r.Warnings, Issue{
			Field:   "extra.model_id",
			Message: "model_id is not set; the inference endpoint path must contain the model id if not supplied",
			Fix:     "Set embedding.extra.model_id to your deployed model identifier, or rely on the inference-id being part of the endpoint URL.",
		})
	}
	if cfg.TimeoutMs == 0 {
		r.Warnings = append(r.Warnings, Issue{
			Field:   "timeout_ms",
			Message: "timeout_ms is 0 (no per-call cap on top of ctx); Stage-2 SLA is 50ms per candidate",
			Fix:     "Set embedding.timeout_ms to a value <= 500 to keep within Stage-2 latency budgets.",
		})
	} else if cfg.TimeoutMs > 500 {
		r.Errors = append(r.Errors, Issue{
			Field:   "timeout_ms",
			Message: fmt.Sprintf("timeout_ms=%d exceeds Stage-2 SLA (500ms per call)", cfg.TimeoutMs),
			Fix:     "Reduce embedding.timeout_ms to <= 500, or rely on the per-run budget caps (rubric A2.5–A2.7) to fail closed.",
		})
	}
	r.OK = len(r.Errors) == 0
	return r
}

// Init allocates the http.Client and reads credentials from env. Idempotent.
func (h *HuaweiCloud) Init(ctx context.Context, cfg ProviderConfig) error {
	if h.httpClient != nil {
		return nil
	}
	if report := h.Preflight(cfg); !report.OK {
		return report
	}
	if cfg.Dim == 0 {
		cfg.Dim = DefaultDim
	}
	h.endpoint = cfg.Endpoint
	h.projectID = cfg.ProjectID
	h.modelID = cfg.Extra["model_id"]
	h.dim = cfg.Dim
	if cfg.TimeoutMs == 0 {
		h.timeout = 500 * time.Millisecond
	} else {
		h.timeout = time.Duration(cfg.TimeoutMs) * time.Millisecond
	}
	h.httpClient = &http.Client{Timeout: h.timeout}

	creds := os.Getenv(cfg.AuthEnv)
	if strings.Contains(creds, "|") {
		parts := strings.SplitN(creds, "|", 2)
		h.ak = strings.TrimSpace(parts[0])
		h.sk = strings.TrimSpace(parts[1])
		h.authKind = authAKSK
	} else {
		h.authToken = creds
		h.authKind = authToken
	}
	return nil
}

// Close releases the http.Client. Idempotent.
func (h *HuaweiCloud) Close() error {
	h.closeOnce.Do(func() { h.httpClient = nil })
	return nil
}

// Health performs a lightweight HEAD on the endpoint. Returns an error if
// the endpoint is unreachable or authentication fails.
func (h *HuaweiCloud) Health(ctx context.Context) error {
	if h.httpClient == nil {
		return fmt.Errorf("huaweicloud-modelarts not initialized")
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, h.endpoint, nil)
	if err != nil {
		return err
	}
	h.signRequest(req, "", "GET")
	resp, err := h.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("health check failed: %w. Fix: verify endpoint reachability and credentials", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 500 {
		return fmt.Errorf("health check: upstream %d. Fix: retry, or check the cloud service status", resp.StatusCode)
	}
	return nil
}

// Embed calls the inference endpoint and parses the returned vectors.
func (h *HuaweiCloud) Embed(ctx context.Context, text string) ([]float32, error) {
	if h.httpClient == nil {
		return nil, fmt.Errorf("huaweicloud-modelarts not initialized")
	}
	if len(text) > MaxInputBytes {
		return nil, fmt.Errorf("input length %d exceeds security cap %d. Fix: shorten the query", len(text), MaxInputBytes)
	}
	body := map[string]any{
		"inputs": []map[string]any{{
			"data":   []string{text},
			"params": map[string]any{},
		}},
	}
	bodyBytes, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, h.endpoint, bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Project-Id", h.projectID)
	h.signRequest(req, string(bodyBytes), "POST")

	resp, err := h.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("huaweicloud-modelarts call failed: %w. Fix: check network egress, credentials, or switch to fallback chain", err)
	}
	defer resp.Body.Close()
	respBytes, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("huaweicloud-modelarts returned %d: %s. Fix: see the model deployment status; common codes are 401 (bad credential), 404 (wrong endpoint), 429 (rate limited)", resp.StatusCode, truncate(string(respBytes), 200))
	}
	var parsed struct {
		Outputs []struct {
			Data  [][]float64 `json:"data"`
			Shape []int       `json:"shape"`
		} `json:"outputs"`
	}
	if err := json.Unmarshal(respBytes, &parsed); err != nil {
		return nil, fmt.Errorf("parse response: %w", err)
	}
	if len(parsed.Outputs) == 0 || len(parsed.Outputs[0].Data) == 0 {
		return nil, fmt.Errorf("response carried no vectors; check that the inference model returns embedding outputs")
	}
	raw := parsed.Outputs[0].Data[0]
	out := make([]float32, len(raw))
	for i, x := range raw {
		out[i] = float32(x)
	}
	return out, nil
}

// signRequest attaches HWS V1 signature headers when AK/SK is configured.
// For token-based auth, it sets the X-Auth-Token header.
func (h *HuaweiCloud) signRequest(req *http.Request, body, method string) {
	if h.authKind == authToken {
		req.Header.Set("X-Auth-Token", h.authToken)
		return
	}
	// HWS V1 signature (see huaweicloud_hws.go for the canonical encoder).
	contentSHA := sha256Hex(body)
	signed := "host;x-auth-ak;x-auth-content-sha256;x-project-id"
	req.Header.Set("Host", req.URL.Host)
	req.Header.Set("X-Signed-Headers", signed)
	canonical := hwsV1CanonicalString(req, contentSHA, signed)
	stringToSign := "HWS-HMAC-SHA256\n" + canonical
	mac := hmac.New(sha256.New, []byte(h.sk))
	mac.Write([]byte(stringToSign))
	sig := hex.EncodeToString(mac.Sum(nil))
	req.Header.Set("X-Auth-AK", h.ak)
	req.Header.Set("X-Auth-Signature", sig)
	req.Header.Set("X-Auth-Content-SHA256", contentSHA)
}

func sha256Hex(s string) string {
	h := sha256.Sum256([]byte(s))
	return hex.EncodeToString(h[:])
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "...(truncated)"
}

// Score is implemented via Embed + cosine.
func (h *HuaweiCloud) Score(ctx context.Context, query, doc string) (float64, error) {
	qv, err := h.Embed(ctx, query)
	if err != nil {
		return 0, err
	}
	dv, err := h.Embed(ctx, doc)
	if err != nil {
		return 0, err
	}
	return cosine(qv, dv), nil
}
