package playground

import (
	"github.com/gin-gonic/gin"

	"github.com/sasrgita/crm-juridico/internal/shared/module"
)

// RegisterRoutes wires all playground endpoints behind auth+tenant middleware.
// The caller should only invoke this when the playground flag is enabled.
func (h *Handler) RegisterRoutes(router *gin.Engine, mw module.Middlewares) {
	g := router.Group("/tenant/ai/playground")
	g.Use(mw.Auth, mw.Tenant)
	g.GET("", h.RenderPage)
	g.GET("/conversation/:contact_id", h.RenderConversation)
	g.POST("/conversation/:contact_id/send", h.HandleSendAsLead)
	g.POST("/conversation/:contact_id/reset", h.HandleReset)
}
