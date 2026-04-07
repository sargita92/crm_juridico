package http

import "github.com/gin-gonic/gin"

// RegisterRoutes wires the automation HTTP routes into the provided router.
func (h *Handler) RegisterRoutes(
	router *gin.Engine,
	authMw gin.HandlerFunc,
	tenantMw gin.HandlerFunc,
	requirePerm func(string, string) gin.HandlerFunc,
) {
	funnels := router.Group("/tenant/leads/funnels")
	funnels.Use(authMw, tenantMw)
	funnels.GET("/:id/automations", requirePerm("automations", "manage"), h.ListByFunnel)
	funnels.POST("/:id/automations", requirePerm("automations", "manage"), h.Create)

	autos := router.Group("/tenant/leads/automations")
	autos.Use(authMw, tenantMw)
	autos.GET("/:id", requirePerm("automations", "manage"), h.GetDetail)
	autos.PUT("/:id", requirePerm("automations", "manage"), h.Update)
	autos.DELETE("/:id", requirePerm("automations", "manage"), h.Delete)
	autos.POST("/:id/toggle", requirePerm("automations", "manage"), h.Toggle)
	autos.GET("/:id/logs", requirePerm("automations", "manage"), h.GetLogs)
}

// RegisterPageRoutes wires the automation HTML page routes.
func (h *PageHandler) RegisterPageRoutes(
	router *gin.Engine,
	authMw gin.HandlerFunc,
	tenantMw gin.HandlerFunc,
	requirePerm func(string, string) gin.HandlerFunc,
) {
	pages := router.Group("/tenant/automations")
	pages.Use(authMw, tenantMw, requirePerm("automations", "manage"))
	pages.GET("", h.ListPage)
	pages.GET("/table", h.RenderTable)
	pages.GET("/create-form", h.RenderCreateForm)
	pages.GET("/fields", h.RenderFields)
	pages.GET("/:id/form", h.RenderEditForm)
	pages.POST("", h.HandleCreate)
	pages.PUT("/:id", h.HandleUpdate)
	pages.DELETE("/:id", h.HandleDelete)
	pages.POST("/:id/toggle", h.HandleToggle)
	pages.GET("/:id/logs", h.RenderLogs)
}
