package infrastructure

import (
	"context"
	"os"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/sasrgita/crm-juridico/internal/shared/database"
	"github.com/sasrgita/crm-juridico/internal/shared/testhelper"
	tenantDomain "github.com/sasrgita/crm-juridico/internal/tenant/domain"
	tenantInfra "github.com/sasrgita/crm-juridico/internal/tenant/infrastructure"
	"github.com/sasrgita/crm-juridico/internal/whatsapp/domain"
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

func setupDB(t *testing.T) *gorm.DB {
	t.Helper()
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	db := sharedContainer.DB(t)
	log := sharedContainer.Logger()

	err := database.RunMigrations(db, log, "file://"+testhelper.MigrationsPath())
	require.NoError(t, err)

	db.Exec("DELETE FROM messages")
	db.Exec("DELETE FROM conversations")
	db.Exec("DELETE FROM contacts")
	db.Exec("DELETE FROM whatsapp_sessions")
	db.Exec("DELETE FROM specialist_documents")
	db.Exec("DELETE FROM specialist_mcps")
	db.Exec("DELETE FROM specialist_tenants")
	db.Exec("DELETE FROM scoring_configs")
	db.Exec("DELETE FROM steps")
	db.Exec("DELETE FROM guardrails")
	db.Exec("DELETE FROM specialists")
	db.Exec("DELETE FROM documents")
	db.Exec("DELETE FROM mcp_servers")
	db.Exec("DELETE FROM tenant_block_history")
	db.Exec("DELETE FROM user_tenants")
	db.Exec("DELETE FROM users")
	db.Exec("DELETE FROM tenants")

	return db
}

func createTenant(t *testing.T, db *gorm.DB) *tenantDomain.Tenant {
	t.Helper()
	repo := tenantInfra.NewGormTenantRepository(db)
	tenant, err := tenantDomain.NewTenant(uuid.New().String(), "Escritorio Teste", tenantDomain.TenantTypePJ, uuid.New().String()[:20])
	require.NoError(t, err)
	require.NoError(t, repo.Create(context.Background(), tenant))
	return tenant
}

func TestGormContactRepository_Create_And_FindByID(t *testing.T) {
	db := setupDB(t)
	repo := NewGormContactRepository(db)
	tenant := createTenant(t, db)
	ctx := context.Background()

	contact, err := domain.NewContact(uuid.New().String(), tenant.ID, "Maria Silva", "+5511999990001", "5511999990001@s.whatsapp.net")
	require.NoError(t, err)

	err = repo.Create(ctx, contact)
	require.NoError(t, err)

	found, err := repo.FindByID(ctx, contact.ID)
	require.NoError(t, err)
	assert.Equal(t, contact.ID, found.ID)
	assert.Equal(t, tenant.ID, found.TenantID)
	assert.Equal(t, "Maria Silva", found.Name)
	assert.Equal(t, "+5511999990001", found.Phone)
	assert.Equal(t, "5511999990001@s.whatsapp.net", found.WhatsAppID)
}

func TestGormContactRepository_FindByID_NotFound(t *testing.T) {
	db := setupDB(t)
	repo := NewGormContactRepository(db)

	_, err := repo.FindByID(context.Background(), "nonexistent")
	assert.ErrorIs(t, err, domain.ErrContactNotFound)
}

func TestGormContactRepository_FindByWhatsAppID(t *testing.T) {
	db := setupDB(t)
	repo := NewGormContactRepository(db)
	tenant := createTenant(t, db)
	ctx := context.Background()

	contact, _ := domain.NewContact(uuid.New().String(), tenant.ID, "Maria", "+5511999990001", "5511999990001@s.whatsapp.net")
	require.NoError(t, repo.Create(ctx, contact))

	found, err := repo.FindByWhatsAppID(ctx, tenant.ID, "5511999990001@s.whatsapp.net")
	require.NoError(t, err)
	assert.Equal(t, contact.ID, found.ID)
}

func TestGormContactRepository_FindByWhatsAppID_NotFound(t *testing.T) {
	db := setupDB(t)
	repo := NewGormContactRepository(db)
	tenant := createTenant(t, db)

	_, err := repo.FindByWhatsAppID(context.Background(), tenant.ID, "nonexistent@s.whatsapp.net")
	assert.ErrorIs(t, err, domain.ErrContactNotFound)
}

func TestGormContactRepository_FindByWhatsAppID_IsolatedByTenant(t *testing.T) {
	db := setupDB(t)
	repo := NewGormContactRepository(db)
	tenantA := createTenant(t, db)
	tenantB := createTenant(t, db)
	ctx := context.Background()

	contact, _ := domain.NewContact(uuid.New().String(), tenantA.ID, "Maria", "+5511999990001", "5511999990001@s.whatsapp.net")
	require.NoError(t, repo.Create(ctx, contact))

	_, err := repo.FindByWhatsAppID(ctx, tenantB.ID, "5511999990001@s.whatsapp.net")
	assert.ErrorIs(t, err, domain.ErrContactNotFound)
}

func TestGormContactRepository_Update(t *testing.T) {
	db := setupDB(t)
	repo := NewGormContactRepository(db)
	tenant := createTenant(t, db)
	ctx := context.Background()

	contact, _ := domain.NewContact(uuid.New().String(), tenant.ID, "Maria", "+5511999990001", "jid@s.whatsapp.net")
	require.NoError(t, repo.Create(ctx, contact))

	contact.UpdateName("Maria Silva")
	require.NoError(t, repo.Update(ctx, contact))

	found, err := repo.FindByID(ctx, contact.ID)
	require.NoError(t, err)
	assert.Equal(t, "Maria Silva", found.Name)
}

func TestGormContactRepository_DuplicateWhatsAppID_SameTenant_ReturnsError(t *testing.T) {
	db := setupDB(t)
	repo := NewGormContactRepository(db)
	tenant := createTenant(t, db)
	ctx := context.Background()

	c1, _ := domain.NewContact(uuid.New().String(), tenant.ID, "Maria", "+5511999990001", "same@s.whatsapp.net")
	require.NoError(t, repo.Create(ctx, c1))

	c2, _ := domain.NewContact(uuid.New().String(), tenant.ID, "Joao", "+5511999990002", "same@s.whatsapp.net")
	err := repo.Create(ctx, c2)
	assert.Error(t, err)
}

func TestGormContactRepository_SameWhatsAppID_DifferentTenants_OK(t *testing.T) {
	db := setupDB(t)
	repo := NewGormContactRepository(db)
	tenantA := createTenant(t, db)
	tenantB := createTenant(t, db)
	ctx := context.Background()

	c1, _ := domain.NewContact(uuid.New().String(), tenantA.ID, "Maria", "+5511999990001", "same@s.whatsapp.net")
	require.NoError(t, repo.Create(ctx, c1))

	c2, _ := domain.NewContact(uuid.New().String(), tenantB.ID, "Maria", "+5511999990001", "same@s.whatsapp.net")
	err := repo.Create(ctx, c2)
	assert.NoError(t, err)
}
