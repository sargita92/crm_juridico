package http

import "github.com/gin-gonic/gin"

// RegisterRoutes attaches all notification routes to the given engine.
func (h *Handler) RegisterRoutes(router *gin.Engine, authMw, tenantMw gin.HandlerFunc) {
	notif := router.Group("/notifications")
	notif.Use(authMw, tenantMw)
	{
		notif.GET("/stream", h.StreamNotifications)
		notif.GET("", h.ListNotifications)
		notif.GET("/unread-count", h.UnreadCount)
		notif.POST("/:id/read", h.MarkRead)
		notif.POST("/read-all", h.MarkAllRead)
		notif.GET("/preferences", h.GetPreferences)
		notif.PUT("/preferences", h.UpdatePreferences)
	}
}
