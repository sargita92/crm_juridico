package infrastructure

import (
	"context"
	"os"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/sasrgita/crm-juridico/internal/mcp/domain"
	"github.com/sasrgita/crm-juridico/internal/shared/database"
	"github.com/sasrgita/crm-juridico/internal/shared/testhelper"
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

func setupRepos(t *testing.T) (*GormMcpServerRepository, *GormSpecialistMcpRepository, *gorm.DB) {
	t.Helper()
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	db := sharedContainer.DB(t)
	log := sharedContainer.Logger()
	err := database.RunMigrations(db, log, "file://"+testhelper.MigrationsPath())
	require.NoError(t, err)

	db.Exec("DELETE FROM specialist_mcps")
	db.Exec("DELETE FROM mcp_servers")
	db.Exec("DELETE FROM specialist_documents")
	db.Exec("DELETE FROM documents")
	db.Exec("DELETE FROM specialist_tenants")
	db.Exec("DELETE FROM specialists")

	return NewGormMcpServerRepository(db), NewGormSpecialistMcpRepository(db), db
}

func createTestMcp(t *testing.T, repo *GormMcpServerRepository, name, url string) *domain.McpServer {
	t.Helper()
	m, err := domain.NewMcpServer(uuid.New().String(), name, url, "", "")
	require.NoError(t, err)
	require.NoError(t, repo.Create(context.Background(), m))
	return m
}

func createTestSpecialist(t *testing.T, db *gorm.DB, name string) string {
	t.Helper()
	id := uuid.New().String()
	require.NoError(t, db.Exec("INSERT INTO specialists (id, name, description, prompt, status) VALUES (?, ?, '', 'test', 'active')", id, name).Error)
	return id
}

// --- McpServer Repository ---

func TestGormMcpServerRepository_Create_And_FindByID(t *testing.T) {
	repo, _, _ := setupRepos(t)
	m := createTestMcp(t, repo, "API Processos", "https://api.processos.com/mcp")

	found, err := repo.FindByID(context.Background(), m.ID)
	require.NoError(t, err)
	assert.Equal(t, m.Name, found.Name)
	assert.Equal(t, m.URL, found.URL)
}

func TestGormMcpServerRepository_FindByID_NotFound(t *testing.T) {
	repo, _, _ := setupRepos(t)
	_, err := repo.FindByID(context.Background(), "nonexistent")
	assert.ErrorIs(t, err, domain.ErrMcpNotFound)
}

func TestGormMcpServerRepository_Update_Success(t *testing.T) {
	repo, _, _ := setupRepos(t)
	ctx := context.Background()
	m := createTestMcp(t, repo, "Antigo", "https://old.com")

	require.NoError(t, m.Update("Novo", "https://new.com", `{"key":"val"}`, ""))
	require.NoError(t, repo.Update(ctx, m))

	found, _ := repo.FindByID(ctx, m.ID)
	assert.Equal(t, "Novo", found.Name)
	assert.Equal(t, "https://new.com", found.URL)
}

func TestGormMcpServerRepository_Delete_Success(t *testing.T) {
	repo, _, _ := setupRepos(t)
	ctx := context.Background()
	m := createTestMcp(t, repo, "Para excluir", "https://example.com")

	require.NoError(t, repo.Delete(ctx, m.ID))
	_, err := repo.FindByID(ctx, m.ID)
	assert.ErrorIs(t, err, domain.ErrMcpNotFound)
}

func TestGormMcpServerRepository_Delete_NotFound(t *testing.T) {
	repo, _, _ := setupRepos(t)
	err := repo.Delete(context.Background(), "nonexistent")
	assert.ErrorIs(t, err, domain.ErrMcpNotFound)
}

func TestGormMcpServerRepository_FindWithFilter_All(t *testing.T) {
	repo, _, _ := setupRepos(t)
	createTestMcp(t, repo, "MCP A", "https://a.com")
	createTestMcp(t, repo, "MCP B", "https://b.com")

	result, err := repo.FindWithFilter(context.Background(), domain.McpFilter{Page: 1, Limit: 20})
	require.NoError(t, err)
	assert.Equal(t, int64(2), result.Total)
}

func TestGormMcpServerRepository_FindWithFilter_Search(t *testing.T) {
	repo, _, _ := setupRepos(t)
	createTestMcp(t, repo, "API Processos", "https://processos.com")
	createTestMcp(t, repo, "API Contratos", "https://contratos.com")

	result, err := repo.FindWithFilter(context.Background(), domain.McpFilter{Search: "Processos", Page: 1, Limit: 20})
	require.NoError(t, err)
	assert.Equal(t, int64(1), result.Total)
}

func TestGormMcpServerRepository_FindByIDs(t *testing.T) {
	repo, _, _ := setupRepos(t)
	m1 := createTestMcp(t, repo, "MCP 1", "https://1.com")
	m2 := createTestMcp(t, repo, "MCP 2", "https://2.com")
	createTestMcp(t, repo, "MCP 3", "https://3.com")

	found, err := repo.FindByIDs(context.Background(), []string{m1.ID, m2.ID})
	require.NoError(t, err)
	assert.Len(t, found, 2)
}

// --- SpecialistMcp Repository ---

func TestGormSpecialistMcpRepository_Associate_And_Exists(t *testing.T) {
	mcpRepo, specMcpRepo, db := setupRepos(t)
	specID := createTestSpecialist(t, db, "Especialista")
	m := createTestMcp(t, mcpRepo, "MCP", "https://example.com")

	require.NoError(t, specMcpRepo.Associate(context.Background(), specID, m.ID))
	exists, err := specMcpRepo.Exists(context.Background(), specID, m.ID)
	require.NoError(t, err)
	assert.True(t, exists)
}

func TestGormSpecialistMcpRepository_Associate_Duplicate(t *testing.T) {
	mcpRepo, specMcpRepo, db := setupRepos(t)
	specID := createTestSpecialist(t, db, "Especialista")
	m := createTestMcp(t, mcpRepo, "MCP", "https://example.com")

	require.NoError(t, specMcpRepo.Associate(context.Background(), specID, m.ID))
	err := specMcpRepo.Associate(context.Background(), specID, m.ID)
	assert.ErrorIs(t, err, domain.ErrMcpAlreadyAssociated)
}

func TestGormSpecialistMcpRepository_Dissociate_Success(t *testing.T) {
	mcpRepo, specMcpRepo, db := setupRepos(t)
	ctx := context.Background()
	specID := createTestSpecialist(t, db, "Especialista")
	m := createTestMcp(t, mcpRepo, "MCP", "https://example.com")

	require.NoError(t, specMcpRepo.Associate(ctx, specID, m.ID))
	require.NoError(t, specMcpRepo.Dissociate(ctx, specID, m.ID))
	exists, _ := specMcpRepo.Exists(ctx, specID, m.ID)
	assert.False(t, exists)
}

func TestGormSpecialistMcpRepository_Dissociate_NotAssociated(t *testing.T) {
	_, specMcpRepo, _ := setupRepos(t)
	err := specMcpRepo.Dissociate(context.Background(), "s-1", "m-1")
	assert.ErrorIs(t, err, domain.ErrMcpNotAssociated)
}

func TestGormSpecialistMcpRepository_DissociateAllByMcpID(t *testing.T) {
	mcpRepo, specMcpRepo, db := setupRepos(t)
	ctx := context.Background()
	s1 := createTestSpecialist(t, db, "E1")
	s2 := createTestSpecialist(t, db, "E2")
	m := createTestMcp(t, mcpRepo, "Shared", "https://shared.com")

	require.NoError(t, specMcpRepo.Associate(ctx, s1, m.ID))
	require.NoError(t, specMcpRepo.Associate(ctx, s2, m.ID))
	require.NoError(t, specMcpRepo.DissociateAllByMcpID(ctx, m.ID))

	e1, _ := specMcpRepo.Exists(ctx, s1, m.ID)
	e2, _ := specMcpRepo.Exists(ctx, s2, m.ID)
	assert.False(t, e1)
	assert.False(t, e2)
}

func TestGormSpecialistMcpRepository_FindMcpIDsBySpecialistID(t *testing.T) {
	mcpRepo, specMcpRepo, db := setupRepos(t)
	ctx := context.Background()
	specID := createTestSpecialist(t, db, "E1")
	m1 := createTestMcp(t, mcpRepo, "M1", "https://1.com")
	m2 := createTestMcp(t, mcpRepo, "M2", "https://2.com")

	require.NoError(t, specMcpRepo.Associate(ctx, specID, m1.ID))
	require.NoError(t, specMcpRepo.Associate(ctx, specID, m2.ID))

	ids, err := specMcpRepo.FindMcpIDsBySpecialistID(ctx, specID)
	require.NoError(t, err)
	assert.Len(t, ids, 2)
}

func TestGormSpecialistMcpRepository_CountByMcpID(t *testing.T) {
	mcpRepo, specMcpRepo, db := setupRepos(t)
	ctx := context.Background()
	s1 := createTestSpecialist(t, db, "E1")
	s2 := createTestSpecialist(t, db, "E2")
	m := createTestMcp(t, mcpRepo, "Shared", "https://shared.com")

	require.NoError(t, specMcpRepo.Associate(ctx, s1, m.ID))
	require.NoError(t, specMcpRepo.Associate(ctx, s2, m.ID))

	count, err := specMcpRepo.CountByMcpID(ctx, m.ID)
	require.NoError(t, err)
	assert.Equal(t, 2, count)
}
