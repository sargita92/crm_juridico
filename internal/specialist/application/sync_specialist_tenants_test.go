package application

import (
	"context"
	"sort"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/sasrgita/crm-juridico/internal/specialist/domain"
	tenantdomain "github.com/sasrgita/crm-juridico/internal/tenant/domain"
)

func setupSyncTest(t *testing.T) (*mockSpecialistRepo, *mockSpecialistTenantRepo, *mockTenantRepo, *SyncSpecialistTenantsUseCase) {
	t.Helper()
	specRepo := newMockSpecialistRepo()
	stRepo := newMockSpecialistTenantRepo()
	tenantRepo := newMockTenantRepo()
	uc := NewSyncSpecialistTenantsUseCase(specRepo, tenantRepo, stRepo)
	return specRepo, stRepo, tenantRepo, uc
}

func TestSyncSpecialistTenantsUseCase_AddsNew(t *testing.T) {
	specRepo, stRepo, tenantRepo, uc := setupSyncTest(t)

	s, _ := domain.NewSpecialist("spec-1", "Assistente", "desc", "prompt")
	specRepo.addSpecialist(s)
	t1, _ := tenantdomain.NewTenant("tenant-1", "Alpha", tenantdomain.TenantTypePJ, "11.111.111/0001-11")
	tenantRepo.addTenant(t1)

	result, err := uc.Execute(context.Background(), SyncSpecialistTenantsInput{
		SpecialistID: "spec-1",
		TenantIDs:    []string{"tenant-1"},
	})

	require.NoError(t, err)
	assert.Equal(t, 1, result.Added)
	assert.Equal(t, 0, result.Removed)

	ids, _ := stRepo.FindTenantIDsBySpecialistID(context.Background(), "spec-1")
	assert.Equal(t, []string{"tenant-1"}, ids)
}

func TestSyncSpecialistTenantsUseCase_SingleAssociate_BecomesDefault(t *testing.T) {
	specRepo, stRepo, tenantRepo, uc := setupSyncTest(t)

	s, _ := domain.NewSpecialist("spec-1", "Assistente", "desc", "prompt")
	specRepo.addSpecialist(s)
	t1, _ := tenantdomain.NewTenant("tenant-1", "Alpha", tenantdomain.TenantTypePJ, "11.111.111/0001-11")
	tenantRepo.addTenant(t1)

	_, err := uc.Execute(context.Background(), SyncSpecialistTenantsInput{
		SpecialistID: "spec-1",
		TenantIDs:    []string{"tenant-1"},
	})
	require.NoError(t, err)

	def, err := stRepo.FindDefaultByTenantID(context.Background(), "tenant-1")
	require.NoError(t, err, "syncing a single specialist onto a tenant must make it routable (default)")
	assert.Equal(t, "spec-1", def)
}

func TestSyncSpecialistTenantsUseCase_RemovesMissing(t *testing.T) {
	specRepo, stRepo, tenantRepo, uc := setupSyncTest(t)

	s, _ := domain.NewSpecialist("spec-1", "Assistente", "desc", "prompt")
	specRepo.addSpecialist(s)
	t1, _ := tenantdomain.NewTenant("tenant-1", "A", tenantdomain.TenantTypePJ, "11.111.111/0001-11")
	t2, _ := tenantdomain.NewTenant("tenant-2", "B", tenantdomain.TenantTypePJ, "22.222.222/0001-22")
	tenantRepo.addTenant(t1)
	tenantRepo.addTenant(t2)
	require.NoError(t, stRepo.Associate(context.Background(), "spec-1", "tenant-1"))
	require.NoError(t, stRepo.Associate(context.Background(), "spec-1", "tenant-2"))

	result, err := uc.Execute(context.Background(), SyncSpecialistTenantsInput{
		SpecialistID: "spec-1",
		TenantIDs:    []string{"tenant-1"},
	})

	require.NoError(t, err)
	assert.Equal(t, 0, result.Added)
	assert.Equal(t, 1, result.Removed)

	ids, _ := stRepo.FindTenantIDsBySpecialistID(context.Background(), "spec-1")
	assert.Equal(t, []string{"tenant-1"}, ids)
}

func TestSyncSpecialistTenantsUseCase_AddAndRemoveTogether(t *testing.T) {
	specRepo, stRepo, tenantRepo, uc := setupSyncTest(t)

	s, _ := domain.NewSpecialist("spec-1", "Assistente", "desc", "prompt")
	specRepo.addSpecialist(s)
	for _, id := range []string{"tenant-1", "tenant-2", "tenant-3"} {
		tnt, _ := tenantdomain.NewTenant(id, id, tenantdomain.TenantTypePJ, "11.111.111/000"+id[len(id)-1:]+"-11")
		tenantRepo.addTenant(tnt)
	}
	require.NoError(t, stRepo.Associate(context.Background(), "spec-1", "tenant-1"))
	require.NoError(t, stRepo.Associate(context.Background(), "spec-1", "tenant-2"))

	result, err := uc.Execute(context.Background(), SyncSpecialistTenantsInput{
		SpecialistID: "spec-1",
		TenantIDs:    []string{"tenant-2", "tenant-3"},
	})

	require.NoError(t, err)
	assert.Equal(t, 1, result.Added)
	assert.Equal(t, 1, result.Removed)

	ids, _ := stRepo.FindTenantIDsBySpecialistID(context.Background(), "spec-1")
	sort.Strings(ids)
	assert.Equal(t, []string{"tenant-2", "tenant-3"}, ids)
}

func TestSyncSpecialistTenantsUseCase_NoChanges(t *testing.T) {
	specRepo, stRepo, tenantRepo, uc := setupSyncTest(t)

	s, _ := domain.NewSpecialist("spec-1", "Assistente", "desc", "prompt")
	specRepo.addSpecialist(s)
	t1, _ := tenantdomain.NewTenant("tenant-1", "A", tenantdomain.TenantTypePJ, "11.111.111/0001-11")
	tenantRepo.addTenant(t1)
	require.NoError(t, stRepo.Associate(context.Background(), "spec-1", "tenant-1"))

	result, err := uc.Execute(context.Background(), SyncSpecialistTenantsInput{
		SpecialistID: "spec-1",
		TenantIDs:    []string{"tenant-1"},
	})

	require.NoError(t, err)
	assert.Equal(t, 0, result.Added)
	assert.Equal(t, 0, result.Removed)
}

func TestSyncSpecialistTenantsUseCase_EmptySelection_RemovesAll(t *testing.T) {
	specRepo, stRepo, tenantRepo, uc := setupSyncTest(t)

	s, _ := domain.NewSpecialist("spec-1", "Assistente", "desc", "prompt")
	specRepo.addSpecialist(s)
	t1, _ := tenantdomain.NewTenant("tenant-1", "A", tenantdomain.TenantTypePJ, "11.111.111/0001-11")
	tenantRepo.addTenant(t1)
	require.NoError(t, stRepo.Associate(context.Background(), "spec-1", "tenant-1"))

	result, err := uc.Execute(context.Background(), SyncSpecialistTenantsInput{
		SpecialistID: "spec-1",
		TenantIDs:    []string{},
	})

	require.NoError(t, err)
	assert.Equal(t, 1, result.Removed)

	ids, _ := stRepo.FindTenantIDsBySpecialistID(context.Background(), "spec-1")
	assert.Empty(t, ids)
}

func TestSyncSpecialistTenantsUseCase_DeduplicatesInput(t *testing.T) {
	specRepo, _, tenantRepo, uc := setupSyncTest(t)

	s, _ := domain.NewSpecialist("spec-1", "Assistente", "desc", "prompt")
	specRepo.addSpecialist(s)
	t1, _ := tenantdomain.NewTenant("tenant-1", "A", tenantdomain.TenantTypePJ, "11.111.111/0001-11")
	tenantRepo.addTenant(t1)

	result, err := uc.Execute(context.Background(), SyncSpecialistTenantsInput{
		SpecialistID: "spec-1",
		TenantIDs:    []string{"tenant-1", "tenant-1"},
	})

	require.NoError(t, err)
	assert.Equal(t, 1, result.Added)
}

func TestSyncSpecialistTenantsUseCase_SpecialistNotFound(t *testing.T) {
	_, _, _, uc := setupSyncTest(t)

	_, err := uc.Execute(context.Background(), SyncSpecialistTenantsInput{
		SpecialistID: "nonexistent",
		TenantIDs:    []string{"tenant-1"},
	})

	assert.ErrorIs(t, err, domain.ErrSpecialistNotFound)
}

func TestSyncSpecialistTenantsUseCase_SpecialistInactive(t *testing.T) {
	specRepo, _, _, uc := setupSyncTest(t)

	s, _ := domain.NewSpecialist("spec-1", "Assistente", "desc", "prompt")
	_ = s.Deactivate()
	specRepo.addSpecialist(s)

	_, err := uc.Execute(context.Background(), SyncSpecialistTenantsInput{
		SpecialistID: "spec-1",
		TenantIDs:    []string{"tenant-1"},
	})

	assert.ErrorIs(t, err, domain.ErrSpecialistInactive)
}

func TestSyncSpecialistTenantsUseCase_TenantNotFound(t *testing.T) {
	specRepo, _, _, uc := setupSyncTest(t)

	s, _ := domain.NewSpecialist("spec-1", "Assistente", "desc", "prompt")
	specRepo.addSpecialist(s)

	_, err := uc.Execute(context.Background(), SyncSpecialistTenantsInput{
		SpecialistID: "spec-1",
		TenantIDs:    []string{"missing"},
	})

	assert.ErrorIs(t, err, tenantdomain.ErrTenantNotFound)
}

func TestSyncSpecialistTenantsUseCase_TenantInactive(t *testing.T) {
	specRepo, _, tenantRepo, uc := setupSyncTest(t)

	s, _ := domain.NewSpecialist("spec-1", "Assistente", "desc", "prompt")
	specRepo.addSpecialist(s)
	tnt, _ := tenantdomain.NewTenant("tenant-1", "Inativo", tenantdomain.TenantTypePJ, "11.111.111/0001-11")
	_ = tnt.Deactivate()
	tenantRepo.addTenant(tnt)

	_, err := uc.Execute(context.Background(), SyncSpecialistTenantsInput{
		SpecialistID: "spec-1",
		TenantIDs:    []string{"tenant-1"},
	})

	assert.ErrorIs(t, err, tenantdomain.ErrTenantInactive)
}

func TestSyncSpecialistTenantsUseCase_KeepingAssociatedInactive_OK(t *testing.T) {
	// Tenant inativo já associado: pode permanecer na seleção (não tenta re-validar).
	specRepo, stRepo, tenantRepo, uc := setupSyncTest(t)

	s, _ := domain.NewSpecialist("spec-1", "Assistente", "desc", "prompt")
	specRepo.addSpecialist(s)
	tnt, _ := tenantdomain.NewTenant("tenant-1", "Inativo", tenantdomain.TenantTypePJ, "11.111.111/0001-11")
	tenantRepo.addTenant(tnt)
	require.NoError(t, stRepo.Associate(context.Background(), "spec-1", "tenant-1"))
	_ = tnt.Deactivate()
	tenantRepo.tenants["tenant-1"] = tnt

	result, err := uc.Execute(context.Background(), SyncSpecialistTenantsInput{
		SpecialistID: "spec-1",
		TenantIDs:    []string{"tenant-1"},
	})

	require.NoError(t, err)
	assert.Equal(t, 0, result.Added)
	assert.Equal(t, 0, result.Removed)
}

func TestSyncSpecialistTenantsUseCase_TooManyTenants(t *testing.T) {
	specRepo, _, _, uc := setupSyncTest(t)

	s, _ := domain.NewSpecialist("spec-1", "Assistente", "desc", "prompt")
	specRepo.addSpecialist(s)

	ids := make([]string, 0, 501)
	for i := 0; i < 501; i++ {
		ids = append(ids, "t")
	}
	_, err := uc.Execute(context.Background(), SyncSpecialistTenantsInput{
		SpecialistID: "spec-1",
		TenantIDs:    ids,
	})

	assert.ErrorIs(t, err, ErrTooManyTenants)
}
