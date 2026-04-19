package http

import (
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"github.com/sasrgita/crm-juridico/internal/pagamentos/application"
	"github.com/sasrgita/crm-juridico/internal/pagamentos/domain"
	"github.com/sasrgita/crm-juridico/internal/shared/module"
)

// Handler orquestra os endpoints HTTP do modulo pagamentos — admin global
// (/admin/pagamentos), aba por tenant (/admin/tenants/:id/pagamentos) e
// portal do tenant (/pagamentos). Os metodos das rotas ficam em arquivos
// complementares (admin_routes.go / tenant_routes.go).
type Handler struct {
	listTenant *application.ListTenantPayments
	listAll    *application.ListAllPayments
	summary    *application.GetTenantFinancialSummary
	register   *application.RegisterManualPayment
	pay        *application.MarkPaymentAsPaid
	cancel     *application.CancelPayment
	billing    domain.TenantBillingRepository
	log        *zap.Logger
}

func NewHandler(
	listTenant *application.ListTenantPayments,
	listAll *application.ListAllPayments,
	summary *application.GetTenantFinancialSummary,
	register *application.RegisterManualPayment,
	pay *application.MarkPaymentAsPaid,
	cancel *application.CancelPayment,
	billing domain.TenantBillingRepository,
	log *zap.Logger,
) *Handler {
	if log == nil {
		log = zap.NewNop()
	}
	return &Handler{
		listTenant: listTenant,
		listAll:    listAll,
		summary:    summary,
		register:   register,
		pay:        pay,
		cancel:     cancel,
		billing:    billing,
		log:        log,
	}
}

// RegisterRoutes monta as rotas admin + portal tenant. As funcoes concretas
// ficam em admin_routes.go e tenant_routes.go.
func (h *Handler) RegisterRoutes(router *gin.Engine, mw module.Middlewares) {
	h.registerAdminRoutes(router, mw)
	h.registerTenantPortalRoutes(router, mw)
}
