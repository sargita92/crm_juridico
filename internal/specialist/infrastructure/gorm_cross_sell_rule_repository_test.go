package infrastructure

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/sasrgita/crm-juridico/internal/shared/database"
	"github.com/sasrgita/crm-juridico/internal/shared/testhelper"
	"github.com/sasrgita/crm-juridico/internal/specialist/domain"
)

func setupCrossSellRuleRepo(t *testing.T) (*GormCrossSellRuleRepository, *gorm.DB) {
	t.Helper()
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	db := sharedContainer.DB(t)
	log := sharedContainer.Logger()

	err := database.RunMigrations(db, log, "file://"+testhelper.MigrationsPath())
	require.NoError(t, err)

	db.Exec("DELETE FROM cross_sell_rules")
	db.Exec("DELETE FROM scoring_configs")
	db.Exec("DELETE FROM guardrails")
	db.Exec("DELETE FROM steps")
	db.Exec("DELETE FROM specialist_tenants")
	db.Exec("DELETE FROM specialists")
	db.Exec("DELETE FROM products")
	db.Exec("DELETE FROM tenant_block_history")
	db.Exec("DELETE FROM user_tenants")
	db.Exec("DELETE FROM users")
	db.Exec("DELETE FROM tenants")

	return NewGormCrossSellRuleRepository(db), db
}

func createTestSpecialistForCSR(t *testing.T, db *gorm.DB) string {
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

func createTestTenantForCSR(t *testing.T, db *gorm.DB) string {
	t.Helper()
	id := uuid.New().String()
	err := db.Table("tenants").Create(map[string]interface{}{
		"id":           id,
		"name":         "Tenant CSR",
		"type":         "pj",
		"document":     uuid.New().String()[:18],
		"status":       "active",
		"block_reason": "",
	}).Error
	require.NoError(t, err)
	return id
}

func createTestProductForCSR(t *testing.T, db *gorm.DB, tenantID string) string {
	t.Helper()
	id := uuid.New().String()
	now := time.Now()
	err := db.Table("products").Create(map[string]interface{}{
		"id":          id,
		"name":        "Test Product",
		"description": "desc",
		"keywords":    "[]",
		"active":      true,
		"created_at":  now,
		"updated_at":  now,
	}).Error
	require.NoError(t, err)
	return id
}

// --- CrossSellRule Repository Tests ---

func TestGormCrossSellRule_RoundTripWithJSONTrigger(t *testing.T) {
	repo, db := setupCrossSellRuleRepo(t)
	ctx := context.Background()

	specialistID := createTestSpecialistForCSR(t, db)
	tenantID := createTestTenantForCSR(t, db)
	productID := createTestProductForCSR(t, db, tenantID)

	rule, err := domain.NewCrossSellRule(
		uuid.New().String(),
		specialistID,
		0,
		domain.CrossSellTriggerKeyword,
		domain.KeywordTrigger{Termos: []string{"trabalhista"}},
		productID,
	)
	require.NoError(t, err)
	require.NoError(t, repo.Save(ctx, rule))

	list, err := repo.ListActiveBySpecialistOrdered(ctx, specialistID)
	require.NoError(t, err)
	require.Len(t, list, 1)

	kw, ok := list[0].TriggerConfig.(domain.KeywordTrigger)
	require.True(t, ok, "expected KeywordTrigger")
	assert.Equal(t, []string{"trabalhista"}, kw.Termos)
}

func TestGormCrossSellRule_StepAnswerTriggerRoundTrip(t *testing.T) {
	repo, db := setupCrossSellRuleRepo(t)
	ctx := context.Background()

	specialistID := createTestSpecialistForCSR(t, db)
	tenantID := createTestTenantForCSR(t, db)
	productID := createTestProductForCSR(t, db, tenantID)

	rule, err := domain.NewCrossSellRule(
		uuid.New().String(),
		specialistID,
		0,
		domain.CrossSellTriggerStepAnswer,
		domain.StepAnswerTrigger{StepID: "step-abc", Regex: `^sim$`},
		productID,
	)
	require.NoError(t, err)
	require.NoError(t, repo.Save(ctx, rule))

	found, err := repo.FindByID(ctx, rule.ID)
	require.NoError(t, err)

	sa, ok := found.TriggerConfig.(domain.StepAnswerTrigger)
	require.True(t, ok, "expected StepAnswerTrigger")
	assert.Equal(t, "step-abc", sa.StepID)
	assert.Equal(t, `^sim$`, sa.Regex)
}

func TestGormCrossSellRule_OrderingMultipleRules(t *testing.T) {
	repo, db := setupCrossSellRuleRepo(t)
	ctx := context.Background()

	specialistID := createTestSpecialistForCSR(t, db)
	tenantID := createTestTenantForCSR(t, db)
	productID := createTestProductForCSR(t, db, tenantID)

	for _, ordem := range []int{2, 0, 1} {
		rule, err := domain.NewCrossSellRule(
			uuid.New().String(),
			specialistID,
			ordem,
			domain.CrossSellTriggerKeyword,
			domain.KeywordTrigger{Termos: []string{"termo"}},
			productID,
		)
		require.NoError(t, err)
		require.NoError(t, repo.Save(ctx, rule))
	}

	list, err := repo.ListBySpecialistID(ctx, specialistID)
	require.NoError(t, err)
	require.Len(t, list, 3)
	assert.Equal(t, 0, list[0].Ordem)
	assert.Equal(t, 1, list[1].Ordem)
	assert.Equal(t, 2, list[2].Ordem)
}

func TestGormCrossSellRule_InactiveFilteredByListActive(t *testing.T) {
	repo, db := setupCrossSellRuleRepo(t)
	ctx := context.Background()

	specialistID := createTestSpecialistForCSR(t, db)
	tenantID := createTestTenantForCSR(t, db)
	productID := createTestProductForCSR(t, db, tenantID)

	active, err := domain.NewCrossSellRule(
		uuid.New().String(),
		specialistID,
		0,
		domain.CrossSellTriggerKeyword,
		domain.KeywordTrigger{Termos: []string{"ativo"}},
		productID,
	)
	require.NoError(t, err)
	require.NoError(t, repo.Save(ctx, active))

	inactive, err := domain.NewCrossSellRule(
		uuid.New().String(),
		specialistID,
		1,
		domain.CrossSellTriggerKeyword,
		domain.KeywordTrigger{Termos: []string{"inativo"}},
		productID,
	)
	require.NoError(t, err)
	inactive.Ativo = false
	require.NoError(t, repo.Save(ctx, inactive))

	// ListBySpecialistID returns all
	all, err := repo.ListBySpecialistID(ctx, specialistID)
	require.NoError(t, err)
	assert.Len(t, all, 2)

	// ListActiveBySpecialistOrdered returns only active
	activeList, err := repo.ListActiveBySpecialistOrdered(ctx, specialistID)
	require.NoError(t, err)
	require.Len(t, activeList, 1)
	assert.True(t, activeList[0].Ativo)
}

func TestGormCrossSellRule_Delete(t *testing.T) {
	repo, db := setupCrossSellRuleRepo(t)
	ctx := context.Background()

	specialistID := createTestSpecialistForCSR(t, db)
	tenantID := createTestTenantForCSR(t, db)
	productID := createTestProductForCSR(t, db, tenantID)

	rule, err := domain.NewCrossSellRule(
		uuid.New().String(),
		specialistID,
		0,
		domain.CrossSellTriggerKeyword,
		domain.KeywordTrigger{Termos: []string{"delete-me"}},
		productID,
	)
	require.NoError(t, err)
	require.NoError(t, repo.Save(ctx, rule))

	require.NoError(t, repo.Delete(ctx, rule.ID))

	_, err = repo.FindByID(ctx, rule.ID)
	assert.ErrorIs(t, err, domain.ErrCrossSellRuleNotFound)
}

func TestGormCrossSellRule_Delete_NotFound(t *testing.T) {
	repo, _ := setupCrossSellRuleRepo(t)

	err := repo.Delete(context.Background(), uuid.New().String())
	assert.ErrorIs(t, err, domain.ErrCrossSellRuleNotFound)
}

func TestGormCrossSellRule_FindByID_NotFound(t *testing.T) {
	repo, _ := setupCrossSellRuleRepo(t)

	_, err := repo.FindByID(context.Background(), uuid.New().String())
	assert.ErrorIs(t, err, domain.ErrCrossSellRuleNotFound)
}

func TestGormCrossSellRule_SaveUpdatesExisting(t *testing.T) {
	repo, db := setupCrossSellRuleRepo(t)
	ctx := context.Background()

	specialistID := createTestSpecialistForCSR(t, db)
	tenantID := createTestTenantForCSR(t, db)
	productID := createTestProductForCSR(t, db, tenantID)

	rule, err := domain.NewCrossSellRule(
		uuid.New().String(),
		specialistID,
		0,
		domain.CrossSellTriggerKeyword,
		domain.KeywordTrigger{Termos: []string{"original"}},
		productID,
	)
	require.NoError(t, err)
	require.NoError(t, repo.Save(ctx, rule))

	// Update: change Ordem and deactivate
	rule.Ordem = 5
	rule.Ativo = false
	require.NoError(t, repo.Save(ctx, rule))

	found, err := repo.FindByID(ctx, rule.ID)
	require.NoError(t, err)
	assert.Equal(t, 5, found.Ordem)
	assert.False(t, found.Ativo)
}
