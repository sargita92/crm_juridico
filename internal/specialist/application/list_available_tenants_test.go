package application

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	tenantdomain "github.com/sasrgita/crm-juridico/internal/tenant/domain"
)

func TestListAvailableTenantsUseCase_Success(t *testing.T) {
	stRepo := newMockSpecialistTenantRepo()
	tenantRepo := newMockTenantRepo()

	t1, _ := tenantdomain.NewTenant("tenant-1", "A", tenantdomain.TenantTypePJ, "11.111.111/0001-11")
	t2, _ := tenantdomain.NewTenant("tenant-2", "B", tenantdomain.TenantTypePJ, "22.222.222/0001-22")
	t3, _ := tenantdomain.NewTenant("tenant-3", "C", tenantdomain.TenantTypePJ, "33.333.333/0001-33")
	tenantRepo.addTenant(t1)
	tenantRepo.addTenant(t2)
	tenantRepo.addTenant(t3)

	// tenant-1 already associated
	_ = stRepo.Associate(context.Background(), "spec-1", "tenant-1")

	uc := NewListAvailableTenantsUseCase(stRepo, tenantRepo)
	items, err := uc.Execute(context.Background(), "spec-1", "")

	require.NoError(t, err)
	assert.Len(t, items, 2)

	ids := make(map[string]bool)
	for _, item := range items {
		ids[item.TenantID] = true
	}
	assert.False(t, ids["tenant-1"])
	assert.True(t, ids["tenant-2"])
	assert.True(t, ids["tenant-3"])
}

func TestListAvailableTenantsUseCase_ExcludesInactiveTenants(t *testing.T) {
	stRepo := newMockSpecialistTenantRepo()
	tenantRepo := newMockTenantRepo()

	active, _ := tenantdomain.NewTenant("tenant-1", "Ativo", tenantdomain.TenantTypePJ, "11.111.111/0001-11")
	inactive, _ := tenantdomain.NewTenant("tenant-2", "Inativo", tenantdomain.TenantTypePJ, "22.222.222/0001-22")
	_ = inactive.Deactivate()
	tenantRepo.addTenant(active)
	tenantRepo.addTenant(inactive)

	uc := NewListAvailableTenantsUseCase(stRepo, tenantRepo)
	items, err := uc.Execute(context.Background(), "spec-1", "")

	require.NoError(t, err)
	assert.Len(t, items, 1)
	assert.Equal(t, "tenant-1", items[0].TenantID)
}

func TestListAvailableTenantsUseCase_AllAssociated_ReturnsEmpty(t *testing.T) {
	stRepo := newMockSpecialistTenantRepo()
	tenantRepo := newMockTenantRepo()

	t1, _ := tenantdomain.NewTenant("tenant-1", "A", tenantdomain.TenantTypePJ, "11.111.111/0001-11")
	tenantRepo.addTenant(t1)

	_ = stRepo.Associate(context.Background(), "spec-1", "tenant-1")

	uc := NewListAvailableTenantsUseCase(stRepo, tenantRepo)
	items, err := uc.Execute(context.Background(), "spec-1", "")

	require.NoError(t, err)
	assert.Empty(t, items)
}
