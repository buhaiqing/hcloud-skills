package embedder

import (
	"net/http"
	"sort"
	"strings"
)

// hwsV1CanonicalString assembles the canonical string the Huawei Cloud HWS V1
// algorithm signs. The line layout follows the public ModelArts example:
//
//	HTTPMethod\n
//	CanonicalURI\n
//	CanonicalQueryString\n
//	CanonicalHeaders\n
//	SignedHeaders\n
//	HexEncode(Hash(RequestPayload))
//
// where CanonicalHeaders is the joined values of the headers listed in
// SignedHeaders, one per line in the order of SignedHeaders.
func hwsV1CanonicalString(req *http.Request, contentSHA, signedHeaders string) string {
	method := strings.ToUpper(req.Method)
	uri := req.URL.Path
	if uri == "" {
		uri = "/"
	}
	query := canonicalQueryString(req.URL.Query())
	names := splitSigned(signedHeaders)
	var headerValues []string
	for _, name := range names {
		headerValues = append(headerValues, canonicalHeaderValue(req.Header.Get(name)))
	}
	headers := strings.Join(headerValues, "\n")
	return strings.Join([]string{method, uri, query, headers, signedHeaders, contentSHA}, "\n")
}

func canonicalQueryString(params map[string][]string) string {
	if len(params) == 0 {
		return ""
	}
	keys := make([]string, 0, len(params))
	for k := range params {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	var b strings.Builder
	for _, key := range keys {
		for _, value := range params[key] {
			if b.Len() > 0 {
				b.WriteByte('&')
			}
			b.WriteString(uriEncode(key, false))
			b.WriteByte('=')
			b.WriteString(uriEncode(value, false))
		}
	}
	return b.String()
}

func splitSigned(signed string) []string {
	if signed == "" {
		return nil
	}
	seen := make(map[string]bool)
	var out []string
	for _, raw := range strings.Split(signed, ";") {
		name := strings.ToLower(strings.TrimSpace(raw))
		if name == "" || seen[name] {
			continue
		}
		seen[name] = true
		out = append(out, name)
	}
	sort.Strings(out)
	return out
}

func canonicalHeaderValue(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	var b strings.Builder
	inSpace := false
	for _, r := range value {
		if r == ' ' || r == '\t' {
			if !inSpace {
				b.WriteByte(' ')
				inSpace = true
			}
			continue
		}
		b.WriteRune(r)
		inSpace = false
	}
	return b.String()
}

func uriEncode(s string, strict bool) string {
	var b strings.Builder
	for _, r := range s {
		switch {
		case r >= '0' && r <= '9', r >= 'A' && r <= 'Z', r >= 'a' && r <= 'z',
			r == '-', r == '_', r == '.', r == '~':
			b.WriteRune(r)
		case r == '/' && !strict:
			b.WriteRune(r)
		default:
			const hex = "0123456789ABCDEF"
			if r < 0x80 {
				b.WriteByte('%')
				b.WriteByte(hex[(r>>4)&0xF])
				b.WriteByte(hex[r&0xF])
			} else {
				for _, b2 := range []byte(string(r)) {
					b.WriteByte('%')
					b.WriteByte(hex[(b2>>4)&0xF])
					b.WriteByte(hex[b2&0xF])
				}
			}
		}
	}
	return b.String()
}
