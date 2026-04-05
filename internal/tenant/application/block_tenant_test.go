package application

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/sasrgita/crm-juridico/internal/tenant/domain"
)

func TestBlockTenantUseCase_Success(t *testing.T) {
	repo := newMockTenantRepo()
	historyRepo := newMockBlockHistoryRepo()
	tenant, _ := domain.NewTenant("id-1", "Escritório", domain.TenantTypePJ, "111")
	repo.addTenant(tenant)

	uc := NewBlockTenantUseCase(repo, historyRepo)

	err := uc.Execute(context.Background(), BlockTenantInput{ID: "id-1", Reason: "Inadimplência", PerformedBy: "admin-1"})
	require.NoError(t, err)

	found, _ := repo.FindByID(context.Background(), "id-1")
	assert.Equal(t, domain.TenantStatusBlocked, found.Status)
	assert.Equal(t, "Inadimplência", found.BlockReason)
}

func TestBlockTenantUseCase_SavesHistoryEntry(t *testing.T) {
	repo := newMockTenantRepo()
	historyRepo := newMockBlockHistoryRepo()
	tenant, _ := domain.NewTenant("id-1", "Escritório", domain.TenantTypePJ, "111")
	repo.addTenant(tenant)

	uc := NewBlockTenantUseCase(repo, historyRepo)

	err := uc.Execute(context.Background(), BlockTenantInput{ID: "id-1", Reason: "Inadimplência", PerformedBy: "admin-1"})
	require.NoError(t, err)

	require.Len(t, historyRepo.entries, 1)
	assert.Equal(t, domain.BlockActionBlock, historyRepo.entries[0].Action)
	assert.Equal(t, "Inadimplência", historyRepo.entries[0].Reason)
	assert.Equal(t, "admin-1", historyRepo.entries[0].PerformedBy)
	assert.Equal(t, "id-1", historyRepo.entries[0].TenantID)
}

func TestBlockTenantUseCase_NotFound(t *testing.T) {
	repo := newMockTenantRepo()
	historyRepo := newMockBlockHistoryRepo()
	uc := NewBlockTenantUseCase(repo, historyRepo)

	err := uc.Execute(context.Background(), BlockTenantInput{ID: "nonexistent", Reason: "Motivo", PerformedBy: "admin-1"})
	assert.ErrorIs(t, err, domain.ErrTenantNotFound)
}

func TestBlockTenantUseCase_EmptyReason(t *testing.T) {
	repo := newMockTenantRepo()
	historyRepo := newMockBlockHistoryRepo()
	tenant, _ := domain.NewTenant("id-1", "Escritório", domain.TenantTypePJ, "111")
	repo.addTenant(tenant)

	uc := NewBlockTenantUseCase(repo, historyRepo)

	err := uc.Execute(context.Background(), BlockTenantInput{ID: "id-1", Reason: "", PerformedBy: "admin-1"})
	assert.ErrorIs(t, err, domain.ErrBlockReasonRequired)
}

func TestBlockTenantUseCase_AlreadyBlocked(t *testing.T) {
	repo := newMockTenantRepo()
	historyRepo := newMockBlockHistoryRepo()
	tenant, _ := domain.NewTenant("id-1", "Escritório", domain.TenantTypePJ, "111")
	_ = tenant.Block("Motivo anterior")
	repo.addTenant(tenant)

	uc := NewBlockTenantUseCase(repo, historyRepo)

	err := uc.Execute(context.Background(), BlockTenantInput{ID: "id-1", Reason: "Novo motivo", PerformedBy: "admin-1"})
	assert.ErrorIs(t, err, domain.ErrTenantAlreadyBlocked)
}

func TestBlockTenantUseCase_EmptyPerformedBy(t *testing.T) {
	repo := newMockTenantRepo()
	historyRepo := newMockBlockHistoryRepo()
	tenant, _ := domain.NewTenant("id-1", "Escritório", domain.TenantTypePJ, "111")
	repo.addTenant(tenant)

	uc := NewBlockTenantUseCase(repo, historyRepo)

	err := uc.Execute(context.Background(), BlockTenantInput{ID: "id-1", Reason: "Motivo", PerformedBy: ""})
	assert.ErrorIs(t, err, domain.ErrHistoryPerformedByRequired)
}
