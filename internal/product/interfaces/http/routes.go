package http

import "github.com/gin-gonic/gin"

func (h *Handler) RegisterRoutes(router *gin.Engine, authMw, tenantMw, adminMw gin.HandlerFunc) {
	// Admin routes — product CRUD (admin-only)
	admin := router.Group("/admin/products")
	admin.Use(authMw, adminMw)

	admin.GET("", h.RenderAdminProductList)
	admin.GET("/new", h.RenderAdminProductForm)
	admin.POST("", h.HandleAdminCreateProduct)
	admin.GET("/:id/edit", h.RenderAdminProductForm)
	admin.PUT("/:id", h.HandleAdminUpdateProduct)
	admin.POST("/:id/toggle", h.HandleAdminToggleProduct)
	admin.DELETE("/:id", h.HandleAdminDeleteProduct)

	// Funnel-product links (admin)
	admin.POST("/:id/funnels", h.HandleAdminLinkFunnel)
	admin.DELETE("/:id/funnels/:funnelId", h.HandleAdminUnlinkFunnel)
	admin.PUT("/:id/funnels/:funnelId/priority", h.HandleAdminUpdatePriority)

	// Tenant-product associations (admin)
	admin.POST("/:id/tenants", h.HandleAdminAssociateTenant)
	admin.DELETE("/:id/tenants/:tenantId", h.HandleAdminDisassociateTenant)

	// Tenant routes — product list + funnel linking
	tenant := router.Group("/tenant/products")
	tenant.Use(authMw, tenantMw)

	tenant.GET("", h.RenderTenantProductList)
	tenant.POST("/:id/funnels", h.HandleTenantLinkFunnel)
	tenant.DELETE("/:id/funnels/:funnelId", h.HandleTenantUnlinkFunnel)
	tenant.PUT("/:id/funnels/:funnelId/priority", h.HandleTenantUpdatePriority)
}
