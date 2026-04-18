package http

import "github.com/gin-gonic/gin"

// RegisterRoutes registers all permission-related routes on the given router.
func (h *Handler) RegisterRoutes(
	router *gin.Engine,
	page *PageHandler,
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

	team := router.Group("/tenant/team")
	team.Use(authMw, tenantMw)
	{
		team.GET("/groups", requirePerm("groups", "manage"), page.GroupsPage)
		team.GET("/groups/table", requirePerm("groups", "manage"), page.GroupsTable)
		team.GET("/groups/new-modal", requirePerm("groups", "manage"), page.GroupNewModal)
		team.POST("/groups", requirePerm("groups", "manage"), page.CreateGroupHTML)
		team.GET("/groups/:id", requirePerm("groups", "manage"), page.GroupDetail)
		team.GET("/groups/:id/section/:name", requirePerm("groups", "manage"), page.GroupSection)
		team.POST("/groups/:id/permissions", requirePerm("groups", "manage"), page.SetGroupPermissionsHTML)
		team.POST("/groups/:id/funnels", requirePerm("groups", "manage"), page.SetGroupFunnelsHTML)
		team.POST("/groups/:id/view-profiles/:fid", requirePerm("funnels", "customize"), page.SetViewProfileHTML)
		team.POST("/groups/:id/load-balance", requirePerm("groups", "manage"), page.SetLoadBalanceHTML)
	}
}
