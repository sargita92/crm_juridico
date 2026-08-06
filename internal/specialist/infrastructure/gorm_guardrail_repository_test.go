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
	db.Exec("DELETE FROM specialist_guardrails")
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

// --- Guardrail Library CRUD ---

func TestGormGuardrailRepository_Create_And_FindByID(t *testing.T) {
	repo, _ := setupGuardrailRepo(t)
	ctx := context.Background()

	g, err := domain.NewGuardrail(uuid.New().String(), "nome-teste", domain.GuardrailTypeForbiddenTopics, "Nao falar sobre concorrentes", "Desculpe, nao posso ajudar com isso.")
	require.NoError(t, err)

	require.NoError(t, repo.Create(ctx, g))

	found, err := repo.FindByID(ctx, g.ID)
	require.NoError(t, err)
	assert.Equal(t, g.ID, found.ID)
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
	repo, _ := setupGuardrailRepo(t)
	ctx := context.Background()

	g, err := domain.NewGuardrail(uuid.New().String(), "nome-teste", domain.GuardrailTypeForbiddenTopics, "Regra original", "Mensagem original")
	require.NoError(t, err)
	require.NoError(t, repo.Create(ctx, g))

	require.NoError(t, g.Update("nome-teste", domain.GuardrailTypeScopeLimit, "Regra atualizada", "Mensagem atualizada"))
	require.NoError(t, repo.Update(ctx, g))

	found, err := repo.FindByID(ctx, g.ID)
	require.NoError(t, err)
	assert.Equal(t, domain.GuardrailTypeScopeLimit, found.Type)
	assert.Equal(t, "Regra atualizada", found.Rule)
	assert.Equal(t, "Mensagem atualizada", found.Message)
}

func TestGormGuardrailRepository_Delete_Success(t *testing.T) {
	repo, _ := setupGuardrailRepo(t)
	ctx := context.Background()

	g, err := domain.NewGuardrail(uuid.New().String(), "nome-teste", domain.GuardrailTypeResponseTone, "Sempre ser educado", "")
	require.NoError(t, err)
	require.NoError(t, repo.Create(ctx, g))

	require.NoError(t, repo.Delete(ctx, g.ID))

	_, err = repo.FindByID(ctx, g.ID)
	assert.ErrorIs(t, err, domain.ErrGuardrailNotFound)
}

func TestGormGuardrailRepository_Delete_NotFound(t *testing.T) {
	repo, _ := setupGuardrailRepo(t)

	err := repo.Delete(context.Background(), uuid.New().String())
	assert.ErrorIs(t, err, domain.ErrGuardrailNotFound)
}

func TestGormGuardrailRepository_FindAll(t *testing.T) {
	repo, _ := setupGuardrailRepo(t)
	ctx := context.Background()

	g1, _ := domain.NewGuardrail(uuid.New().String(), "a", domain.GuardrailTypeForbiddenTopics, "r1", "")
	g2, _ := domain.NewGuardrail(uuid.New().String(), "b", domain.GuardrailTypeScopeLimit, "r2", "")
	require.NoError(t, repo.Create(ctx, g1))
	require.NoError(t, repo.Create(ctx, g2))

	all, err := repo.FindAll(ctx)
	require.NoError(t, err)
	assert.Len(t, all, 2)
}

// --- Attachment (join table) ---

func TestGormGuardrailRepository_Attach_And_FindBySpecialistID(t *testing.T) {
	repo, db := setupGuardrailRepo(t)
	ctx := context.Background()
	specialistID := createTestSpecialistForGuardrail(t, db)

	g1, _ := domain.NewGuardrail(uuid.New().String(), "g1", domain.GuardrailTypeForbiddenTopics, "Nao falar de politica", "")
	g2, _ := domain.NewGuardrail(uuid.New().String(), "g2", domain.GuardrailTypeScopeLimit, "Somente juridico", "Fora do escopo")
	require.NoError(t, repo.Create(ctx, g1))
	require.NoError(t, repo.Create(ctx, g2))

	require.NoError(t, repo.Attach(ctx, specialistID, g1.ID))
	require.NoError(t, repo.Attach(ctx, specialistID, g2.ID))

	guardrails, err := repo.FindBySpecialistID(ctx, specialistID)
	require.NoError(t, err)
	assert.Len(t, guardrails, 2)
	ids := []string{guardrails[0].ID, guardrails[1].ID}
	assert.Contains(t, ids, g1.ID)
	assert.Contains(t, ids, g2.ID)
}

// The core reuse behavior: one library guardrail attached to two specialists is
// returned for both.
func TestGormGuardrailRepository_SharedAcrossSpecialists(t *testing.T) {
	repo, db := setupGuardrailRepo(t)
	ctx := context.Background()
	spec1 := createTestSpecialistForGuardrail(t, db)
	spec2 := createTestSpecialistForGuardrail(t, db)

	g, _ := domain.NewGuardrail(uuid.New().String(), "compartilhado", domain.GuardrailTypeForbiddenTopics, "regra", "")
	require.NoError(t, repo.Create(ctx, g))
	require.NoError(t, repo.Attach(ctx, spec1, g.ID))
	require.NoError(t, repo.Attach(ctx, spec2, g.ID))

	l1, err := repo.FindBySpecialistID(ctx, spec1)
	require.NoError(t, err)
	l2, err := repo.FindBySpecialistID(ctx, spec2)
	require.NoError(t, err)
	require.Len(t, l1, 1)
	require.Len(t, l2, 1)
	assert.Equal(t, g.ID, l1[0].ID)
	assert.Equal(t, g.ID, l2[0].ID)

	count, err := repo.CountSpecialistsByGuardrailID(ctx, g.ID)
	require.NoError(t, err)
	assert.Equal(t, 2, count)
}

func TestGormGuardrailRepository_Attach_Duplicate(t *testing.T) {
	repo, db := setupGuardrailRepo(t)
	ctx := context.Background()
	specialistID := createTestSpecialistForGuardrail(t, db)

	g, _ := domain.NewGuardrail(uuid.New().String(), "g", domain.GuardrailTypeForbiddenTopics, "regra", "")
	require.NoError(t, repo.Create(ctx, g))
	require.NoError(t, repo.Attach(ctx, specialistID, g.ID))

	err := repo.Attach(ctx, specialistID, g.ID)
	assert.ErrorIs(t, err, domain.ErrGuardrailAlreadyAttached)
}

func TestGormGuardrailRepository_Detach(t *testing.T) {
	repo, db := setupGuardrailRepo(t)
	ctx := context.Background()
	spec1 := createTestSpecialistForGuardrail(t, db)
	spec2 := createTestSpecialistForGuardrail(t, db)

	g, _ := domain.NewGuardrail(uuid.New().String(), "g", domain.GuardrailTypeForbiddenTopics, "regra", "")
	require.NoError(t, repo.Create(ctx, g))
	require.NoError(t, repo.Attach(ctx, spec1, g.ID))
	require.NoError(t, repo.Attach(ctx, spec2, g.ID))

	require.NoError(t, repo.Detach(ctx, spec1, g.ID))

	l1, _ := repo.FindBySpecialistID(ctx, spec1)
	l2, _ := repo.FindBySpecialistID(ctx, spec2)
	assert.Empty(t, l1)
	assert.Len(t, l2, 1)

	// Guardrail remains in the library after detach.
	_, err := repo.FindByID(ctx, g.ID)
	require.NoError(t, err)
}

func TestGormGuardrailRepository_FindBySpecialistID_Empty(t *testing.T) {
	repo, db := setupGuardrailRepo(t)
	ctx := context.Background()
	specialistID := createTestSpecialistForGuardrail(t, db)

	guardrails, err := repo.FindBySpecialistID(ctx, specialistID)
	require.NoError(t, err)
	assert.Empty(t, guardrails)
}

func TestGormGuardrailRepository_CountSpecialists_Zero(t *testing.T) {
	repo, _ := setupGuardrailRepo(t)
	ctx := context.Background()

	g, _ := domain.NewGuardrail(uuid.New().String(), "g", domain.GuardrailTypeForbiddenTopics, "regra", "")
	require.NoError(t, repo.Create(ctx, g))

	count, err := repo.CountSpecialistsByGuardrailID(ctx, g.ID)
	require.NoError(t, err)
	assert.Equal(t, 0, count)
}
