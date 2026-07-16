// Package clientid classifies downstream API callers from User-Agent and session headers.
package clientid

import (
	"strings"
)

// Known client type IDs stored on request audits and shown on the dashboard.
// Product IDs follow common CLI/agent names; runtime IDs (go/python/…) match
// agent-traffic-classifier "programmatic" class (library UA, not a product).
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
	RooCode    = "roo_code"
	Windsurf   = "windsurf"
	ZCode      = "zcode"
	GeminiCLI  = "gemini_cli" // Google-Gemini-CLI (agent-traffic-classifier)
	Kiro       = "kiro"       // Amazon Kiro-CLI
	MCP        = "mcp"        // ModelContextProtocol clients
	Copilot    = "copilot"    // VS Code / GitHub Copilot agent UA contains "Code/"
	OpenAISDK  = "openai_sdk"
	Anthropic  = "anthropic_sdk"
	NodeHTTP   = "node"
	PythonHTTP = "python"
	GoHTTP     = "go"
	JavaHTTP   = "java"
	RustHTTP   = "rust"
	RubyHTTP   = "ruby"
	PerlHTTP   = "perl"
	Curl       = "curl"
	Wget       = "wget"
	// Legacy is empty client_type on audits written before client detection existed.
	Legacy  = "legacy"
	Unknown = "unknown"
)

// Labels maps stable IDs to short dashboard labels (codex:60 style).
// "Go 客户端" = programmatic Go net/http default UA, not a mystery product.
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
	RooCode:    "Roo Code",
	Windsurf:   "Windsurf",
	ZCode:      "ZCode",
	GeminiCLI:  "Gemini CLI",
	Kiro:       "Kiro",
	MCP:        "MCP Client",
	Copilot:    "GitHub Copilot",
	OpenAISDK:  "OpenAI SDK",
	Anthropic:  "Anthropic SDK",
	NodeHTTP:   "Node / undici",
	PythonHTTP: "Python HTTP",
	GoHTTP:     "Go 客户端",
	JavaHTTP:   "Java / OkHttp",
	RustHTTP:   "Rust HTTP",
	RubyHTTP:   "Ruby HTTP",
	PerlHTTP:   "Perl",
	Curl:       "curl",
	Wget:       "Wget",
	Legacy:     "历史请求",
	Unknown:    "未知客户端",
}

// Detect classifies a caller from User-Agent and optional request headers.
// Headers keys should be lower-case. Empty input yields Unknown (not Legacy —
// Legacy is only for persisted empty client_type rows).
func Detect(userAgent string, headers map[string]string) string {
	ua := strings.ToLower(strings.TrimSpace(userAgent))
	if headers == nil {
		headers = map[string]string{}
	}
	originator := strings.ToLower(strings.TrimSpace(headerValue(headers, "originator")))
	xApp := strings.ToLower(strings.TrimSpace(headerValue(headers, "x-app")))
	xClientName := strings.ToLower(strings.TrimSpace(headerValue(headers, "x-client-name")))
	xClientTitle := strings.ToLower(strings.TrimSpace(headerValue(headers, "x-client-title")))
	xTitle := strings.ToLower(strings.TrimSpace(headerValue(headers, "x-title")))
	stainlessPkg := strings.ToLower(strings.TrimSpace(headerValue(headers, "x-stainless-package-version")))
	_ = stainlessPkg

	// Explicit product / session headers (most reliable).
	if hasHeader(headers, "x-claude-code-session-id") ||
		matchAny(xApp, "claude-code", "claude_code") ||
		matchAny(headerValue(headers, "anthropic-beta"), "claude-code") {
		return ClaudeCode
	}
	if hasHeader(headers, "x-codex-window-id") || hasHeader(headers, "x-codex-session-id") ||
		matchAny(originator, "codex") ||
		matchAny(xApp, "codex") {
		return Codex
	}
	if hasHeader(headers, "x-grok-conv-id") || hasHeader(headers, "x-grok-conversation-id") {
		if matchAny(ua, "grok-cli", "grok cli", "xai-grok", "grok-shell", "xai-sdk", "grok-pager", "grok-build") || ua == "" || matchAny(originator, "grok") {
			return GrokCLI
		}
	}

	// Explicit client identity headers (downstream apps should set one of these).
	if id := detectFromProductToken(xClientName); id != Unknown {
		return id
	}
	if id := detectFromProductToken(xClientTitle); id != Unknown {
		return id
	}
	if id := detectFromProductToken(xTitle); id != Unknown {
		return id
	}
	if id := detectFromProductToken(xApp); id != Unknown {
		return id
	}

	// Product / agent UA tokens first (patterns aligned with agent-traffic-classifier
	// defaults/agents.ts + bots.json "agent" category, plus our gateway-specific list).
	switch {
	// Anthropic coding agents (ATC: Claude-User / Claude-Agent)
	case matchAny(ua, "claude-user", "claude-agent", "claude-code", "claude-cli", "claude_code", "claude code", "@anthropic-ai/claude-code"):
		return ClaudeCode
	case matchAny(ua, "codex_cli_rs", "codex-cli", "openai-codex", "gpt-codex", "openai codex", "codex/") ||
		matchAny(originator, "codex_cli_rs", "codex-cli"):
		return Codex
	case matchAny(ua, "google-gemini-cli", "gemini-cli", "gemini cli"):
		return GeminiCLI
	case matchAny(ua, "kiro-cli", "kiro/"):
		return Kiro
	case matchAny(ua, "modelcontextprotocol", "mcp-client", "mcp/"):
		return MCP
	// VS Code Copilot agent traffic often includes "Code/" (ATC pattern); avoid bare "code".
	case matchAny(ua, "github copilot", "copilot/") || (strings.Contains(ua, "code/") && matchAny(ua, "vscode", "visual studio", "copilot")):
		return Copilot
	case matchAny(ua, "hermes-agent", "hermes-cli", "hermes/", "nous-hermes", "openhermes", "hermes agent"):
		return Hermes
	case matchAny(ua, "opencode", "open-code", "sst/opencode", "anomalyco/opencode"):
		return OpenCode
	case matchAny(ua, "grok-cli", "grok cli", "xai-grok", "grok-shell", "xai-sdk", "xai/", "grok-pager", "grok-build"):
		return GrokCLI
	case matchAny(ua, "cline", "claude-dev", "roo-cline"):
		return Cline
	case matchAny(ua, "roo-code", "roocode", "roo code"):
		return RooCode
	case matchAny(ua, "cursor/", "cursor-", "cursor "):
		return Cursor
	case matchAny(ua, "continue/", "continue.dev", "continuedev"):
		return Continue
	case matchAny(ua, "windsurf", "codeium"):
		return Windsurf
	case matchAny(ua, "zcode", "z-code", "z_code"):
		return ZCode
	case matchAny(ua, "aider"):
		return Aider
	case matchAny(ua, "openai-python", "openai-node", "openai-go", "openai-java", "openai-php", "openai/") ||
		(matchAny(headerValue(headers, "x-stainless-lang"), "python", "js", "node", "go", "java") && hasHeader(headers, "x-stainless-package-version")):
		// Stainless-generated OpenAI SDKs set x-stainless-* headers.
		return OpenAISDK
	case matchAny(ua, "anthropic/", "anthropic-sdk", "anthropic-python", "anthropic-typescript", "@anthropic-ai/sdk"):
		return Anthropic
	// Path-ish hints from anthropic-version alone: Claude Code and many agents hit Messages API.
	// Prefer Claude Code when UA is empty/generic node and anthropic-version is present —
	// pure Anthropic SDK usually sets anthropic/ or @anthropic-ai/sdk in UA.
	case hasHeader(headers, "anthropic-version") && (ua == "" || matchAny(ua, "node", "undici", "axios", "fetch")):
		return ClaudeCode
	// Programmatic HTTP libraries (agent-traffic-classifier DEFAULT_PROGRAMMATIC).
	// These are language runtimes, not product names — UI labels explain that.
	case matchAny(ua, "python-httpx", "python-requests", "aiohttp/", "httpx/", "python-urllib", "urllib", "trafilatura"):
		return PythonHTTP
	case matchAny(ua, "node-fetch", "undici", "axios/", "got/", "got (", "node.js", "nodejs", "bun/"):
		return NodeHTTP
	// Go default transport — this is the "Go 客户端" you see in audits (not mysterious).
	case matchAny(ua, "go-http-client", "go-resty", "fasthttp", "go-http/", "golang/", "net/http", "colly"):
		return GoHTTP
	case matchAny(ua, "okhttp", "apache-httpclient", "java/"):
		return JavaHTTP
	case matchAny(ua, "reqwest", "rustls", "ureq/"):
		return RustHTTP
	case matchAny(ua, "http.rb", "ruby", "faraday"):
		return RubyHTTP
	case matchAny(ua, "libwww-perl", "lwp::", "www-mechanize"):
		return PerlHTTP
	case matchAny(ua, "curl/", "curl "):
		return Curl
	case matchAny(ua, "wget/", "wget "):
		return Wget
	// Federated social stacks that call OpenAI-compatible gateways.
	case matchAny(ua, "misskey/", "sharkey/", "megalodon/", "firefish/", "iceshrimp/"):
		return OpenAISDK
	}
	if ua == "" {
		// Empty UA: still try stainless / anthropic-version already handled above.
		// Common silent Go callers leave UA blank depending on library config.
		if matchAny(headerValue(headers, "x-stainless-lang"), "go") {
			return OpenAISDK
		}
		return Unknown
	}
	return Unknown
}

// detectFromProductToken maps free-form product names (headers) to known IDs.
func detectFromProductToken(token string) string {
	token = strings.ToLower(strings.TrimSpace(token))
	if token == "" {
		return Unknown
	}
	switch {
	case matchAny(token, "claude-code", "claude_code", "claude code", "claude-cli", "claude-user", "claude-agent"):
		return ClaudeCode
	case matchAny(token, "codex"):
		return Codex
	case matchAny(token, "gemini-cli", "gemini_cli", "google-gemini"):
		return GeminiCLI
	case matchAny(token, "kiro"):
		return Kiro
	case matchAny(token, "mcp", "modelcontextprotocol"):
		return MCP
	case matchAny(token, "copilot"):
		return Copilot
	case matchAny(token, "hermes"):
		return Hermes
	case matchAny(token, "opencode", "open-code"):
		return OpenCode
	case matchAny(token, "grok-cli", "grok_cli", "grok cli", "grok-pager", "grok-shell", "grok-build", "xai"):
		return GrokCLI
	case matchAny(token, "cline"):
		return Cline
	case matchAny(token, "roo-code", "roocode"):
		return RooCode
	case matchAny(token, "cursor"):
		return Cursor
	case matchAny(token, "continue"):
		return Continue
	case matchAny(token, "windsurf", "codeium"):
		return Windsurf
	case matchAny(token, "aider"):
		return Aider
	case matchAny(token, "misskey", "sharkey", "megalodon"):
		return OpenAISDK
	case matchAny(token, "python"):
		return PythonHTTP
	case matchAny(token, "node", "nodejs"):
		return NodeHTTP
	case matchAny(token, "go", "golang"):
		return GoHTTP
	case matchAny(token, "curl"):
		return Curl
	case matchAny(token, "wget"):
		return Wget
	default:
		return Unknown
	}
}

// Label returns a short human label for a client type id.
func Label(id string) string {
	if label, ok := Labels[id]; ok {
		return label
	}
	if id == "" {
		return Labels[Legacy]
	}
	return id
}

// NormalizeStored maps persisted client_type values for aggregation.
// Empty means audits written before client detection → Legacy.
func NormalizeStored(id string) string {
	id = strings.TrimSpace(id)
	if id == "" {
		return Legacy
	}
	return id
}

func hasHeader(headers map[string]string, name string) bool {
	return headerValue(headers, name) != ""
}

func headerValue(headers map[string]string, name string) string {
	if value := strings.TrimSpace(headers[name]); value != "" {
		return value
	}
	return strings.TrimSpace(headers[strings.ToLower(name)])
}

func matchAny(haystack string, needles ...string) bool {
	haystack = strings.ToLower(haystack)
	for _, needle := range needles {
		if needle != "" && strings.Contains(haystack, strings.ToLower(needle)) {
			return true
		}
	}
	return false
}
