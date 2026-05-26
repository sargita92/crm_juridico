package dashboard

import (
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"gorm.io/gorm"

	authdomain "github.com/sasrgita/crm-juridico/internal/auth/domain"
	"github.com/sasrgita/crm-juridico/internal/dashboard/application"
	"github.com/sasrgita/crm-juridico/internal/dashboard/infrastructure"
	dashboardhttp "github.com/sasrgita/crm-juridico/internal/dashboard/interfaces/http"
	pagamentosDomain "github.com/sasrgita/crm-juridico/internal/pagamentos/domain"
	"github.com/sasrgita/crm-juridico/internal/shared/module"
)

type Config struct {
	ServiceChecks []infrastructure.ServiceCheck // bloco 4 admin
}

type Module struct {
	tenantUC *application.GetTenantDashboard
	adminUC  *application.GetAdminDashboard
	handler  *dashboardhttp.Handler
	log      *zap.Logger
}

// NewModule monta o módulo dashboard. Não recebe UserRepository — GormUserLookup
// usa diretamente o db (pega só o name por ID, não precisa do contrato de domínio).
func NewModule(
	db *gorm.DB,
	userTenants authdomain.UserTenantRepository,
	payments pagamentosDomain.PaymentRepository,
	log *zap.Logger,
	cfg Config,
) *Module {
	if log == nil {
		log = zap.NewNop()
	}

	tenantRepo := infrastructure.NewGormTenantStatsRepo(db)
	adminRepo := infrastructure.NewGormAdminStatsRepo(db, payments)
	infraProv := infrastructure.NewPrometheusStatsProvider(nil, cfg.ServiceChecks)
	ul := infrastructure.NewGormUserLookup(db)
	clk := application.SystemClock{}

	tenantUC := application.NewGetTenantDashboard(tenantRepo, ul, clk)
	adminUC := application.NewGetAdminDashboard(adminRepo, infraProv, clk)

	handler := dashboardhttp.NewHandler(tenantUC, adminUC, userTenants, ul, log)

	return &Module{
		tenantUC: tenantUC,
		adminUC:  adminUC,
		handler:  handler,
		log:      log,
	}
}

func (m *Module) Name() string { return "dashboard" }

func (m *Module) RegisterRoutes(router *gin.Engine, mw module.Middlewares) {
	m.handler.RegisterRoutes(router, mw)
}
