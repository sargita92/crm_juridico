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
	tenant, _ := domain.NewTenant("id-1", "Escritório", domain.TenantTypePJ, "111")
	repo.addTenant(tenant)

	uc := NewBlockTenantUseCase(repo)

	err := uc.Execute(context.Background(), BlockTenantInput{ID: "id-1", Reason: "Inadimplência"})
	require.NoError(t, err)

	found, _ := repo.FindByID(context.Background(), "id-1")
	assert.Equal(t, domain.TenantStatusBlocked, found.Status)
	assert.Equal(t, "Inadimplência", found.BlockReason)
}

func TestBlockTenantUseCase_NotFound(t *testing.T) {
	repo := newMockTenantRepo()
	uc := NewBlockTenantUseCase(repo)

	err := uc.Execute(context.Background(), BlockTenantInput{ID: "nonexistent", Reason: "Motivo"})
	assert.ErrorIs(t, err, domain.ErrTenantNotFound)
}

func TestBlockTenantUseCase_EmptyReason(t *testing.T) {
	repo := newMockTenantRepo()
	tenant, _ := domain.NewTenant("id-1", "Escritório", domain.TenantTypePJ, "111")
	repo.addTenant(tenant)

	uc := NewBlockTenantUseCase(repo)

	err := uc.Execute(context.Background(), BlockTenantInput{ID: "id-1", Reason: ""})
	assert.ErrorIs(t, err, domain.ErrBlockReasonRequired)
}

func TestBlockTenantUseCase_AlreadyBlocked(t *testing.T) {
	repo := newMockTenantRepo()
	tenant, _ := domain.NewTenant("id-1", "Escritório", domain.TenantTypePJ, "111")
	_ = tenant.Block("Motivo anterior")
	repo.addTenant(tenant)

	uc := NewBlockTenantUseCase(repo)

	err := uc.Execute(context.Background(), BlockTenantInput{ID: "id-1", Reason: "Novo motivo"})
	assert.ErrorIs(t, err, domain.ErrTenantAlreadyBlocked)
}
