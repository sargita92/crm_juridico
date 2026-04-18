package http

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"github.com/sasrgita/crm-juridico/internal/auth/application"
	permapp "github.com/sasrgita/crm-juridico/internal/permission/application"
	"github.com/sasrgita/crm-juridico/internal/shared/middleware"
)

// PageHandler renders HTML pages for the "Equipe > Usuários" tab.
type PageHandler struct {
	inviteUC        *application.InviteUserUseCase
	manageUsers     *application.ManageUsersUseCase
	listGroupsUC    *permapp.ListGroupsUseCase
	manageUserPerms *permapp.ManagePermissionsUseCase
	resolverUC      *permapp.ResolvePermissionUseCase
	log             *zap.Logger
}

func NewPageHandler(
	inviteUC *application.InviteUserUseCase,
	manageUsers *application.ManageUsersUseCase,
	listGroupsUC *permapp.ListGroupsUseCase,
	manageUserPerms *permapp.ManagePermissionsUseCase,
	resolverUC *permapp.ResolvePermissionUseCase,
	log *zap.Logger,
) *PageHandler {
	return &PageHandler{
		inviteUC:        inviteUC,
		manageUsers:     manageUsers,
		listGroupsUC:    listGroupsUC,
		manageUserPerms: manageUserPerms,
		resolverUC:      resolverUC,
		log:             log,
	}
}

// permissionCatalog is the complete list of (resource, action) pairs the UI exposes
// for individual user permissions. Kept in sync with backend middleware expectations.
var permissionCatalog = []struct{ Resource, Action string }{
	{"leads", "read"}, {"leads", "create"}, {"leads", "update"}, {"leads", "delete"},
	{"users", "read"}, {"users", "update"}, {"users", "delete"},
	{"groups", "manage"},
	{"funnels", "read"}, {"funnels", "customize"},
	{"automations", "manage"},
	{"products", "manage"},
	{"specialists", "manage"},
	{"invites", "create"}, {"invites", "read"}, {"invites", "delete"},
	{"settings", "manage"},
}

type modalPerm struct {
	Resource string
	Action   string
	Granted  bool
}

// RedirectToUsers handles GET /tenant/team.
func (h *PageHandler) RedirectToUsers(c *gin.Context) {
	c.Redirect(http.StatusFound, "/tenant/team/users")
}

// UsersPage renders the shell + users tab.
func (h *PageHandler) UsersPage(c *gin.Context) {
	tenantID := middleware.GetTenantID(c.Request.Context())

	users, err := h.manageUsers.ListTenantUsers(c.Request.Context(), tenantID)
	if err != nil {
		h.log.Error("failed to list users", zap.Error(err))
		c.Status(http.StatusInternalServerError)
		return
	}
	invites, err := h.inviteUC.ListInvites(c.Request.Context(), tenantID)
	if err != nil {
		h.log.Warn("failed to list invites", zap.Error(err))
		invites = nil
	}

	c.HTML(http.StatusOK, "team/shell.html", gin.H{
		"ActiveTab": "users",
		"Users":     users,
		"Invites":   invites,
	})
}

// --- Stubs (implemented in later tasks) ---

// UsersTable renders only the users + invites table fragment (for HTMX refreshes).
func (h *PageHandler) UsersTable(c *gin.Context) {
	tenantID := middleware.GetTenantID(c.Request.Context())

	users, err := h.manageUsers.ListTenantUsers(c.Request.Context(), tenantID)
	if err != nil {
		h.log.Error("failed to list users", zap.Error(err))
		c.Status(http.StatusInternalServerError)
		return
	}
	invites, err := h.inviteUC.ListInvites(c.Request.Context(), tenantID)
	if err != nil {
		h.log.Warn("failed to list invites", zap.Error(err))
		invites = nil
	}

	c.HTML(http.StatusOK, "team/users_table.html", gin.H{"Users": users, "Invites": invites})
}

// InviteNewModal renders the "Convidar Usuário" modal with the list of groups.
func (h *PageHandler) InviteNewModal(c *gin.Context) {
	tenantID := middleware.GetTenantID(c.Request.Context())

	groups, err := h.listGroupsUC.Execute(c.Request.Context(), tenantID)
	if err != nil {
		h.log.Error("failed to list groups for invite modal", zap.Error(err))
		c.Status(http.StatusInternalServerError)
		return
	}

	c.HTML(http.StatusOK, "team/invite_new_modal.html", gin.H{"Groups": groups})
}

// CreateInvite handles POST /tenant/team/invites — form submission from the modal.
func (h *PageHandler) CreateInvite(c *gin.Context) {
	claims := middleware.GetClaims(c.Request.Context())
	tenantID := middleware.GetTenantID(c.Request.Context())

	if claims == nil {
		c.Status(http.StatusUnauthorized)
		return
	}

	groupIDs := c.PostFormArray("group_ids")
	days := 7
	if n, err := strconv.Atoi(c.PostForm("expires_in_days")); err == nil && n > 0 {
		days = n
	}

	out, err := h.inviteUC.GenerateInvite(c.Request.Context(), tenantID, claims.UserID, groupIDs, days)
	if err != nil {
		h.log.Error("failed to generate invite", zap.Error(err))
		c.HTML(http.StatusInternalServerError, "team/invite_new_modal.html", gin.H{
			"Error":  "Falha ao gerar convite",
			"Groups": nil,
		})
		return
	}

	scheme := "http"
	if c.Request.TLS != nil {
		scheme = "https"
	}
	inviteURL := scheme + "://" + c.Request.Host + "/invite/" + out.Token

	c.HTML(http.StatusOK, "team/invite_success.html", gin.H{"InviteURL": inviteURL})
}

// UserPermissionsModal renders the individual-permissions modal for a user.
func (h *PageHandler) UserPermissionsModal(c *gin.Context) {
	tenantID := middleware.GetTenantID(c.Request.Context())
	userID := c.Param("id")

	granted, err := h.manageUserPerms.GetUserPermissions(c.Request.Context(), tenantID, userID)
	if err != nil {
		h.log.Error("failed to load user permissions", zap.Error(err))
		c.Status(http.StatusInternalServerError)
		return
	}
	grantedSet := make(map[string]bool, len(granted))
	for _, p := range granted {
		grantedSet[p.Resource+":"+p.Action] = true
	}

	all := make([]modalPerm, len(permissionCatalog))
	for i, p := range permissionCatalog {
		all[i] = modalPerm{
			Resource: p.Resource,
			Action:   p.Action,
			Granted:  grantedSet[p.Resource+":"+p.Action],
		}
	}

	c.HTML(http.StatusOK, "team/user_permissions_modal.html", gin.H{
		"UserID":   userID,
		"AllPerms": all,
	})
}

// SetUserPermissionsHTML handles POST /tenant/team/users/:id/permissions.
// Form field `perms` carries values like "resource:action".
func (h *PageHandler) SetUserPermissionsHTML(c *gin.Context) {
	tenantID := middleware.GetTenantID(c.Request.Context())
	userID := c.Param("id")

	items := c.PostFormArray("perms")
	inputs := make([]permapp.PermissionInput, 0, len(items))
	for _, it := range items {
		parts := strings.SplitN(it, ":", 2)
		if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
			continue
		}
		inputs = append(inputs, permapp.PermissionInput{Resource: parts[0], Action: parts[1]})
	}

	if err := h.manageUserPerms.SetUserPermissions(c.Request.Context(), tenantID, userID, inputs); err != nil {
		h.log.Error("failed to set user permissions", zap.Error(err))
		c.Status(http.StatusUnprocessableEntity)
		return
	}

	c.Header("HX-Trigger", "refreshTeam")
	c.Status(http.StatusNoContent)
}

func (h *PageHandler) UserWhatsAppModal(c *gin.Context)      { c.Status(http.StatusNotImplemented) }
func (h *PageHandler) SetUserWhatsApp(c *gin.Context)        { c.Status(http.StatusNotImplemented) }
