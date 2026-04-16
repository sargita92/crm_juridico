package infrastructure

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/sasrgita/crm-juridico/internal/shared/database"
	"github.com/sasrgita/crm-juridico/internal/shared/testhelper"
	"github.com/sasrgita/crm-juridico/internal/specialist/domain"
)

func setupToolRepo(t *testing.T) (*GormSpecialistToolRepository, *gorm.DB) {
	t.Helper()
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	db := sharedContainer.DB(t)
	log := sharedContainer.Logger()

	err := database.RunMigrations(db, log, "file://"+testhelper.MigrationsPath())
	require.NoError(t, err)

	db.Exec("DELETE FROM specialist_tools")
	db.Exec("DELETE FROM specialist_tenants")
	db.Exec("DELETE FROM specialists")
	db.Exec("DELETE FROM tenant_block_history")
	db.Exec("DELETE FROM user_tenants")
	db.Exec("DELETE FROM users")
	db.Exec("DELETE FROM tenants")

	return NewGormSpecialistToolRepository(db), db
}

func createTestSpecialistForTool(t *testing.T, db *gorm.DB) string {
	t.Helper()
	id := uuid.New().String()
	err := db.Table("specialists").Create(map[string]interface{}{
		"id":          id,
		"name":        "Test Specialist",
		"description": "desc",
		"prompt":      "prompt",
		"status":      "active",
	}).Error
	require.NoError(t, err)
	return id
}

// --- SpecialistTool Repository Tests ---

func TestGormSpecialistToolRepository_Associate_WithValidData_ReturnsNoError(t *testing.T) {
	repo, db := setupToolRepo(t)

	ctx := context.Background()
	specialistID := createTestSpecialistForTool(t, db)

	err := repo.Associate(ctx, specialistID, "search_web")
	require.NoError(t, err)

	err = repo.Associate(ctx, specialistID, "send_email")
	require.NoError(t, err)

	names, err := repo.FindToolNamesBySpecialistID(ctx, specialistID)
	require.NoError(t, err)
	assert.Len(t, names, 2)
	assert.Contains(t, names, "search_web")
	assert.Contains(t, names, "send_email")
}

func TestGormSpecialistToolRepository_Dissociate_WithAssociatedTool_ReturnsNoError(t *testing.T) {
	repo, db := setupToolRepo(t)

	ctx := context.Background()
	specialistID := createTestSpecialistForTool(t, db)

	require.NoError(t, repo.Associate(ctx, specialistID, "search_web"))

	err := repo.Dissociate(ctx, specialistID, "search_web")
	require.NoError(t, err)

	names, err := repo.FindToolNamesBySpecialistID(ctx, specialistID)
	require.NoError(t, err)
	assert.Empty(t, names)
}

func TestGormSpecialistToolRepository_Associate_WithDuplicateTool_ReturnsErrToolAlreadyAssociated(t *testing.T) {
	repo, db := setupToolRepo(t)

	ctx := context.Background()
	specialistID := createTestSpecialistForTool(t, db)

	require.NoError(t, repo.Associate(ctx, specialistID, "search_web"))

	err := repo.Associate(ctx, specialistID, "search_web")
	assert.ErrorIs(t, err, domain.ErrToolAlreadyAssociated)
}

func TestGormSpecialistToolRepository_Dissociate_WithNonExistentTool_ReturnsErrToolNotAssociated(t *testing.T) {
	repo, db := setupToolRepo(t)

	ctx := context.Background()
	specialistID := createTestSpecialistForTool(t, db)

	err := repo.Dissociate(ctx, specialistID, "nonexistent_tool")
	assert.ErrorIs(t, err, domain.ErrToolNotAssociated)
}

func TestGormSpecialistToolRepository_FindAll_WithAssociatedTools_ReturnsFullEntities(t *testing.T) {
	repo, db := setupToolRepo(t)

	ctx := context.Background()
	specialistID := createTestSpecialistForTool(t, db)

	require.NoError(t, repo.Associate(ctx, specialistID, "search_web"))
	require.NoError(t, repo.Associate(ctx, specialistID, "send_email"))

	tools, err := repo.FindAll(ctx, specialistID)
	require.NoError(t, err)
	assert.Len(t, tools, 2)

	for _, tool := range tools {
		assert.NotEmpty(t, tool.ID)
		assert.Equal(t, specialistID, tool.SpecialistID)
		assert.NotEmpty(t, tool.ToolName)
		assert.False(t, tool.CreatedAt.IsZero())
	}

	toolNames := []string{tools[0].ToolName, tools[1].ToolName}
	assert.Contains(t, toolNames, "search_web")
	assert.Contains(t, toolNames, "send_email")
}
