package application

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/sasrgita/crm-juridico/internal/tenant/domain"
)

func TestGetBlockHistoryUseCase_Success(t *testing.T) {
	repo := newMockTenantRepo()
	historyRepo := newMockBlockHistoryRepo()
	tenant, _ := domain.NewTenant("id-1", "Escritório", domain.TenantTypePJ, "111")
	repo.addTenant(tenant)

	entry1, _ := domain.NewBlockHistoryEntry("h-1", "id-1", domain.BlockActionBlock, "Inadimplencia", "admin-1")
	entry2, _ := domain.NewBlockHistoryEntry("h-2", "id-1", domain.BlockActionUnblock, "Regularizado", "admin-2")
	historyRepo.entries = append(historyRepo.entries, *entry1, *entry2)

	uc := NewGetBlockHistoryUseCase(repo, historyRepo)

	result, err := uc.Execute(context.Background(), "id-1")
	require.NoError(t, err)
	require.Len(t, result, 2)
	assert.Equal(t, "block", result[0].Action)
	assert.Equal(t, "admin-1", result[0].PerformedBy)
	assert.Equal(t, "unblock", result[1].Action)
	assert.Equal(t, "admin-2", result[1].PerformedBy)
}

func TestGetBlockHistoryUseCase_TenantNotFound(t *testing.T) {
	repo := newMockTenantRepo()
	historyRepo := newMockBlockHistoryRepo()
	uc := NewGetBlockHistoryUseCase(repo, historyRepo)

	_, err := uc.Execute(context.Background(), "nonexistent")
	assert.ErrorIs(t, err, domain.ErrTenantNotFound)
}

func TestGetBlockHistoryUseCase_EmptyHistory(t *testing.T) {
	repo := newMockTenantRepo()
	historyRepo := newMockBlockHistoryRepo()
	tenant, _ := domain.NewTenant("id-1", "Escritório", domain.TenantTypePJ, "111")
	repo.addTenant(tenant)

	uc := NewGetBlockHistoryUseCase(repo, historyRepo)

	result, err := uc.Execute(context.Background(), "id-1")
	require.NoError(t, err)
	assert.Empty(t, result)
}
