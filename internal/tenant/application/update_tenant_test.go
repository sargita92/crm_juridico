package application

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	auditdomain "github.com/sasrgita/crm-juridico/internal/audit/domain"
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

func TestUpdateTenantUseCase_PublishesAuditWithDiff(t *testing.T) {
	repo := newMockTenantRepo()
	tenant, _ := domain.NewTenant("id-1", "Old", domain.TenantTypePF, "111")
	repo.addTenant(tenant)

	spy := &spyAuditPublisher{}
	uc := NewUpdateTenantUseCase(repo)
	uc.SetAuditPublisher(spy)

	_, err := uc.Execute(ctxWithAdminClaims("admin-1", "admin@crm.com"), UpdateTenantInput{
		ID: "id-1", Name: "New", Type: "PJ", Document: "222",
	})
	require.NoError(t, err)

	require.Len(t, spy.calls, 1)
	call := spy.calls[0]
	assert.Equal(t, auditdomain.ActionTenantUpdated, call.Action)
	assert.Equal(t, "tenant", call.Entity)
	require.NotNil(t, call.EntityID)
	assert.Equal(t, "id-1", *call.EntityID)
	assert.Equal(t, "admin@crm.com", call.ActorEmail)

	require.NotNil(t, call.Metadata)
	diffRaw, ok := call.Metadata["diff"]
	require.True(t, ok, "metadata.diff ausente")
	diff, ok := diffRaw.(auditdomain.Metadata)
	require.True(t, ok, "metadata.diff deve ser auditdomain.Metadata")

	nameDiff, ok := diff["Name"].(map[string]any)
	require.True(t, ok, "diff[Name] deve existir")
	assert.Equal(t, "Old", nameDiff["antes"])
	assert.Equal(t, "New", nameDiff["depois"])

	typeDiff, ok := diff["Type"].(map[string]any)
	require.True(t, ok, "diff[Type] deve existir")
	assert.Equal(t, "PF", typeDiff["antes"])
	assert.Equal(t, "PJ", typeDiff["depois"])
}

func TestUpdateTenantUseCase_NoChange_DoesNotPublish(t *testing.T) {
	repo := newMockTenantRepo()
	tenant, _ := domain.NewTenant("id-1", "Same", domain.TenantTypePJ, "111")
	repo.addTenant(tenant)

	spy := &spyAuditPublisher{}
	uc := NewUpdateTenantUseCase(repo)
	uc.SetAuditPublisher(spy)

	// Sem mudanca real (mesmos valores)
	_, err := uc.Execute(ctxWithAdminClaims("a", "a@x.com"), UpdateTenantInput{
		ID: "id-1", Name: "Same", Type: "PJ", Document: "111",
	})
	require.NoError(t, err)
	assert.Empty(t, spy.calls, "publish nao deve ocorrer quando diff e vazio")
}

func TestUpdateTenantUseCase_RepoFailure_DoesNotPublish(t *testing.T) {
	repo := newMockTenantRepo()
	tenant, _ := domain.NewTenant("id-1", "Old", domain.TenantTypePJ, "111")
	repo.addTenant(tenant)
	repo.updateErr = errors.New("db down")

	spy := &spyAuditPublisher{}
	uc := NewUpdateTenantUseCase(repo)
	uc.SetAuditPublisher(spy)

	_, err := uc.Execute(ctxWithAdminClaims("a", "a@x.com"), UpdateTenantInput{
		ID: "id-1", Name: "New", Type: "PJ", Document: "222",
	})
	require.Error(t, err)
	assert.Empty(t, spy.calls)
}

func TestUpdateTenantUseCase_PublisherError_DoesNotAbort(t *testing.T) {
	repo := newMockTenantRepo()
	tenant, _ := domain.NewTenant("id-1", "Old", domain.TenantTypePJ, "111")
	repo.addTenant(tenant)

	spy := &spyAuditPublisher{publishErr: errors.New("audit failure")}
	uc := NewUpdateTenantUseCase(repo)
	uc.SetAuditPublisher(spy)

	out, err := uc.Execute(ctxWithAdminClaims("a", "a@x.com"), UpdateTenantInput{
		ID: "id-1", Name: "New", Type: "PJ", Document: "222",
	})
	require.NoError(t, err)
	require.NotNil(t, out)
}
