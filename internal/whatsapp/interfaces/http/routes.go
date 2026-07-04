package http

import "github.com/gin-gonic/gin"

func (h *Handler) RegisterRoutes(router *gin.Engine, authMw, tenantMw gin.HandlerFunc) {
	tenant := router.Group("/tenant/whatsapp")
	tenant.Use(authMw, tenantMw)

	tenant.GET("", h.RenderPage)
	tenant.GET("/qr", h.RenderQR)
	tenant.GET("/qr.png", h.ServeQRImage)
	tenant.GET("/status", h.RenderStatus)
	tenant.GET("/unread-badge", h.RenderUnreadBadge)
	tenant.GET("/conversations", h.RenderConversations)
	tenant.GET("/conversations/:id", h.RenderChat)
	tenant.GET("/conversations/:id/messages/new", h.RenderNewMessages)
	tenant.POST("/conversations/:id/messages", h.HandleSendMessage)
	tenant.GET("/conversations/:id/notes", h.RenderNotesPanel)
	tenant.POST("/conversations/:id/notes", h.HandleCreateNote)
	tenant.POST("/connect", h.HandleConnect)
	tenant.POST("/disconnect", h.HandleDisconnect)
	// SSE migrou para o stream unificado /tenant/stream (F26): uma única conexão por página.
}
