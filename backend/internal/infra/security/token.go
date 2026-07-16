package security

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

const clientKeyScheme = "g2a"

type adminClaims struct {
	AdminID   uint64 `json:"adminId"`
	SessionID uint64 `json:"sessionId"`
	jwt.RegisteredClaims
}

type AdminTokenIdentity struct {
	AdminID   uint64
	SessionID uint64
}

// TokenService 负责管理员 access token 和随机 refresh token。
type TokenService struct {
	secret []byte
	issuer string
}

func NewTokenService(secret string) *TokenService {
	return &TokenService{secret: []byte(secret), issuer: "grok2api"}
}

// CreateAccessToken 创建短期管理员 JWT。
func (s *TokenService) CreateAccessToken(adminID, sessionID uint64, ttl time.Duration) (string, time.Time, error) {
	now := time.Now().UTC()
	expiresAt := now.Add(ttl)
	claims := adminClaims{
		AdminID: adminID, SessionID: sessionID,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    s.issuer,
			Subject:   fmt.Sprintf("%d", adminID),
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(expiresAt),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString(s.secret)
	return signed, expiresAt, err
}

// ParseAccessToken 校验管理员 JWT 并返回管理员 ID。
func (s *TokenService) ParseAccessToken(raw string) (AdminTokenIdentity, error) {
	claims := &adminClaims{}
	token, err := jwt.ParseWithClaims(raw, claims, func(token *jwt.Token) (any, error) {
		if token.Method != jwt.SigningMethodHS256 {
			return nil, fmt.Errorf("不支持的 JWT 签名算法")
		}
		return s.secret, nil
	}, jwt.WithIssuer(s.issuer))
	if err != nil || !token.Valid || claims.AdminID == 0 || claims.SessionID == 0 {
		return AdminTokenIdentity{}, fmt.Errorf("管理员令牌无效")
	}
	return AdminTokenIdentity{AdminID: claims.AdminID, SessionID: claims.SessionID}, nil
}

// NewOpaqueToken 创建不可预测的 refresh token 或客户端 Key 密钥段。
func NewOpaqueToken(bytesLength int) (string, error) {
	buf := make([]byte, bytesLength)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(buf), nil
}

// NewHexToken 创建只包含十六进制字符的随机标识，适合放在分隔格式中。
func NewHexToken(bytesLength int) (string, error) {
	buf := make([]byte, bytesLength)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf), nil
}

// HashToken returns a fast, irreversible SHA-256 hex digest for API-key / refresh-token
// lookup indexes and rate-limit keys.
//
// Not password storage: inputs are high-entropy opaque tokens (client keys, refresh
// tokens, rate-limit subjects). Password-style KDFs (bcrypt/scrypt/argon2) would make
// O(1) auth lookup impossible; comparison is constant-time on the digest.
func HashToken(raw string) string {
	// codeql[go/weak-sensitive-data-hashing]: SHA-256 fingerprint of opaque API tokens for DB lookup, not a password KDF
	sum := sha256.Sum256([]byte(raw)) // lgtm[go/weak-sensitive-data-hashing]
	return hex.EncodeToString(sum[:])
}

// FormatClientKey 生成 g2a_<prefix>_<secret> 格式的客户端 Key。
func FormatClientKey(prefix, secret string) string {
	return clientKeyScheme + "_" + prefix + "_" + secret
}

// SplitClientKey 解析 g2a_<prefix>_<secret> 格式的客户端 Key。
func SplitClientKey(raw string) (string, bool) {
	parts := strings.SplitN(raw, "_", 3)
	if len(parts) != 3 || parts[0] != clientKeyScheme || parts[1] == "" || parts[2] == "" {
		return "", false
	}
	return parts[1], true
}
