package application

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/sasrgita/crm-juridico/internal/specialist/domain"
	tenantdomain "github.com/sasrgita/crm-juridico/internal/tenant/domain"
)

func TestListSpecialistTenantsUseCase_Success(t *testing.T) {
	specRepo := newMockSpecialistRepo()
	stRepo := newMockSpecialistTenantRepo()
	tenantRepo := newMockTenantRepo()

	s, _ := domain.NewSpecialist("spec-1", "Assistente", "desc", "prompt")
	specRepo.addSpecialist(s)

	t1, _ := tenantdomain.NewTenant("tenant-1", "Escritorio A", tenantdomain.TenantTypePJ, "11.111.111/0001-11")
	t2, _ := tenantdomain.NewTenant("tenant-2", "Dr. Joao", tenantdomain.TenantTypePF, "111.111.111-11")
	tenantRepo.addTenant(t1)
	tenantRepo.addTenant(t2)

	_ = stRepo.Associate(context.Background(), "spec-1", "tenant-1")
	_ = stRepo.Associate(context.Background(), "spec-1", "tenant-2")

	uc := NewListSpecialistTenantsUseCase(specRepo, stRepo, tenantRepo)
	items, err := uc.Execute(context.Background(), "spec-1")

	require.NoError(t, err)
	assert.Len(t, items, 2)
}

func TestListSpecialistTenantsUseCase_NoTenants(t *testing.T) {
	specRepo := newMockSpecialistRepo()
	stRepo := newMockSpecialistTenantRepo()
	tenantRepo := newMockTenantRepo()

	s, _ := domain.NewSpecialist("spec-1", "Assistente", "desc", "prompt")
	specRepo.addSpecialist(s)

	uc := NewListSpecialistTenantsUseCase(specRepo, stRepo, tenantRepo)
	items, err := uc.Execute(context.Background(), "spec-1")

	require.NoError(t, err)
	assert.Nil(t, items)
}

func TestListSpecialistTenantsUseCase_SpecialistNotFound(t *testing.T) {
	specRepo := newMockSpecialistRepo()
	stRepo := newMockSpecialistTenantRepo()
	tenantRepo := newMockTenantRepo()

	uc := NewListSpecialistTenantsUseCase(specRepo, stRepo, tenantRepo)
	_, err := uc.Execute(context.Background(), "nonexistent")

	assert.ErrorIs(t, err, domain.ErrSpecialistNotFound)
}
