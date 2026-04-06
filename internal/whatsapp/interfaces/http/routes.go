package http

import "github.com/gin-gonic/gin"

func (h *Handler) RegisterRoutes(router *gin.Engine, authMw, tenantMw gin.HandlerFunc) {
	tenant := router.Group("/tenant/whatsapp")
	tenant.Use(authMw, tenantMw)

	tenant.GET("", h.RenderPage)
	tenant.GET("/qr", h.RenderQR)
	tenant.GET("/qr.png", h.ServeQRImage)
	tenant.GET("/status", h.RenderStatus)
	tenant.GET("/conversations", h.RenderConversations)
	tenant.GET("/conversations/:id", h.RenderChat)
	tenant.GET("/conversations/:id/messages/new", h.RenderNewMessages)
	tenant.POST("/conversations/:id/messages", h.HandleSendMessage)
	tenant.POST("/connect", h.HandleConnect)
	tenant.POST("/disconnect", h.HandleDisconnect)
	tenant.GET("/events", h.HandleSSE)
}
