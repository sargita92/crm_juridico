package application

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/sasrgita/crm-juridico/internal/tenant/domain"
)

func TestDeactivateTenantUseCase_Success(t *testing.T) {
	repo := newMockTenantRepo()
	tenant, _ := domain.NewTenant("id-1", "Escritório", domain.TenantTypePJ, "111")
	repo.addTenant(tenant)

	uc := NewDeactivateTenantUseCase(repo)

	err := uc.Execute(context.Background(), "id-1")
	require.NoError(t, err)

	found, _ := repo.FindByID(context.Background(), "id-1")
	assert.Equal(t, domain.TenantStatusInactive, found.Status)
}

func TestDeactivateTenantUseCase_NotFound(t *testing.T) {
	repo := newMockTenantRepo()
	uc := NewDeactivateTenantUseCase(repo)

	err := uc.Execute(context.Background(), "nonexistent")
	assert.ErrorIs(t, err, domain.ErrTenantNotFound)
}

func TestDeactivateTenantUseCase_AlreadyInactive(t *testing.T) {
	repo := newMockTenantRepo()
	tenant, _ := domain.NewTenant("id-1", "Escritório", domain.TenantTypePJ, "111")
	_ = tenant.Deactivate()
	repo.addTenant(tenant)

	uc := NewDeactivateTenantUseCase(repo)

	err := uc.Execute(context.Background(), "id-1")
	assert.ErrorIs(t, err, domain.ErrTenantInactive)
}

func TestDeactivateTenantUseCase_FromBlocked(t *testing.T) {
	repo := newMockTenantRepo()
	tenant, _ := domain.NewTenant("id-1", "Escritório", domain.TenantTypePJ, "111")
	_ = tenant.Block("Motivo")
	repo.addTenant(tenant)

	uc := NewDeactivateTenantUseCase(repo)

	err := uc.Execute(context.Background(), "id-1")
	require.NoError(t, err)

	found, _ := repo.FindByID(context.Background(), "id-1")
	assert.Equal(t, domain.TenantStatusInactive, found.Status)
}
