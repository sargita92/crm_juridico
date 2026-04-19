package infrastructure

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/sasrgita/crm-juridico/internal/pagamentos/domain"
	"github.com/sasrgita/crm-juridico/internal/shared/database"
	"github.com/sasrgita/crm-juridico/internal/shared/testhelper"
	tenantDomain "github.com/sasrgita/crm-juridico/internal/tenant/domain"
	tenantInfra "github.com/sasrgita/crm-juridico/internal/tenant/infrastructure"
)

func setupBillingRepo(t *testing.T) (*GormTenantBillingRepository, *gorm.DB) {
	t.Helper()
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	db := sharedContainer.DB(t)
	log := sharedContainer.Logger()
	require.NoError(t, database.RunMigrations(db, log, "file://"+testhelper.MigrationsPath()))
	db.Exec("DELETE FROM payments")
	db.Exec("DELETE FROM tenant_block_history")
	db.Exec("DELETE FROM user_tenants")
	db.Exec("DELETE FROM users")
	db.Exec("DELETE FROM tenants")
	return NewGormTenantBillingRepository(db), db
}

func TestTenantBilling_GetByID_Mensal(t *testing.T) {
	repo, db := setupBillingRepo(t)
	tn, _ := tenantDomain.NewTenant(uuid.New().String(), "T", tenantDomain.TenantTypePJ, uuid.New().String()[:20])
	valor := int64(50000)
	dia := uint8(10)
	start := time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC)
	tn.SetBillingConfig("mensal", &valor, &dia, &start, true)
	require.NoError(t, tenantInfra.NewGormTenantRepository(db).Create(context.Background(), tn))

	tb, err := repo.GetByID(context.Background(), tn.ID)
	require.NoError(t, err)
	assert.True(t, tb.Active)
	assert.Equal(t, domain.PlanMensal, tb.Config.Plano)
	require.NotNil(t, tb.Config.ValorCents)
	assert.Equal(t, int64(50000), *tb.Config.ValorCents)
	assert.True(t, tb.Config.ExibirPagamentos)
}

func TestTenantBilling_GetByID_NotFound(t *testing.T) {
	repo, _ := setupBillingRepo(t)
	_, err := repo.GetByID(context.Background(), uuid.NewString())
	assert.ErrorIs(t, err, domain.ErrTenantNotFound)
}

func TestTenantBilling_ListActiveBillable_SkipsVitalicio(t *testing.T) {
	repo, db := setupBillingRepo(t)
	tenantRepo := tenantInfra.NewGormTenantRepository(db)

	// mensal ativo
	m, _ := tenantDomain.NewTenant(uuid.New().String(), "M", tenantDomain.TenantTypePJ, uuid.New().String()[:20])
	valor := int64(50000)
	dia := uint8(10)
	start := time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC)
	m.SetBillingConfig("mensal", &valor, &dia, &start, true)
	require.NoError(t, tenantRepo.Create(context.Background(), m))

	// vitalicio - nao aparece
	v, _ := tenantDomain.NewTenant(uuid.New().String(), "V", tenantDomain.TenantTypePJ, uuid.New().String()[:20])
	v.SetBillingConfig("vitalicio", nil, nil, nil, false)
	require.NoError(t, tenantRepo.Create(context.Background(), v))

	list, err := repo.ListActiveBillable(context.Background())
	require.NoError(t, err)
	require.Len(t, list, 1)
	assert.Equal(t, m.ID, list[0].TenantID)
}

func TestTenantBilling_ListActiveBillable_SkipsBlocked(t *testing.T) {
	repo, db := setupBillingRepo(t)
	tenantRepo := tenantInfra.NewGormTenantRepository(db)

	blocked, _ := tenantDomain.NewTenant(uuid.New().String(), "B", tenantDomain.TenantTypePJ, uuid.New().String()[:20])
	valor := int64(50000)
	dia := uint8(10)
	start := time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC)
	blocked.SetBillingConfig("mensal", &valor, &dia, &start, true)
	require.NoError(t, blocked.Block("inadimplente"))
	require.NoError(t, tenantRepo.Create(context.Background(), blocked))

	list, err := repo.ListActiveBillable(context.Background())
	require.NoError(t, err)
	assert.Len(t, list, 0)
}
