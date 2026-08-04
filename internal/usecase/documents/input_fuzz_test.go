package documents

import (
	"encoding/json"
	"strings"
	"testing"
)

// FuzzDocumentInputs проверяет устойчивость create/patch application input к произвольным байтам.
func FuzzDocumentInputs(f *testing.F) {
	for _, seed := range [][]byte{[]byte(`{}`), []byte(`{"identity":"q"}`), []byte(`{"description":null}`), []byte(`null`), []byte(`[]`), {0xff, 0x00}} {
		f.Add(seed)
	}
	f.Fuzz(func(t *testing.T, payload []byte) {
		if len(payload) > 64*1024 {
			t.Skip()
		}
		_, _ = NewCreateInputJSON(payload)
		_, _ = NewPatchInputJSON(payload)
	})
}

// FuzzSecretValidation проверяет рекурсивные JSON-структуры и неизменный запрет credential material.
func FuzzSecretValidation(f *testing.F) {
	for _, seed := range []string{
		`{"credentialRefs":{"client":"vault://client"}}`,
		`{"clientSecret":"value"}`,
		`{"nested":[{"password":"value"}]}`,
		`{"tokenEndpoint":"https://issuer/token"}`,
	} {
		f.Add(seed)
	}
	f.Fuzz(func(t *testing.T, raw string) {
		if len(raw) > 64*1024 {
			t.Skip()
		}
		var value any
		if json.Unmarshal([]byte(raw), &value) != nil {
			return
		}
		err := validateSecrets(value)
		if containsForbiddenSecretKey(value) && err == nil {
			t.Fatal("fuzz input с secret-полем был принят")
		}
	})
}

func containsForbiddenSecretKey(value any) bool {
	switch typed := value.(type) {
	case map[string]any:
		for key, item := range typed {
			normalized := strings.ToLower(strings.NewReplacer("_", "", "-", "").Replace(key))
			if normalized == "credentialrefs" {
				continue
			}
			if normalized == "manualtoken" || strings.Contains(normalized, "password") || strings.Contains(normalized, "clientsecret") || strings.Contains(normalized, "accesstoken") || strings.Contains(normalized, "refreshtoken") || normalized == "bearertoken" || normalized == "secret" {
				return true
			}
			if containsForbiddenSecretKey(item) {
				return true
			}
		}
	case []any:
		for _, item := range typed {
			if containsForbiddenSecretKey(item) {
				return true
			}
		}
	}
	return false
}
