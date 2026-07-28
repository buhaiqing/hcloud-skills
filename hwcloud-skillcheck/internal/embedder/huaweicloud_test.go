package embedder

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
)

func TestHuaweiCloudPreflightAggregatesConfigurationErrors(t *testing.T) {
	cfg := ProviderConfig{ProviderName: "huaweicloud-modelarts", Endpoint: "http://localhost/model", AuthEnv: "MISSING_AUTH", Dim: 1, TimeoutMs: 501}
	report := (&HuaweiCloud{}).Preflight(cfg)
	if report.OK || len(report.Errors) < 4 {
		t.Fatalf("expected multiple errors, got %+v", report)
	}
	for _, issue := range report.Errors {
		if issue.Fix == "" {
			t.Fatalf("error lacks Fix: %+v", issue)
		}
	}
}

func TestHuaweiCloudPreflightRejectsBadURLAndCredential(t *testing.T) {
	t.Setenv("MODELARTS_AUTH", "short")
	report := (&HuaweiCloud{}).Preflight(ProviderConfig{Endpoint: "://bad", AuthEnv: "MODELARTS_AUTH", Dim: 384, TimeoutMs: 500})
	if report.OK || len(report.Errors) == 0 || len(report.Warnings) == 0 {
		t.Fatalf("unexpected report: %+v", report)
	}
}

func TestHuaweiCloudEmbedHTTPSRoundTrip(t *testing.T) {
	transport := roundTripFunc(func(r *http.Request) (*http.Response, error) {
		if r.Header.Get("X-Auth-Token") != "test-token-value" {
			t.Fatalf("missing token header")
		}
		return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`{"outputs":[{"data":[[0.25,0.75]],"shape":[1,2]}]}`)), Header: make(http.Header)}, nil
	})
	t.Setenv("MODELARTS_AUTH", "test-token-value")
	cfg := ProviderConfig{ProviderName: "huaweicloud-modelarts", Endpoint: "https://modelarts.example.test/v1/infers/test", AuthEnv: "MODELARTS_AUTH", Dim: 384, TimeoutMs: 500, Extra: map[string]string{"model_id": "test"}}
	emb, err := PreflightAndInit(context.Background(), cfg)
	if err != nil {
		t.Fatal(err)
	}
	cloud := emb.(*HuaweiCloud)
	cloud.httpClient.Transport = transport
	vector, err := cloud.Embed(context.Background(), "hello")
	if err != nil {
		t.Fatal(err)
	}
	if len(vector) != 2 || vector[0] != 0.25 || vector[1] != 0.75 {
		t.Fatalf("unexpected vector: %v", vector)
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(request *http.Request) (*http.Response, error) {
	return f(request)
}

func TestHuaweiCloudAKSKParsing(t *testing.T) {
	t.Setenv("MODELARTS_AUTH", "access-key|secret-key")
	cfg := ProviderConfig{Endpoint: "https://example.com/model", AuthEnv: "MODELARTS_AUTH", Dim: 384, TimeoutMs: 500, Extra: map[string]string{"model_id": "test"}}
	h := &HuaweiCloud{}
	if err := h.Init(context.Background(), cfg); err != nil {
		t.Fatal(err)
	}
	if h.authKind != authAKSK || h.ak != "access-key" || h.sk != "secret-key" {
		t.Fatalf("credentials parsed incorrectly: kind=%v ak=%q sk=%q", h.authKind, h.ak, h.sk)
	}
	if strings.Contains(h.Preflight(cfg).Error(), "secret-key") {
		t.Fatal("preflight leaked credential")
	}
}
