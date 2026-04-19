package middleware

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"
)

// TenantBillingFlagFetcher resolves whether the tenant should see the
// "Pagamentos" sidebar entry. Implemented by the pagamentos module
// (PortalAccessChecker / billing repo). Pure function-typed to avoid
// importing the pagamentos package from shared/middleware.
type TenantBillingFlagFetcher func(ctx context.Context, tenantID string) (showPagamentos bool)

// SidebarFlags emite cookies "ux-only" lidos por JS no client para esconder
// itens da sidebar quando não fazem sentido para o tenant atual. Os cookies
// NÃO são fonte de autorização — toda decisão de acesso real continua no
// middleware/handler do recurso.
//
// Pré-requisitos: Auth + RequireTenant já executados (claims/tenantID disponíveis).
func SidebarFlags(showPagamentos TenantBillingFlagFetcher) gin.HandlerFunc {
	return func(c *gin.Context) {
		claims := GetClaims(c.Request.Context())
		if claims == nil || claims.TenantID == "" {
			c.Next()
			return
		}
		val := "1"
		if showPagamentos != nil && !showPagamentos(c.Request.Context(), claims.TenantID) {
			val = "0"
		}
		// Cookie de sessão (sem MaxAge), HttpOnly=false para JS ler, SameSite=Lax,
		// Path=/ para cobrir todas as rotas tenant.
		c.SetSameSite(http.SameSiteLaxMode)
		c.SetCookie("ux_show_pagamentos", val, 0, "/", "", false, false)
		c.Next()
	}
}
