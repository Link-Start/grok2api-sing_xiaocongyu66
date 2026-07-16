package middleware

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"

	clientkeydomain "github.com/chenyme/grok2api/backend/internal/domain/clientkey"
	"github.com/chenyme/grok2api/backend/internal/infra/security"
	"github.com/chenyme/grok2api/backend/internal/pkg/clientid"
	"github.com/gin-gonic/gin"
)

const RequestIDKey = "requestId"
const maxRequestIDLength = 64

// RequestID 为每个请求生成稳定关联 ID，并写入响应头。
func RequestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		requestID := strings.TrimSpace(c.GetHeader("X-Request-ID"))
		if !validRequestID(requestID) {
			requestID, _ = security.NewOpaqueToken(12)
			if requestID == "" {
				requestID = "req-" + strconv.FormatInt(time.Now().UnixNano(), 36)
			}
		}
		c.Set(RequestIDKey, requestID)
		c.Header("X-Request-ID", requestID)
		c.Next()
	}
}

// validRequestID 只接受适合写入日志和审计索引的短 ASCII 标识。
func validRequestID(value string) bool {
	if value == "" || len(value) > maxRequestIDLength {
		return false
	}
	for index := range len(value) {
		character := value[index]
		if (character >= 'a' && character <= 'z') || (character >= 'A' && character <= 'Z') || (character >= '0' && character <= '9') {
			continue
		}
		switch character {
		case '-', '_', '.', ':':
		default:
			return false
		}
	}
	return true
}

// Timeout 为 HTTP 请求设置统一生命周期上限。
func Timeout(duration time.Duration) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(c.Request.Context(), duration)
		defer cancel()
		c.Request = c.Request.WithContext(ctx)
		c.Next()
	}
}

// MaxBodyBytes 对所有请求体应用统一硬上限，避免管理端绑定无界读取。
func MaxBodyBytes(limit int64) gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.Request.Body != nil && limit > 0 {
			c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, limit)
		}
		c.Next()
	}
}

// SecurityHeaders 为 API 和媒体响应添加通用浏览器安全边界。
func SecurityHeaders() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("X-Content-Type-Options", "nosniff")
		c.Header("X-Frame-Options", "DENY")
		c.Header("Referrer-Policy", "no-referrer")
		c.Header("Permissions-Policy", "camera=(), microphone=(), geolocation=()")
		c.Header("Cross-Origin-Opener-Policy", "same-origin")
		c.Header("Cross-Origin-Resource-Policy", "same-site")
		// HSTS only when the request is already TLS (or terminated with HTTPS scheme).
		if c.Request.TLS != nil || strings.EqualFold(c.GetHeader("X-Forwarded-Proto"), "https") {
			c.Header("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
		}
		c.Next()
	}
}

func safeClientIP(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	// Strip zone / port forms and accept only parseable IPs.
	host := value
	if h, _, err := net.SplitHostPort(value); err == nil {
		host = h
	}
	host = strings.Trim(host, "[]")
	if ip := net.ParseIP(host); ip != nil {
		return ip.String()
	}
	return ""
}

// classifiedRoute returns only string constants so access logs never sink request-tainted path text.
func classifiedRoute(c *gin.Context) string {
	switch c.FullPath() {
	case "/healthz":
		return "healthz"
	case "/readyz":
		return "readyz"
	case "/v1/models":
		return "v1_models"
	case "/v1/chat/completions":
		return "v1_chat_completions"
	case "/v1/responses":
		return "v1_responses"
	case "/responses":
		return "responses"
	case "/v1/messages":
		return "v1_messages"
	case "/v1/images/generations":
		return "v1_images_generations"
	case "/v1/images/edits":
		return "v1_images_edits"
	case "/v1/videos/generations":
		return "v1_videos_generations"
	case "/v1/videos/:requestId":
		return "v1_videos_get"
	case "/v1/media/images/:assetId":
		return "v1_media_image"
	case "/api/admin/v1/auth/login":
		return "admin_auth_login"
	case "/api/admin/v1/auth/refresh":
		return "admin_auth_refresh"
	case "/api/admin/v1/auth/logout":
		return "admin_auth_logout"
	case "/api/admin/v1/me":
		return "admin_me"
	case "/api/admin/v1/me/password":
		return "admin_me_password"
	case "/api/admin/v1/accounts":
		return "admin_accounts"
	case "/api/admin/v1/accounts/summary":
		return "admin_accounts_summary"
	case "/api/admin/v1/accounts/:id":
		return "admin_account"
	case "/api/admin/v1/models":
		return "admin_models"
	case "/api/admin/v1/client-keys":
		return "admin_client_keys"
	case "/api/admin/v1/client-keys/:id":
		return "admin_client_key"
	case "/api/admin/v1/client-keys/:id/secret":
		return "admin_client_key_secret"
	case "/api/admin/v1/request-audits":
		return "admin_request_audits"
	case "/api/admin/v1/request-audits/:id":
		return "admin_request_audit"
	case "/api/admin/v1/request-audits/summary":
		return "admin_request_audits_summary"
	case "/api/admin/v1/dashboard":
		return "admin_dashboard"
	case "/api/admin/v1/settings":
		return "admin_settings"
	case "/api/admin/v1/system":
		return "admin_system"
	case "/api/admin/v1/egress-nodes":
		return "admin_egress_nodes"
	case "/api/admin/v1/egress-nodes/report":
		return "admin_egress_report"
	case "/api/admin/v1/egress-nodes/:id":
		return "admin_egress_node"
	case "/api/admin/v1/egress-nodes/:id/test":
		return "admin_egress_node_test"
	case "/api/admin/v1/media/images":
		return "admin_media_images"
	case "/api/admin/v1/media/videos":
		return "admin_media_videos"
	case "":
		return "unmatched"
	default:
		return "other"
	}
}

func classifiedClientType(value string) string {
	switch value {
	case "claude_code", "codex", "grok_cli", "hermes", "opencode", "cline", "cursor",
		"continue", "aider", "roo_code", "windsurf", "gemini_cli", "kiro", "mcp",
		"copilot", "openai_sdk", "anthropic_sdk", "node", "python", "go", "java",
		"rust", "ruby", "perl", "curl", "wget", "legacy", "unknown":
		return value
	default:
		return "other"
	}
}

// AccessLog 记录路径、状态、耗时与调用方元数据，不读取请求或响应正文。
func AccessLog(logger *slog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		startedAt := time.Now()
		c.Next()
		requestID, _ := c.Get(RequestIDKey)
		userAgent := strings.TrimSpace(c.Request.UserAgent())
		headers := map[string]string{}
		for _, name := range []string{
			"x-claude-code-session-id", "x-codex-window-id", "x-codex-session-id",
			"x-grok-conv-id", "x-grok-conversation-id",
			"originator", "x-app", "anthropic-version", "anthropic-beta",
			"x-stainless-lang", "x-stainless-package-version",
		} {
			if value := strings.TrimSpace(c.GetHeader(name)); value != "" {
				headers[strings.ToLower(name)] = value
			}
		}
		// Detection still uses UA/headers; logging only uses an allowlisted constant class.
		clientType := classifiedClientType(clientid.Detect(userAgent, headers))
		logMethod := c.Request.Method
		switch logMethod {
		case http.MethodGet, http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete, http.MethodHead, http.MethodOptions:
		default:
			logMethod = "OTHER"
		}
		reqID := fmt.Sprint(requestID)
		if !validRequestID(reqID) {
			reqID = "-"
		}
		attrs := []any{
			"request_id", reqID,
			"method", logMethod,
			"route", classifiedRoute(c),
			"status", c.Writer.Status(),
			"duration_ms", time.Since(startedAt).Milliseconds(),
			"client_ip", safeClientIP(c.ClientIP()),
			"client_type", clientType,
			"user_agent_len", len(userAgent),
			"bytes_out", c.Writer.Size(),
		}
		if keyValue, ok := c.Get(ClientKey); ok {
			if key, ok := keyValue.(clientkeydomain.Key); ok {
				// IDs only — never free-form key names in access logs.
				attrs = append(attrs, "client_key_id", key.ID)
			}
		}
		logger.Info("http_request", attrs...)
	}
}
