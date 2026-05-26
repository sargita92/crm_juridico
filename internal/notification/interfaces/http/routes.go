package http

import "github.com/gin-gonic/gin"

// RegisterRoutes attaches the JSON/API notification routes.
func (h *Handler) RegisterRoutes(router *gin.Engine, authMw, tenantMw gin.HandlerFunc) {
	// Stream SSE unificado por página (F26): um único endpoint que entrega
	// notificações E encaminha os demais eventos (new-message, conversation-update,
	// lead-*). Substitui /notifications/stream e /tenant/whatsapp/events, para que
	// cada página mantenha apenas UMA conexão SSE.
	stream := router.Group("/tenant")
	stream.Use(authMw, tenantMw)
	stream.GET("/stream", h.StreamNotifications)

	notif := router.Group("/notifications")
	notif.Use(authMw, tenantMw)
	{
		notif.GET("", h.ListNotifications)
		notif.GET("/unread-count", h.UnreadCount)
		notif.POST("/:id/read", h.MarkRead)
		notif.POST("/read-all", h.MarkAllRead)
		notif.GET("/preferences", h.GetPreferences)
		notif.PUT("/preferences", h.UpdatePreferences)
	}
}

// RegisterPageRoutes attaches the HTML page routes under /tenant/notifications.
func (p *PageHandler) RegisterPageRoutes(router *gin.Engine, authMw, tenantMw gin.HandlerFunc) {
	pages := router.Group("/tenant/notifications")
	pages.Use(authMw, tenantMw)
	{
		pages.GET("", p.RenderPage)
		pages.GET("/list", p.RenderList)
		pages.GET("/dropdown", p.RenderDropdown)
		pages.GET("/badge", p.RenderBadge)
	}
}
