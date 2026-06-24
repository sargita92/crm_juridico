package application

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/sasrgita/crm-juridico/internal/specialist/domain"
	tenantdomain "github.com/sasrgita/crm-juridico/internal/tenant/domain"
)

func TestListManageableTenantsUseCase_FlagsAssociations(t *testing.T) {
	specRepo := newMockSpecialistRepo()
	stRepo := newMockSpecialistTenantRepo()
	tenantRepo := newMockTenantRepo()

	s, _ := domain.NewSpecialist("spec-1", "Assistente", "desc", "prompt")
	specRepo.addSpecialist(s)

	t1, _ := tenantdomain.NewTenant("tenant-1", "Alpha", tenantdomain.TenantTypePJ, "11.111.111/0001-11")
	t2, _ := tenantdomain.NewTenant("tenant-2", "Beta", tenantdomain.TenantTypePJ, "22.222.222/0001-22")
	t3, _ := tenantdomain.NewTenant("tenant-3", "Gamma", tenantdomain.TenantTypePJ, "33.333.333/0001-33")
	tenantRepo.addTenant(t1)
	tenantRepo.addTenant(t2)
	tenantRepo.addTenant(t3)

	require.NoError(t, stRepo.Associate(context.Background(), "spec-1", "tenant-1"))

	uc := NewListManageableTenantsUseCase(specRepo, stRepo, tenantRepo)
	items, err := uc.Execute(context.Background(), "spec-1", "")

	require.NoError(t, err)
	require.Len(t, items, 3)

	byID := make(map[string]ManageableTenantItem, len(items))
	for _, it := range items {
		byID[it.TenantID] = it
	}
	assert.True(t, byID["tenant-1"].IsAssociated)
	assert.False(t, byID["tenant-2"].IsAssociated)
	assert.False(t, byID["tenant-3"].IsAssociated)
}

func TestListManageableTenantsUseCase_OmitsInactiveUnassociated(t *testing.T) {
	specRepo := newMockSpecialistRepo()
	stRepo := newMockSpecialistTenantRepo()
	tenantRepo := newMockTenantRepo()

	s, _ := domain.NewSpecialist("spec-1", "Assistente", "desc", "prompt")
	specRepo.addSpecialist(s)

	active, _ := tenantdomain.NewTenant("tenant-1", "Ativo", tenantdomain.TenantTypePJ, "11.111.111/0001-11")
	inactive, _ := tenantdomain.NewTenant("tenant-2", "Inativo", tenantdomain.TenantTypePJ, "22.222.222/0001-22")
	_ = inactive.Deactivate()
	tenantRepo.addTenant(active)
	tenantRepo.addTenant(inactive)

	uc := NewListManageableTenantsUseCase(specRepo, stRepo, tenantRepo)
	items, err := uc.Execute(context.Background(), "spec-1", "")

	require.NoError(t, err)
	require.Len(t, items, 1)
	assert.Equal(t, "tenant-1", items[0].TenantID)
}

func TestListManageableTenantsUseCase_KeepsInactiveIfAssociated(t *testing.T) {
	specRepo := newMockSpecialistRepo()
	stRepo := newMockSpecialistTenantRepo()
	tenantRepo := newMockTenantRepo()

	s, _ := domain.NewSpecialist("spec-1", "Assistente", "desc", "prompt")
	specRepo.addSpecialist(s)

	inactive, _ := tenantdomain.NewTenant("tenant-2", "Inativo", tenantdomain.TenantTypePJ, "22.222.222/0001-22")
	_ = inactive.Deactivate()
	tenantRepo.addTenant(inactive)

	require.NoError(t, stRepo.Associate(context.Background(), "spec-1", "tenant-2"))

	uc := NewListManageableTenantsUseCase(specRepo, stRepo, tenantRepo)
	items, err := uc.Execute(context.Background(), "spec-1", "")

	require.NoError(t, err)
	require.Len(t, items, 1)
	assert.Equal(t, "tenant-2", items[0].TenantID)
	assert.True(t, items[0].IsAssociated)
}

func TestListManageableTenantsUseCase_SpecialistNotFound(t *testing.T) {
	specRepo := newMockSpecialistRepo()
	stRepo := newMockSpecialistTenantRepo()
	tenantRepo := newMockTenantRepo()

	uc := NewListManageableTenantsUseCase(specRepo, stRepo, tenantRepo)
	_, err := uc.Execute(context.Background(), "nonexistent", "")

	assert.ErrorIs(t, err, domain.ErrSpecialistNotFound)
}

func TestListManageableTenantsUseCase_MarksDefault(t *testing.T) {
	specRepo := newMockSpecialistRepo()
	stRepo := newMockSpecialistTenantRepo()
	tenantRepo := newMockTenantRepo()

	s, _ := domain.NewSpecialist("spec-1", "Assistente", "desc", "prompt")
	specRepo.addSpecialist(s)

	t1, _ := tenantdomain.NewTenant("tenant-1", "Alpha", tenantdomain.TenantTypePJ, "11.111.111/0001-11")
	tenantRepo.addTenant(t1)
	require.NoError(t, stRepo.Associate(context.Background(), "spec-1", "tenant-1"))
	stRepo.markDefault("spec-1", "tenant-1")

	uc := NewListManageableTenantsUseCase(specRepo, stRepo, tenantRepo)
	items, err := uc.Execute(context.Background(), "spec-1", "")

	require.NoError(t, err)
	require.Len(t, items, 1)
	assert.True(t, items[0].IsAssociated)
	assert.True(t, items[0].IsDefault)
}
