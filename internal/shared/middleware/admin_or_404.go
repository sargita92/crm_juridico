package middleware

import (
	"context"
	"net/http"
	"net/url"

	"github.com/gin-gonic/gin"

	"github.com/sasrgita/crm-juridico/internal/auth/domain"
)

// AdminOr404 e a variante de RequireAdmin usada exclusivamente pelas rotas
// /admin/logs (F12). Diferencas em relacao a RequireAdmin (que continua
// existindo para outras rotas admin que respondem JSON):
//
//  1. Nao autenticado (claims == nil): em vez de 401 JSON, redireciona via
//     302 para /admin/login?return=<path-original> (decisao do usuario
//     2026-04-24 — observacao 3 do briefing). E uma rota HTML server-rendered;
//     a redirect entrega UX coerente com o resto do painel admin.
//
//  2. Autenticado nao-admin (role != "admin"): em vez de 403 JSON, responde
//     404 e renderiza `audit/not_found.html` (Story 5 criterio 4 — OWASP
//     A01: nao revelar a existencia da rota a usuarios sem permissao).
//
// O 404 generico e o MESMO body usado pelos handlers em log inexistente,
// para que o cenario S5-C08 (resposta identica) seja satisfeito apenas
// inspecionando o status code + template — sem detalhe diferenciador no
// corpo.
func AdminOr404() gin.HandlerFunc {
	return func(c *gin.Context) {
		claims := GetClaims(c.Request.Context())

		if claims == nil {
			// Nao autenticado: redireciona para /admin/login mantendo a rota
			// original em ?return= para preservar UX de "voltar ao que tentava
			// fazer" apos o login.
			redirectURL := "/admin/login?return=" + url.QueryEscape(c.Request.URL.RequestURI())
			c.Redirect(http.StatusFound, redirectURL)
			c.Abort()
			return
		}

		if claims.Role != domain.UserRoleAdmin {
			// Autenticado mas sem privilegio: 404 generico identico ao usado
			// pelos handlers para id inexistente. Body neutro, sem revelar
			// que a rota existe.
			c.HTML(http.StatusNotFound, "admin/audit/not_found.html", gin.H{})
			c.Abort()
			return
		}

		c.Next()
	}
}

// AdminPageAuth e a versao "page-friendly" de Auth: tenta validar o token
// (cookie ou Bearer) e popular as claims; quando NAO ha token ou ele e
// invalido, retorna a request com claims vazias para o proximo middleware
// resolver. Usado em conjunto com AdminOr404 nas rotas /admin/logs* — a
// decisao de status code (302 ou 404) cabe ao AdminOr404, nao a este.
//
// E criado para nao quebrar a semantica do `Auth` existente (401 JSON),
// que continua sendo usado pelas demais rotas admin (JSON-first).
func AdminPageAuth(tokenProvider domain.TokenProvider) gin.HandlerFunc {
	return func(c *gin.Context) {
		tokenStr := extractTokenForPage(c)
		if tokenStr == "" {
			// Sem token: deixa o AdminOr404 redirecionar.
			c.Next()
			return
		}

		claims, err := tokenProvider.Validate(tokenStr)
		if err != nil || claims == nil {
			// Token invalido/expirado: tratamos como nao autenticado para que
			// o AdminOr404 redirecione consistentemente com S5-C09.
			c.Next()
			return
		}

		c.Set("user_id", claims.UserID)
		c.Set("role", string(claims.Role))
		c.Set("tenant_id", claims.TenantID)
		ctx := context.WithValue(c.Request.Context(), claimsKey{}, claims)
		c.Request = c.Request.WithContext(ctx)
		c.Next()
	}
}

// extractTokenForPage espelha extractToken (auth.go) sem expor o helper
// nao-exportado. Mantido aqui para evitar circularidade caso a logica de
// extracao precise divergir no futuro.
func extractTokenForPage(c *gin.Context) string {
	if token, err := c.Cookie("token"); err == nil && token != "" {
		return token
	}
	auth := c.GetHeader("Authorization")
	const bearer = "Bearer "
	if len(auth) > len(bearer) && auth[:len(bearer)] == bearer {
		return auth[len(bearer):]
	}
	return ""
}
