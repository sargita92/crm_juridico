package infrastructure

import (
	"strings"
	"context"
	"os"
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

var sharedContainer *testhelper.MySQLContainer

func TestMain(m *testing.M) {
	short := false
	for _, arg := range os.Args {
		if arg == "-test.short" || arg == "-short" || strings.HasPrefix(arg, "-test.short=") || strings.HasPrefix(arg, "-short=") {
			short = true
			break
		}
	}

	if !short {
		ctx := context.Background()
		sharedContainer = testhelper.NewMySQLContainerForMain(ctx)
		code := m.Run()
		_ = sharedContainer.Container.Terminate(ctx)
		os.Exit(code)
	}

	os.Exit(m.Run())
}

func setupTenantRepo(t *testing.T) (*GormTenantRepository, *gorm.DB) {
	t.Helper()
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	db := sharedContainer.DB(t)
	log := sharedContainer.Logger()

	err := database.RunMigrations(db, log, "file://"+testhelper.MigrationsPath())
	require.NoError(t, err)

	// Clean tables for test isolation
	db.Exec("DELETE FROM tenant_block_history")
	db.Exec("DELETE FROM user_tenants")
	db.Exec("DELETE FROM users")
	db.Exec("DELETE FROM tenants")

	return NewGormTenantRepository(db), db
}

func TestGormTenantRepository_Create_And_FindByID(t *testing.T) {
	repo, _ := setupTenantRepo(t)

	tenant, err := domain.NewTenant(uuid.New().String(), "Escritório Silva", domain.TenantTypePJ, "12.345.678/0001-00")
	require.NoError(t, err)

	err = repo.Create(context.Background(), tenant)
	require.NoError(t, err)

	found, err := repo.FindByID(context.Background(), tenant.ID)
	require.NoError(t, err)
	assert.Equal(t, tenant.ID, found.ID)
	assert.Equal(t, tenant.Name, found.Name)
	assert.Equal(t, tenant.Type, found.Type)
	assert.Equal(t, tenant.Document, found.Document)
	assert.Equal(t, domain.TenantStatusActive, found.Status)
}

func TestGormTenantRepository_Create_DuplicateDocument_ReturnsError(t *testing.T) {
	repo, _ := setupTenantRepo(t)

	ctx := context.Background()
	t1, _ := domain.NewTenant(uuid.New().String(), "Escritório A", domain.TenantTypePJ, "12.345.678/0001-00")
	t2, _ := domain.NewTenant(uuid.New().String(), "Escritório B", domain.TenantTypePJ, "12.345.678/0001-00")

	require.NoError(t, repo.Create(ctx, t1))
	err := repo.Create(ctx, t2)
	assert.ErrorIs(t, err, domain.ErrTenantDocumentExists)
}

func TestGormTenantRepository_FindByID_NotFound(t *testing.T) {
	repo, _ := setupTenantRepo(t)

	_, err := repo.FindByID(context.Background(), "nonexistent-id")
	assert.ErrorIs(t, err, domain.ErrTenantNotFound)
}

func TestGormTenantRepository_FindByIDs(t *testing.T) {
	repo, _ := setupTenantRepo(t)

	ctx := context.Background()
	t1, _ := domain.NewTenant(uuid.New().String(), "Escritório A", domain.TenantTypePJ, "11.111.111/0001-11")
	t2, _ := domain.NewTenant(uuid.New().String(), "Dr. João", domain.TenantTypePF, "111.111.111-11")
	require.NoError(t, repo.Create(ctx, t1))
	require.NoError(t, repo.Create(ctx, t2))

	tenants, err := repo.FindByIDs(ctx, []string{t1.ID, t2.ID})
	require.NoError(t, err)
	assert.Len(t, tenants, 2)
}

func TestGormTenantRepository_FindByIDs_EmptySlice(t *testing.T) {
	repo, _ := setupTenantRepo(t)

	tenants, err := repo.FindByIDs(context.Background(), []string{})
	require.NoError(t, err)
	assert.Nil(t, tenants)
}

func TestGormTenantRepository_FindAll_ReturnsActiveOnly(t *testing.T) {
	repo, _ := setupTenantRepo(t)

	ctx := context.Background()
	active, _ := domain.NewTenant(uuid.New().String(), "Ativo", domain.TenantTypePJ, "22.222.222/0001-22")
	blocked, _ := domain.NewTenant(uuid.New().String(), "Bloqueado", domain.TenantTypePJ, "33.333.333/0001-33")
	blocked.Status = domain.TenantStatusBlocked

	require.NoError(t, repo.Create(ctx, active))
	require.NoError(t, repo.Create(ctx, blocked))

	tenants, err := repo.FindAll(ctx)
	require.NoError(t, err)
	assert.Len(t, tenants, 1)
	assert.Equal(t, active.ID, tenants[0].ID)
}

func TestGormTenantRepository_Update_Success(t *testing.T) {
	repo, _ := setupTenantRepo(t)

	ctx := context.Background()
	tenant, _ := domain.NewTenant(uuid.New().String(), "Nome Antigo", domain.TenantTypePF, "111.111.111-11")
	require.NoError(t, repo.Create(ctx, tenant))

	require.NoError(t, tenant.Update("Nome Novo", domain.TenantTypePJ, "22.222.222/0001-22"))
	err := repo.Update(ctx, tenant)
	require.NoError(t, err)

	found, err := repo.FindByID(ctx, tenant.ID)
	require.NoError(t, err)
	assert.Equal(t, "Nome Novo", found.Name)
	assert.Equal(t, domain.TenantTypePJ, found.Type)
	assert.Equal(t, "22.222.222/0001-22", found.Document)
}

func TestGormTenantRepository_Update_DuplicateDocument_ReturnsError(t *testing.T) {
	repo, _ := setupTenantRepo(t)

	ctx := context.Background()
	t1, _ := domain.NewTenant(uuid.New().String(), "A", domain.TenantTypePF, "111.111.111-11")
	t2, _ := domain.NewTenant(uuid.New().String(), "B", domain.TenantTypePF, "222.222.222-22")
	require.NoError(t, repo.Create(ctx, t1))
	require.NoError(t, repo.Create(ctx, t2))

	require.NoError(t, t2.Update("B", domain.TenantTypePF, "111.111.111-11"))
	err := repo.Update(ctx, t2)
	assert.ErrorIs(t, err, domain.ErrTenantDocumentExists)
}

func TestGormTenantRepository_Update_BlockAndUnblock(t *testing.T) {
	repo, _ := setupTenantRepo(t)

	ctx := context.Background()
	tenant, _ := domain.NewTenant(uuid.New().String(), "Escritório", domain.TenantTypePJ, "33.333.333/0001-33")
	require.NoError(t, repo.Create(ctx, tenant))

	require.NoError(t, tenant.Block("Inadimplência"))
	require.NoError(t, repo.Update(ctx, tenant))

	found, _ := repo.FindByID(ctx, tenant.ID)
	assert.Equal(t, domain.TenantStatusBlocked, found.Status)
	assert.Equal(t, "Inadimplência", found.BlockReason)

	require.NoError(t, tenant.Unblock("Pagamento regularizado"))
	require.NoError(t, repo.Update(ctx, tenant))

	found, _ = repo.FindByID(ctx, tenant.ID)
	assert.Equal(t, domain.TenantStatusActive, found.Status)
	assert.Empty(t, found.BlockReason)
}

func TestGormTenantRepository_FindWithFilter_SearchByName(t *testing.T) {
	repo, _ := setupTenantRepo(t)

	ctx := context.Background()
	t1, _ := domain.NewTenant(uuid.New().String(), "Escritório ABC", domain.TenantTypePJ, "11.111.111/0001-11")
	t2, _ := domain.NewTenant(uuid.New().String(), "Escritório XYZ", domain.TenantTypePJ, "22.222.222/0001-22")
	require.NoError(t, repo.Create(ctx, t1))
	require.NoError(t, repo.Create(ctx, t2))

	result, err := repo.FindWithFilter(ctx, domain.TenantFilter{Search: "ABC", Page: 1, Limit: 20})
	require.NoError(t, err)
	assert.Equal(t, int64(1), result.Total)
	assert.Equal(t, "Escritório ABC", result.Tenants[0].Name)
}

func TestGormTenantRepository_FindWithFilter_SearchByDocument(t *testing.T) {
	repo, _ := setupTenantRepo(t)

	ctx := context.Background()
	t1, _ := domain.NewTenant(uuid.New().String(), "Escritório A", domain.TenantTypePJ, "44.444.444/0001-44")
	require.NoError(t, repo.Create(ctx, t1))

	result, err := repo.FindWithFilter(ctx, domain.TenantFilter{Search: "44.444", Page: 1, Limit: 20})
	require.NoError(t, err)
	assert.Equal(t, int64(1), result.Total)
}

func TestGormTenantRepository_FindWithFilter_FilterByStatus(t *testing.T) {
	repo, _ := setupTenantRepo(t)

	ctx := context.Background()
	active, _ := domain.NewTenant(uuid.New().String(), "Ativo", domain.TenantTypePJ, "55.555.555/0001-55")
	blocked, _ := domain.NewTenant(uuid.New().String(), "Bloqueado", domain.TenantTypePJ, "66.666.666/0001-66")
	require.NoError(t, repo.Create(ctx, active))
	require.NoError(t, repo.Create(ctx, blocked))
	require.NoError(t, blocked.Block("Motivo"))
	require.NoError(t, repo.Update(ctx, blocked))

	result, err := repo.FindWithFilter(ctx, domain.TenantFilter{Status: domain.TenantStatusBlocked, Page: 1, Limit: 20})
	require.NoError(t, err)
	assert.Equal(t, int64(1), result.Total)
	assert.Equal(t, "Bloqueado", result.Tenants[0].Name)
}

func TestGormTenantRepository_FindWithFilter_FilterByType(t *testing.T) {
	repo, _ := setupTenantRepo(t)

	ctx := context.Background()
	pj, _ := domain.NewTenant(uuid.New().String(), "PJ Tenant", domain.TenantTypePJ, "77.777.777/0001-77")
	pf, _ := domain.NewTenant(uuid.New().String(), "PF Tenant", domain.TenantTypePF, "888.888.888-88")
	require.NoError(t, repo.Create(ctx, pj))
	require.NoError(t, repo.Create(ctx, pf))

	result, err := repo.FindWithFilter(ctx, domain.TenantFilter{Type: domain.TenantTypePF, Page: 1, Limit: 20})
	require.NoError(t, err)
	assert.Equal(t, int64(1), result.Total)
	assert.Equal(t, "PF Tenant", result.Tenants[0].Name)
}

func TestGormTenantRepository_FindWithFilter_Pagination(t *testing.T) {
	repo, _ := setupTenantRepo(t)

	ctx := context.Background()
	for i := 0; i < 5; i++ {
		tenant, _ := domain.NewTenant(uuid.New().String(), "Tenant", domain.TenantTypePJ, uuid.New().String()[:20])
		require.NoError(t, repo.Create(ctx, tenant))
	}

	result, err := repo.FindWithFilter(ctx, domain.TenantFilter{Page: 1, Limit: 2})
	require.NoError(t, err)
	assert.Equal(t, int64(5), result.Total)
	assert.Len(t, result.Tenants, 2)
	assert.Equal(t, 1, result.Page)
	assert.Equal(t, 2, result.Limit)
}

func TestGormTenantRepository_FindWithFilter_DefaultPageAndLimit(t *testing.T) {
	repo, _ := setupTenantRepo(t)

	result, err := repo.FindWithFilter(context.Background(), domain.TenantFilter{})
	require.NoError(t, err)
	assert.Equal(t, 1, result.Page)
	assert.Equal(t, 20, result.Limit)
}

func TestGormTenantRepository_FindByDocument_Success(t *testing.T) {
	repo, _ := setupTenantRepo(t)

	ctx := context.Background()
	tenant, _ := domain.NewTenant(uuid.New().String(), "Escritório", domain.TenantTypePJ, "99.999.999/0001-99")
	require.NoError(t, repo.Create(ctx, tenant))

	found, err := repo.FindByDocument(ctx, "99.999.999/0001-99")
	require.NoError(t, err)
	assert.Equal(t, tenant.ID, found.ID)
}

func TestGormTenantRepository_FindByDocument_NotFound(t *testing.T) {
	repo, _ := setupTenantRepo(t)

	_, err := repo.FindByDocument(context.Background(), "00.000.000/0000-00")
	assert.ErrorIs(t, err, domain.ErrTenantNotFound)
}

func TestTenantRepository_BillingFields_Roundtrip(t *testing.T) {
	repo, _ := setupTenantRepo(t)

	tn, err := domain.NewTenant(uuid.New().String(), "BF", domain.TenantTypePJ, uuid.New().String()[:20])
	require.NoError(t, err)
	valor := int64(75000)
	dia := uint8(15)
	start := time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC)
	tn.SetBillingConfig("anual", &valor, &dia, &start, false)
	require.NoError(t, repo.Create(context.Background(), tn))

	got, err := repo.FindByID(context.Background(), tn.ID)
	require.NoError(t, err)
	assert.Equal(t, "anual", got.Plano)
	require.NotNil(t, got.ValorCobrancaCents)
	assert.Equal(t, int64(75000), *got.ValorCobrancaCents)
	require.NotNil(t, got.DiaVencimento)
	assert.Equal(t, uint8(15), *got.DiaVencimento)
	require.NotNil(t, got.DataInicioCobranca)
	assert.False(t, got.ExibirPagamentos)
}
