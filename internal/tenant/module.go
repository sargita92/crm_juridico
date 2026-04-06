package tenant

import (
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/sasrgita/crm-juridico/internal/tenant/application"
	"github.com/sasrgita/crm-juridico/internal/tenant/domain"
	"github.com/sasrgita/crm-juridico/internal/tenant/infrastructure"
	tenanthttp "github.com/sasrgita/crm-juridico/internal/tenant/interfaces/http"
)

type Module struct {
	tenantRepo domain.TenantRepository
	handler    *tenanthttp.Handler
}

func NewModule(db *gorm.DB) *Module {
	tenantRepo := infrastructure.NewGormTenantRepository(db)
	blockHistoryRepo := infrastructure.NewGormBlockHistoryRepository(db)

	createUC := application.NewCreateTenantUseCase(tenantRepo)
	listUC := application.NewListTenantsUseCase(tenantRepo)
	getUC := application.NewGetTenantUseCase(tenantRepo)
	updateUC := application.NewUpdateTenantUseCase(tenantRepo)
	deactivateUC := application.NewDeactivateTenantUseCase(tenantRepo)
	blockUC := application.NewBlockTenantUseCase(tenantRepo, blockHistoryRepo)
	unblockUC := application.NewUnblockTenantUseCase(tenantRepo, blockHistoryRepo)
	getBlockHistoryUC := application.NewGetBlockHistoryUseCase(tenantRepo, blockHistoryRepo)

	handler := tenanthttp.NewHandler(
		createUC, listUC, getUC,
		updateUC, deactivateUC,
		blockUC, unblockUC,
		getBlockHistoryUC,
	)

	return &Module{
		tenantRepo: tenantRepo,
		handler:    handler,
	}
}

func (m *Module) Name() string { return "tenant" }

func (m *Module) RegisterRoutes(router *gin.Engine, authMw, adminMw gin.HandlerFunc) {
	m.handler.RegisterRoutes(router, authMw, adminMw)
}

func (m *Module) TenantRepo() domain.TenantRepository {
	return m.tenantRepo
}
