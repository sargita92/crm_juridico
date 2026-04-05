package application

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

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
