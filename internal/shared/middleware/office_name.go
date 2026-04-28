package middleware

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"
)

// OfficeNameCookie é o cookie "ux-only" que carrega o nome do escritório
// (Tenant.Name) para a sidebar/topbar do tenant. Lido por JS no client.
// NÃO é fonte de autorização — apenas display.
const OfficeNameCookie = "crm_office_name"

type officeNameKey struct{}

// OfficeNameResolver retorna o nome do escritório (Tenant.Name) para um
// tenantID. Função tipada para evitar dependência do módulo tenant em
// shared/middleware (mesmo padrão do TenantBillingFlagFetcher).
type OfficeNameResolver func(ctx context.Context, tenantID string) (string, error)

// OfficeName resolve o nome do escritório do tenant atual e o expõe de duas
// formas:
//   - cookie ux-only `crm_office_name` (URL-encoded UTF-8) consumido por JS
//     na sidebar/topbar.
//   - request context (GetOfficeName) para handlers Go que queiram usar.
//
// Pré-requisitos: Auth + RequireTenant já executados (claims/tenantID
// disponíveis). Sem claims, o middleware é no-op (cookie não é setado).
// Falhas do resolver não abortam a request — apenas omitem o cookie.
func OfficeName(resolver OfficeNameResolver) gin.HandlerFunc {
	return func(c *gin.Context) {
		if resolver == nil {
			c.Next()
			return
		}
		claims := GetClaims(c.Request.Context())
		if claims == nil || claims.TenantID == "" {
			c.Next()
			return
		}
		name, err := resolver(c.Request.Context(), claims.TenantID)
		if err != nil || name == "" {
			c.Next()
			return
		}
		c.SetSameSite(http.SameSiteLaxMode)
		// gin.Context.SetCookie já aplica url.QueryEscape no value, então passamos
		// o nome cru. Cliente decodifica com decodeURIComponent(v.replace(/\+/g,'%20')).
		c.SetCookie(OfficeNameCookie, name, 0, "/", "", false, false)

		ctx := context.WithValue(c.Request.Context(), officeNameKey{}, name)
		c.Request = c.Request.WithContext(ctx)
		c.Next()
	}
}

// GetOfficeName retorna o nome do escritório armazenado pelo middleware
// OfficeName, ou string vazia se ausente.
func GetOfficeName(ctx context.Context) string {
	if name, ok := ctx.Value(officeNameKey{}).(string); ok {
		return name
	}
	return ""
}
