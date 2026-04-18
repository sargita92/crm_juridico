package http

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	authapp "github.com/sasrgita/crm-juridico/internal/auth/application"
	funnelapp "github.com/sasrgita/crm-juridico/internal/funnel/application"
	funneldomain "github.com/sasrgita/crm-juridico/internal/funnel/domain"
	"github.com/sasrgita/crm-juridico/internal/permission/application"
	"github.com/sasrgita/crm-juridico/internal/shared/middleware"
)

var groupPermActions = []string{"read", "create", "update", "delete", "manage", "customize"}
var groupPermResources = []string{"leads", "users", "groups", "funnels", "automations", "products", "specialists", "invites", "settings"}
var groupPermAvailable = map[string]map[string]bool{
	"leads":       {"read": true, "create": true, "update": true, "delete": true},
	"users":       {"read": true, "update": true, "delete": true},
	"groups":      {"manage": true},
	"funnels":     {"read": true, "update": true, "customize": true},
	"automations": {"manage": true},
	"products":    {"manage": true},
	"specialists": {"manage": true},
	"invites":     {"create": true, "read": true, "delete": true},
	"settings":    {"manage": true},
}

type permCell struct {
	Resource  string
	Action    string
	Available bool
	Granted   bool
}

type permRow struct {
	Resource string
	Cells    []permCell
}

type funnelOpt struct {
	ID       string
	Name     string
	Assigned bool
}

type columnOpt struct {
	ID      string
	Name    string
	Visible bool
}

type funnelWithCols struct {
	ID      string
	Name    string
	Columns []columnOpt
}

// PageHandler renders HTML pages for the "Equipe > Grupos" tab and group detail.
type PageHandler struct {
	createGroup   *application.CreateGroupUseCase
	listGroups    *application.ListGroupsUseCase
	getGroup      *application.GetGroupUseCase
	updateGroup   *application.UpdateGroupUseCase
	deleteGroup   *application.DeleteGroupUseCase
	manageMembers *application.ManageMembersUseCase
	managePerms   *application.ManagePermissionsUseCase
	manageVP      *application.ManageViewProfilesUseCase
	manageGF      *application.ManageGroupFunnelsUseCase
	loadBalanceUC *authapp.ManageLoadBalanceUseCase
	listFunnelsUC *funnelapp.ListFunnelsUseCase
	columnRepo    funneldomain.ColumnRepository
	usersListUC   *authapp.ManageUsersUseCase
	log           *zap.Logger
}

func NewPageHandler(
	createGroup *application.CreateGroupUseCase,
	listGroups *application.ListGroupsUseCase,
	getGroup *application.GetGroupUseCase,
	updateGroup *application.UpdateGroupUseCase,
	deleteGroup *application.DeleteGroupUseCase,
	manageMembers *application.ManageMembersUseCase,
	managePerms *application.ManagePermissionsUseCase,
	manageVP *application.ManageViewProfilesUseCase,
	manageGF *application.ManageGroupFunnelsUseCase,
	loadBalanceUC *authapp.ManageLoadBalanceUseCase,
	listFunnelsUC *funnelapp.ListFunnelsUseCase,
	columnRepo funneldomain.ColumnRepository,
	usersListUC *authapp.ManageUsersUseCase,
	log *zap.Logger,
) *PageHandler {
	return &PageHandler{
		createGroup:   createGroup,
		listGroups:    listGroups,
		getGroup:      getGroup,
		updateGroup:   updateGroup,
		deleteGroup:   deleteGroup,
		manageMembers: manageMembers,
		managePerms:   managePerms,
		manageVP:      manageVP,
		manageGF:      manageGF,
		loadBalanceUC: loadBalanceUC,
		listFunnelsUC: listFunnelsUC,
		columnRepo:    columnRepo,
		usersListUC:   usersListUC,
		log:           log,
	}
}

// --- Groups tab (working) ---

// GroupsPage renders the shell + groups tab.
func (h *PageHandler) GroupsPage(c *gin.Context) {
	tenantID := middleware.GetTenantID(c.Request.Context())
	groups, err := h.listGroups.Execute(c.Request.Context(), tenantID)
	if err != nil {
		h.log.Error("failed to list groups", zap.Error(err))
		c.Status(http.StatusInternalServerError)
		return
	}
	c.HTML(http.StatusOK, "team/shell.html", gin.H{
		"ActiveTab": "groups",
		"Groups":    groups,
	})
}

// GroupsTable renders only the groups table fragment.
func (h *PageHandler) GroupsTable(c *gin.Context) {
	tenantID := middleware.GetTenantID(c.Request.Context())
	groups, err := h.listGroups.Execute(c.Request.Context(), tenantID)
	if err != nil {
		h.log.Error("failed to list groups", zap.Error(err))
		c.Status(http.StatusInternalServerError)
		return
	}
	c.HTML(http.StatusOK, "team/groups_table.html", gin.H{"Groups": groups})
}

// GroupNewModal renders the "Novo Grupo" modal.
func (h *PageHandler) GroupNewModal(c *gin.Context) {
	c.HTML(http.StatusOK, "team/group_new_modal.html", nil)
}

// CreateGroupHTML handles POST /tenant/team/groups.
func (h *PageHandler) CreateGroupHTML(c *gin.Context) {
	tenantID := middleware.GetTenantID(c.Request.Context())
	out, err := h.createGroup.Execute(c.Request.Context(), application.CreateGroupInput{
		TenantID:    tenantID,
		Name:        c.PostForm("name"),
		Description: c.PostForm("description"),
	})
	if err != nil {
		h.log.Error("failed to create group", zap.Error(err))
		c.Status(http.StatusUnprocessableEntity)
		return
	}
	c.Header("HX-Redirect", "/tenant/team/groups/"+out.ID)
	c.Status(http.StatusOK)
}

// --- Group detail scaffold (Tasks 13-18) ---

// GroupDetail renders the detail page for a single group (header + section cards).
func (h *PageHandler) GroupDetail(c *gin.Context) {
	tenantID := middleware.GetTenantID(c.Request.Context())
	groupID := c.Param("id")

	group, err := h.getGroup.Execute(c.Request.Context(), tenantID, groupID)
	if err != nil {
		h.log.Warn("group not found", zap.String("group_id", groupID), zap.Error(err))
		c.Status(http.StatusNotFound)
		return
	}

	sections := []struct {
		Slug  string
		Title string
	}{
		{"members", "👥 Membros"},
		{"permissions", "🔐 Permissões"},
		{"funnels", "🎯 Funis atribuídos"},
		{"view-profiles", "👁️ Perfis de visualização"},
		{"load-balance", "⚖️ Load Balance"},
	}

	c.HTML(http.StatusOK, "team/group_detail.html", gin.H{
		"Group":    group,
		"Sections": sections,
	})
}

// GroupSection dispatches to the appropriate section renderer.
func (h *PageHandler) GroupSection(c *gin.Context) {
	name := c.Param("name")
	switch name {
	case "members":
		h.sectionMembers(c)
	case "permissions":
		h.sectionPermissions(c)
	case "funnels":
		h.sectionFunnels(c)
	case "view-profiles":
		h.sectionViewProfiles(c)
	case "load-balance":
		h.sectionLoadBalance(c)
	default:
		c.Status(http.StatusNotFound)
	}
}

// sectionMembers renders the members section with member list and add form.
func (h *PageHandler) sectionMembers(c *gin.Context) {
	ctx := c.Request.Context()
	tenantID := middleware.GetTenantID(ctx)
	groupID := c.Param("id")

	members, err := h.manageMembers.ListMembers(ctx, groupID)
	if err != nil {
		h.log.Error("failed to list members", zap.String("group_id", groupID), zap.Error(err))
		c.Status(http.StatusInternalServerError)
		return
	}

	allUsers, err := h.usersListUC.ListTenantUsers(ctx, tenantID)
	if err != nil {
		h.log.Error("failed to list tenant users", zap.Error(err))
		c.Status(http.StatusInternalServerError)
		return
	}

	memberIDs := make(map[string]bool, len(members))
	for _, m := range members {
		memberIDs[m.ID] = true
	}

	candidates := make([]authapp.UserOutput, 0, len(allUsers))
	for _, u := range allUsers {
		if !memberIDs[u.ID] {
			candidates = append(candidates, u)
		}
	}

	c.HTML(http.StatusOK, "team/group_section_members.html", gin.H{
		"GroupID":    groupID,
		"Members":    members,
		"Candidates": candidates,
	})
}

func (h *PageHandler) sectionPermissions(c *gin.Context) {
	groupID := c.Param("id")

	granted, err := h.managePerms.GetGroupPermissions(c.Request.Context(), groupID)
	if err != nil {
		h.log.Error("failed to get group permissions", zap.String("group_id", groupID), zap.Error(err))
		c.Status(http.StatusInternalServerError)
		return
	}
	gset := make(map[string]bool, len(granted))
	for _, p := range granted {
		gset[p.Resource+":"+p.Action] = true
	}

	rows := make([]permRow, 0, len(groupPermResources))
	for _, r := range groupPermResources {
		cells := make([]permCell, len(groupPermActions))
		for i, a := range groupPermActions {
			available := groupPermAvailable[r][a]
			cells[i] = permCell{
				Resource:  r,
				Action:    a,
				Available: available,
				Granted:   available && gset[r+":"+a],
			}
		}
		rows = append(rows, permRow{Resource: r, Cells: cells})
	}

	c.HTML(http.StatusOK, "team/group_section_permissions.html", gin.H{
		"GroupID": groupID,
		"Actions": groupPermActions,
		"Rows":    rows,
	})
}
func (h *PageHandler) sectionFunnels(c *gin.Context) {
	ctx := c.Request.Context()
	tenantID := middleware.GetTenantID(ctx)
	groupID := c.Param("id")

	funnels, err := h.listFunnelsUC.Execute(ctx, tenantID)
	if err != nil {
		h.log.Error("failed to list funnels", zap.Error(err))
		c.Status(http.StatusInternalServerError)
		return
	}

	assigned, err := h.manageGF.ListByGroup(ctx, groupID)
	if err != nil {
		h.log.Error("failed to list group funnels", zap.Error(err))
		c.Status(http.StatusInternalServerError)
		return
	}
	aset := make(map[string]bool, len(assigned))
	for _, a := range assigned {
		aset[a.FunnelID] = true
	}

	opts := make([]funnelOpt, len(funnels))
	for i, f := range funnels {
		opts[i] = funnelOpt{ID: f.ID, Name: f.Name, Assigned: aset[f.ID]}
	}

	c.HTML(http.StatusOK, "team/group_section_funnels.html", gin.H{
		"GroupID": groupID,
		"Funnels": opts,
	})
}
func (h *PageHandler) sectionViewProfiles(c *gin.Context) {
	ctx := c.Request.Context()
	tenantID := middleware.GetTenantID(ctx)
	groupID := c.Param("id")

	assignedFunnels, err := h.manageGF.ListByGroup(ctx, groupID)
	if err != nil {
		h.log.Error("failed to list group funnels", zap.Error(err))
		c.Status(http.StatusInternalServerError)
		return
	}

	profiles, err := h.manageVP.ListByGroup(ctx, groupID)
	if err != nil {
		h.log.Error("failed to list view profiles", zap.Error(err))
		c.Status(http.StatusInternalServerError)
		return
	}
	// Map: funnelID -> set of visibleColumnIDs.
	visibleByFunnel := make(map[string]map[string]bool, len(profiles))
	for _, p := range profiles {
		m := make(map[string]bool, len(p.VisibleColumns))
		for _, cid := range p.VisibleColumns {
			m[cid] = true
		}
		visibleByFunnel[p.FunnelID] = m
	}

	// Build funnel-name map to enrich display.
	allFunnels, err := h.listFunnelsUC.Execute(ctx, tenantID)
	if err != nil {
		h.log.Error("failed to list funnels", zap.Error(err))
		c.Status(http.StatusInternalServerError)
		return
	}
	nameByID := make(map[string]string, len(allFunnels))
	for _, f := range allFunnels {
		nameByID[f.ID] = f.Name
	}

	result := make([]funnelWithCols, 0, len(assignedFunnels))
	for _, gf := range assignedFunnels {
		cols, err := h.columnRepo.FindByFunnelID(ctx, gf.FunnelID)
		if err != nil {
			h.log.Warn("failed to list columns", zap.String("funnel_id", gf.FunnelID), zap.Error(err))
			continue
		}

		colOpts := make([]columnOpt, len(cols))
		visibleSet, hasProfile := visibleByFunnel[gf.FunnelID]
		for i, col := range cols {
			visible := true // default: all columns visible when no profile is set
			if hasProfile {
				visible = visibleSet[col.ID]
			}
			colOpts[i] = columnOpt{ID: col.ID, Name: col.Name, Visible: visible}
		}

		result = append(result, funnelWithCols{
			ID:      gf.FunnelID,
			Name:    nameByID[gf.FunnelID],
			Columns: colOpts,
		})
	}

	c.HTML(http.StatusOK, "team/group_section_view_profiles.html", gin.H{
		"GroupID": groupID,
		"Funnels": result,
	})
}
func (h *PageHandler) sectionLoadBalance(c *gin.Context)  { c.Status(http.StatusNotImplemented) }

func (h *PageHandler) SetGroupPermissionsHTML(c *gin.Context) {
	tenantID := middleware.GetTenantID(c.Request.Context())
	groupID := c.Param("id")

	items := c.PostFormArray("perms")
	inputs := make([]application.PermissionInput, 0, len(items))
	for _, it := range items {
		parts := strings.SplitN(it, ":", 2)
		if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
			continue
		}
		inputs = append(inputs, application.PermissionInput{Resource: parts[0], Action: parts[1]})
	}

	if err := h.managePerms.SetGroupPermissions(c.Request.Context(), tenantID, groupID, inputs); err != nil {
		h.log.Error("failed to set group permissions", zap.Error(err))
		c.Status(http.StatusUnprocessableEntity)
		return
	}

	c.Status(http.StatusOK)
}

func (h *PageHandler) SetGroupFunnelsHTML(c *gin.Context) {
	ctx := c.Request.Context()
	groupID := c.Param("id")

	funnelIDs := c.PostFormArray("funnel_ids")

	for _, fid := range funnelIDs {
		if err := h.manageGF.SetGroupFunnel(ctx, application.GroupFunnelInput{
			GroupID:   groupID,
			FunnelID:  fid,
			ColumnIDs: nil,
		}); err != nil {
			h.log.Error("failed to set group funnel", zap.String("funnel_id", fid), zap.Error(err))
			c.Status(http.StatusUnprocessableEntity)
			return
		}
	}

	c.Status(http.StatusOK)
}
func (h *PageHandler) SetViewProfileHTML(c *gin.Context) {
	groupID := c.Param("id")
	funnelID := c.Param("fid")
	cols := c.PostFormArray("visible_columns")

	if err := h.manageVP.SetViewProfile(c.Request.Context(), application.ViewProfileInput{
		GroupID:        groupID,
		FunnelID:       funnelID,
		VisibleColumns: cols,
	}); err != nil {
		h.log.Error("failed to set view profile", zap.Error(err))
		c.Status(http.StatusUnprocessableEntity)
		return
	}

	c.Status(http.StatusOK)
}
func (h *PageHandler) SetLoadBalanceHTML(c *gin.Context)      { c.Status(http.StatusNotImplemented) }
