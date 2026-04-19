package http

import (
	"github.com/gin-gonic/gin"

	"github.com/sasrgita/crm-juridico/internal/shared/module"
)

// SetPortalMiddleware injeta o middleware de autorizacao do portal (tenant
// com plano cobravel + exibir_pagamentos=true + usuario com payments:view
// ou owner/admin). Deve ser chamado pelo module antes de RegisterRoutes.
func (h *Handler) SetPortalMiddleware(mw gin.HandlerFunc) {
	h.portalMw = mw
}

func (h *Handler) registerTenantPortalRoutes(router *gin.Engine, mw module.Middlewares) {
	if h.portalMw == nil {
		// sem middleware de portal, nao expomos as rotas (seguranca em primeiro lugar)
		return
	}
	g := router.Group("/pagamentos")
	g.Use(mw.Auth, mw.Tenant, h.portalMw)
	g.GET("", h.TenantList)
}
