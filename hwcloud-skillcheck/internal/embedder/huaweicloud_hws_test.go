package embedder

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"strings"
	"testing"
)

func TestHWSV1CanonicalOrdering(t *testing.T) {
	req, err := http.NewRequest(http.MethodPost, "https://modelarts.cn-north-4.myhuaweicloud.com/v1/infers/abc?foo=bar&baz=qux", strings.NewReader("{}"))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Host", req.URL.Host)
	req.Header.Set("X-Auth-AK", "AK-EXAMPLE")
	req.Header.Set("X-Auth-Content-SHA256", "44136fa355b3678a1146ad16f7e8649e94fb4fc21fe77e8310c060f61caaff8a")
	req.Header.Set("X-Project-Id", "project-1")
	signed := "host;x-auth-ak;x-auth-content-sha256;x-project-id"
	canonical := hwsV1CanonicalString(req, "44136fa355b3678a1146ad16f7e8649e94fb4fc21fe77e8310c060f61caaff8a", signed)
	want := "POST\n/v1/infers/abc\nbaz=qux&foo=bar\nmodelarts.cn-north-4.myhuaweicloud.com\nAK-EXAMPLE\n44136fa355b3678a1146ad16f7e8649e94fb4fc21fe77e8310c060f61caaff8a\nproject-1\nhost;x-auth-ak;x-auth-content-sha256;x-project-id\n44136fa355b3678a1146ad16f7e8649e94fb4fc21fe77e8310c060f61caaff8a"
	if canonical != want {
		t.Fatalf("canonical mismatch:\n got=%q\nwant=%q", canonical, want)
	}
}

func TestHWSV1SignatureStable(t *testing.T) {
	req, err := http.NewRequest(http.MethodPost, "https://modelarts.cn-north-4.myhuaweicloud.com/v1/infers/abc", strings.NewReader("{}"))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Host", req.URL.Host)
	req.Header.Set("X-Auth-AK", "ak")
	req.Header.Set("X-Auth-Content-SHA256", "44136fa355b3678a1146ad16f7e8649e94fb4fc21fe77e8310c060f61caaff8a")
	req.Header.Set("X-Project-Id", "pid")
	signed := "host;x-auth-ak;x-auth-content-sha256;x-project-id"
	canonical := hwsV1CanonicalString(req, "44136fa355b3678a1146ad16f7e8649e94fb4fc21fe77e8310c060f61caaff8a", signed)
	stringToSign := "HWS-HMAC-SHA256\n" + canonical
	mac := hmac.New(sha256.New, []byte("sk"))
	mac.Write([]byte(stringToSign))
	sig := hex.EncodeToString(mac.Sum(nil))
	if len(sig) != 64 {
		t.Fatalf("unexpected signature length: %d", len(sig))
	}
}
