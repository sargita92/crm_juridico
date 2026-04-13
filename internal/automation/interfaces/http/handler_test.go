package http

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	automationapp "github.com/sasrgita/crm-juridico/internal/automation/application"
	"github.com/sasrgita/crm-juridico/internal/shared/middleware"
)

// setupAPIEnv builds a Gin router wired to the JSON API Handler with in-memory repos.
func setupAPIEnv(t *testing.T) (*gin.Engine, *mockAutomationRepo, *mockLogRepo) {
	t.Helper()
	gin.SetMode(gin.TestMode)

	autoRepo := newMockAutomationRepo()
	logRepo := &mockLogRepo{}
	crudUC := automationapp.NewCRUDUseCase(autoRepo, logRepo)
	handler := NewHandler(crudUC, zap.NewNop())

	router := gin.New()
	router.Use(func(c *gin.Context) {
		ctx := middleware.SetTenantIDForTest(c.Request.Context(), testTenantID)
		c.Request = c.Request.WithContext(ctx)
		c.Next()
	})

	funnels := router.Group("/tenant/leads/funnels")
	funnels.GET("/:id/automations", handler.ListByFunnel)
	funnels.POST("/:id/automations", handler.Create)

	autos := router.Group("/tenant/leads/automations")
	autos.GET("/:id", handler.GetDetail)
	autos.PUT("/:id", handler.Update)
	autos.DELETE("/:id", handler.Delete)
	autos.POST("/:id/toggle", handler.Toggle)
	autos.GET("/:id/logs", handler.GetLogs)

	return router, autoRepo, logRepo
}

func apiDo(router *gin.Engine, method, path, body string) *httptest.ResponseRecorder {
	w := httptest.NewRecorder()
	var req *http.Request
	if body != "" {
		req = httptest.NewRequest(method, path, strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
	} else {
		req = httptest.NewRequest(method, path, nil)
	}
	router.ServeHTTP(w, req)
	return w
}

func TestAPI_Create_Success_PortugueseTemplate(t *testing.T) {
	router, autoRepo, _ := setupAPIEnv(t)

	body := `{
		"column_id": "col-1",
		"type": "auto_message",
		"config": {"template": "Olá, sua ação foi recebida — aguarde retorno"},
		"priority": 3
	}`
	w := apiDo(router, http.MethodPost, "/tenant/leads/funnels/fn-1/automations", body)
	require.Equal(t, http.StatusCreated, w.Code)

	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, "auto_message", resp["type"])

	require.Len(t, autoRepo.items, 1)
	for _, a := range autoRepo.items {
		assert.Equal(t, "Olá, sua ação foi recebida — aguarde retorno", a.Config["template"])
		assert.Equal(t, testTenantID, a.TenantID)
		assert.Equal(t, "fn-1", a.FunnelID)
	}
}

func TestAPI_Create_InvalidJSON(t *testing.T) {
	router, _, _ := setupAPIEnv(t)
	w := apiDo(router, http.MethodPost, "/tenant/leads/funnels/fn-1/automations", "{not json")
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestAPI_Create_ValidationError(t *testing.T) {
	router, _, _ := setupAPIEnv(t)
	body := `{"type": "invalid_type", "column_id": "col-1"}`
	w := apiDo(router, http.MethodPost, "/tenant/leads/funnels/fn-1/automations", body)
	assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
}

func TestAPI_ListByFunnel_Empty(t *testing.T) {
	router, _, _ := setupAPIEnv(t)
	w := apiDo(router, http.MethodGet, "/tenant/leads/funnels/fn-1/automations", "")
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "[]", strings.TrimSpace(w.Body.String()))
}

func TestAPI_ListByFunnel_WithItems(t *testing.T) {
	router, _, _ := setupAPIEnv(t)

	createBody := `{"column_id":"col-1","type":"auto_note","config":{"template":"nota"},"priority":1}`
	apiDo(router, http.MethodPost, "/tenant/leads/funnels/fn-1/automations", createBody)

	w := apiDo(router, http.MethodGet, "/tenant/leads/funnels/fn-1/automations", "")
	require.Equal(t, http.StatusOK, w.Code)

	var list []map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &list))
	require.Len(t, list, 1)
	assert.Equal(t, "auto_note", list[0]["type"])
}

func TestAPI_GetDetail_Success(t *testing.T) {
	router, _, _ := setupAPIEnv(t)

	create := apiDo(router, http.MethodPost,
		"/tenant/leads/funnels/fn-1/automations",
		`{"column_id":"col-1","type":"auto_message","config":{"template":"ação"},"priority":1}`)
	require.Equal(t, http.StatusCreated, create.Code)

	var created map[string]interface{}
	require.NoError(t, json.Unmarshal(create.Body.Bytes(), &created))
	id := created["id"].(string)

	w := apiDo(router, http.MethodGet, "/tenant/leads/automations/"+id, "")
	require.Equal(t, http.StatusOK, w.Code)

	var got map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &got))
	assert.Equal(t, id, got["id"])

	config := got["config"].(map[string]interface{})
	assert.Equal(t, "ação", config["template"])
}

func TestAPI_GetDetail_NotFound(t *testing.T) {
	router, _, _ := setupAPIEnv(t)
	w := apiDo(router, http.MethodGet, "/tenant/leads/automations/missing", "")
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestAPI_Update_Success(t *testing.T) {
	router, _, _ := setupAPIEnv(t)

	create := apiDo(router, http.MethodPost,
		"/tenant/leads/funnels/fn-1/automations",
		`{"column_id":"col-1","type":"auto_message","config":{"template":"old"},"priority":1}`)
	var created map[string]interface{}
	_ = json.Unmarshal(create.Body.Bytes(), &created)
	id := created["id"].(string)

	w := apiDo(router, http.MethodPut, "/tenant/leads/automations/"+id,
		`{"column_id":"col-2","config":{"template":"São José"},"priority":7}`)
	require.Equal(t, http.StatusOK, w.Code)

	var got map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &got))
	assert.Equal(t, float64(7), got["priority"])
	assert.Equal(t, "col-2", got["column_id"])
	config := got["config"].(map[string]interface{})
	assert.Equal(t, "São José", config["template"])
}

func TestAPI_Update_InvalidJSON(t *testing.T) {
	router, _, _ := setupAPIEnv(t)
	w := apiDo(router, http.MethodPut, "/tenant/leads/automations/any", "{{")
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestAPI_Update_NotFound(t *testing.T) {
	router, _, _ := setupAPIEnv(t)
	w := apiDo(router, http.MethodPut, "/tenant/leads/automations/missing",
		`{"column_id":"c","priority":1}`)
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestAPI_Delete_Success(t *testing.T) {
	router, autoRepo, _ := setupAPIEnv(t)

	create := apiDo(router, http.MethodPost,
		"/tenant/leads/funnels/fn-1/automations",
		`{"column_id":"col-1","type":"auto_note","config":{"template":"x"},"priority":1}`)
	var created map[string]interface{}
	_ = json.Unmarshal(create.Body.Bytes(), &created)
	id := created["id"].(string)

	w := apiDo(router, http.MethodDelete, "/tenant/leads/automations/"+id, "")
	assert.Equal(t, http.StatusNoContent, w.Code)
	assert.Len(t, autoRepo.items, 0)
}

func TestAPI_Delete_NotFound(t *testing.T) {
	router, _, _ := setupAPIEnv(t)
	w := apiDo(router, http.MethodDelete, "/tenant/leads/automations/missing", "")
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestAPI_Toggle_Success(t *testing.T) {
	router, autoRepo, _ := setupAPIEnv(t)

	create := apiDo(router, http.MethodPost,
		"/tenant/leads/funnels/fn-1/automations",
		`{"column_id":"col-1","type":"auto_note","config":{"template":"x"},"priority":1}`)
	var created map[string]interface{}
	_ = json.Unmarshal(create.Body.Bytes(), &created)
	id := created["id"].(string)

	assert.True(t, autoRepo.items[id].Active)

	w := apiDo(router, http.MethodPost, "/tenant/leads/automations/"+id+"/toggle", "")
	assert.Equal(t, http.StatusOK, w.Code)
	assert.False(t, autoRepo.items[id].Active)
}

func TestAPI_Toggle_NotFound(t *testing.T) {
	router, _, _ := setupAPIEnv(t)
	w := apiDo(router, http.MethodPost, "/tenant/leads/automations/missing/toggle", "")
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestAPI_GetLogs_Empty(t *testing.T) {
	router, _, _ := setupAPIEnv(t)
	w := apiDo(router, http.MethodGet, "/tenant/leads/automations/any/logs", "")
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestAPI_GetLogs_CustomPagination(t *testing.T) {
	router, _, _ := setupAPIEnv(t)
	w := apiDo(router, http.MethodGet, "/tenant/leads/automations/any/logs?limit=5&offset=2", "")
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestAPI_GetLogs_InvalidPaginationIgnored(t *testing.T) {
	router, _, _ := setupAPIEnv(t)
	w := apiDo(router, http.MethodGet, "/tenant/leads/automations/any/logs?limit=abc&offset=-1", "")
	assert.Equal(t, http.StatusOK, w.Code)
}
