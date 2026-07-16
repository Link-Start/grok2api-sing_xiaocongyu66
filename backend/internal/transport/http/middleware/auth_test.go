package middleware

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	clientkeyapp "github.com/chenyme/grok2api/backend/internal/application/clientkey"
	"github.com/gin-gonic/gin"
)

func TestClientRuntimeStoreFailureUsesServiceUnavailable(t *testing.T) {
	err := errors.Join(clientkeyapp.ErrRuntimeUnavailable, errors.New("redis unavailable"))
	if status := clientErrorStatus(err); status != http.StatusServiceUnavailable {
		t.Fatalf("status = %d", status)
	}
	if code := clientErrorCode(err); code != "runtime_store_unavailable" {
		t.Fatalf("code = %q", code)
	}
	if message := clientErrorMessage(err); message == err.Error() {
		t.Fatal("runtime implementation detail leaked to client")
	}
}

func TestBearerTokenAcceptsCaseInsensitiveSchemeAndWhitespace(t *testing.T) {
	token, ok := bearerToken("  bearer\tsecret-token  ")
	if !ok || token != "secret-token" {
		t.Fatalf("token = %q, ok = %v", token, ok)
	}
	for _, value := range []string{"", "Bearer", "Basic token", "Bearer token extra"} {
		if _, ok := bearerToken(value); ok {
			t.Fatalf("header %q unexpectedly accepted", value)
		}
	}
}

func TestExtractClientAPIKeyCustomHeader(t *testing.T) {
	gin.SetMode(gin.TestMode)
	const key = "g2a_15fc1704968b_vhHjniwKU_x3ROA1JbWo8U3G5YbMGFPT"
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", nil)
	req.Header.Set("congyu_15fc", key)
	c.Request = req

	if got := extractClientAPIKey(c, []string{"congyu_15fc"}); got != key {
		t.Fatalf("custom header key = %q", got)
	}
}

func TestExtractClientAPIKeyPrefersBearerOverCustom(t *testing.T) {
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", nil)
	req.Header.Set("Authorization", "Bearer g2a_from_bearer_secret")
	req.Header.Set("congyu_15fc", "g2a_from_custom_secret")
	c.Request = req

	if got := extractClientAPIKey(c, []string{"congyu_15fc"}); got != "g2a_from_bearer_secret" {
		t.Fatalf("prefer bearer, got %q", got)
	}
}

func TestExtractClientAPIKeyCustomHeaderBearerPrefix(t *testing.T) {
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", nil)
	req.Header.Set("congyu_15fc", "Bearer g2a_custom_bearer_value")
	c.Request = req

	if got := extractClientAPIKey(c, []string{"congyu_15fc"}); got != "g2a_custom_bearer_value" {
		t.Fatalf("custom bearer-style header = %q", got)
	}
}

func TestNormalizeAPIKeyHeadersDropsBuiltinsAndDupes(t *testing.T) {
	got := normalizeAPIKeyHeaders([]string{" congyu_15fc ", "Authorization", "X-API-Key", "congyu_15fc", "My-Key"})
	if len(got) != 2 || got[0] != "congyu_15fc" || got[1] != "My-Key" {
		t.Fatalf("normalized = %#v", got)
	}
}
