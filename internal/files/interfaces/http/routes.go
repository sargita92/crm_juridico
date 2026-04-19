package http

import "github.com/gin-gonic/gin"

// RegisterRoutes wires the files endpoints under /tenant/files and also the
// lead-detail summary route under /tenant/leads. All routes are behind auth
// + tenant middlewares and require the `files:view` permission.
func (h *Handler) RegisterRoutes(
	router *gin.Engine,
	authMw gin.HandlerFunc,
	tenantMw gin.HandlerFunc,
	requirePerm func(string, string) gin.HandlerFunc,
) {
	files := router.Group("/tenant/files")
	files.Use(authMw, tenantMw, requirePerm("files", "view"))
	files.GET("", h.ListPage)
	files.GET("/list", h.ListFragment)
	files.GET("/:id/preview", h.PreviewDrawer)
	files.GET("/:id/download", h.Download)
	files.GET("/:id/thumbnail", h.Thumbnail)

	leads := router.Group("/tenant/leads")
	leads.Use(authMw, tenantMw, requirePerm("files", "view"))
	leads.GET("/:id/files-summary", h.LeadFilesSummary)
}
