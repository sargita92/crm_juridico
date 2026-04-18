package http

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	authapp "github.com/sasrgita/crm-juridico/internal/auth/application"
	funnelapp "github.com/sasrgita/crm-juridico/internal/funnel/application"
	funneldomain "github.com/sasrgita/crm-juridico/internal/funnel/domain"
	"github.com/sasrgita/crm-juridico/internal/permission/application"
	"github.com/sasrgita/crm-juridico/internal/shared/middleware"
)

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

// --- Stubs (implemented in Tasks 13-18) ---

func (h *PageHandler) GroupDetail(c *gin.Context)             { c.Status(http.StatusNotImplemented) }
func (h *PageHandler) GroupSection(c *gin.Context)            { c.Status(http.StatusNotImplemented) }
func (h *PageHandler) SetGroupPermissionsHTML(c *gin.Context) { c.Status(http.StatusNotImplemented) }
func (h *PageHandler) SetGroupFunnelsHTML(c *gin.Context)     { c.Status(http.StatusNotImplemented) }
func (h *PageHandler) SetViewProfileHTML(c *gin.Context)      { c.Status(http.StatusNotImplemented) }
func (h *PageHandler) SetLoadBalanceHTML(c *gin.Context)      { c.Status(http.StatusNotImplemented) }
