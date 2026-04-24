package application

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	auditdomain "github.com/sasrgita/crm-juridico/internal/audit/domain"
	"github.com/sasrgita/crm-juridico/internal/tenant/domain"
)

func TestUnblockTenantUseCase_Success(t *testing.T) {
	repo := newMockTenantRepo()
	historyRepo := newMockBlockHistoryRepo()
	tenant, _ := domain.NewTenant("id-1", "Escritório", domain.TenantTypePJ, "111")
	_ = tenant.Block("Inadimplência")
	repo.addTenant(tenant)

	uc := NewUnblockTenantUseCase(repo, historyRepo)

	err := uc.Execute(context.Background(), UnblockTenantInput{ID: "id-1", Reason: "Pagamento regularizado", PerformedBy: "admin-1"})
	require.NoError(t, err)

	found, _ := repo.FindByID(context.Background(), "id-1")
	assert.Equal(t, domain.TenantStatusActive, found.Status)
	assert.Empty(t, found.BlockReason)
}

func TestUnblockTenantUseCase_SavesHistoryEntry(t *testing.T) {
	repo := newMockTenantRepo()
	historyRepo := newMockBlockHistoryRepo()
	tenant, _ := domain.NewTenant("id-1", "Escritório", domain.TenantTypePJ, "111")
	_ = tenant.Block("Inadimplência")
	repo.addTenant(tenant)

	uc := NewUnblockTenantUseCase(repo, historyRepo)

	err := uc.Execute(context.Background(), UnblockTenantInput{ID: "id-1", Reason: "Regularizado", PerformedBy: "admin-2"})
	require.NoError(t, err)

	require.Len(t, historyRepo.entries, 1)
	assert.Equal(t, domain.BlockActionUnblock, historyRepo.entries[0].Action)
	assert.Equal(t, "Regularizado", historyRepo.entries[0].Reason)
	assert.Equal(t, "admin-2", historyRepo.entries[0].PerformedBy)
	assert.Equal(t, "id-1", historyRepo.entries[0].TenantID)
}

func TestUnblockTenantUseCase_NotFound(t *testing.T) {
	repo := newMockTenantRepo()
	historyRepo := newMockBlockHistoryRepo()
	uc := NewUnblockTenantUseCase(repo, historyRepo)

	err := uc.Execute(context.Background(), UnblockTenantInput{ID: "nonexistent", Reason: "Motivo", PerformedBy: "admin-1"})
	assert.ErrorIs(t, err, domain.ErrTenantNotFound)
}

func TestUnblockTenantUseCase_EmptyReason(t *testing.T) {
	repo := newMockTenantRepo()
	historyRepo := newMockBlockHistoryRepo()
	tenant, _ := domain.NewTenant("id-1", "Escritório", domain.TenantTypePJ, "111")
	_ = tenant.Block("Motivo")
	repo.addTenant(tenant)

	uc := NewUnblockTenantUseCase(repo, historyRepo)

	err := uc.Execute(context.Background(), UnblockTenantInput{ID: "id-1", Reason: "", PerformedBy: "admin-1"})
	assert.ErrorIs(t, err, domain.ErrUnblockReasonRequired)
}

func TestUnblockTenantUseCase_NotBlocked(t *testing.T) {
	repo := newMockTenantRepo()
	historyRepo := newMockBlockHistoryRepo()
	tenant, _ := domain.NewTenant("id-1", "Escritório", domain.TenantTypePJ, "111")
	repo.addTenant(tenant)

	uc := NewUnblockTenantUseCase(repo, historyRepo)

	err := uc.Execute(context.Background(), UnblockTenantInput{ID: "id-1", Reason: "Motivo", PerformedBy: "admin-1"})
	assert.ErrorIs(t, err, domain.ErrTenantNotBlocked)
}

func TestUnblockTenantUseCase_EmptyPerformedBy(t *testing.T) {
	repo := newMockTenantRepo()
	historyRepo := newMockBlockHistoryRepo()
	tenant, _ := domain.NewTenant("id-1", "Escritório", domain.TenantTypePJ, "111")
	_ = tenant.Block("Motivo")
	repo.addTenant(tenant)

	uc := NewUnblockTenantUseCase(repo, historyRepo)

	err := uc.Execute(context.Background(), UnblockTenantInput{ID: "id-1", Reason: "Motivo", PerformedBy: ""})
	assert.ErrorIs(t, err, domain.ErrHistoryPerformedByRequired)
}

func TestUnblockTenantUseCase_PublishesAuditOnSuccess(t *testing.T) {
	repo := newMockTenantRepo()
	historyRepo := newMockBlockHistoryRepo()
	tenant, _ := domain.NewTenant("id-1", "X", domain.TenantTypePJ, "111")
	_ = tenant.Block("Motivo antigo")
	repo.addTenant(tenant)

	spy := &spyAuditPublisher{}
	uc := NewUnblockTenantUseCase(repo, historyRepo)
	uc.SetAuditPublisher(spy)

	err := uc.Execute(ctxWithAdminClaims("admin-1", "admin@crm.com"), UnblockTenantInput{
		ID: "id-1", Reason: "Pago", PerformedBy: "admin-1",
	})
	require.NoError(t, err)

	require.Len(t, spy.calls, 1)
	call := spy.calls[0]
	assert.Equal(t, auditdomain.ActionTenantUnblocked, call.Action)
	assert.Equal(t, "tenant", call.Entity)
	require.NotNil(t, call.EntityID)
	assert.Equal(t, "id-1", *call.EntityID)
	require.NotNil(t, call.TenantID)
	assert.Equal(t, "id-1", *call.TenantID)
	assert.Equal(t, "admin@crm.com", call.ActorEmail)
	require.NotNil(t, call.UserID)
	assert.Equal(t, "admin-1", *call.UserID)
	require.NotNil(t, call.Metadata)
	assert.Equal(t, "Pago", call.Metadata["reason"])
}

func TestUnblockTenantUseCase_RepoFailure_DoesNotPublish(t *testing.T) {
	repo := newMockTenantRepo()
	historyRepo := newMockBlockHistoryRepo()
	tenant, _ := domain.NewTenant("id-1", "X", domain.TenantTypePJ, "111")
	_ = tenant.Block("Motivo")
	repo.addTenant(tenant)
	repo.updateErr = errors.New("db down")

	spy := &spyAuditPublisher{}
	uc := NewUnblockTenantUseCase(repo, historyRepo)
	uc.SetAuditPublisher(spy)

	err := uc.Execute(ctxWithAdminClaims("a", "a@x.com"), UnblockTenantInput{
		ID: "id-1", Reason: "Motivo", PerformedBy: "a",
	})
	require.Error(t, err)
	assert.Empty(t, spy.calls)
}

func TestUnblockTenantUseCase_PublisherError_DoesNotAbort(t *testing.T) {
	repo := newMockTenantRepo()
	historyRepo := newMockBlockHistoryRepo()
	tenant, _ := domain.NewTenant("id-1", "X", domain.TenantTypePJ, "111")
	_ = tenant.Block("Motivo")
	repo.addTenant(tenant)

	spy := &spyAuditPublisher{publishErr: errors.New("audit fail")}
	uc := NewUnblockTenantUseCase(repo, historyRepo)
	uc.SetAuditPublisher(spy)

	err := uc.Execute(ctxWithAdminClaims("a", "a@x.com"), UnblockTenantInput{
		ID: "id-1", Reason: "Motivo", PerformedBy: "a",
	})
	require.NoError(t, err)
}
