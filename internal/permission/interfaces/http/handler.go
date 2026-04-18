package http

import (
	"context"
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	authapp "github.com/sasrgita/crm-juridico/internal/auth/application"
	authdomain "github.com/sasrgita/crm-juridico/internal/auth/domain"
	"github.com/sasrgita/crm-juridico/internal/permission/application"
	"github.com/sasrgita/crm-juridico/internal/shared/middleware"
)

// loadBalanceUseCase is a local interface so tests can stub the concrete use case.
type loadBalanceUseCase interface {
	GetByGroup(ctx context.Context, tenantID, groupID string) (*authdomain.LoadBalanceConfig, error)
	SetByGroup(ctx context.Context, in authapp.SetLoadBalanceInput) (*authdomain.LoadBalanceConfig, error)
}

// Handler holds all permission-related use cases and provides HTTP handlers.
type Handler struct {
	createGroupUC   *application.CreateGroupUseCase
	getGroupUC      *application.GetGroupUseCase
	updateGroupUC   *application.UpdateGroupUseCase
	listGroupsUC    *application.ListGroupsUseCase
	deleteGroupUC   *application.DeleteGroupUseCase
	manageMembersUC *application.ManageMembersUseCase
	managePermsUC   *application.ManagePermissionsUseCase
	manageVPUC      *application.ManageViewProfilesUseCase
	manageGFUC      *application.ManageGroupFunnelsUseCase
	loadBalanceUC   loadBalanceUseCase
	log             *zap.Logger
}

// NewHandler constructs a Handler with all required use cases.
func NewHandler(
	createGroupUC *application.CreateGroupUseCase,
	getGroupUC *application.GetGroupUseCase,
	updateGroupUC *application.UpdateGroupUseCase,
	listGroupsUC *application.ListGroupsUseCase,
	deleteGroupUC *application.DeleteGroupUseCase,
	manageMembersUC *application.ManageMembersUseCase,
	managePermsUC *application.ManagePermissionsUseCase,
	manageVPUC *application.ManageViewProfilesUseCase,
	manageGFUC *application.ManageGroupFunnelsUseCase,
	loadBalanceUC loadBalanceUseCase,
	log *zap.Logger,
) *Handler {
	return &Handler{
		createGroupUC:   createGroupUC,
		getGroupUC:      getGroupUC,
		updateGroupUC:   updateGroupUC,
		listGroupsUC:    listGroupsUC,
		deleteGroupUC:   deleteGroupUC,
		manageMembersUC: manageMembersUC,
		managePermsUC:   managePermsUC,
		manageVPUC:      manageVPUC,
		manageGFUC:      manageGFUC,
		loadBalanceUC:   loadBalanceUC,
		log:             log,
	}
}

// --- Groups ---

// ListGroups returns all permission groups for the current tenant.
func (h *Handler) ListGroups(c *gin.Context) {
	tenantID := middleware.GetTenantID(c.Request.Context())

	groups, err := h.listGroupsUC.Execute(c.Request.Context(), tenantID)
	if err != nil {
		h.log.Error("failed to list groups", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list groups"})
		return
	}

	c.JSON(http.StatusOK, groups)
}

// GetGroup returns a single permission group by ID, scoped to the current tenant.
func (h *Handler) GetGroup(c *gin.Context) {
	tenantID := middleware.GetTenantID(c.Request.Context())
	id := c.Param("id")

	out, err := h.getGroupUC.Execute(c.Request.Context(), tenantID, id)
	if err != nil {
		h.log.Error("failed to get group", zap.String("id", id), zap.Error(err))
		c.JSON(http.StatusNotFound, gin.H{"error": "group not found"})
		return
	}

	c.JSON(http.StatusOK, out)
}

// CreateGroup creates a new permission group for the current tenant.
func (h *Handler) CreateGroup(c *gin.Context) {
	tenantID := middleware.GetTenantID(c.Request.Context())

	var req struct {
		Name        string `json:"name" form:"name"`
		Description string `json:"description" form:"description"`
	}
	if err := c.ShouldBind(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	out, err := h.createGroupUC.Execute(c.Request.Context(), application.CreateGroupInput{
		TenantID:    tenantID,
		Name:        req.Name,
		Description: req.Description,
	})
	if err != nil {
		h.log.Error("failed to create group", zap.Error(err))
		c.JSON(http.StatusUnprocessableEntity, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, out)
}

// UpdateGroup updates an existing permission group.
func (h *Handler) UpdateGroup(c *gin.Context) {
	tenantID := middleware.GetTenantID(c.Request.Context())
	id := c.Param("id")

	var req struct {
		Name        string `json:"name" form:"name"`
		Description string `json:"description" form:"description"`
	}
	if err := c.ShouldBind(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	out, err := h.updateGroupUC.Execute(c.Request.Context(), application.UpdateGroupInput{
		TenantID:    tenantID,
		ID:          id,
		Name:        req.Name,
		Description: req.Description,
	})
	if err != nil {
		h.log.Error("failed to update group", zap.String("id", id), zap.Error(err))
		c.JSON(http.StatusNotFound, gin.H{"error": "group not found"})
		return
	}

	c.JSON(http.StatusOK, out)
}

// DeleteGroup deletes a permission group by ID.
func (h *Handler) DeleteGroup(c *gin.Context) {
	tenantID := middleware.GetTenantID(c.Request.Context())
	id := c.Param("id")

	if err := h.deleteGroupUC.Execute(c.Request.Context(), tenantID, id); err != nil {
		h.log.Error("failed to delete group", zap.String("id", id), zap.Error(err))
		c.JSON(http.StatusNotFound, gin.H{"error": "group not found"})
		return
	}

	c.Status(http.StatusNoContent)
}

// --- Members ---

// ListMembers returns all members of a group.
func (h *Handler) ListMembers(c *gin.Context) {
	groupID := c.Param("id")

	members, err := h.manageMembersUC.ListMembers(c.Request.Context(), groupID)
	if err != nil {
		h.log.Error("failed to list members", zap.String("group_id", groupID), zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list members"})
		return
	}

	c.JSON(http.StatusOK, members)
}

// AddMember adds a user to a group.
func (h *Handler) AddMember(c *gin.Context) {
	tenantID := middleware.GetTenantID(c.Request.Context())
	groupID := c.Param("id")

	var req struct {
		UserID string `json:"user_id" form:"user_id"`
	}
	if err := c.ShouldBind(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.manageMembersUC.AddMember(c.Request.Context(), req.UserID, groupID, tenantID); err != nil {
		h.log.Error("failed to add member", zap.String("group_id", groupID), zap.String("user_id", req.UserID), zap.Error(err))
		c.JSON(http.StatusUnprocessableEntity, gin.H{"error": err.Error()})
		return
	}

	c.Status(http.StatusCreated)
}

// RemoveMember removes a user from a group.
func (h *Handler) RemoveMember(c *gin.Context) {
	groupID := c.Param("id")
	userID := c.Param("uid")

	if err := h.manageMembersUC.RemoveMember(c.Request.Context(), userID, groupID); err != nil {
		h.log.Error("failed to remove member", zap.String("group_id", groupID), zap.String("user_id", userID), zap.Error(err))
		c.JSON(http.StatusUnprocessableEntity, gin.H{"error": err.Error()})
		return
	}

	c.Status(http.StatusNoContent)
}

// --- Permissions ---

// GetGroupPermissions returns all permissions assigned to a group.
func (h *Handler) GetGroupPermissions(c *gin.Context) {
	groupID := c.Param("id")

	perms, err := h.managePermsUC.GetGroupPermissions(c.Request.Context(), groupID)
	if err != nil {
		h.log.Error("failed to get group permissions", zap.String("group_id", groupID), zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get group permissions"})
		return
	}

	c.JSON(http.StatusOK, perms)
}

// SetGroupPermissions replaces all permissions for a group.
func (h *Handler) SetGroupPermissions(c *gin.Context) {
	tenantID := middleware.GetTenantID(c.Request.Context())
	groupID := c.Param("id")

	var req []struct {
		Resource string `json:"resource" form:"resource"`
		Action   string `json:"action" form:"action"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	inputs := make([]application.PermissionInput, len(req))
	for i, r := range req {
		inputs[i] = application.PermissionInput{
			Resource: r.Resource,
			Action:   r.Action,
		}
	}

	if err := h.managePermsUC.SetGroupPermissions(c.Request.Context(), tenantID, groupID, inputs); err != nil {
		h.log.Error("failed to set group permissions", zap.String("group_id", groupID), zap.Error(err))
		c.JSON(http.StatusUnprocessableEntity, gin.H{"error": err.Error()})
		return
	}

	c.Status(http.StatusOK)
}

// GetUserPermissions returns all individual permissions assigned to a user within the current tenant.
func (h *Handler) GetUserPermissions(c *gin.Context) {
	tenantID := middleware.GetTenantID(c.Request.Context())
	userID := c.Param("id")

	perms, err := h.managePermsUC.GetUserPermissions(c.Request.Context(), tenantID, userID)
	if err != nil {
		h.log.Error("failed to get user permissions", zap.String("user_id", userID), zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get user permissions"})
		return
	}

	c.JSON(http.StatusOK, perms)
}

// SetUserPermissions replaces all individual permissions for a user within the current tenant.
func (h *Handler) SetUserPermissions(c *gin.Context) {
	tenantID := middleware.GetTenantID(c.Request.Context())
	userID := c.Param("id")

	var req []struct {
		Resource string `json:"resource" form:"resource"`
		Action   string `json:"action" form:"action"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	inputs := make([]application.PermissionInput, len(req))
	for i, r := range req {
		inputs[i] = application.PermissionInput{
			Resource: r.Resource,
			Action:   r.Action,
		}
	}

	if err := h.managePermsUC.SetUserPermissions(c.Request.Context(), tenantID, userID, inputs); err != nil {
		h.log.Error("failed to set user permissions", zap.String("user_id", userID), zap.Error(err))
		c.JSON(http.StatusUnprocessableEntity, gin.H{"error": err.Error()})
		return
	}

	c.Status(http.StatusOK)
}

// --- View Profiles ---

// ListViewProfiles returns all view profiles for a group.
func (h *Handler) ListViewProfiles(c *gin.Context) {
	groupID := c.Param("id")

	profiles, err := h.manageVPUC.ListByGroup(c.Request.Context(), groupID)
	if err != nil {
		h.log.Error("failed to list view profiles", zap.String("group_id", groupID), zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list view profiles"})
		return
	}

	c.JSON(http.StatusOK, profiles)
}

// SetViewProfile creates or updates the view profile for a group+funnel combination.
func (h *Handler) SetViewProfile(c *gin.Context) {
	groupID := c.Param("id")
	funnelID := c.Param("fid")

	var req struct {
		VisibleColumns []string `json:"visible_columns" form:"visible_columns"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.manageVPUC.SetViewProfile(c.Request.Context(), application.ViewProfileInput{
		GroupID:        groupID,
		FunnelID:       funnelID,
		VisibleColumns: req.VisibleColumns,
	}); err != nil {
		h.log.Error("failed to set view profile", zap.String("group_id", groupID), zap.String("funnel_id", funnelID), zap.Error(err))
		c.JSON(http.StatusUnprocessableEntity, gin.H{"error": err.Error()})
		return
	}

	c.Status(http.StatusOK)
}

// --- Group Funnels ---

// ListGroupFunnels returns all funnel associations for a group.
func (h *Handler) ListGroupFunnels(c *gin.Context) {
	groupID := c.Param("id")

	funnels, err := h.manageGFUC.ListByGroup(c.Request.Context(), groupID)
	if err != nil {
		h.log.Error("failed to list group funnels", zap.String("group_id", groupID), zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list group funnels"})
		return
	}

	c.JSON(http.StatusOK, funnels)
}

// SetGroupFunnels sets one or more funnel associations for a group.
// Each entry in the request body will be upserted individually.
func (h *Handler) SetGroupFunnels(c *gin.Context) {
	groupID := c.Param("id")

	var req []struct {
		FunnelID  string   `json:"funnel_id" form:"funnel_id"`
		ColumnIDs []string `json:"column_ids" form:"column_ids"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	for _, r := range req {
		if err := h.manageGFUC.SetGroupFunnel(c.Request.Context(), application.GroupFunnelInput{
			GroupID:   groupID,
			FunnelID:  r.FunnelID,
			ColumnIDs: r.ColumnIDs,
		}); err != nil {
			h.log.Error("failed to set group funnel",
				zap.String("group_id", groupID),
				zap.String("funnel_id", r.FunnelID),
				zap.Error(err),
			)
			c.JSON(http.StatusUnprocessableEntity, gin.H{"error": err.Error()})
			return
		}
	}

	c.Status(http.StatusOK)
}

// --- Load Balance ---

func (h *Handler) GetLoadBalance(c *gin.Context) {
	tenantID := middleware.GetTenantID(c.Request.Context())
	groupID := c.Param("id")

	cfg, err := h.loadBalanceUC.GetByGroup(c.Request.Context(), tenantID, groupID)
	if err != nil {
		if errors.Is(err, authapp.ErrGroupNotInTenant) {
			c.JSON(http.StatusNotFound, gin.H{"error": "group not found"})
			return
		}
		if errors.Is(err, authdomain.ErrLoadBalanceNotFound) {
			c.JSON(http.StatusOK, gin.H{"algorithm": "round_robin", "active": false, "exists": false})
			return
		}
		h.log.Error("failed to get load balance", zap.String("group_id", groupID), zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get load balance"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"algorithm": string(cfg.Algorithm),
		"active":    cfg.Active,
		"exists":    true,
	})
}

func (h *Handler) SetLoadBalance(c *gin.Context) {
	tenantID := middleware.GetTenantID(c.Request.Context())
	groupID := c.Param("id")

	var req struct {
		Algorithm string `json:"algorithm"`
		Active    bool   `json:"active"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	cfg, err := h.loadBalanceUC.SetByGroup(c.Request.Context(), authapp.SetLoadBalanceInput{
		TenantID:  tenantID,
		GroupID:   groupID,
		Algorithm: authdomain.LoadBalanceAlgorithm(req.Algorithm),
		Active:    req.Active,
	})
	if err != nil {
		if errors.Is(err, authapp.ErrGroupNotInTenant) {
			c.JSON(http.StatusNotFound, gin.H{"error": "group not found"})
			return
		}
		if errors.Is(err, authdomain.ErrInvalidAlgorithm) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid algorithm"})
			return
		}
		h.log.Error("failed to set load balance", zap.String("group_id", groupID), zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to set load balance"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"algorithm": string(cfg.Algorithm),
		"active":    cfg.Active,
	})
}
