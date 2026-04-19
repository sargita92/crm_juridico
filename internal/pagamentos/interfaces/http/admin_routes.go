package http

import (
	"github.com/gin-gonic/gin"

	"github.com/sasrgita/crm-juridico/internal/shared/module"
)

func (h *Handler) registerAdminRoutes(router *gin.Engine, mw module.Middlewares) {
	admin := router.Group("/admin")
	admin.Use(mw.Auth, mw.Admin)

	admin.GET("/payment", h.AdminListGlobal)
	admin.POST("/payment/:id/pagar", h.AdminMarkAsPaid)
	admin.POST("/payment/:id/cancelar", h.AdminCancel)

	admin.GET("/tenants/:id/payment", h.AdminListTenant)
	admin.GET("/tenants/:id/payment/novo", h.AdminFormNovo)
	admin.POST("/tenants/:id/payment", h.AdminCreateAvulso)
	admin.GET("/tenants/:id/payment/resumo", h.AdminTenantSummary)
}
