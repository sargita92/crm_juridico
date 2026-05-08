package http

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/sasrgita/crm-juridico/internal/specialist/application"
	"github.com/sasrgita/crm-juridico/internal/specialist/domain"
	specialistinfra "github.com/sasrgita/crm-juridico/internal/specialist/infrastructure"
)

// jsonReq builds a JSON request with optional auth cookie.
func jsonReq(method, path string, body any, cookies ...*http.Cookie) *http.Request {
	var b []byte
	if body != nil {
		var err error
		b, err = json.Marshal(body)
		if err != nil {
			panic(err)
		}
	}
	req := httptest.NewRequest(method, path, bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	for _, c := range cookies {
		req.AddCookie(c)
	}
	return req
}

// seedCrossSellRule creates a cross-sell rule directly in the DB.
func seedCrossSellRule(t *testing.T, env *testEnv, specialistID, targetProductID string, ordem int) *domain.CrossSellRule {
	t.Helper()
	rule, err := domain.NewCrossSellRule(
		uuid.New().String(),
		specialistID,
		ordem,
		domain.CrossSellTriggerKeyword,
		domain.KeywordTrigger{Termos: []string{"test"}},
		targetProductID,
	)
	require.NoError(t, err)
	repo := specialistinfra.NewGormCrossSellRuleRepository(env.db)
	require.NoError(t, repo.Save(context.Background(), rule))
	return rule
}

// --- LIST ---

func TestCrossSellRule_List_NoToken_Returns401(t *testing.T) {
	env := setupTestEnv(t)
	s := env.seedSpecialist(t, "Spec", "prompt")

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/admin/specialists/"+s.ID+"/cross-sell-rules", nil)
	env.router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestCrossSellRule_List_UserRole_Returns403(t *testing.T) {
	env := setupTestEnv(t)
	s := env.seedSpecialist(t, "Spec", "prompt")
	token := env.userToken(t)

	w := httptest.NewRecorder()
	req := getReq("/admin/specialists/"+s.ID+"/cross-sell-rules", tokenCookie(token))
	env.router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusForbidden, w.Code)
}

func TestCrossSellRule_List_Success(t *testing.T) {
	env := setupTestEnv(t)
	s := env.seedSpecialist(t, "Spec List", "prompt")
	token := env.adminToken(t)
	productID := uuid.New().String()

	seedCrossSellRule(t, env, s.ID, productID, 1)
	seedCrossSellRule(t, env, s.ID, productID, 2)

	w := httptest.NewRecorder()
	req := getReq("/admin/specialists/"+s.ID+"/cross-sell-rules", tokenCookie(token))
	env.router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	rules, ok := resp["rules"].([]any)
	require.True(t, ok)
	assert.Len(t, rules, 2)
}

// --- CREATE ---

func TestCrossSellRule_Create_Success(t *testing.T) {
	env := setupTestEnv(t)
	s := env.seedSpecialist(t, "Spec Create", "prompt")
	token := env.adminToken(t)

	body := map[string]any{
		"trigger_type": "keyword",
		"trigger_config": map[string]any{
			"termos": []string{"contrato", "seguro"},
		},
		"target_product_id": uuid.New().String(),
		"ativo":             true,
	}

	w := httptest.NewRecorder()
	req := jsonReq(http.MethodPost, "/admin/specialists/"+s.ID+"/cross-sell-rules", body, tokenCookie(token))
	env.router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)

	var out application.CrossSellRuleOutput
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &out))
	assert.Equal(t, s.ID, out.SpecialistID)
	assert.Equal(t, "keyword", out.TriggerType)
	assert.Equal(t, 1, out.Ordem)

	// Verify persisted
	repo := specialistinfra.NewGormCrossSellRuleRepository(env.db)
	rules, err := repo.ListBySpecialistID(context.Background(), s.ID)
	require.NoError(t, err)
	assert.Len(t, rules, 1)
}

func TestCrossSellRule_Create_InvalidKeywords_Returns400(t *testing.T) {
	env := setupTestEnv(t)
	s := env.seedSpecialist(t, "Spec Bad Create", "prompt")
	token := env.adminToken(t)

	body := map[string]any{
		"trigger_type": "keyword",
		"trigger_config": map[string]any{
			"termos": []string{}, // empty keywords — invalid
		},
		"target_product_id": uuid.New().String(),
	}

	w := httptest.NewRecorder()
	req := jsonReq(http.MethodPost, "/admin/specialists/"+s.ID+"/cross-sell-rules", body, tokenCookie(token))
	env.router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestCrossSellRule_Create_MissingTargetProduct_Returns400(t *testing.T) {
	env := setupTestEnv(t)
	s := env.seedSpecialist(t, "Spec Bad Target", "prompt")
	token := env.adminToken(t)

	body := map[string]any{
		"trigger_type": "keyword",
		"trigger_config": map[string]any{
			"termos": []string{"test"},
		},
		"target_product_id": "",
	}

	w := httptest.NewRecorder()
	req := jsonReq(http.MethodPost, "/admin/specialists/"+s.ID+"/cross-sell-rules", body, tokenCookie(token))
	env.router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestCrossSellRule_Create_SpecialistNotFound_Returns404(t *testing.T) {
	env := setupTestEnv(t)
	token := env.adminToken(t)

	body := map[string]any{
		"trigger_type": "keyword",
		"trigger_config": map[string]any{
			"termos": []string{"test"},
		},
		"target_product_id": uuid.New().String(),
	}

	w := httptest.NewRecorder()
	req := jsonReq(http.MethodPost, "/admin/specialists/"+uuid.New().String()+"/cross-sell-rules", body, tokenCookie(token))
	env.router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

// --- UPDATE ---

func TestCrossSellRule_Update_Success(t *testing.T) {
	env := setupTestEnv(t)
	s := env.seedSpecialist(t, "Spec Update", "prompt")
	token := env.adminToken(t)
	productID := uuid.New().String()

	rule := seedCrossSellRule(t, env, s.ID, productID, 1)

	newProductID := uuid.New().String()
	body := map[string]any{
		"trigger_type": "keyword",
		"trigger_config": map[string]any{
			"termos": []string{"atualizado"},
		},
		"target_product_id": newProductID,
		"ativo":             false,
	}

	w := httptest.NewRecorder()
	req := jsonReq(http.MethodPut, "/admin/specialists/"+s.ID+"/cross-sell-rules/"+rule.ID, body, tokenCookie(token))
	env.router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var out application.CrossSellRuleOutput
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &out))
	assert.Equal(t, newProductID, out.TargetProductID)
	assert.False(t, out.Ativo)
}

func TestCrossSellRule_Update_NotFound_Returns404(t *testing.T) {
	env := setupTestEnv(t)
	s := env.seedSpecialist(t, "Spec Update NF", "prompt")
	token := env.adminToken(t)

	body := map[string]any{
		"trigger_type": "keyword",
		"trigger_config": map[string]any{
			"termos": []string{"test"},
		},
		"target_product_id": uuid.New().String(),
		"ativo":             true,
	}

	w := httptest.NewRecorder()
	req := jsonReq(http.MethodPut, "/admin/specialists/"+s.ID+"/cross-sell-rules/"+uuid.New().String(), body, tokenCookie(token))
	env.router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

// --- DELETE ---

func TestCrossSellRule_Delete_Success(t *testing.T) {
	env := setupTestEnv(t)
	s := env.seedSpecialist(t, "Spec Delete", "prompt")
	token := env.adminToken(t)
	productID := uuid.New().String()

	rule := seedCrossSellRule(t, env, s.ID, productID, 1)

	w := httptest.NewRecorder()
	req := deleteReq("/admin/specialists/"+s.ID+"/cross-sell-rules/"+rule.ID, tokenCookie(token))
	env.router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	// Verify removed
	repo := specialistinfra.NewGormCrossSellRuleRepository(env.db)
	rules, err := repo.ListBySpecialistID(context.Background(), s.ID)
	require.NoError(t, err)
	assert.Len(t, rules, 0)
}

func TestCrossSellRule_Delete_NotFound_Returns404(t *testing.T) {
	env := setupTestEnv(t)
	s := env.seedSpecialist(t, "Spec Delete NF", "prompt")
	token := env.adminToken(t)

	w := httptest.NewRecorder()
	req := deleteReq("/admin/specialists/"+s.ID+"/cross-sell-rules/"+uuid.New().String(), tokenCookie(token))
	env.router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

// --- MOVE UP / DOWN ---

func TestCrossSellRule_MoveUp_Success(t *testing.T) {
	env := setupTestEnv(t)
	s := env.seedSpecialist(t, "Spec Move Up", "prompt")
	token := env.adminToken(t)
	productID := uuid.New().String()

	rule1 := seedCrossSellRule(t, env, s.ID, productID, 1)
	rule2 := seedCrossSellRule(t, env, s.ID, productID, 2)

	w := httptest.NewRecorder()
	req := jsonReq(http.MethodPost, "/admin/specialists/"+s.ID+"/cross-sell-rules/"+rule2.ID+"/move-up", nil, tokenCookie(token))
	env.router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	repo := specialistinfra.NewGormCrossSellRuleRepository(env.db)
	r1, err := repo.FindByID(context.Background(), rule1.ID)
	require.NoError(t, err)
	r2, err := repo.FindByID(context.Background(), rule2.ID)
	require.NoError(t, err)

	// rule2 should now be at position 1, rule1 at position 2
	assert.Equal(t, 1, r2.Ordem)
	assert.Equal(t, 2, r1.Ordem)
}

func TestCrossSellRule_MoveDown_Success(t *testing.T) {
	env := setupTestEnv(t)
	s := env.seedSpecialist(t, "Spec Move Down", "prompt")
	token := env.adminToken(t)
	productID := uuid.New().String()

	rule1 := seedCrossSellRule(t, env, s.ID, productID, 1)
	rule2 := seedCrossSellRule(t, env, s.ID, productID, 2)

	w := httptest.NewRecorder()
	req := jsonReq(http.MethodPost, "/admin/specialists/"+s.ID+"/cross-sell-rules/"+rule1.ID+"/move-down", nil, tokenCookie(token))
	env.router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	repo := specialistinfra.NewGormCrossSellRuleRepository(env.db)
	r1, err := repo.FindByID(context.Background(), rule1.ID)
	require.NoError(t, err)
	r2, err := repo.FindByID(context.Background(), rule2.ID)
	require.NoError(t, err)

	// rule1 should now be at position 2, rule2 at position 1
	assert.Equal(t, 2, r1.Ordem)
	assert.Equal(t, 1, r2.Ordem)
}

func TestCrossSellRule_MoveUp_AlreadyFirst_Returns400(t *testing.T) {
	env := setupTestEnv(t)
	s := env.seedSpecialist(t, "Spec Move Up First", "prompt")
	token := env.adminToken(t)
	productID := uuid.New().String()

	rule := seedCrossSellRule(t, env, s.ID, productID, 1)

	w := httptest.NewRecorder()
	req := jsonReq(http.MethodPost, "/admin/specialists/"+s.ID+"/cross-sell-rules/"+rule.ID+"/move-up", nil, tokenCookie(token))
	env.router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestCrossSellRule_MoveDown_AlreadyLast_Returns400(t *testing.T) {
	env := setupTestEnv(t)
	s := env.seedSpecialist(t, "Spec Move Down Last", "prompt")
	token := env.adminToken(t)
	productID := uuid.New().String()

	rule := seedCrossSellRule(t, env, s.ID, productID, 1)

	w := httptest.NewRecorder()
	req := jsonReq(http.MethodPost, "/admin/specialists/"+s.ID+"/cross-sell-rules/"+rule.ID+"/move-down", nil, tokenCookie(token))
	env.router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}
