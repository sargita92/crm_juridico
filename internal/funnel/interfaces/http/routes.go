package http

import "github.com/gin-gonic/gin"

func (h *Handler) RegisterRoutes(router *gin.Engine, authMw, tenantMw gin.HandlerFunc) {
	tenant := router.Group("/tenant/leads")
	tenant.Use(authMw, tenantMw)

	// Kanban
	tenant.GET("", h.RenderKanbanPage)
	tenant.GET("/kanban", h.RenderKanbanContent)

	// Funnels
	tenant.GET("/funnels", h.RenderFunnelList)
	tenant.GET("/funnels/new", h.RenderFunnelForm)
	tenant.POST("/funnels", h.HandleCreateFunnel)
	tenant.GET("/funnels/:id", h.RenderFunnelDetail)
	tenant.GET("/funnels/:id/edit", h.RenderFunnelForm)
	tenant.PUT("/funnels/:id", h.HandleUpdateFunnel)
	tenant.POST("/funnels/:id/toggle", h.HandleToggleFunnel)

	// Columns
	tenant.GET("/funnels/:id/columns", h.RenderColumnsSection)
	tenant.GET("/funnels/:id/columns/new", h.RenderColumnForm)
	tenant.POST("/funnels/:id/columns", h.HandleCreateColumn)
	tenant.DELETE("/funnels/:id/columns/:cid", h.HandleDeleteColumn)
	tenant.POST("/funnels/:id/columns/:cid/move-up", h.HandleMoveColumnUp)
	tenant.POST("/funnels/:id/columns/:cid/move-down", h.HandleMoveColumnDown)

	// Leads
	tenant.GET("/:id", h.RenderLeadDetail)
	tenant.GET("/:id/move-form", h.RenderLeadMoveForm)
	tenant.POST("/:id/move", h.HandleMoveLead)
	tenant.POST("/:id/notes", h.HandleCreateNote)
	tenant.GET("/:id/product-form", h.RenderLeadProductForm)
	tenant.PUT("/:id/product", h.HandleSetLeadProduct)
}
