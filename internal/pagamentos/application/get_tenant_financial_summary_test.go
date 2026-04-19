package application_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/sasrgita/crm-juridico/internal/pagamentos/application"
	"github.com/sasrgita/crm-juridico/internal/pagamentos/domain"
)

func billingFor(tenantID string, plano domain.Plan) *domain.TenantBilling {
	valor := int64(5000)
	dia := uint8(10)
	start := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	cfg := domain.BillingConfig{
		Plano:              plano,
		ExibirPagamentos:   true,
		ValorCents:         &valor,
		DiaVencimento:      &dia,
		DataInicioCobranca: &start,
	}
	if plano == domain.PlanVitalicio || plano == domain.PlanExterno {
		cfg.ValorCents = nil
		cfg.DiaVencimento = nil
		cfg.DataInicioCobranca = nil
	}
	return &domain.TenantBilling{TenantID: tenantID, Active: true, Config: cfg}
}

func TestGetTenantFinancialSummary_Atrasado_HasPriority(t *testing.T) {
	repo := newFakeRepo()
	billing := newFakeBillingRepo()
	billing.byID["t1"] = billingFor("t1", domain.PlanMensal)
	repo.summaryResult = &domain.Summary{
		TotalPagoAnoCents:  10000,
		TotalPendenteCents: 2000,
		TotalAtrasadoCents: 3000,
		HasPendente:        true,
		HasAtrasado:        true,
	}
	uc := application.NewGetTenantFinancialSummary(repo, billing, &fakeClock{now: time.Now()})

	out, err := uc.Execute(context.Background(), "t1")
	require.NoError(t, err)
	assert.Equal(t, application.BadgeAtrasado, out.Badge)
	assert.Equal(t, int64(10000), out.TotalPagoAnoCents)
	assert.Equal(t, int64(3000), out.TotalAtrasadoCents)
}

func TestGetTenantFinancialSummary_Pendente(t *testing.T) {
	repo := newFakeRepo()
	billing := newFakeBillingRepo()
	billing.byID["t1"] = billingFor("t1", domain.PlanMensal)
	repo.summaryResult = &domain.Summary{
		TotalPendenteCents: 2000,
		HasPendente:        true,
	}
	uc := application.NewGetTenantFinancialSummary(repo, billing, &fakeClock{})
	out, err := uc.Execute(context.Background(), "t1")
	require.NoError(t, err)
	assert.Equal(t, application.BadgePendente, out.Badge)
}

func TestGetTenantFinancialSummary_EmDia(t *testing.T) {
	repo := newFakeRepo()
	billing := newFakeBillingRepo()
	billing.byID["t1"] = billingFor("t1", domain.PlanMensal)
	repo.summaryResult = &domain.Summary{}
	uc := application.NewGetTenantFinancialSummary(repo, billing, &fakeClock{})
	out, err := uc.Execute(context.Background(), "t1")
	require.NoError(t, err)
	assert.Equal(t, application.BadgeEmDia, out.Badge)
}

func TestGetTenantFinancialSummary_SemCobranca_VitalicioSemDebito(t *testing.T) {
	repo := newFakeRepo()
	billing := newFakeBillingRepo()
	billing.byID["t1"] = billingFor("t1", domain.PlanVitalicio)
	repo.summaryResult = &domain.Summary{}
	uc := application.NewGetTenantFinancialSummary(repo, billing, &fakeClock{})
	out, err := uc.Execute(context.Background(), "t1")
	require.NoError(t, err)
	assert.Equal(t, application.BadgeSemCobranca, out.Badge)
}

func TestGetTenantFinancialSummary_VitalicioComAvulsoAtrasado_MostraAtrasado(t *testing.T) {
	repo := newFakeRepo()
	billing := newFakeBillingRepo()
	billing.byID["t1"] = billingFor("t1", domain.PlanVitalicio)
	repo.summaryResult = &domain.Summary{
		TotalAtrasadoCents: 500,
		HasAtrasado:        true,
	}
	uc := application.NewGetTenantFinancialSummary(repo, billing, &fakeClock{})
	out, err := uc.Execute(context.Background(), "t1")
	require.NoError(t, err)
	assert.Equal(t, application.BadgeAtrasado, out.Badge)
}

func TestGetTenantFinancialSummary_BillingNotFound_PropagatesError(t *testing.T) {
	repo := newFakeRepo()
	billing := newFakeBillingRepo()
	uc := application.NewGetTenantFinancialSummary(repo, billing, &fakeClock{})
	_, err := uc.Execute(context.Background(), "missing")
	assert.ErrorIs(t, err, domain.ErrTenantNotFound)
}
