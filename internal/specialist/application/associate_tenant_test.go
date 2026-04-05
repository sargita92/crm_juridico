package application

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/sasrgita/crm-juridico/internal/specialist/domain"
	tenantdomain "github.com/sasrgita/crm-juridico/internal/tenant/domain"
)

func TestAssociateTenantUseCase_Success(t *testing.T) {
	specRepo := newMockSpecialistRepo()
	tenantRepo := newMockTenantRepo()
	stRepo := newMockSpecialistTenantRepo()

	s, _ := domain.NewSpecialist("spec-1", "Assistente", "desc", "prompt")
	specRepo.addSpecialist(s)

	tenant, _ := tenantdomain.NewTenant("tenant-1", "Escritorio", tenantdomain.TenantTypePJ, "11.111.111/0001-11")
	tenantRepo.addTenant(tenant)

	uc := NewAssociateTenantUseCase(specRepo, tenantRepo, stRepo)
	err := uc.Execute(context.Background(), AssociateTenantInput{
		SpecialistID: "spec-1",
		TenantIDs:    []string{"tenant-1"},
	})

	require.NoError(t, err)

	exists, _ := stRepo.Exists(context.Background(), "spec-1", "tenant-1")
	assert.True(t, exists)
}

func TestAssociateTenantUseCase_MultipleTenants(t *testing.T) {
	specRepo := newMockSpecialistRepo()
	tenantRepo := newMockTenantRepo()
	stRepo := newMockSpecialistTenantRepo()

	s, _ := domain.NewSpecialist("spec-1", "Assistente", "desc", "prompt")
	specRepo.addSpecialist(s)

	t1, _ := tenantdomain.NewTenant("tenant-1", "A", tenantdomain.TenantTypePJ, "11.111.111/0001-11")
	t2, _ := tenantdomain.NewTenant("tenant-2", "B", tenantdomain.TenantTypePJ, "22.222.222/0001-22")
	tenantRepo.addTenant(t1)
	tenantRepo.addTenant(t2)

	uc := NewAssociateTenantUseCase(specRepo, tenantRepo, stRepo)
	err := uc.Execute(context.Background(), AssociateTenantInput{
		SpecialistID: "spec-1",
		TenantIDs:    []string{"tenant-1", "tenant-2"},
	})

	require.NoError(t, err)

	ids, _ := stRepo.FindTenantIDsBySpecialistID(context.Background(), "spec-1")
	assert.Len(t, ids, 2)
}

func TestAssociateTenantUseCase_SpecialistNotFound(t *testing.T) {
	specRepo := newMockSpecialistRepo()
	tenantRepo := newMockTenantRepo()
	stRepo := newMockSpecialistTenantRepo()

	uc := NewAssociateTenantUseCase(specRepo, tenantRepo, stRepo)
	err := uc.Execute(context.Background(), AssociateTenantInput{
		SpecialistID: "nonexistent",
		TenantIDs:    []string{"tenant-1"},
	})

	assert.ErrorIs(t, err, domain.ErrSpecialistNotFound)
}

func TestAssociateTenantUseCase_SpecialistInactive(t *testing.T) {
	specRepo := newMockSpecialistRepo()
	tenantRepo := newMockTenantRepo()
	stRepo := newMockSpecialistTenantRepo()

	s, _ := domain.NewSpecialist("spec-1", "Assistente", "desc", "prompt")
	_ = s.Deactivate()
	specRepo.addSpecialist(s)

	uc := NewAssociateTenantUseCase(specRepo, tenantRepo, stRepo)
	err := uc.Execute(context.Background(), AssociateTenantInput{
		SpecialistID: "spec-1",
		TenantIDs:    []string{"tenant-1"},
	})

	assert.ErrorIs(t, err, domain.ErrSpecialistInactive)
}

func TestAssociateTenantUseCase_TenantNotFound(t *testing.T) {
	specRepo := newMockSpecialistRepo()
	tenantRepo := newMockTenantRepo()
	stRepo := newMockSpecialistTenantRepo()

	s, _ := domain.NewSpecialist("spec-1", "Assistente", "desc", "prompt")
	specRepo.addSpecialist(s)

	uc := NewAssociateTenantUseCase(specRepo, tenantRepo, stRepo)
	err := uc.Execute(context.Background(), AssociateTenantInput{
		SpecialistID: "spec-1",
		TenantIDs:    []string{"nonexistent"},
	})

	assert.ErrorIs(t, err, tenantdomain.ErrTenantNotFound)
}

func TestAssociateTenantUseCase_TenantInactive(t *testing.T) {
	specRepo := newMockSpecialistRepo()
	tenantRepo := newMockTenantRepo()
	stRepo := newMockSpecialistTenantRepo()

	s, _ := domain.NewSpecialist("spec-1", "Assistente", "desc", "prompt")
	specRepo.addSpecialist(s)

	tenant, _ := tenantdomain.NewTenant("tenant-1", "Escritorio", tenantdomain.TenantTypePJ, "11.111.111/0001-11")
	_ = tenant.Deactivate()
	tenantRepo.addTenant(tenant)

	uc := NewAssociateTenantUseCase(specRepo, tenantRepo, stRepo)
	err := uc.Execute(context.Background(), AssociateTenantInput{
		SpecialistID: "spec-1",
		TenantIDs:    []string{"tenant-1"},
	})

	assert.ErrorIs(t, err, tenantdomain.ErrTenantInactive)
}

func TestAssociateTenantUseCase_AlreadyAssociated_Skips(t *testing.T) {
	specRepo := newMockSpecialistRepo()
	tenantRepo := newMockTenantRepo()
	stRepo := newMockSpecialistTenantRepo()

	s, _ := domain.NewSpecialist("spec-1", "Assistente", "desc", "prompt")
	specRepo.addSpecialist(s)

	tenant, _ := tenantdomain.NewTenant("tenant-1", "Escritorio", tenantdomain.TenantTypePJ, "11.111.111/0001-11")
	tenantRepo.addTenant(tenant)

	_ = stRepo.Associate(context.Background(), "spec-1", "tenant-1")

	uc := NewAssociateTenantUseCase(specRepo, tenantRepo, stRepo)
	err := uc.Execute(context.Background(), AssociateTenantInput{
		SpecialistID: "spec-1",
		TenantIDs:    []string{"tenant-1"},
	})

	require.NoError(t, err)
}
