package pagamentos

import (
	"context"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"gorm.io/gorm"

	"github.com/sasrgita/crm-juridico/internal/pagamentos/application"
	"github.com/sasrgita/crm-juridico/internal/pagamentos/domain"
	"github.com/sasrgita/crm-juridico/internal/pagamentos/infrastructure"
	pagamentoshttp "github.com/sasrgita/crm-juridico/internal/pagamentos/interfaces/http"
	"github.com/sasrgita/crm-juridico/internal/shared/module"
)

// SetPermissionChecker liga o resolver de permissoes ao middleware do portal
// tenant. Chamado no wiring do cmd/api apos a criacao dos modulos auth
// e permission (quebra do ciclo de dependencies).
func (m *Module) SetPermissionChecker(perm pagamentoshttp.PermissionChecker) {
	checker := pagamentoshttp.NewPortalAccessChecker(m.billingRepo, perm)
	m.handler.SetPortalMiddleware(checker.Middleware())
}

type Config struct {
	CronSpec  string
	GraceDays int
	Location  *time.Location
}

type Module struct {
	paymentRepo  *infrastructure.GormPaymentRepository
	billingRepo  *infrastructure.GormTenantBillingRepository
	summaryUC    *application.GetTenantFinancialSummary
	listTenantUC *application.ListTenantPayments
	listAllUC    *application.ListAllPayments
	registerUC   *application.RegisterManualPayment
	payUC        *application.MarkPaymentAsPaid
	cancelUC     *application.CancelPayment
	scheduler    *infrastructure.BillingScheduler
	handler      *pagamentoshttp.Handler
	log          *zap.Logger
}

func NewModule(db *gorm.DB, log *zap.Logger, cfg Config) *Module {
	if log == nil {
		log = zap.NewNop()
	}
	if cfg.GraceDays < 0 {
		cfg.GraceDays = 0
	}
	if cfg.Location == nil {
		loc, err := time.LoadLocation("America/Sao_Paulo")
		if err != nil {
			loc = time.UTC
		}
		cfg.Location = loc
	}

	paymentRepo := infrastructure.NewGormPaymentRepository(db)
	billingRepo := infrastructure.NewGormTenantBillingRepository(db)
	cal := domain.NewBrazilHolidayCalendar()
	clk := application.SystemClock{}
	idGen := application.UUIDGenerator{}

	summaryUC := application.NewGetTenantFinancialSummary(paymentRepo, billingRepo, clk)
	listTenantUC := application.NewListTenantPayments(paymentRepo)
	listAllUC := application.NewListAllPayments(paymentRepo)
	registerUC := application.NewRegisterManualPayment(paymentRepo, idGen, clk)
	payUC := application.NewMarkPaymentAsPaid(paymentRepo, clk)
	cancelUC := application.NewCancelPayment(paymentRepo, clk)
	genUC := application.NewGenerateRecurringPayments(paymentRepo, billingRepo, cal, idGen, clk)
	refUC := application.NewRefreshOverdueStatuses(paymentRepo, cal, cfg.GraceDays, clk)
	scheduler := infrastructure.NewBillingScheduler(cfg.CronSpec, genUC, refUC, log, cfg.Location)

	handler := pagamentoshttp.NewHandler(
		listTenantUC, listAllUC, summaryUC, registerUC, payUC, cancelUC, billingRepo, paymentRepo, log,
	)

	return &Module{
		paymentRepo:  paymentRepo,
		billingRepo:  billingRepo,
		summaryUC:    summaryUC,
		listTenantUC: listTenantUC,
		listAllUC:    listAllUC,
		registerUC:   registerUC,
		payUC:        payUC,
		cancelUC:     cancelUC,
		scheduler:    scheduler,
		handler:      handler,
		log:          log,
	}
}

func (m *Module) Name() string { return "pagamentos" }

func (m *Module) RegisterRoutes(router *gin.Engine, mw module.Middlewares) {
	m.handler.RegisterRoutes(router, mw)
}

func (m *Module) StartScheduler() error { return m.scheduler.Start() }
func (m *Module) StopScheduler()        { m.scheduler.Stop() }

// Summary expõe o UC de resumo financeiro para outros módulos (ex: aba
// Pagamentos no detalhe do tenant em tenant/).
func (m *Module) Summary() *application.GetTenantFinancialSummary { return m.summaryUC }

// PaymentRepo expõe o repositório para uso cruzado (dashboard admin).
func (m *Module) PaymentRepo() domain.PaymentRepository { return m.paymentRepo }

// ShowsPortalForTenant retorna se o tenant tem billing config ativa para
// exibir a aba Pagamentos. Usado pelo middleware SidebarFlags para esconder
// o link na sidebar quando o portal não está disponível. UX-only — autorização
// continua no PortalAccessChecker.
func (m *Module) ShowsPortalForTenant(ctx context.Context, tenantID string) bool {
	tb, err := m.billingRepo.GetByID(ctx, tenantID)
	if err != nil {
		return false
	}
	return tb.Config.ShowsPortalMenu()
}
