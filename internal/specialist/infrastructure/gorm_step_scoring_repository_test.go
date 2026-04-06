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

func setupStepRepo(t *testing.T) (*GormStepRepository, *gorm.DB) {
	t.Helper()
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	db := sharedContainer.DB(t)
	log := sharedContainer.Logger()

	err := database.RunMigrations(db, log, "file://"+testhelper.MigrationsPath())
	require.NoError(t, err)

	db.Exec("DELETE FROM scoring_configs")
	db.Exec("DELETE FROM guardrails")
	db.Exec("DELETE FROM steps")
	db.Exec("DELETE FROM specialist_tenants")
	db.Exec("DELETE FROM specialists")
	db.Exec("DELETE FROM tenant_block_history")
	db.Exec("DELETE FROM user_tenants")
	db.Exec("DELETE FROM users")
	db.Exec("DELETE FROM tenants")

	return NewGormStepRepository(db), db
}

func setupScoringRepo(t *testing.T) (*GormScoringConfigRepository, *gorm.DB) {
	t.Helper()
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	db := sharedContainer.DB(t)
	log := sharedContainer.Logger()

	err := database.RunMigrations(db, log, "file://"+testhelper.MigrationsPath())
	require.NoError(t, err)

	db.Exec("DELETE FROM scoring_configs")
	db.Exec("DELETE FROM guardrails")
	db.Exec("DELETE FROM steps")
	db.Exec("DELETE FROM specialist_tenants")
	db.Exec("DELETE FROM specialists")
	db.Exec("DELETE FROM tenant_block_history")
	db.Exec("DELETE FROM user_tenants")
	db.Exec("DELETE FROM users")
	db.Exec("DELETE FROM tenants")

	return NewGormScoringConfigRepository(db), db
}

func createTestSpecialistForStep(t *testing.T, db *gorm.DB) string {
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

// --- Step Repository Tests ---

func TestGormStepRepository_Create_And_FindByID(t *testing.T) {
	repo, db := setupStepRepo(t)

	ctx := context.Background()
	specialistID := createTestSpecialistForStep(t, db)

	step, err := domain.NewStep(uuid.New().String(), specialistID, 1, "Qual o nome do cliente?", domain.StepDataTypeFreeText, true, 10)
	require.NoError(t, err)

	err = repo.Create(ctx, step)
	require.NoError(t, err)

	found, err := repo.FindByID(ctx, step.ID)
	require.NoError(t, err)
	assert.Equal(t, step.ID, found.ID)
	assert.Equal(t, step.SpecialistID, found.SpecialistID)
	assert.Equal(t, 1, found.OrderIndex)
	assert.Equal(t, "Qual o nome do cliente?", found.Text)
	assert.Equal(t, domain.StepDataTypeFreeText, found.DataType)
	assert.True(t, found.Required)
	assert.Equal(t, 10, found.Score)
}

func TestGormStepRepository_FindByID_NotFound(t *testing.T) {
	repo, _ := setupStepRepo(t)

	_, err := repo.FindByID(context.Background(), uuid.New().String())
	assert.ErrorIs(t, err, domain.ErrStepNotFound)
}

func TestGormStepRepository_Update_Success(t *testing.T) {
	repo, db := setupStepRepo(t)

	ctx := context.Background()
	specialistID := createTestSpecialistForStep(t, db)

	step, err := domain.NewStep(uuid.New().String(), specialistID, 1, "Pergunta original", domain.StepDataTypeFreeText, true, 5)
	require.NoError(t, err)
	require.NoError(t, repo.Create(ctx, step))

	require.NoError(t, step.Update("Pergunta atualizada", domain.StepDataTypeNumber, false, 20))
	err = repo.Update(ctx, step)
	require.NoError(t, err)

	found, err := repo.FindByID(ctx, step.ID)
	require.NoError(t, err)
	assert.Equal(t, "Pergunta atualizada", found.Text)
	assert.Equal(t, domain.StepDataTypeNumber, found.DataType)
	assert.False(t, found.Required)
	assert.Equal(t, 20, found.Score)
}

func TestGormStepRepository_Delete_Success(t *testing.T) {
	repo, db := setupStepRepo(t)

	ctx := context.Background()
	specialistID := createTestSpecialistForStep(t, db)

	step, err := domain.NewStep(uuid.New().String(), specialistID, 1, "Pergunta para deletar", domain.StepDataTypeFreeText, true, 0)
	require.NoError(t, err)
	require.NoError(t, repo.Create(ctx, step))

	err = repo.Delete(ctx, step.ID)
	require.NoError(t, err)

	_, err = repo.FindByID(ctx, step.ID)
	assert.ErrorIs(t, err, domain.ErrStepNotFound)
}

func TestGormStepRepository_FindBySpecialistID_OrderedByIndex(t *testing.T) {
	repo, db := setupStepRepo(t)

	ctx := context.Background()
	specialistID := createTestSpecialistForStep(t, db)

	step3, err := domain.NewStep(uuid.New().String(), specialistID, 3, "Pergunta 3", domain.StepDataTypeFreeText, true, 0)
	require.NoError(t, err)
	require.NoError(t, repo.Create(ctx, step3))

	step1, err := domain.NewStep(uuid.New().String(), specialistID, 1, "Pergunta 1", domain.StepDataTypeFreeText, true, 0)
	require.NoError(t, err)
	require.NoError(t, repo.Create(ctx, step1))

	step2, err := domain.NewStep(uuid.New().String(), specialistID, 2, "Pergunta 2", domain.StepDataTypeNumber, false, 10)
	require.NoError(t, err)
	require.NoError(t, repo.Create(ctx, step2))

	steps, err := repo.FindBySpecialistID(ctx, specialistID)
	require.NoError(t, err)
	require.Len(t, steps, 3)
	assert.Equal(t, 1, steps[0].OrderIndex)
	assert.Equal(t, 2, steps[1].OrderIndex)
	assert.Equal(t, 3, steps[2].OrderIndex)
}

func TestGormStepRepository_GetMaxOrderIndex(t *testing.T) {
	repo, db := setupStepRepo(t)

	ctx := context.Background()
	specialistID := createTestSpecialistForStep(t, db)

	// When no steps exist, should return 0
	maxOrder, err := repo.GetMaxOrderIndex(ctx, specialistID)
	require.NoError(t, err)
	assert.Equal(t, 0, maxOrder)

	// After creating steps, should return the highest order index
	step1, err := domain.NewStep(uuid.New().String(), specialistID, 1, "Pergunta 1", domain.StepDataTypeFreeText, true, 0)
	require.NoError(t, err)
	require.NoError(t, repo.Create(ctx, step1))

	step2, err := domain.NewStep(uuid.New().String(), specialistID, 2, "Pergunta 2", domain.StepDataTypeFreeText, true, 0)
	require.NoError(t, err)
	require.NoError(t, repo.Create(ctx, step2))

	step5, err := domain.NewStep(uuid.New().String(), specialistID, 5, "Pergunta 5", domain.StepDataTypeFreeText, true, 0)
	require.NoError(t, err)
	require.NoError(t, repo.Create(ctx, step5))

	maxOrder, err = repo.GetMaxOrderIndex(ctx, specialistID)
	require.NoError(t, err)
	assert.Equal(t, 5, maxOrder)
}

func TestGormStepRepository_ReorderAfterDelete(t *testing.T) {
	repo, db := setupStepRepo(t)

	ctx := context.Background()
	specialistID := createTestSpecialistForStep(t, db)

	step1, err := domain.NewStep(uuid.New().String(), specialistID, 1, "Pergunta 1", domain.StepDataTypeFreeText, true, 0)
	require.NoError(t, err)
	require.NoError(t, repo.Create(ctx, step1))

	step2, err := domain.NewStep(uuid.New().String(), specialistID, 2, "Pergunta 2", domain.StepDataTypeFreeText, true, 0)
	require.NoError(t, err)
	require.NoError(t, repo.Create(ctx, step2))

	step3, err := domain.NewStep(uuid.New().String(), specialistID, 3, "Pergunta 3", domain.StepDataTypeFreeText, true, 0)
	require.NoError(t, err)
	require.NoError(t, repo.Create(ctx, step3))

	// Simulate deleting step at index 1 and reordering the rest
	require.NoError(t, repo.Delete(ctx, step1.ID))
	err = repo.ReorderAfterDelete(ctx, specialistID, 1)
	require.NoError(t, err)

	// step2 (was index 2) should now be index 1
	found2, err := repo.FindByID(ctx, step2.ID)
	require.NoError(t, err)
	assert.Equal(t, 1, found2.OrderIndex)

	// step3 (was index 3) should now be index 2
	found3, err := repo.FindByID(ctx, step3.ID)
	require.NoError(t, err)
	assert.Equal(t, 2, found3.OrderIndex)
}

func TestGormStepRepository_SwapOrder(t *testing.T) {
	repo, db := setupStepRepo(t)

	ctx := context.Background()
	specialistID := createTestSpecialistForStep(t, db)

	step1, err := domain.NewStep(uuid.New().String(), specialistID, 1, "Pergunta 1", domain.StepDataTypeFreeText, true, 0)
	require.NoError(t, err)
	require.NoError(t, repo.Create(ctx, step1))

	step2, err := domain.NewStep(uuid.New().String(), specialistID, 2, "Pergunta 2", domain.StepDataTypeFreeText, true, 0)
	require.NoError(t, err)
	require.NoError(t, repo.Create(ctx, step2))

	err = repo.SwapOrder(ctx, step1.ID, 1, step2.ID, 2)
	require.NoError(t, err)

	found1, err := repo.FindByID(ctx, step1.ID)
	require.NoError(t, err)
	assert.Equal(t, 2, found1.OrderIndex)

	found2, err := repo.FindByID(ctx, step2.ID)
	require.NoError(t, err)
	assert.Equal(t, 1, found2.OrderIndex)
}

// --- Scoring Config Repository Tests ---

func TestGormScoringConfigRepository_CreateOrUpdate_New(t *testing.T) {
	repo, db := setupScoringRepo(t)

	ctx := context.Background()
	specialistID := createTestSpecialistForStep(t, db)

	config, err := domain.NewScoringConfig(uuid.New().String(), specialistID, 70)
	require.NoError(t, err)

	err = repo.CreateOrUpdate(ctx, config)
	require.NoError(t, err)

	found, err := repo.FindBySpecialistID(ctx, specialistID)
	require.NoError(t, err)
	assert.Equal(t, config.ID, found.ID)
	assert.Equal(t, specialistID, found.SpecialistID)
	assert.Equal(t, 70, found.Threshold)
}

func TestGormScoringConfigRepository_CreateOrUpdate_Existing(t *testing.T) {
	repo, db := setupScoringRepo(t)

	ctx := context.Background()
	specialistID := createTestSpecialistForStep(t, db)

	config, err := domain.NewScoringConfig(uuid.New().String(), specialistID, 50)
	require.NoError(t, err)
	require.NoError(t, repo.CreateOrUpdate(ctx, config))

	// Update the threshold
	require.NoError(t, config.UpdateThreshold(80, 100))
	err = repo.CreateOrUpdate(ctx, config)
	require.NoError(t, err)

	found, err := repo.FindBySpecialistID(ctx, specialistID)
	require.NoError(t, err)
	assert.Equal(t, config.ID, found.ID)
	assert.Equal(t, 80, found.Threshold)
}

func TestGormScoringConfigRepository_FindBySpecialistID_NotFound(t *testing.T) {
	repo, db := setupScoringRepo(t)

	ctx := context.Background()
	specialistID := createTestSpecialistForStep(t, db)

	_, err := repo.FindBySpecialistID(ctx, specialistID)
	assert.ErrorIs(t, err, domain.ErrScoringConfigNotFound)
}
