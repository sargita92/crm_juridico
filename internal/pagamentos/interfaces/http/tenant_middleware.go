package http

import (
	"context"
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/sasrgita/crm-juridico/internal/pagamentos/domain"
)

// PermissionChecker abstrai o ResolvePermissionUseCase. O HasPermission ja
// inclui owner bypass + admin bypass + grupos/individuals.
type PermissionChecker interface {
	HasPermission(ctx context.Context, userID, tenantID, resource, action string) (bool, error)
}

// PortalAccessChecker autoriza /tenant/payment no portal do tenant verificando:
//  1. tenant tem plano cobravel e exibir_pagamentos = true (caso contrario 404)
//  2. usuario atual tem "payments:view" (ou e owner/admin via Resolver).
type PortalAccessChecker struct {
	billing    domain.TenantBillingRepository
	permission PermissionChecker
}

func NewPortalAccessChecker(billing domain.TenantBillingRepository, perm PermissionChecker) *PortalAccessChecker {
	return &PortalAccessChecker{billing: billing, permission: perm}
}

func (c *PortalAccessChecker) Middleware() gin.HandlerFunc {
	return func(gc *gin.Context) {
		tenantID := currentTenantID(gc)
		userID := currentUserID(gc)
		if tenantID == "" || userID == "" {
			gc.AbortWithStatus(http.StatusUnauthorized)
			return
		}

		tb, err := c.billing.GetByID(gc.Request.Context(), tenantID)
		if err != nil {
			if errors.Is(err, domain.ErrTenantNotFound) {
				gc.AbortWithStatus(http.StatusNotFound)
				return
			}
			gc.AbortWithStatus(http.StatusInternalServerError)
			return
		}
		if !tb.Config.ShowsPortalMenu() {
			// Tenant existe mas pagamentos não está disponível (sem plano cobrável
			// ou exibir_pagamentos=false). Render empty state em vez de 404 cru.
			gc.Abort()
			gc.HTML(http.StatusNotFound, "pagamentos/portal_unavailable.html", nil)
			return
		}

		allowed, err := c.permission.HasPermission(gc.Request.Context(), userID, tenantID, "payments", "view")
		if err != nil {
			gc.AbortWithStatus(http.StatusInternalServerError)
			return
		}
		if !allowed {
			gc.AbortWithStatus(http.StatusForbidden)
			return
		}
		gc.Next()
	}
}

func currentTenantID(c *gin.Context) string {
	if v, ok := c.Get("tenant_id"); ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}
