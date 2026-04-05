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
	tenant, _ := domain.NewTenant("id-1", "Escritório", domain.TenantTypePJ, "111")
	_ = tenant.Block("Inadimplência")
	repo.addTenant(tenant)

	uc := NewUnblockTenantUseCase(repo)

	err := uc.Execute(context.Background(), UnblockTenantInput{ID: "id-1", Reason: "Pagamento regularizado"})
	require.NoError(t, err)

	found, _ := repo.FindByID(context.Background(), "id-1")
	assert.Equal(t, domain.TenantStatusActive, found.Status)
	assert.Empty(t, found.BlockReason)
}

func TestUnblockTenantUseCase_NotFound(t *testing.T) {
	repo := newMockTenantRepo()
	uc := NewUnblockTenantUseCase(repo)

	err := uc.Execute(context.Background(), UnblockTenantInput{ID: "nonexistent", Reason: "Motivo"})
	assert.ErrorIs(t, err, domain.ErrTenantNotFound)
}

func TestUnblockTenantUseCase_EmptyReason(t *testing.T) {
	repo := newMockTenantRepo()
	tenant, _ := domain.NewTenant("id-1", "Escritório", domain.TenantTypePJ, "111")
	_ = tenant.Block("Motivo")
	repo.addTenant(tenant)

	uc := NewUnblockTenantUseCase(repo)

	err := uc.Execute(context.Background(), UnblockTenantInput{ID: "id-1", Reason: ""})
	assert.ErrorIs(t, err, domain.ErrUnblockReasonRequired)
}

func TestUnblockTenantUseCase_NotBlocked(t *testing.T) {
	repo := newMockTenantRepo()
	tenant, _ := domain.NewTenant("id-1", "Escritório", domain.TenantTypePJ, "111")
	repo.addTenant(tenant)

	uc := NewUnblockTenantUseCase(repo)

	err := uc.Execute(context.Background(), UnblockTenantInput{ID: "id-1", Reason: "Motivo"})
	assert.ErrorIs(t, err, domain.ErrTenantNotBlocked)
}
