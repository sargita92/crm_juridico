package application

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	pagdomain "github.com/sasrgita/crm-juridico/internal/pagamentos/domain"
	"github.com/sasrgita/crm-juridico/internal/tenant/domain"
)

func TestUpdateTenantUseCase_Success(t *testing.T) {
	repo := newMockTenantRepo()
	tenant, _ := domain.NewTenant("id-1", "Nome Antigo", domain.TenantTypePF, "111")
	repo.addTenant(tenant)

	uc := NewUpdateTenantUseCase(repo)

	output, err := uc.Execute(context.Background(), UpdateTenantInput{
		ID:       "id-1",
		Name:     "Nome Novo",
		Type:     "PJ",
		Document: "222",
	})

	require.NoError(t, err)
	assert.Equal(t, "Nome Novo", output.Name)
	assert.Equal(t, "PJ", output.Type)
	assert.Equal(t, "222", output.Document)
}

func TestUpdateTenantUseCase_NotFound(t *testing.T) {
	repo := newMockTenantRepo()
	uc := NewUpdateTenantUseCase(repo)

	_, err := uc.Execute(context.Background(), UpdateTenantInput{
		ID:       "nonexistent",
		Name:     "Nome",
		Type:     "PJ",
		Document: "111",
	})

	assert.ErrorIs(t, err, domain.ErrTenantNotFound)
}

func TestUpdateTenantUseCase_EmptyName_ReturnsError(t *testing.T) {
	repo := newMockTenantRepo()
	tenant, _ := domain.NewTenant("id-1", "Nome", domain.TenantTypePJ, "111")
	repo.addTenant(tenant)

	uc := NewUpdateTenantUseCase(repo)

	_, err := uc.Execute(context.Background(), UpdateTenantInput{
		ID:       "id-1",
		Name:     "",
		Type:     "PJ",
		Document: "111",
	})

	assert.ErrorIs(t, err, domain.ErrTenantNameRequired)
}

func TestUpdateTenantUseCase_BillingMensal_PersistsFields(t *testing.T) {
	repo := newMockTenantRepo()
	tenant, _ := domain.NewTenant("id-1", "Nome", domain.TenantTypePJ, "111")
	repo.addTenant(tenant)

	uc := NewUpdateTenantUseCase(repo)

	valor := int64(5000)
	dia := uint8(10)
	start := time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC)

	output, err := uc.Execute(context.Background(), UpdateTenantInput{
		ID: "id-1", Name: "Nome", Type: "PJ", Document: "111",
		Billing: &UpdateTenantBilling{
			Plano:              "mensal",
			ValorCobrancaCents: &valor,
			DiaVencimento:      &dia,
			DataInicioCobranca: &start,
			ExibirPagamentos:   true,
		},
	})
	require.NoError(t, err)
	assert.Equal(t, "mensal", output.Plano)
	require.NotNil(t, output.ValorCobrancaCents)
	assert.Equal(t, int64(5000), *output.ValorCobrancaCents)
	require.NotNil(t, output.DiaVencimento)
	assert.Equal(t, uint8(10), *output.DiaVencimento)
	require.NotNil(t, output.DataInicioCobranca)
	assert.True(t, output.DataInicioCobranca.Equal(start))
	assert.True(t, output.ExibirPagamentos)
}

func TestUpdateTenantUseCase_BillingMensal_SemValor_ReturnsError(t *testing.T) {
	repo := newMockTenantRepo()
	tenant, _ := domain.NewTenant("id-1", "Nome", domain.TenantTypePJ, "111")
	repo.addTenant(tenant)

	uc := NewUpdateTenantUseCase(repo)

	dia := uint8(10)
	start := time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC)

	_, err := uc.Execute(context.Background(), UpdateTenantInput{
		ID: "id-1", Name: "Nome", Type: "PJ", Document: "111",
		Billing: &UpdateTenantBilling{
			Plano:              "mensal",
			DiaVencimento:      &dia,
			DataInicioCobranca: &start,
			ExibirPagamentos:   true,
		},
	})
	assert.ErrorIs(t, err, pagdomain.ErrValorInvalido)
}

func TestUpdateTenantUseCase_BillingVitalicio_IgnoresValor(t *testing.T) {
	repo := newMockTenantRepo()
	tenant, _ := domain.NewTenant("id-1", "Nome", domain.TenantTypePJ, "111")
	// pre-existing billing that should be wiped by switching to vitalicio
	valorOld := int64(9999)
	diaOld := uint8(5)
	startOld := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	tenant.SetBillingConfig("mensal", &valorOld, &diaOld, &startOld, true)
	repo.addTenant(tenant)

	uc := NewUpdateTenantUseCase(repo)

	valor := int64(123)
	output, err := uc.Execute(context.Background(), UpdateTenantInput{
		ID: "id-1", Name: "Nome", Type: "PJ", Document: "111",
		Billing: &UpdateTenantBilling{
			Plano:              "vitalicio",
			ValorCobrancaCents: &valor,
			ExibirPagamentos:   false,
		},
	})
	require.NoError(t, err)
	assert.Equal(t, "vitalicio", output.Plano)
	assert.Nil(t, output.ValorCobrancaCents)
	assert.Nil(t, output.DiaVencimento)
	assert.Nil(t, output.DataInicioCobranca)
	assert.False(t, output.ExibirPagamentos)
}

func TestUpdateTenantUseCase_BillingInvalidPlano_ReturnsError(t *testing.T) {
	repo := newMockTenantRepo()
	tenant, _ := domain.NewTenant("id-1", "Nome", domain.TenantTypePJ, "111")
	repo.addTenant(tenant)

	uc := NewUpdateTenantUseCase(repo)

	_, err := uc.Execute(context.Background(), UpdateTenantInput{
		ID: "id-1", Name: "Nome", Type: "PJ", Document: "111",
		Billing: &UpdateTenantBilling{Plano: "outro"},
	})
	assert.ErrorIs(t, err, pagdomain.ErrInvalidPlano)
}

func TestUpdateTenantUseCase_DuplicateDocument_ReturnsError(t *testing.T) {
	repo := newMockTenantRepo()
	t1, _ := domain.NewTenant("id-1", "A", domain.TenantTypePJ, "111")
	t2, _ := domain.NewTenant("id-2", "B", domain.TenantTypePJ, "222")
	repo.addTenant(t1)
	repo.addTenant(t2)

	uc := NewUpdateTenantUseCase(repo)

	_, err := uc.Execute(context.Background(), UpdateTenantInput{
		ID:       "id-2",
		Name:     "B",
		Type:     "PJ",
		Document: "111",
	})

	assert.ErrorIs(t, err, domain.ErrTenantDocumentExists)
}
