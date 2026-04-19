package application

import (
	"context"

	"github.com/sasrgita/crm-juridico/internal/pagamentos/domain"
)

type FinancialBadge string

const (
	BadgeEmDia       FinancialBadge = "em_dia"
	BadgePendente    FinancialBadge = "pendente"
	BadgeAtrasado    FinancialBadge = "atrasado"
	BadgeSemCobranca FinancialBadge = "sem_cobranca"
)

type FinancialSummaryOutput struct {
	Badge              FinancialBadge
	TotalPagoAnoCents  int64
	TotalPendenteCents int64
	TotalAtrasadoCents int64
}

type GetTenantFinancialSummary struct {
	payments domain.PaymentRepository
	billing  domain.TenantBillingRepository
	clock    Clock
}

func NewGetTenantFinancialSummary(payments domain.PaymentRepository, billing domain.TenantBillingRepository, clock Clock) *GetTenantFinancialSummary {
	return &GetTenantFinancialSummary{payments: payments, billing: billing, clock: clock}
}

func (uc *GetTenantFinancialSummary) Execute(ctx context.Context, tenantID string) (*FinancialSummaryOutput, error) {
	tb, err := uc.billing.GetByID(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	sum, err := uc.payments.Summary(ctx, tenantID, uc.clock.Now())
	if err != nil {
		return nil, err
	}
	out := &FinancialSummaryOutput{
		TotalPagoAnoCents:  sum.TotalPagoAnoCents,
		TotalPendenteCents: sum.TotalPendenteCents,
		TotalAtrasadoCents: sum.TotalAtrasadoCents,
	}
	switch {
	case !tb.Config.GenerateRecurring() && sum.TotalPendenteCents == 0 && sum.TotalAtrasadoCents == 0:
		out.Badge = BadgeSemCobranca
	case sum.HasAtrasado:
		out.Badge = BadgeAtrasado
	case sum.HasPendente:
		out.Badge = BadgePendente
	default:
		out.Badge = BadgeEmDia
	}
	return out, nil
}
