package http

import (
	"github.com/gin-gonic/gin"

	"github.com/sasrgita/crm-juridico/internal/shared/module"
)

func (h *Handler) registerAdminRoutes(router *gin.Engine, mw module.Middlewares) {
	admin := router.Group("/admin")
	admin.Use(mw.Auth, mw.Admin)

	admin.GET("/pagamentos", h.AdminListGlobal)
	admin.POST("/pagamentos/:id/pagar", h.AdminMarkAsPaid)
	admin.POST("/pagamentos/:id/cancelar", h.AdminCancel)

	admin.GET("/tenants/:id/pagamentos", h.AdminListTenant)
	admin.GET("/tenants/:id/pagamentos/novo", h.AdminFormNovo)
	admin.POST("/tenants/:id/pagamentos", h.AdminCreateAvulso)
	admin.GET("/tenants/:id/pagamentos/resumo", h.AdminTenantSummary)
}
