package infrastructure

import (
	"context"
	"os"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/sasrgita/crm-juridico/internal/shared/database"
	"github.com/sasrgita/crm-juridico/internal/shared/testhelper"
	"github.com/sasrgita/crm-juridico/internal/specialist/domain"
	tenantdomain "github.com/sasrgita/crm-juridico/internal/tenant/domain"
)

var sharedContainer *testhelper.MySQLContainer

func TestMain(m *testing.M) {
	short := false
	for _, arg := range os.Args {
		if arg == "-test.short" || arg == "-short" {
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

func setupRepos(t *testing.T) (*GormSpecialistRepository, *GormSpecialistTenantRepository, *gorm.DB) {
	t.Helper()
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	db := sharedContainer.DB(t)
	log := sharedContainer.Logger()

	err := database.RunMigrations(db, log, "file://"+testhelper.MigrationsPath())
	require.NoError(t, err)

	db.Exec("DELETE FROM specialist_tenants")
	db.Exec("DELETE FROM specialists")
	db.Exec("DELETE FROM tenant_block_history")
	db.Exec("DELETE FROM user_tenants")
	db.Exec("DELETE FROM users")
	db.Exec("DELETE FROM tenants")

	return NewGormSpecialistRepository(db), NewGormSpecialistTenantRepository(db), db
}

func createTestTenant(t *testing.T, db *gorm.DB, name, document string) *tenantdomain.Tenant {
	t.Helper()
	tenant, err := tenantdomain.NewTenant(uuid.New().String(), name, tenantdomain.TenantTypePJ, document)
	require.NoError(t, err)

	model := struct {
		ID          string `gorm:"primaryKey;column:id;type:char(36)"`
		Name        string `gorm:"column:name"`
		Type        string `gorm:"column:type"`
		Document    string `gorm:"column:document"`
		Status      string `gorm:"column:status"`
		BlockReason string `gorm:"column:block_reason"`
	}{
		ID:       tenant.ID,
		Name:     tenant.Name,
		Type:     string(tenant.Type),
		Document: tenant.Document,
		Status:   string(tenant.Status),
	}

	err = db.Table("tenants").Create(&model).Error
	require.NoError(t, err)
	return tenant
}

// --- Specialist Repository Tests ---

func TestGormSpecialistRepository_Create_And_FindByID(t *testing.T) {
	repo, _, _ := setupRepos(t)

	s, err := domain.NewSpecialist(uuid.New().String(), "Assistente Juridico", "Atende leads", "Voce e um assistente...")
	require.NoError(t, err)

	err = repo.Create(context.Background(), s)
	require.NoError(t, err)

	found, err := repo.FindByID(context.Background(), s.ID)
	require.NoError(t, err)
	assert.Equal(t, s.ID, found.ID)
	assert.Equal(t, s.Name, found.Name)
	assert.Equal(t, s.Description, found.Description)
	assert.Equal(t, s.Prompt, found.Prompt)
	assert.Equal(t, domain.SpecialistStatusActive, found.Status)
}

func TestGormSpecialistRepository_FindByID_NotFound(t *testing.T) {
	repo, _, _ := setupRepos(t)

	_, err := repo.FindByID(context.Background(), "nonexistent-id")
	assert.ErrorIs(t, err, domain.ErrSpecialistNotFound)
}

func TestGormSpecialistRepository_Update_Success(t *testing.T) {
	repo, _, _ := setupRepos(t)

	ctx := context.Background()
	s, _ := domain.NewSpecialist(uuid.New().String(), "Nome Antigo", "desc", "prompt antigo")
	require.NoError(t, repo.Create(ctx, s))

	require.NoError(t, s.Update("Nome Novo", "desc nova", "prompt novo"))
	err := repo.Update(ctx, s)
	require.NoError(t, err)

	found, err := repo.FindByID(ctx, s.ID)
	require.NoError(t, err)
	assert.Equal(t, "Nome Novo", found.Name)
	assert.Equal(t, "desc nova", found.Description)
	assert.Equal(t, "prompt novo", found.Prompt)
}

func TestGormSpecialistRepository_Update_DeactivateAndActivate(t *testing.T) {
	repo, _, _ := setupRepos(t)

	ctx := context.Background()
	s, _ := domain.NewSpecialist(uuid.New().String(), "Especialista", "desc", "prompt")
	require.NoError(t, repo.Create(ctx, s))

	require.NoError(t, s.Deactivate())
	require.NoError(t, repo.Update(ctx, s))

	found, _ := repo.FindByID(ctx, s.ID)
	assert.Equal(t, domain.SpecialistStatusInactive, found.Status)

	require.NoError(t, s.Activate())
	require.NoError(t, repo.Update(ctx, s))

	found, _ = repo.FindByID(ctx, s.ID)
	assert.Equal(t, domain.SpecialistStatusActive, found.Status)
}

func TestGormSpecialistRepository_FindWithFilter_SearchByName(t *testing.T) {
	repo, _, _ := setupRepos(t)

	ctx := context.Background()
	s1, _ := domain.NewSpecialist(uuid.New().String(), "Assistente Juridico", "", "prompt1")
	s2, _ := domain.NewSpecialist(uuid.New().String(), "Assistente Comercial", "", "prompt2")
	require.NoError(t, repo.Create(ctx, s1))
	require.NoError(t, repo.Create(ctx, s2))

	result, err := repo.FindWithFilter(ctx, domain.SpecialistFilter{Search: "Juridico", Page: 1, Limit: 20})
	require.NoError(t, err)
	assert.Equal(t, int64(1), result.Total)
	assert.Equal(t, "Assistente Juridico", result.Specialists[0].Name)
}

func TestGormSpecialistRepository_FindWithFilter_FilterByStatus(t *testing.T) {
	repo, _, _ := setupRepos(t)

	ctx := context.Background()
	active, _ := domain.NewSpecialist(uuid.New().String(), "Ativo", "", "prompt1")
	inactive, _ := domain.NewSpecialist(uuid.New().String(), "Inativo", "", "prompt2")
	require.NoError(t, repo.Create(ctx, active))
	require.NoError(t, repo.Create(ctx, inactive))
	require.NoError(t, inactive.Deactivate())
	require.NoError(t, repo.Update(ctx, inactive))

	result, err := repo.FindWithFilter(ctx, domain.SpecialistFilter{Status: domain.SpecialistStatusInactive, Page: 1, Limit: 20})
	require.NoError(t, err)
	assert.Equal(t, int64(1), result.Total)
	assert.Equal(t, "Inativo", result.Specialists[0].Name)
}

func TestGormSpecialistRepository_FindWithFilter_Pagination(t *testing.T) {
	repo, _, _ := setupRepos(t)

	ctx := context.Background()
	for i := 0; i < 5; i++ {
		s, _ := domain.NewSpecialist(uuid.New().String(), "Especialista", "", "prompt")
		require.NoError(t, repo.Create(ctx, s))
	}

	result, err := repo.FindWithFilter(ctx, domain.SpecialistFilter{Page: 1, Limit: 2})
	require.NoError(t, err)
	assert.Equal(t, int64(5), result.Total)
	assert.Len(t, result.Specialists, 2)
	assert.Equal(t, 1, result.Page)
	assert.Equal(t, 2, result.Limit)
}

func TestGormSpecialistRepository_FindWithFilter_DefaultPageAndLimit(t *testing.T) {
	repo, _, _ := setupRepos(t)

	result, err := repo.FindWithFilter(context.Background(), domain.SpecialistFilter{})
	require.NoError(t, err)
	assert.Equal(t, 1, result.Page)
	assert.Equal(t, 20, result.Limit)
}

// --- SpecialistTenant Repository Tests ---

func TestGormSpecialistTenantRepository_Associate_And_FindTenantIDs(t *testing.T) {
	specRepo, stRepo, db := setupRepos(t)

	ctx := context.Background()
	s, _ := domain.NewSpecialist(uuid.New().String(), "Especialista", "", "prompt")
	require.NoError(t, specRepo.Create(ctx, s))

	tenant := createTestTenant(t, db, "Escritorio A", "11.111.111/0001-11")

	err := stRepo.Associate(ctx, s.ID, tenant.ID)
	require.NoError(t, err)

	ids, err := stRepo.FindTenantIDsBySpecialistID(ctx, s.ID)
	require.NoError(t, err)
	assert.Len(t, ids, 1)
	assert.Equal(t, tenant.ID, ids[0])
}

func TestGormSpecialistTenantRepository_Associate_Duplicate_ReturnsError(t *testing.T) {
	specRepo, stRepo, db := setupRepos(t)

	ctx := context.Background()
	s, _ := domain.NewSpecialist(uuid.New().String(), "Especialista", "", "prompt")
	require.NoError(t, specRepo.Create(ctx, s))

	tenant := createTestTenant(t, db, "Escritorio A", "11.111.111/0001-11")

	require.NoError(t, stRepo.Associate(ctx, s.ID, tenant.ID))
	err := stRepo.Associate(ctx, s.ID, tenant.ID)
	assert.ErrorIs(t, err, domain.ErrTenantAlreadyAssociated)
}

func TestGormSpecialistTenantRepository_Dissociate_Success(t *testing.T) {
	specRepo, stRepo, db := setupRepos(t)

	ctx := context.Background()
	s, _ := domain.NewSpecialist(uuid.New().String(), "Especialista", "", "prompt")
	require.NoError(t, specRepo.Create(ctx, s))

	tenant := createTestTenant(t, db, "Escritorio A", "11.111.111/0001-11")
	require.NoError(t, stRepo.Associate(ctx, s.ID, tenant.ID))

	err := stRepo.Dissociate(ctx, s.ID, tenant.ID)
	require.NoError(t, err)

	ids, err := stRepo.FindTenantIDsBySpecialistID(ctx, s.ID)
	require.NoError(t, err)
	assert.Empty(t, ids)
}

func TestGormSpecialistTenantRepository_Dissociate_NotAssociated_ReturnsError(t *testing.T) {
	specRepo, stRepo, _ := setupRepos(t)

	ctx := context.Background()
	s, _ := domain.NewSpecialist(uuid.New().String(), "Especialista", "", "prompt")
	require.NoError(t, specRepo.Create(ctx, s))

	err := stRepo.Dissociate(ctx, s.ID, "nonexistent-tenant-id")
	assert.ErrorIs(t, err, domain.ErrTenantNotAssociated)
}

func TestGormSpecialistTenantRepository_FindSpecialistIDsByTenantID(t *testing.T) {
	specRepo, stRepo, db := setupRepos(t)

	ctx := context.Background()
	s1, _ := domain.NewSpecialist(uuid.New().String(), "Especialista 1", "", "prompt1")
	s2, _ := domain.NewSpecialist(uuid.New().String(), "Especialista 2", "", "prompt2")
	require.NoError(t, specRepo.Create(ctx, s1))
	require.NoError(t, specRepo.Create(ctx, s2))

	tenant := createTestTenant(t, db, "Escritorio A", "11.111.111/0001-11")
	require.NoError(t, stRepo.Associate(ctx, s1.ID, tenant.ID))
	require.NoError(t, stRepo.Associate(ctx, s2.ID, tenant.ID))

	ids, err := stRepo.FindSpecialistIDsByTenantID(ctx, tenant.ID)
	require.NoError(t, err)
	assert.Len(t, ids, 2)
}

func TestGormSpecialistTenantRepository_Exists(t *testing.T) {
	specRepo, stRepo, db := setupRepos(t)

	ctx := context.Background()
	s, _ := domain.NewSpecialist(uuid.New().String(), "Especialista", "", "prompt")
	require.NoError(t, specRepo.Create(ctx, s))

	tenant := createTestTenant(t, db, "Escritorio A", "11.111.111/0001-11")

	exists, err := stRepo.Exists(ctx, s.ID, tenant.ID)
	require.NoError(t, err)
	assert.False(t, exists)

	require.NoError(t, stRepo.Associate(ctx, s.ID, tenant.ID))

	exists, err = stRepo.Exists(ctx, s.ID, tenant.ID)
	require.NoError(t, err)
	assert.True(t, exists)
}
