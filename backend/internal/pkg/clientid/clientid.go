// Package clientid classifies downstream API callers from User-Agent and session headers.
package clientid

import (
	"strings"
)

// Known client type IDs stored on request audits and shown on the dashboard.
const (
	ClaudeCode = "claude_code"
	Codex      = "codex"
	GrokCLI    = "grok_cli"
	Hermes     = "hermes"
	OpenCode   = "opencode"
	Cline      = "cline"
	Cursor     = "cursor"
	Continue   = "continue"
	Aider      = "aider"
	OpenAISDK  = "openai_sdk"
	Anthropic  = "anthropic_sdk"
	PythonHTTP = "python"
	Curl       = "curl"
	Unknown    = "unknown"
)

// Labels maps stable IDs to short dashboard labels (codex:60 style).
var Labels = map[string]string{
	ClaudeCode: "Claude Code",
	Codex:      "Codex",
	GrokCLI:    "Grok CLI",
	Hermes:     "Hermes",
	OpenCode:   "OpenCode",
	Cline:      "Cline",
	Cursor:     "Cursor",
	Continue:   "Continue",
	Aider:      "Aider",
	OpenAISDK:  "OpenAI SDK",
	Anthropic:  "Anthropic SDK",
	PythonHTTP: "Python",
	Curl:       "curl",
	Unknown:    "Unknown",
}

// Detect classifies a caller from User-Agent and optional request headers.
// Headers keys should be lower-case. Empty input yields Unknown.
func Detect(userAgent string, headers map[string]string) string {
	ua := strings.ToLower(strings.TrimSpace(userAgent))
	if headers == nil {
		headers = map[string]string{}
	}
	// Explicit session / product headers first (most reliable).
	if hasHeader(headers, "x-claude-code-session-id") {
		return ClaudeCode
	}
	if hasHeader(headers, "x-codex-window-id") || hasHeader(headers, "x-codex-session-id") {
		return Codex
	}
	if hasHeader(headers, "x-grok-conv-id") || hasHeader(headers, "x-grok-conversation-id") {
		if matchAny(ua, "grok-cli", "grok cli", "xai-grok", "grok-shell") || ua == "" {
			return GrokCLI
		}
	}

	// User-Agent product tokens (order: multi-agent / IDE clients before generic SDKs).
	switch {
	case matchAny(ua, "claude-code", "claude-cli", "claude code"):
		return ClaudeCode
	case matchAny(ua, "openai-codex", "codex-cli", "codex/", "gpt-codex", "openai codex"):
		return Codex
	case matchAny(ua, "hermes-agent", "hermes/", "nous-hermes", "openhermes"):
		return Hermes
	case matchAny(ua, "opencode", "open-code", "sst/opencode"):
		return OpenCode
	case matchAny(ua, "grok-cli", "grok cli", "xai-grok", "grok-shell", "xai-sdk"):
		return GrokCLI
	case matchAny(ua, "cline", "claude-dev"):
		return Cline
	case matchAny(ua, "cursor/", "cursor-"):
		return Cursor
	case matchAny(ua, "continue/", "continue.dev"):
		return Continue
	case matchAny(ua, "aider"):
		return Aider
	case matchAny(ua, "openai-python", "openai-node", "openai-go", "openai-java", "openai/"):
		return OpenAISDK
	case matchAny(ua, "anthropic/", "anthropic-sdk", "anthropic-python", "anthropic-typescript"):
		return Anthropic
	case matchAny(ua, "python-httpx", "python-requests", "aiohttp/", "httpx/"):
		return PythonHTTP
	case matchAny(ua, "curl/"):
		return Curl
	}
	if ua == "" {
		return Unknown
	}
	return Unknown
}

// Label returns a short human label for a client type id.
func Label(id string) string {
	if label, ok := Labels[id]; ok {
		return label
	}
	if id == "" {
		return Labels[Unknown]
	}
	return id
}

func hasHeader(headers map[string]string, name string) bool {
	value := strings.TrimSpace(headers[name])
	if value == "" {
		value = strings.TrimSpace(headers[strings.ToLower(name)])
	}
	return value != ""
}

func matchAny(haystack string, needles ...string) bool {
	for _, needle := range needles {
		if needle != "" && strings.Contains(haystack, needle) {
			return true
		}
	}
	return false
}
