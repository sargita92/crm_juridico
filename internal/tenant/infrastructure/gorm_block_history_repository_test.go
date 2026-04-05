package infrastructure

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/sasrgita/crm-juridico/internal/shared/database"
	"github.com/sasrgita/crm-juridico/internal/shared/testhelper"
	"github.com/sasrgita/crm-juridico/internal/tenant/domain"
)

func setupBlockHistoryRepo(t *testing.T) (*GormBlockHistoryRepository, *GormTenantRepository, *gorm.DB) {
	t.Helper()
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	db := sharedContainer.DB(t)
	log := sharedContainer.Logger()

	err := database.RunMigrations(db, log, "file://"+testhelper.MigrationsPath())
	require.NoError(t, err)

	db.Exec("DELETE FROM tenant_block_history")
	db.Exec("DELETE FROM user_tenants")
	db.Exec("DELETE FROM users")
	db.Exec("DELETE FROM tenants")

	return NewGormBlockHistoryRepository(db), NewGormTenantRepository(db), db
}

func createTestTenant(t *testing.T, tenantRepo *GormTenantRepository) *domain.Tenant {
	t.Helper()
	tenant, err := domain.NewTenant(uuid.New().String(), "Tenant Test", domain.TenantTypePJ, uuid.New().String()[:20])
	require.NoError(t, err)
	require.NoError(t, tenantRepo.Create(context.Background(), tenant))
	return tenant
}

func TestGormBlockHistoryRepository_Save_And_FindByTenantID(t *testing.T) {
	repo, tenantRepo, _ := setupBlockHistoryRepo(t)
	ctx := context.Background()

	tenant := createTestTenant(t, tenantRepo)

	entry1, err := domain.NewBlockHistoryEntry(uuid.New().String(), tenant.ID, domain.BlockActionBlock, "Inadimplencia", "admin-1")
	require.NoError(t, err)
	require.NoError(t, repo.Save(ctx, entry1))

	time.Sleep(time.Second)

	entry2, err := domain.NewBlockHistoryEntry(uuid.New().String(), tenant.ID, domain.BlockActionUnblock, "Regularizado", "admin-1")
	require.NoError(t, err)
	require.NoError(t, repo.Save(ctx, entry2))

	history, err := repo.FindByTenantID(ctx, tenant.ID)
	require.NoError(t, err)
	require.Len(t, history, 2)

	// DESC order: unblock first, block second
	assert.Equal(t, domain.BlockActionUnblock, history[0].Action)
	assert.Equal(t, "Regularizado", history[0].Reason)
	assert.Equal(t, domain.BlockActionBlock, history[1].Action)
	assert.Equal(t, "Inadimplencia", history[1].Reason)
}

func TestGormBlockHistoryRepository_FindByTenantID_Empty(t *testing.T) {
	repo, tenantRepo, _ := setupBlockHistoryRepo(t)
	ctx := context.Background()

	tenant := createTestTenant(t, tenantRepo)

	history, err := repo.FindByTenantID(ctx, tenant.ID)
	require.NoError(t, err)
	assert.Empty(t, history)
}

func TestGormBlockHistoryRepository_FindByTenantID_IsolatesTenants(t *testing.T) {
	repo, tenantRepo, _ := setupBlockHistoryRepo(t)
	ctx := context.Background()

	tenantA := createTestTenant(t, tenantRepo)
	tenantB, err := domain.NewTenant(uuid.New().String(), "Tenant B", domain.TenantTypePF, uuid.New().String()[:20])
	require.NoError(t, err)
	require.NoError(t, tenantRepo.Create(ctx, tenantB))

	entry, err := domain.NewBlockHistoryEntry(uuid.New().String(), tenantA.ID, domain.BlockActionBlock, "Motivo A", "admin-1")
	require.NoError(t, err)
	require.NoError(t, repo.Save(ctx, entry))

	historyB, err := repo.FindByTenantID(ctx, tenantB.ID)
	require.NoError(t, err)
	assert.Empty(t, historyB)

	historyA, err := repo.FindByTenantID(ctx, tenantA.ID)
	require.NoError(t, err)
	assert.Len(t, historyA, 1)
}
