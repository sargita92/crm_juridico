package http

import "github.com/gin-gonic/gin"

func (h *Handler) RegisterRoutes(router *gin.Engine, authMw, tenantMw gin.HandlerFunc) {
	tenant := router.Group("/tenant/products")
	tenant.Use(authMw, tenantMw)

	tenant.GET("", h.RenderProductList)
	tenant.GET("/new", h.RenderProductForm)
	tenant.POST("", h.HandleCreateProduct)
	tenant.GET("/:id/edit", h.RenderProductForm)
	tenant.PUT("/:id", h.HandleUpdateProduct)
	tenant.POST("/:id/toggle", h.HandleToggleProduct)

	// Funnel-product links
	tenant.POST("/:id/funnels", h.HandleLinkFunnel)
	tenant.DELETE("/:id/funnels/:funnelId", h.HandleUnlinkFunnel)
	tenant.PUT("/:id/funnels/:funnelId/priority", h.HandleUpdatePriority)
}
