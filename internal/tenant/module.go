package tenant

import (
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	auditapp "github.com/sasrgita/crm-juridico/internal/audit/application"
	"github.com/sasrgita/crm-juridico/internal/shared/module"
	"github.com/sasrgita/crm-juridico/internal/tenant/application"
	"github.com/sasrgita/crm-juridico/internal/tenant/domain"
	"github.com/sasrgita/crm-juridico/internal/tenant/infrastructure"
	tenanthttp "github.com/sasrgita/crm-juridico/internal/tenant/interfaces/http"
)

type Module struct {
	tenantRepo domain.TenantRepository
	handler    *tenanthttp.Handler

	// Use cases referenciados aqui apenas para SetAuditPublisher (F12 Step 6).
	// O handler ja recebe os mesmos ponteiros — a injecao do publisher ocorre
	// nos UCs, nao no handler.
	createUC     *application.CreateTenantUseCase
	updateUC     *application.UpdateTenantUseCase
	deactivateUC *application.DeactivateTenantUseCase
	blockUC      *application.BlockTenantUseCase
	unblockUC    *application.UnblockTenantUseCase
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
		tenantRepo:   tenantRepo,
		handler:      handler,
		createUC:     createUC,
		updateUC:     updateUC,
		deactivateUC: deactivateUC,
		blockUC:      blockUC,
		unblockUC:    unblockUC,
	}
}

func (m *Module) Name() string { return "tenant" }

func (m *Module) RegisterRoutes(router *gin.Engine, mw module.Middlewares) {
	m.handler.RegisterRoutes(router, mw.Auth, mw.Admin)
}

func (m *Module) TenantRepo() domain.TenantRepository {
	return m.tenantRepo
}

// SetAuditPublisher propaga o publisher de auditoria para todos os UCs que
// produzem eventos auditaveis (create/update/deactivate/block/unblock).
// Deve ser chamado pelo composition root apos audit.NewModule existir.
//
// `publisher` pode ser nil — cada UC usa NoopPublisher como fallback,
// mantendo o codigo sem nil-checks. Decisao F12 Step 6.
func (m *Module) SetAuditPublisher(publisher auditapp.Publisher) {
	m.createUC.SetAuditPublisher(publisher)
	m.updateUC.SetAuditPublisher(publisher)
	m.deactivateUC.SetAuditPublisher(publisher)
	m.blockUC.SetAuditPublisher(publisher)
	m.unblockUC.SetAuditPublisher(publisher)
}
