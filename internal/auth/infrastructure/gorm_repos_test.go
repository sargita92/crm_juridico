package infrastructure

import (
	"context"
	"os"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/sasrgita/crm-juridico/internal/auth/domain"
	"github.com/sasrgita/crm-juridico/internal/shared/database"
	"github.com/sasrgita/crm-juridico/internal/shared/testhelper"
	tenantdomain "github.com/sasrgita/crm-juridico/internal/tenant/domain"
	tenantinfra "github.com/sasrgita/crm-juridico/internal/tenant/infrastructure"
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

type testRepos struct {
	userRepo       *GormUserRepository
	userTenantRepo *GormUserTenantRepository
	tenantRepo     *tenantinfra.GormTenantRepository
}

func setupAuthRepos(t *testing.T) *testRepos {
	t.Helper()
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	db := sharedContainer.DB(t)
	log := sharedContainer.Logger()

	err := database.RunMigrations(db, log, "file://"+testhelper.MigrationsPath())
	require.NoError(t, err)

	db.Exec("DELETE FROM user_tenants")
	db.Exec("DELETE FROM users")
	db.Exec("DELETE FROM tenants")

	return &testRepos{
		userRepo:       NewGormUserRepository(db),
		userTenantRepo: NewGormUserTenantRepository(db),
		tenantRepo:     tenantinfra.NewGormTenantRepository(db),
	}
}

func createTestTenant(t *testing.T, repos *testRepos, name, doc string) *tenantdomain.Tenant {
	t.Helper()
	tenant, err := tenantdomain.NewTenant(uuid.New().String(), name, tenantdomain.TenantTypePJ, doc)
	require.NoError(t, err)
	require.NoError(t, repos.tenantRepo.Create(context.Background(), tenant))
	return tenant
}

func createTestUser(t *testing.T, repos *testRepos, name, email string) *domain.User {
	t.Helper()
	user, err := domain.NewUser(uuid.New().String(), name, email, "$2a$10$fakehash1234567890123456789012345678901234567890", domain.UserRoleUser)
	require.NoError(t, err)
	require.NoError(t, repos.userRepo.Create(context.Background(), user))
	return user
}

// --- UserRepository tests ---

func TestGormUserRepository_Create_And_FindByEmail(t *testing.T) {
	repos := setupAuthRepos(t)

	user := createTestUser(t, repos, "João Silva", "joao@email.com")

	found, err := repos.userRepo.FindByEmail(context.Background(), "joao@email.com")
	require.NoError(t, err)
	assert.Equal(t, user.ID, found.ID)
	assert.Equal(t, user.Name, found.Name)
	assert.Equal(t, user.Email, found.Email)
	assert.Equal(t, domain.UserStatusActive, found.Status)
}

func TestGormUserRepository_Create_DuplicateEmail_ReturnsError(t *testing.T) {
	repos := setupAuthRepos(t)

	createTestUser(t, repos, "João", "joao@email.com")

	user2, _ := domain.NewUser(uuid.New().String(), "Outro João", "joao@email.com", "$2a$10$hash", domain.UserRoleUser)
	err := repos.userRepo.Create(context.Background(), user2)
	assert.ErrorIs(t, err, domain.ErrUserEmailExists)
}

func TestGormUserRepository_FindByEmail_NotFound(t *testing.T) {
	repos := setupAuthRepos(t)

	_, err := repos.userRepo.FindByEmail(context.Background(), "naoexiste@email.com")
	assert.ErrorIs(t, err, domain.ErrUserNotFound)
}

func TestGormUserRepository_FindByID(t *testing.T) {
	repos := setupAuthRepos(t)

	user := createTestUser(t, repos, "João", "joao@email.com")

	found, err := repos.userRepo.FindByID(context.Background(), user.ID)
	require.NoError(t, err)
	assert.Equal(t, user.Email, found.Email)
}

func TestGormUserRepository_ExistsByEmail(t *testing.T) {
	repos := setupAuthRepos(t)

	createTestUser(t, repos, "João", "joao@email.com")

	exists, err := repos.userRepo.ExistsByEmail(context.Background(), "joao@email.com")
	require.NoError(t, err)
	assert.True(t, exists)

	exists, err = repos.userRepo.ExistsByEmail(context.Background(), "naoexiste@email.com")
	require.NoError(t, err)
	assert.False(t, exists)
}

// --- UserTenantRepository tests ---

func TestGormUserTenantRepository_Associate_And_FindTenantIDs(t *testing.T) {
	repos := setupAuthRepos(t)

	ctx := context.Background()
	tenant1 := createTestTenant(t, repos, "Escritório A", "11.111.111/0001-11")
	tenant2 := createTestTenant(t, repos, "Escritório B", "22.222.222/0001-22")
	user := createTestUser(t, repos, "João", "joao@email.com")

	require.NoError(t, repos.userTenantRepo.Associate(ctx, user.ID, tenant1.ID))
	require.NoError(t, repos.userTenantRepo.Associate(ctx, user.ID, tenant2.ID))

	ids, err := repos.userTenantRepo.FindTenantIDsByUserID(ctx, user.ID)
	require.NoError(t, err)
	assert.Len(t, ids, 2)
	assert.Contains(t, ids, tenant1.ID)
	assert.Contains(t, ids, tenant2.ID)
}

func TestGormUserTenantRepository_FindTenantIDs_NoAssociations(t *testing.T) {
	repos := setupAuthRepos(t)

	ids, err := repos.userTenantRepo.FindTenantIDsByUserID(context.Background(), "nonexistent-user")
	require.NoError(t, err)
	assert.Empty(t, ids)
}

func TestGormUserRepository_PasswordHash_NotPlaintext(t *testing.T) {
	repos := setupAuthRepos(t)

	hasher := NewBcryptHasher()
	hash, err := hasher.Hash("password123")
	require.NoError(t, err)

	user, err := domain.NewUser(uuid.New().String(), "João", "joao@email.com", hash, domain.UserRoleUser)
	require.NoError(t, err)
	require.NoError(t, repos.userRepo.Create(context.Background(), user))

	found, err := repos.userRepo.FindByEmail(context.Background(), "joao@email.com")
	require.NoError(t, err)
	assert.NotEqual(t, "password123", found.PasswordHash)
	assert.NoError(t, hasher.Compare(found.PasswordHash, "password123"))
}
