package auth

import (
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/sasrgita/crm-juridico/internal/auth/application"
	"github.com/sasrgita/crm-juridico/internal/auth/infrastructure"
	authhttp "github.com/sasrgita/crm-juridico/internal/auth/interfaces/http"
	"github.com/sasrgita/crm-juridico/internal/shared/module"
	tenantdomain "github.com/sasrgita/crm-juridico/internal/tenant/domain"
)

// Module wires up the auth domain (login, tenant selection, invites, user management).
type Module struct {
	handler       *authhttp.Handler
	inviteUC      *application.InviteUserUseCase
	manageUsersUC *application.ManageUsersUseCase
	loginUC       *application.LoginUseCase
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

	handler := authhttp.NewHandler(loginUC, selectTenantUC, listTenantsUC, secureCookie)

	return &Module{
		handler:       handler,
		inviteUC:      inviteUC,
		manageUsersUC: manageUsersUC,
		loginUC:       loginUC,
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
	}
}
