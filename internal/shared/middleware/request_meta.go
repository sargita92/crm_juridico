package middleware

import (
	"context"
	"net"
	"strings"

	"github.com/gin-gonic/gin"
)

// MaxUserAgentLength limita o tamanho do User-Agent armazenado no contexto.
// Alinhado ao limite da coluna `audit_logs.user_agent VARCHAR(255)` para evitar
// que o consumidor (ex.: F12 publisher) precise truncar de novo.
const MaxUserAgentLength = 255

type ipKey struct{}
type userAgentKey struct{}

// RequestMeta extrai IP e User-Agent do request HTTP e injeta no
// `context.Context` via chaves tipadas. As features que precisam desses
// metadados (audit, segurança) leem usando os helpers IPFromContext /
// UserAgentFromContext.
//
// IP: respeita `X-Forwarded-For` (primeiro IP valido) e `X-Real-IP`. Se nada
// for valido, cai em `c.ClientIP()` que ja e o padrao da Gin para resolver
// upstream. Strings vazias resultam em "" no contexto (caller decide o
// fallback).
//
// User-Agent: trunca em MaxUserAgentLength para nao estourar a coluna do
// banco em casos de bots agressivos (alguns User-Agents tem 1KB+).
func RequestMeta() gin.HandlerFunc {
	return func(c *gin.Context) {
		ip := extractIP(c)
		ua := truncateUA(c.GetHeader("User-Agent"))

		ctx := context.WithValue(c.Request.Context(), ipKey{}, ip)
		ctx = context.WithValue(ctx, userAgentKey{}, ua)
		c.Request = c.Request.WithContext(ctx)

		c.Next()
	}
}

// IPFromContext retorna o IP injetado pelo middleware RequestMeta.
// Quando o middleware nao foi aplicado, retorna string vazia.
func IPFromContext(ctx context.Context) string {
	if ip, ok := ctx.Value(ipKey{}).(string); ok {
		return ip
	}
	return ""
}

// UserAgentFromContext retorna o User-Agent injetado pelo middleware
// RequestMeta. Quando o middleware nao foi aplicado, retorna string vazia.
func UserAgentFromContext(ctx context.Context) string {
	if ua, ok := ctx.Value(userAgentKey{}).(string); ok {
		return ua
	}
	return ""
}

// extractIP devolve o primeiro IP valido a partir de:
//  1. X-Forwarded-For (primeiro elemento valido apos parse)
//  2. X-Real-IP
//  3. c.ClientIP() (Gin ja considera RemoteAddr / proxy trust)
//
// Retorna string vazia somente se todas as alternativas falharem (ex.: teste
// sintetico sem RemoteAddr).
func extractIP(c *gin.Context) string {
	if xff := c.GetHeader("X-Forwarded-For"); xff != "" {
		for _, part := range strings.Split(xff, ",") {
			candidate := strings.TrimSpace(part)
			if candidate == "" {
				continue
			}
			if ip := net.ParseIP(candidate); ip != nil {
				return candidate
			}
		}
	}

	if xri := strings.TrimSpace(c.GetHeader("X-Real-IP")); xri != "" {
		if ip := net.ParseIP(xri); ip != nil {
			return xri
		}
	}

	return c.ClientIP()
}

// truncateUA garante que o User-Agent caiba na coluna do banco.
// Trunca em runas (nao bytes) para nao quebrar UTF-8 no meio.
func truncateUA(ua string) string {
	if ua == "" {
		return ""
	}
	if len(ua) <= MaxUserAgentLength {
		return ua
	}
	runes := []rune(ua)
	if len(runes) <= MaxUserAgentLength {
		return ua
	}
	return string(runes[:MaxUserAgentLength])
}
