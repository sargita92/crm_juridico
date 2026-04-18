package http

import "github.com/gin-gonic/gin"

// RegisterRoutes registers all permission-related routes on the given router.
func (h *Handler) RegisterRoutes(
	router *gin.Engine,
	authMw, tenantMw gin.HandlerFunc,
	requirePerm func(resource, action string) gin.HandlerFunc,
) {
	groups := router.Group("/tenant/groups")
	groups.Use(authMw, tenantMw)
	{
		groups.GET("", requirePerm("groups", "manage"), h.ListGroups)
		groups.POST("", requirePerm("groups", "manage"), h.CreateGroup)
		groups.GET("/:id", requirePerm("groups", "manage"), h.GetGroup)
		groups.PUT("/:id", requirePerm("groups", "manage"), h.UpdateGroup)
		groups.DELETE("/:id", requirePerm("groups", "manage"), h.DeleteGroup)
		groups.GET("/:id/members", requirePerm("groups", "manage"), h.ListMembers)
		groups.POST("/:id/members", requirePerm("groups", "manage"), h.AddMember)
		groups.DELETE("/:id/members/:uid", requirePerm("groups", "manage"), h.RemoveMember)
		groups.GET("/:id/permissions", requirePerm("groups", "manage"), h.GetGroupPermissions)
		groups.PUT("/:id/permissions", requirePerm("groups", "manage"), h.SetGroupPermissions)
		groups.GET("/:id/view-profiles", requirePerm("funnels", "customize"), h.ListViewProfiles)
		groups.PUT("/:id/view-profiles/:fid", requirePerm("funnels", "customize"), h.SetViewProfile)
		groups.GET("/:id/funnels", requirePerm("groups", "manage"), h.ListGroupFunnels)
		groups.PUT("/:id/funnels", requirePerm("groups", "manage"), h.SetGroupFunnels)
		groups.GET("/:id/load-balance", requirePerm("groups", "manage"), h.GetLoadBalance)
		groups.PUT("/:id/load-balance", requirePerm("groups", "manage"), h.SetLoadBalance)
	}

	users := router.Group("/tenant/users")
	users.Use(authMw, tenantMw)
	{
		users.GET("/:id/permissions", requirePerm("users", "manage"), h.GetUserPermissions)
		users.PUT("/:id/permissions", requirePerm("users", "manage"), h.SetUserPermissions)
	}
}
