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

func setupGuardrailRepo(t *testing.T) (*GormGuardrailRepository, *gorm.DB) {
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

	return NewGormGuardrailRepository(db), db
}

func createTestSpecialistForGuardrail(t *testing.T, db *gorm.DB) string {
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

// --- Guardrail Repository Tests ---

func TestGormGuardrailRepository_Create_And_FindByID(t *testing.T) {
	repo, db := setupGuardrailRepo(t)

	ctx := context.Background()
	specialistID := createTestSpecialistForGuardrail(t, db)

	g, err := domain.NewGuardrail(uuid.New().String(), specialistID, "nome-teste", domain.GuardrailTypeForbiddenTopics, "Nao falar sobre concorrentes", "Desculpe, nao posso ajudar com isso.")
	require.NoError(t, err)

	err = repo.Create(ctx, g)
	require.NoError(t, err)

	found, err := repo.FindByID(ctx, g.ID)
	require.NoError(t, err)
	assert.Equal(t, g.ID, found.ID)
	assert.Equal(t, g.SpecialistID, found.SpecialistID)
	assert.Equal(t, domain.GuardrailTypeForbiddenTopics, found.Type)
	assert.Equal(t, g.Rule, found.Rule)
	assert.Equal(t, g.Message, found.Message)
	assert.True(t, found.Active)
}

func TestGormGuardrailRepository_FindByID_NotFound(t *testing.T) {
	repo, _ := setupGuardrailRepo(t)

	_, err := repo.FindByID(context.Background(), uuid.New().String())
	assert.ErrorIs(t, err, domain.ErrGuardrailNotFound)
}

func TestGormGuardrailRepository_Update_Success(t *testing.T) {
	repo, db := setupGuardrailRepo(t)

	ctx := context.Background()
	specialistID := createTestSpecialistForGuardrail(t, db)

	g, err := domain.NewGuardrail(uuid.New().String(), specialistID, "nome-teste", domain.GuardrailTypeForbiddenTopics, "Regra original", "Mensagem original")
	require.NoError(t, err)
	require.NoError(t, repo.Create(ctx, g))

	require.NoError(t, g.Update("nome-teste", domain.GuardrailTypeScopeLimit, "Regra atualizada", "Mensagem atualizada"))
	err = repo.Update(ctx, g)
	require.NoError(t, err)

	found, err := repo.FindByID(ctx, g.ID)
	require.NoError(t, err)
	assert.Equal(t, domain.GuardrailTypeScopeLimit, found.Type)
	assert.Equal(t, "Regra atualizada", found.Rule)
	assert.Equal(t, "Mensagem atualizada", found.Message)
}

func TestGormGuardrailRepository_Delete_Success(t *testing.T) {
	repo, db := setupGuardrailRepo(t)

	ctx := context.Background()
	specialistID := createTestSpecialistForGuardrail(t, db)

	g, err := domain.NewGuardrail(uuid.New().String(), specialistID, "nome-teste", domain.GuardrailTypeResponseTone, "Sempre ser educado", "")
	require.NoError(t, err)
	require.NoError(t, repo.Create(ctx, g))

	err = repo.Delete(ctx, g.ID)
	require.NoError(t, err)

	_, err = repo.FindByID(ctx, g.ID)
	assert.ErrorIs(t, err, domain.ErrGuardrailNotFound)
}

func TestGormGuardrailRepository_Delete_NotFound(t *testing.T) {
	repo, _ := setupGuardrailRepo(t)

	err := repo.Delete(context.Background(), uuid.New().String())
	assert.ErrorIs(t, err, domain.ErrGuardrailNotFound)
}

func TestGormGuardrailRepository_FindBySpecialistID(t *testing.T) {
	repo, db := setupGuardrailRepo(t)

	ctx := context.Background()
	specialistID := createTestSpecialistForGuardrail(t, db)

	g1, err := domain.NewGuardrail(uuid.New().String(), specialistID, "nome-teste", domain.GuardrailTypeForbiddenTopics, "Nao falar de politica", "")
	require.NoError(t, err)
	require.NoError(t, repo.Create(ctx, g1))

	g2, err := domain.NewGuardrail(uuid.New().String(), specialistID, "nome-teste", domain.GuardrailTypeScopeLimit, "Somente assuntos juridicos", "Fora do escopo")
	require.NoError(t, err)
	require.NoError(t, repo.Create(ctx, g2))

	guardrails, err := repo.FindBySpecialistID(ctx, specialistID)
	require.NoError(t, err)
	assert.Len(t, guardrails, 2)

	ids := []string{guardrails[0].ID, guardrails[1].ID}
	assert.Contains(t, ids, g1.ID)
	assert.Contains(t, ids, g2.ID)
}

func TestGormGuardrailRepository_FindBySpecialistID_Empty(t *testing.T) {
	repo, db := setupGuardrailRepo(t)

	ctx := context.Background()
	specialistID := createTestSpecialistForGuardrail(t, db)

	guardrails, err := repo.FindBySpecialistID(ctx, specialistID)
	require.NoError(t, err)
	assert.Empty(t, guardrails)
}
