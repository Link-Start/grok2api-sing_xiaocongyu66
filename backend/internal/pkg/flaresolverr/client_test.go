package flaresolverr

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestSolveClearanceParsesCookiesAndUA(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1" || r.Method != http.MethodPost {
			t.Fatalf("unexpected request %s %s", r.Method, r.URL.Path)
		}
		var payload map[string]any
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Fatal(err)
		}
		if payload["cmd"] != "request.get" {
			t.Fatalf("cmd=%v", payload["cmd"])
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"status": "ok",
			"solution": map[string]any{
				"userAgent": "Mozilla/5.0 TestAgent",
				"cookies": []map[string]any{
					{"name": "cf_clearance", "value": "abc", "domain": ".grok.com"},
					{"name": "__cf_bm", "value": "bm", "domain": ".grok.com"},
					{"name": "unrelated", "value": "drop", "domain": ".grok.com"},
				},
			},
		})
	}))
	defer server.Close()

	result, err := SolveClearance(context.Background(), server.URL, "https://grok.com/", "", 5*time.Second)
	if err != nil {
		t.Fatal(err)
	}
	if result.UserAgent != "Mozilla/5.0 TestAgent" {
		t.Fatalf("ua=%q", result.UserAgent)
	}
	if !strings.Contains(result.Cookies, "cf_clearance=abc") || !strings.Contains(result.Cookies, "__cf_bm=bm") {
		t.Fatalf("cookies=%q", result.Cookies)
	}
	if strings.Contains(result.Cookies, "unrelated") {
		t.Fatalf("unrelated cookie not sanitized: %q", result.Cookies)
	}
}

func TestSolveClearanceRejectsNonOK(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"status": "error", "message": "challenge failed"})
	}))
	defer server.Close()
	if _, err := SolveClearance(context.Background(), server.URL, "https://grok.com/", "", time.Second); err == nil {
		t.Fatal("expected error")
	}
}
