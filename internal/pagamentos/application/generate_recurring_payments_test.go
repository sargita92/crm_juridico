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

func makeBilling(tenantID string, plano domain.Plan, dia uint8, start time.Time, valorCents int64) domain.TenantBilling {
	cfg := domain.BillingConfig{
		Plano:              plano,
		ValorCents:         &valorCents,
		DiaVencimento:      &dia,
		DataInicioCobranca: &start,
		ExibirPagamentos:   true,
	}
	return domain.TenantBilling{TenantID: tenantID, Active: true, Config: cfg}
}

func TestGenerate_FirstCompetenciaForMensal(t *testing.T) {
	cal := domain.NewBrazilHolidayCalendar()
	repo := newFakeRepo()
	billing := newFakeBillingRepo()
	billing.listed = []domain.TenantBilling{
		makeBilling("t1", domain.PlanMensal, 10, time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC), 50000),
	}
	clk := &fakeClock{now: time.Date(2026, 4, 15, 0, 0, 0, 0, time.UTC)}
	uc := application.NewGenerateRecurringPayments(repo, billing, cal, &fakeIDGen{id: "p1"}, clk)

	n, err := uc.Execute(context.Background())
	require.NoError(t, err)
	assert.Equal(t, 1, n)
	p := repo.payments["p1"]
	require.NotNil(t, p)
	assert.Equal(t, domain.TypeRecorrente, p.Tipo)
	assert.Equal(t, int64(50000), p.ValorCents)
	assert.Equal(t, time.Date(2026, 4, 10, 0, 0, 0, 0, time.UTC), p.DataVencimento)
	require.NotNil(t, p.Competencia)
	assert.Equal(t, time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC), *p.Competencia)
}

func TestGenerate_Idempotent(t *testing.T) {
	cal := domain.NewBrazilHolidayCalendar()
	repo := newFakeRepo()
	billing := newFakeBillingRepo()
	billing.listed = []domain.TenantBilling{
		makeBilling("t1", domain.PlanMensal, 10, time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC), 50000),
	}
	clk := &fakeClock{now: time.Date(2026, 4, 15, 0, 0, 0, 0, time.UTC)}
	uc := application.NewGenerateRecurringPayments(repo, billing, cal, &fakeIDGen{id: "p1"}, clk)

	n1, _ := uc.Execute(context.Background())
	n2, err := uc.Execute(context.Background())
	require.NoError(t, err)
	assert.Equal(t, 1, n1)
	assert.Equal(t, 0, n2)
}

func TestGenerate_AnualUmPorAno(t *testing.T) {
	cal := domain.NewBrazilHolidayCalendar()
	repo := newFakeRepo()
	billing := newFakeBillingRepo()
	// start: abril 2025 → competências esperadas: abr/2025, abr/2026
	billing.listed = []domain.TenantBilling{
		makeBilling("t1", domain.PlanAnual, 10, time.Date(2025, 4, 1, 0, 0, 0, 0, time.UTC), 600000),
	}
	clk := &fakeClock{now: time.Date(2026, 4, 15, 0, 0, 0, 0, time.UTC)}
	// use an id generator that yields unique ids
	idGen := &seqIDGen{prefix: "a-"}
	uc := application.NewGenerateRecurringPayments(repo, billing, cal, idGen, clk)

	n, err := uc.Execute(context.Background())
	require.NoError(t, err)
	assert.Equal(t, 2, n)
}

func TestGenerate_PulaDataInicioFutura(t *testing.T) {
	cal := domain.NewBrazilHolidayCalendar()
	repo := newFakeRepo()
	billing := newFakeBillingRepo()
	billing.listed = []domain.TenantBilling{
		makeBilling("t1", domain.PlanMensal, 10, time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC), 50000),
	}
	clk := &fakeClock{now: time.Date(2026, 4, 15, 0, 0, 0, 0, time.UTC)}
	uc := application.NewGenerateRecurringPayments(repo, billing, cal, &fakeIDGen{id: "p1"}, clk)
	n, err := uc.Execute(context.Background())
	require.NoError(t, err)
	assert.Equal(t, 0, n)
}

func TestGenerate_VencimentoEmFimDeSemana_ProrrogaParaSegunda(t *testing.T) {
	cal := domain.NewBrazilHolidayCalendar()
	repo := newFakeRepo()
	billing := newFakeBillingRepo()
	// dia 10 em maio/2026 cai num domingo → esperado próximo dia útil = 11/05 (segunda)
	billing.listed = []domain.TenantBilling{
		makeBilling("t1", domain.PlanMensal, 10, time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC), 50000),
	}
	clk := &fakeClock{now: time.Date(2026, 5, 15, 0, 0, 0, 0, time.UTC)}
	uc := application.NewGenerateRecurringPayments(repo, billing, cal, &fakeIDGen{id: "p1"}, clk)
	n, err := uc.Execute(context.Background())
	require.NoError(t, err)
	assert.Equal(t, 1, n)
	p := repo.payments["p1"]
	assert.Equal(t, time.Monday, p.DataVencimento.Weekday())
	assert.Equal(t, time.Date(2026, 5, 11, 0, 0, 0, 0, time.UTC), p.DataVencimento)
}

func TestGenerate_SkipsIncompleteBillingConfig(t *testing.T) {
	cal := domain.NewBrazilHolidayCalendar()
	repo := newFakeRepo()
	billing := newFakeBillingRepo()
	// Tenant sem ValorCents → skip
	dia := uint8(10)
	start := time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC)
	billing.listed = []domain.TenantBilling{
		{TenantID: "t1", Active: true, Config: domain.BillingConfig{
			Plano: domain.PlanMensal, DiaVencimento: &dia, DataInicioCobranca: &start,
		}},
	}
	uc := application.NewGenerateRecurringPayments(repo, billing, cal, &fakeIDGen{id: "p1"}, &fakeClock{now: time.Date(2026, 4, 15, 0, 0, 0, 0, time.UTC)})
	n, err := uc.Execute(context.Background())
	require.NoError(t, err)
	assert.Equal(t, 0, n)
}
