package flaresolverr

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// FlareSolverrResult is a successful Cloudflare challenge solution.
type FlareSolverrResult struct {
	Cookies   string
	UserAgent string
	Host      string
}

type flareSolverrRequest struct {
	Cmd        string             `json:"cmd"`
	URL        string             `json:"url"`
	MaxTimeout int                `json:"maxTimeout"`
	Proxy      *flareSolverrProxy `json:"proxy,omitempty"`
}

type flareSolverrProxy struct {
	URL string `json:"url"`
}

type flareSolverrResponse struct {
	Status   string `json:"status"`
	Message  string `json:"message"`
	Solution *struct {
		URL       string `json:"url"`
		UserAgent string `json:"userAgent"`
		Cookies   []struct {
			Name   string `json:"name"`
			Value  string `json:"value"`
			Domain string `json:"domain"`
		} `json:"cookies"`
	} `json:"solution"`
}

// SolveClearance asks FlareSolverr to open targetURL (optionally via proxyURL)
// and returns sanitized Cloudflare cookies + the browser User-Agent used.
func SolveClearance(ctx context.Context, flaresolverrURL, targetURL, proxyURL string, timeout time.Duration) (FlareSolverrResult, error) {
	base := strings.TrimRight(strings.TrimSpace(flaresolverrURL), "/")
	if base == "" {
		return FlareSolverrResult{}, fmt.Errorf("flaresolverr url 为空")
	}
	target := strings.TrimSpace(targetURL)
	if target == "" {
		target = "https://grok.com/"
	}
	if timeout <= 0 {
		timeout = 60 * time.Second
	}
	payload := flareSolverrRequest{
		Cmd:        "request.get",
		URL:        target,
		MaxTimeout: int(timeout / time.Millisecond),
	}
	if strings.TrimSpace(proxyURL) != "" {
		payload.Proxy = &flareSolverrProxy{URL: strings.TrimSpace(proxyURL)}
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return FlareSolverrResult{}, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, base+"/v1", bytes.NewReader(body))
	if err != nil {
		return FlareSolverrResult{}, err
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: timeout + 30*time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return FlareSolverrResult{}, fmt.Errorf("连接 FlareSolverr: %w", err)
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	if err != nil {
		return FlareSolverrResult{}, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return FlareSolverrResult{}, fmt.Errorf("FlareSolverr HTTP %d: %s", resp.StatusCode, truncateForLog(string(raw), 200))
	}
	var parsed flareSolverrResponse
	if err := json.Unmarshal(raw, &parsed); err != nil {
		return FlareSolverrResult{}, fmt.Errorf("解析 FlareSolverr 响应: %w", err)
	}
	if !strings.EqualFold(parsed.Status, "ok") || parsed.Solution == nil {
		msg := strings.TrimSpace(parsed.Message)
		if msg == "" {
			msg = parsed.Status
		}
		return FlareSolverrResult{}, fmt.Errorf("FlareSolverr 未成功: %s", msg)
	}
	host := ""
	if u, err := url.Parse(target); err == nil {
		host = strings.ToLower(u.Hostname())
	}
	parts := make([]string, 0, len(parsed.Solution.Cookies))
	for _, cookie := range parsed.Solution.Cookies {
		name := strings.TrimSpace(cookie.Name)
		if name == "" {
			continue
		}
		if host != "" && cookie.Domain != "" {
			domain := strings.ToLower(strings.TrimPrefix(strings.TrimSpace(cookie.Domain), "."))
			if domain != "" && !strings.HasSuffix(host, domain) {
				continue
			}
		}
		parts = append(parts, name+"="+cookie.Value)
	}
	if len(parts) == 0 {
		// Fall back to all cookies if domain filter removed everything.
		for _, cookie := range parsed.Solution.Cookies {
			if strings.TrimSpace(cookie.Name) == "" {
				continue
			}
			parts = append(parts, cookie.Name+"="+cookie.Value)
		}
	}
	cookies := sanitizeCloudflareCookies(strings.Join(parts, "; "))
	if cookies == "" {
		return FlareSolverrResult{}, fmt.Errorf("FlareSolverr 未返回有效 Cloudflare Cookie")
	}
	return FlareSolverrResult{
		Cookies:   cookies,
		UserAgent: strings.TrimSpace(parsed.Solution.UserAgent),
		Host:      host,
	}, nil
}

func sanitizeCloudflareCookies(value string) string {
	parts := strings.Split(value, ";")
	allowed := make([]string, 0, len(parts))
	seen := make(map[string]struct{}, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		name, cookieValue, ok := strings.Cut(part, "=")
		if !ok {
			continue
		}
		lower := strings.ToLower(strings.TrimSpace(name))
		if lower != "cf_clearance" && lower != "__cf_bm" && lower != "_cfuvid" && !strings.HasPrefix(lower, "cf_chl_") {
			continue
		}
		if _, exists := seen[lower]; exists {
			continue
		}
		seen[lower] = struct{}{}
		allowed = append(allowed, lower+"="+cookieValue)
	}
	return strings.Join(allowed, "; ")
}

func truncateForLog(value string, max int) string {
	value = strings.TrimSpace(value)
	if max <= 0 || len(value) <= max {
		return value
	}
	return value[:max] + "…"
}
