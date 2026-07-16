package account

import (
	"errors"
	"fmt"
	"strings"

	accountdomain "github.com/chenyme/grok2api/backend/internal/domain/account"
	"github.com/chenyme/grok2api/backend/internal/infra/provider"
)

func isRateLimitOrTransient(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, provider.ErrUnauthorized) {
		return false
	}
	msg := strings.ToLower(err.Error())
	for _, needle := range []string{"429", "too many", "rate limit", "ratelimit", "retry later", "timeout", "temporar", "connection reset", "eof", "context deadline"} {
		if strings.Contains(msg, needle) {
			return true
		}
	}
	var refreshErr *provider.CredentialRefreshError
	if errors.As(err, &refreshErr) {
		if refreshErr.Status == 429 {
			return true
		}
		if !refreshErr.Permanent {
			return true
		}
	}
	return false
}

func uniqueSSOBySourceKey(items []accountdomain.Credential) []accountdomain.Credential {
	seen := make(map[string]accountdomain.Credential, len(items))
	order := make([]string, 0, len(items))
	for _, item := range items {
		key := strings.TrimSpace(item.SourceKey)
		if key == "" {
			key = fmt.Sprintf("id:%d", item.ID)
		}
		if prev, ok := seen[key]; ok {
			if betterSSODuplicate(item, prev) {
				seen[key] = item
			}
			continue
		}
		seen[key] = item
		order = append(order, key)
	}
	out := make([]accountdomain.Credential, 0, len(order))
	for _, key := range order {
		out = append(out, seen[key])
	}
	return out
}

func betterSSODuplicate(candidate, existing accountdomain.Credential) bool {
	score := func(v accountdomain.Credential) int {
		n := 0
		if v.Enabled {
			n += 2
		}
		if v.AuthStatus == accountdomain.AuthStatusActive {
			n += 2
		}
		return n
	}
	return score(candidate) > score(existing)
}
