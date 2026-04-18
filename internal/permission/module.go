package permission

import (
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"gorm.io/gorm"

	authapp "github.com/sasrgita/crm-juridico/internal/auth/application"
	"github.com/sasrgita/crm-juridico/internal/permission/application"
	"github.com/sasrgita/crm-juridico/internal/permission/infrastructure"
	permhttp "github.com/sasrgita/crm-juridico/internal/permission/interfaces/http"
	"github.com/sasrgita/crm-juridico/internal/shared/module"
)

// Module wires together the permission feature.
type Module struct {
	handler    *permhttp.Handler
	resolverUC *application.ResolvePermissionUseCase
	vpUC       *application.ManageViewProfilesUseCase
}

// NewModule creates all permission repositories, use cases, and handlers.
func NewModule(db *gorm.DB, log *zap.Logger, loadBalanceUC *authapp.ManageLoadBalanceUseCase) *Module {
	// Repositories
	groupRepo := infrastructure.NewGormPermissionGroupRepository(db)
	ugRepo := infrastructure.NewGormUserGroupRepository(db)
	permRepo := infrastructure.NewGormPermissionRepository(db)
	vpRepo := infrastructure.NewGormViewProfileRepository(db)
	gfRepo := infrastructure.NewGormGroupFunnelRepository(db)

	// Adapters
	ownerChecker := infrastructure.NewOwnerCheckerAdapter(db)
	adminChecker := infrastructure.NewAdminCheckerAdapter(db)

	// Use cases
	resolverUC := application.NewResolvePermissionUseCase(permRepo, ugRepo, ownerChecker, adminChecker)
	createGroupUC := application.NewCreateGroupUseCase(groupRepo)
	getGroupUC := application.NewGetGroupUseCase(groupRepo)
	updateGroupUC := application.NewUpdateGroupUseCase(groupRepo)
	listGroupsUC := application.NewListGroupsUseCase(groupRepo)
	deleteGroupUC := application.NewDeleteGroupUseCase(groupRepo)
	manageMembersUC := application.NewManageMembersUseCase(groupRepo, ugRepo)
	managePermsUC := application.NewManagePermissionsUseCase(permRepo)
	manageVPUC := application.NewManageViewProfilesUseCase(vpRepo)
	manageGFUC := application.NewManageGroupFunnelsUseCase(gfRepo)

	handler := permhttp.NewHandler(
		createGroupUC,
		getGroupUC,
		updateGroupUC,
		listGroupsUC,
		deleteGroupUC,
		manageMembersUC,
		managePermsUC,
		manageVPUC,
		manageGFUC,
		loadBalanceUC,
		log,
	)

	return &Module{
		handler:    handler,
		resolverUC: resolverUC,
		vpUC:       manageVPUC,
	}
}

// Name returns the module identifier.
func (m *Module) Name() string { return "permission" }

// RegisterRoutes registers all permission HTTP routes.
func (m *Module) RegisterRoutes(router *gin.Engine, mw module.Middlewares) {
	m.handler.RegisterRoutes(router, mw.Auth, mw.Tenant, mw.RequirePermission)
}

// Resolver exposes the ResolvePermissionUseCase for use as a middleware checker.
func (m *Module) Resolver() *application.ResolvePermissionUseCase { return m.resolverUC }

// ViewProfileUC exposes the ManageViewProfilesUseCase for cross-module use.
func (m *Module) ViewProfileUC() *application.ManageViewProfilesUseCase { return m.vpUC }
