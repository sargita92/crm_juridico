package http

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/sasrgita/crm-juridico/internal/specialist/domain"
	specialistinfra "github.com/sasrgita/crm-juridico/internal/specialist/infrastructure"
)

// --- OWASP A01: Broken Access Control — CrossSellRule ---

// TestCrossSellRule_Requires401WhenUnauthenticated verifies that all cross-sell
// rule endpoints reject requests without a valid authentication token.
func TestCrossSellRule_Requires401WhenUnauthenticated(t *testing.T) {
	env := setupTestEnv(t)
	s := env.seedSpecialist(t, "OWASP Spec Auth", "prompt")
	ruleID := uuid.New().String()

	routes := []struct {
		method string
		path   string
	}{
		{http.MethodGet, "/admin/specialists/" + s.ID + "/cross-sell-rules"},
		{http.MethodPost, "/admin/specialists/" + s.ID + "/cross-sell-rules"},
		{http.MethodPut, "/admin/specialists/" + s.ID + "/cross-sell-rules/" + ruleID},
		{http.MethodDelete, "/admin/specialists/" + s.ID + "/cross-sell-rules/" + ruleID},
		{http.MethodPost, "/admin/specialists/" + s.ID + "/cross-sell-rules/" + ruleID + "/move-up"},
		{http.MethodPost, "/admin/specialists/" + s.ID + "/cross-sell-rules/" + ruleID + "/move-down"},
	}

	for _, route := range routes {
		t.Run(route.method+" "+route.path, func(t *testing.T) {
			w := httptest.NewRecorder()
			req := httptest.NewRequest(route.method, route.path, nil)
			env.router.ServeHTTP(w, req)
			assert.Equal(t, http.StatusUnauthorized, w.Code)
		})
	}
}

// TestCrossSellRule_Requires403WhenWrongPermission verifies that a non-admin user
// cannot access cross-sell rule endpoints (OWASP A01: Broken Access Control).
func TestCrossSellRule_Requires403WhenWrongPermission(t *testing.T) {
	env := setupTestEnv(t)
	s := env.seedSpecialist(t, "OWASP Spec Perm", "prompt")
	token := env.userToken(t)
	ruleID := uuid.New().String()

	routes := []struct {
		method string
		path   string
	}{
		{http.MethodGet, "/admin/specialists/" + s.ID + "/cross-sell-rules"},
		{http.MethodPost, "/admin/specialists/" + s.ID + "/cross-sell-rules"},
		{http.MethodPut, "/admin/specialists/" + s.ID + "/cross-sell-rules/" + ruleID},
		{http.MethodDelete, "/admin/specialists/" + s.ID + "/cross-sell-rules/" + ruleID},
		{http.MethodPost, "/admin/specialists/" + s.ID + "/cross-sell-rules/" + ruleID + "/move-up"},
		{http.MethodPost, "/admin/specialists/" + s.ID + "/cross-sell-rules/" + ruleID + "/move-down"},
	}

	for _, route := range routes {
		t.Run(route.method+" "+route.path, func(t *testing.T) {
			w := httptest.NewRecorder()
			req := httptest.NewRequest(route.method, route.path, nil)
			req.AddCookie(tokenCookie(token))
			env.router.ServeHTTP(w, req)
			assert.Equal(t, http.StatusForbidden, w.Code)
		})
	}
}

// TestCrossSellRule_RejectsAccessToRuleOfOtherTenant verifies that an admin cannot
// update, delete, or reorder a cross-sell rule that belongs to a different specialist
// (BOLA / IDOR attack — OWASP A01: Broken Access Control).
//
// Scenario: specialistA owns ruleA. An admin attempts to mutate ruleA using
// the URL path of specialistB (cross-specialist / cross-tenant isolation).
func TestCrossSellRule_RejectsAccessToRuleOfOtherTenant(t *testing.T) {
	env := setupTestEnv(t)
	token := env.adminToken(t)

	// Two independent specialists (may be associated with different tenants)
	specA := env.seedSpecialist(t, "OWASP SpecA", "prompt")
	specB := env.seedSpecialist(t, "OWASP SpecB", "prompt")

	productID := uuid.New().String()

	// Seed a rule under specA
	ruleA, err := domain.NewCrossSellRule(
		uuid.New().String(),
		specA.ID,
		1,
		domain.CrossSellTriggerKeyword,
		domain.KeywordTrigger{Termos: []string{"seguro"}},
		productID,
	)
	require.NoError(t, err)
	repo := specialistinfra.NewGormCrossSellRuleRepository(env.db)
	require.NoError(t, repo.Save(context.Background(), ruleA))

	// PUT /admin/specialists/{specB.ID}/cross-sell-rules/{ruleA.ID}
	// ruleA does NOT belong to specB — must be rejected with 403.
	t.Run("update rule belonging to other specialist returns 403", func(t *testing.T) {
		body := map[string]any{
			"trigger_type": "keyword",
			"trigger_config": map[string]any{
				"termos": []string{"outro"},
			},
			"target_product_id": productID,
			"ativo":             true,
		}
		w := httptest.NewRecorder()
		req := jsonReq(http.MethodPut, "/admin/specialists/"+specB.ID+"/cross-sell-rules/"+ruleA.ID, body, tokenCookie(token))
		env.router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusForbidden, w.Code)
	})

	// DELETE /admin/specialists/{specB.ID}/cross-sell-rules/{ruleA.ID}
	// ruleA does NOT belong to specB — must be rejected with 403.
	t.Run("delete rule belonging to other specialist returns 403", func(t *testing.T) {
		w := httptest.NewRecorder()
		req := deleteReq("/admin/specialists/"+specB.ID+"/cross-sell-rules/"+ruleA.ID, tokenCookie(token))
		env.router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusForbidden, w.Code)
	})

	// POST /admin/specialists/{specB.ID}/cross-sell-rules/{ruleA.ID}/move-up
	// ruleA does NOT belong to specB — must be rejected with 403.
	t.Run("move-up rule belonging to other specialist returns 403", func(t *testing.T) {
		w := httptest.NewRecorder()
		req := jsonReq(http.MethodPost, "/admin/specialists/"+specB.ID+"/cross-sell-rules/"+ruleA.ID+"/move-up", nil, tokenCookie(token))
		env.router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusForbidden, w.Code)
	})

	// POST /admin/specialists/{specB.ID}/cross-sell-rules/{ruleA.ID}/move-down
	// ruleA does NOT belong to specB — must be rejected with 403.
	t.Run("move-down rule belonging to other specialist returns 403", func(t *testing.T) {
		w := httptest.NewRecorder()
		req := jsonReq(http.MethodPost, "/admin/specialists/"+specB.ID+"/cross-sell-rules/"+ruleA.ID+"/move-down", nil, tokenCookie(token))
		env.router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusForbidden, w.Code)
	})

	// Verify ruleA was NOT mutated (integrity check after all BOLA attempts)
	t.Run("rule of specA is untouched after BOLA attempts", func(t *testing.T) {
		persisted, err := repo.FindByID(context.Background(), ruleA.ID)
		require.NoError(t, err)
		assert.Equal(t, specA.ID, persisted.SpecialistID)
		assert.Equal(t, domain.KeywordTrigger{Termos: []string{"seguro"}}, persisted.TriggerConfig)
	})
}

// TestCrossSellRule_RejectsTargetProductFromOtherTenant verifies that the
// list endpoint for specialist A does NOT expose rules that belong to specialist B,
// even when specialist B is associated with a different tenant.
//
// This is a data-isolation check: the ListBySpecialistID scope must be respected.
func TestCrossSellRule_RejectsTargetProductFromOtherTenant(t *testing.T) {
	env := setupTestEnv(t)
	token := env.adminToken(t)

	specA := env.seedSpecialist(t, "OWASP IsolA", "prompt")
	specB := env.seedSpecialist(t, "OWASP IsolB", "prompt")

	productAID := uuid.New().String()
	productBID := uuid.New().String()

	repo := specialistinfra.NewGormCrossSellRuleRepository(env.db)

	// Seed a rule under specA with a unique productID
	ruleA, err := domain.NewCrossSellRule(
		uuid.New().String(),
		specA.ID,
		1,
		domain.CrossSellTriggerKeyword,
		domain.KeywordTrigger{Termos: []string{"produto-A-secreto"}},
		productAID,
	)
	require.NoError(t, err)
	require.NoError(t, repo.Save(context.Background(), ruleA))

	// Seed a rule under specB with a different productID
	ruleB, err := domain.NewCrossSellRule(
		uuid.New().String(),
		specB.ID,
		1,
		domain.CrossSellTriggerKeyword,
		domain.KeywordTrigger{Termos: []string{"produto-B-secreto"}},
		productBID,
	)
	require.NoError(t, err)
	require.NoError(t, repo.Save(context.Background(), ruleB))

	// Listing rules for specA must NOT include specB's rule/product
	t.Run("listing specA rules does not expose specB product IDs", func(t *testing.T) {
		w := httptest.NewRecorder()
		req := getReq("/admin/specialists/"+specA.ID+"/cross-sell-rules", tokenCookie(token))
		env.router.ServeHTTP(w, req)
		require.Equal(t, http.StatusOK, w.Code)
		assert.NotContains(t, w.Body.String(), productBID,
			"rules of specB must not appear in specA listing")
		assert.Contains(t, w.Body.String(), productAID,
			"specA's own product must appear in specA listing")
	})

	// Listing rules for specB must NOT include specA's rule/product
	t.Run("listing specB rules does not expose specA product IDs", func(t *testing.T) {
		w := httptest.NewRecorder()
		req := getReq("/admin/specialists/"+specB.ID+"/cross-sell-rules", tokenCookie(token))
		env.router.ServeHTTP(w, req)
		require.Equal(t, http.StatusOK, w.Code)
		assert.NotContains(t, w.Body.String(), productAID,
			"rules of specA must not appear in specB listing")
		assert.Contains(t, w.Body.String(), productBID,
			"specB's own product must appear in specB listing")
	})
}
