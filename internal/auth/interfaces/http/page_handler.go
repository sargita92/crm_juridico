package http

import (
	"net/http"

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
	invites, _ := h.inviteUC.ListInvites(c.Request.Context(), tenantID)

	c.HTML(http.StatusOK, "team/shell.html", gin.H{
		"ActiveTab": "users",
		"Users":     users,
		"Invites":   invites,
	})
}

// --- Stubs (implemented in later tasks) ---

func (h *PageHandler) UsersTable(c *gin.Context)             { c.Status(http.StatusNotImplemented) }
func (h *PageHandler) InviteNewModal(c *gin.Context)         { c.Status(http.StatusNotImplemented) }
func (h *PageHandler) CreateInvite(c *gin.Context)           { c.Status(http.StatusNotImplemented) }
func (h *PageHandler) UserPermissionsModal(c *gin.Context)   { c.Status(http.StatusNotImplemented) }
func (h *PageHandler) SetUserPermissionsHTML(c *gin.Context) { c.Status(http.StatusNotImplemented) }
func (h *PageHandler) UserWhatsAppModal(c *gin.Context)      { c.Status(http.StatusNotImplemented) }
func (h *PageHandler) SetUserWhatsApp(c *gin.Context)        { c.Status(http.StatusNotImplemented) }
