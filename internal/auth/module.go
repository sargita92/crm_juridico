package auth

import (
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"gorm.io/gorm"

	"github.com/sasrgita/crm-juridico/internal/auth/application"
	authdomain "github.com/sasrgita/crm-juridico/internal/auth/domain"
	"github.com/sasrgita/crm-juridico/internal/auth/infrastructure"
	authhttp "github.com/sasrgita/crm-juridico/internal/auth/interfaces/http"
	permapp "github.com/sasrgita/crm-juridico/internal/permission/application"
	"github.com/sasrgita/crm-juridico/internal/shared/module"
	tenantdomain "github.com/sasrgita/crm-juridico/internal/tenant/domain"
)

// Module wires up the auth domain (login, tenant selection, invites, user management).
type Module struct {
	handler         *authhttp.Handler
	pageHandler     *authhttp.PageHandler
	inviteUC        *application.InviteUserUseCase
	manageUsersUC   *application.ManageUsersUseCase
	loginUC         *application.LoginUseCase
	loadBalanceUC   *application.ManageLoadBalanceUseCase
	loadBalanceRepo authdomain.LoadBalanceConfigRepository
	userTenantRepo  authdomain.UserTenantRepository
}

// NewModule builds and returns a fully wired auth Module.
func NewModule(
	db *gorm.DB,
	tenantRepo tenantdomain.TenantRepository,
	jwtSecret string,
	jwtExpiration time.Duration,
	secureCookie bool,
) *Module {
	userRepo := infrastructure.NewGormUserRepository(db)
	userTenantRepo := infrastructure.NewGormUserTenantRepository(db)
	inviteRepo := infrastructure.NewGormInviteTokenRepository(db)
	passwordHasher := infrastructure.NewBcryptHasher()
	tokenProvider := infrastructure.NewJWTProvider(jwtSecret, jwtExpiration)

	loginUC := application.NewLoginUseCase(userRepo, userTenantRepo, tenantRepo, passwordHasher, tokenProvider)
	selectTenantUC := application.NewSelectTenantUseCase(userTenantRepo, tenantRepo, tokenProvider)
	listTenantsUC := application.NewListUserTenantsUseCase(userTenantRepo, tenantRepo)
	inviteUC := application.NewInviteUserUseCase(inviteRepo, userRepo, userTenantRepo, passwordHasher)
	manageUsersUC := application.NewManageUsersUseCase(userRepo, userTenantRepo)

	loadBalanceRepo := infrastructure.NewGormLoadBalanceConfigRepository(db)
	groupChecker := infrastructure.NewGroupTenantCheckerAdapter(db)
	// Overlap checker is wired later via SetLoadBalanceOverlapChecker because it
	// depends on permission repositories that are constructed after this module.
	loadBalanceUC := application.NewManageLoadBalanceUseCase(loadBalanceRepo, groupChecker, nil)

	handler := authhttp.NewHandler(loginUC, selectTenantUC, listTenantsUC, secureCookie)

	return &Module{
		handler:         handler,
		inviteUC:        inviteUC,
		manageUsersUC:   manageUsersUC,
		loginUC:         loginUC,
		loadBalanceUC:   loadBalanceUC,
		loadBalanceRepo: loadBalanceRepo,
		userTenantRepo:  userTenantRepo,
	}
}

// Name implements module.Module.
func (m *Module) Name() string { return "auth" }

// RegisterRoutes implements module.Module.
// The base auth routes (login, logout, select-tenant, dashboard) are registered here.
// Invite and user-management tenant routes are registered via RegisterTenantRoutes.
func (m *Module) RegisterRoutes(router *gin.Engine, mw module.Middlewares) {
	m.handler.RegisterRoutes(router, mw.Auth, mw.Tenant)
	m.registerInvitePublicRoutes(router)
	m.registerTenantRoutes(router, mw)
}

// Handler exposes the underlying HTTP handler (used by main.go for admin login).
func (m *Module) Handler() *authhttp.Handler { return m.handler }

// LoginUseCase exposes the login use case (used by main.go admin login route).
func (m *Module) LoginUseCase() *application.LoginUseCase { return m.loginUC }

// InviteUseCase exposes the invite use case.
func (m *Module) InviteUseCase() *application.InviteUserUseCase { return m.inviteUC }

// ManageUsersUseCase exposes the manage-users use case.
func (m *Module) ManageUsersUseCase() *application.ManageUsersUseCase { return m.manageUsersUC }

// LoadBalanceUseCase exposes the manage-load-balance use case (used cross-module).
func (m *Module) LoadBalanceUseCase() *application.ManageLoadBalanceUseCase { return m.loadBalanceUC }

// LoadBalanceRepo exposes the load-balance config repository so that
// cross-module adapters (e.g. permission's GroupColumnOverlapAdapter) can query
// active configs without depending on gorm directly.
func (m *Module) LoadBalanceRepo() authdomain.LoadBalanceConfigRepository {
	return m.loadBalanceRepo
}

// UserTenantRepo exposes the user-tenant repository so that cross-module
// adapters (e.g. auth's LoadBalancePicker, wired in main.go) can resolve
// tenant membership and owner fallback without duplicating the gorm instance.
func (m *Module) UserTenantRepo() authdomain.UserTenantRepository {
	return m.userTenantRepo
}

// SetLoadBalanceOverlapChecker injects the cross-module overlap checker into
// the ManageLoadBalanceUseCase. Must be called after the permission module is
// available so that ManageLoadBalanceUseCase.SetByGroup can enforce the
// one-active-LB-per-column invariant.
func (m *Module) SetLoadBalanceOverlapChecker(checker application.GroupColumnOverlapChecker) {
	m.loadBalanceUC.SetOverlapChecker(checker)
}

// AttachPermissionDeps wires the PageHandler with use cases from the permission module.
// Must be called after NewModule and before RegisterRoutes — otherwise the /tenant/team/*
// routes will be skipped. Calling twice will overwrite the previous PageHandler; the
// wiring expects a single invocation from main.go.
func (m *Module) AttachPermissionDeps(
	listGroupsUC *permapp.ListGroupsUseCase,
	manageUserPerms *permapp.ManagePermissionsUseCase,
	resolverUC *permapp.ResolvePermissionUseCase,
	log *zap.Logger,
) {
	m.pageHandler = authhttp.NewPageHandler(m.inviteUC, m.manageUsersUC, listGroupsUC, manageUserPerms, resolverUC, log)
}

// registerInvitePublicRoutes registers routes that do not require authentication.
func (m *Module) registerInvitePublicRoutes(router *gin.Engine) {
	router.GET("/invite/:token", m.handleGetInviteInfo)
	router.POST("/invite/:token/accept", m.handleAcceptInvite)
}

// registerTenantRoutes registers invite and user-management routes that require auth + tenant.
func (m *Module) registerTenantRoutes(router *gin.Engine, mw module.Middlewares) {
	tenantGroup := router.Group("/tenant")
	tenantGroup.Use(mw.Auth, mw.Tenant)
	{
		tenantGroup.POST("/invites", mw.RequirePermission("invites", "create"), m.handleGenerateInvite)
		tenantGroup.GET("/invites", mw.RequirePermission("invites", "read"), m.handleListInvites)
		tenantGroup.DELETE("/invites/:id", mw.RequirePermission("invites", "delete"), m.handleRevokeInvite)

		tenantGroup.GET("/users", mw.RequirePermission("users", "read"), m.handleListTenantUsers)
		tenantGroup.DELETE("/users/:id", mw.RequirePermission("users", "delete"), m.handleRemoveUser)
		tenantGroup.PUT("/users/:id/whatsapp", mw.RequirePermission("users", "update"), m.handleSetWhatsAppID)

		// HTML page routes (require pageHandler to be attached via AttachPermissionDeps)
		if m.pageHandler != nil {
			tenantGroup.GET("/team", m.pageHandler.RedirectToUsers)
			tenantGroup.GET("/team/users", mw.RequirePermission("users", "read"), m.pageHandler.UsersPage)
			tenantGroup.GET("/team/users/table", mw.RequirePermission("users", "read"), m.pageHandler.UsersTable)
			tenantGroup.GET("/team/users/:id/permissions-modal", mw.RequirePermission("users", "update"), m.pageHandler.UserPermissionsModal)
			tenantGroup.POST("/team/users/:id/permissions", mw.RequirePermission("users", "update"), m.pageHandler.SetUserPermissionsHTML)
			tenantGroup.GET("/team/users/:id/whatsapp-modal", mw.RequirePermission("users", "update"), m.pageHandler.UserWhatsAppModal)
			tenantGroup.POST("/team/users/:id/whatsapp", mw.RequirePermission("users", "update"), m.pageHandler.SetUserWhatsApp)
			tenantGroup.GET("/team/invites/new-modal", mw.RequirePermission("invites", "create"), m.pageHandler.InviteNewModal)
			tenantGroup.POST("/team/invites", mw.RequirePermission("invites", "create"), m.pageHandler.CreateInvite)
		}
	}
}
