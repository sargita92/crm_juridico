package permission

import (
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"gorm.io/gorm"

	auditapp "github.com/sasrgita/crm-juridico/internal/audit/application"
	authapp "github.com/sasrgita/crm-juridico/internal/auth/application"
	authdomain "github.com/sasrgita/crm-juridico/internal/auth/domain"
	funnelapp "github.com/sasrgita/crm-juridico/internal/funnel/application"
	funneldomain "github.com/sasrgita/crm-juridico/internal/funnel/domain"
	"github.com/sasrgita/crm-juridico/internal/permission/application"
	permdomain "github.com/sasrgita/crm-juridico/internal/permission/domain"
	"github.com/sasrgita/crm-juridico/internal/permission/infrastructure"
	permhttp "github.com/sasrgita/crm-juridico/internal/permission/interfaces/http"
	"github.com/sasrgita/crm-juridico/internal/shared/module"
)

// Module wires together the permission feature.
type Module struct {
	handler         *permhttp.Handler
	pageHandler     *permhttp.PageHandler
	resolverUC      *application.ResolvePermissionUseCase
	vpUC            *application.ManageViewProfilesUseCase
	listGroupsUC    *application.ListGroupsUseCase
	managePermsUC   *application.ManagePermissionsUseCase
	groupFunnelRepo permdomain.GroupFunnelRepository
	userGroupRepo   permdomain.UserGroupRepository
}

// NewModule creates all permission repositories, use cases, and handlers.
func NewModule(
	db *gorm.DB,
	log *zap.Logger,
	loadBalanceUC *authapp.ManageLoadBalanceUseCase,
	listFunnelsUC *funnelapp.ListFunnelsUseCase,
	columnRepo funneldomain.ColumnRepository,
	usersListUC *authapp.ManageUsersUseCase,
	userRepo authdomain.UserRepository,
) *Module {
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
	manageMembersUC := application.NewManageMembersUseCase(groupRepo, ugRepo, userRepo)
	managePermsUC := application.NewManagePermissionsUseCase(permRepo)
	// Injeta userRepo no UC de permissoes para que SetUserPermissions
	// consiga distinguir alvo admin (publica audit) vs tenant user
	// (nao publica) — F12 Step 7.
	managePermsUC.SetUserRepo(userRepo)
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

	pageHandler := permhttp.NewPageHandler(
		createGroupUC,
		listGroupsUC,
		getGroupUC,
		updateGroupUC,
		deleteGroupUC,
		manageMembersUC,
		managePermsUC,
		manageVPUC,
		manageGFUC,
		loadBalanceUC,
		listFunnelsUC,
		columnRepo,
		usersListUC,
		log,
	)

	return &Module{
		handler:         handler,
		pageHandler:     pageHandler,
		resolverUC:      resolverUC,
		vpUC:            manageVPUC,
		listGroupsUC:    listGroupsUC,
		managePermsUC:   managePermsUC,
		groupFunnelRepo: gfRepo,
		userGroupRepo:   ugRepo,
	}
}

// Name returns the module identifier.
func (m *Module) Name() string { return "permission" }

// RegisterRoutes registers all permission HTTP routes.
func (m *Module) RegisterRoutes(router *gin.Engine, mw module.Middlewares) {
	m.handler.RegisterRoutes(router, m.pageHandler, mw.Auth, mw.Tenant, mw.RequirePermission)
}

// Resolver exposes the ResolvePermissionUseCase for use as a middleware checker.
func (m *Module) Resolver() *application.ResolvePermissionUseCase { return m.resolverUC }

// ViewProfileUC exposes the ManageViewProfilesUseCase for cross-module use.
func (m *Module) ViewProfileUC() *application.ManageViewProfilesUseCase { return m.vpUC }

// ListGroupsUseCase exposes ListGroups for cross-module use.
func (m *Module) ListGroupsUseCase() *application.ListGroupsUseCase { return m.listGroupsUC }

// ManagePermissionsUseCase exposes permissions CRUD for cross-module use.
func (m *Module) ManagePermissionsUseCase() *application.ManagePermissionsUseCase {
	return m.managePermsUC
}

// GroupFunnelRepo exposes the group-funnel repository for cross-module adapters
// (e.g. auth's load-balance overlap checker).
func (m *Module) GroupFunnelRepo() permdomain.GroupFunnelRepository {
	return m.groupFunnelRepo
}

// UserGroupRepo exposes the user-group repository for cross-module adapters
// (e.g. auth's LoadBalancePicker, which needs to enumerate group members).
func (m *Module) UserGroupRepo() permdomain.UserGroupRepository {
	return m.userGroupRepo
}

// SetAuditPublisher injeta o publisher de auditoria nos UCs do modulo
// que produzem eventos auditaveis (ManagePermissionsUseCase em F12
// Step 7 — `permission.changed`).
//
// Wired pelo composition root em cmd/api/main.go apos o audit.Module
// existir; nil-safe (UC cai pra NoopPublisher quando nao injetado).
func (m *Module) SetAuditPublisher(p auditapp.Publisher) {
	if m.managePermsUC != nil {
		m.managePermsUC.SetAuditPublisher(p)
	}
}
