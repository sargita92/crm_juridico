package http

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	authapp "github.com/sasrgita/crm-juridico/internal/auth/application"
	authdomain "github.com/sasrgita/crm-juridico/internal/auth/domain"
	"github.com/sasrgita/crm-juridico/internal/shared/middleware"
)

func init() {
	gin.SetMode(gin.TestMode)
}

// --- Stub for loadBalanceUsecase interface ---

type stubLB struct {
	getFn func(ctx context.Context, tenantID, groupID string) (*authdomain.LoadBalanceConfig, error)
	setFn func(ctx context.Context, in authapp.SetLoadBalanceInput) (*authdomain.LoadBalanceConfig, error)
}

func (s *stubLB) GetByGroup(ctx context.Context, tenantID, groupID string) (*authdomain.LoadBalanceConfig, error) {
	return s.getFn(ctx, tenantID, groupID)
}

func (s *stubLB) SetByGroup(ctx context.Context, in authapp.SetLoadBalanceInput) (*authdomain.LoadBalanceConfig, error) {
	return s.setFn(ctx, in)
}

// --- Helper to build a minimal handler with nil use cases (safe for tests that don't invoke them) ---

func newTestHandler(lb loadBalanceUsecase) *Handler {
	log, _ := zap.NewDevelopment()
	return &Handler{
		loadBalanceUC: lb,
		log:           log,
	}
}

// --- Helper to build a gin context with tenantID injected ---

func ginCtxWithTenant(tenantID, groupID, method string, body []byte) (*gin.Context, *httptest.ResponseRecorder) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	var req *http.Request
	if body != nil {
		req = httptest.NewRequest(method, "/tenant/groups/"+groupID+"/load-balance", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
	} else {
		req = httptest.NewRequest(method, "/tenant/groups/"+groupID+"/load-balance", nil)
	}

	req = req.WithContext(middleware.SetTenantIDForTest(req.Context(), tenantID))
	c.Request = req
	c.Params = gin.Params{{Key: "id", Value: groupID}}

	return c, w
}

// --- Tests ---

// TestGetLoadBalance_Defaults verifies that when no config exists (ErrLoadBalanceNotFound)
// the handler returns 200 with default values.
func TestGetLoadBalance_Defaults(t *testing.T) {
	lb := &stubLB{
		getFn: func(_ context.Context, _, _ string) (*authdomain.LoadBalanceConfig, error) {
			return nil, authdomain.ErrLoadBalanceNotFound
		},
	}
	h := newTestHandler(lb)

	c, w := ginCtxWithTenant("tenant-1", "group-1", http.MethodGet, nil)
	h.GetLoadBalance(c)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, "round_robin", resp["algorithm"])
	assert.Equal(t, false, resp["active"])
	assert.Equal(t, false, resp["exists"])
}

// TestSetLoadBalance_InvalidAlgorithm verifies that an unknown algorithm value
// results in a 400 Bad Request response.
func TestSetLoadBalance_InvalidAlgorithm(t *testing.T) {
	lb := &stubLB{
		setFn: func(_ context.Context, _ authapp.SetLoadBalanceInput) (*authdomain.LoadBalanceConfig, error) {
			return nil, authdomain.ErrInvalidAlgorithm
		},
	}
	h := newTestHandler(lb)

	body, _ := json.Marshal(map[string]interface{}{"algorithm": "weird", "active": false})
	c, w := ginCtxWithTenant("tenant-1", "group-1", http.MethodPut, body)
	h.SetLoadBalance(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, "invalid algorithm", resp["error"])
}

// TestSetLoadBalance_GroupNotInTenant verifies that when the group does not belong
// to the tenant the handler returns 404.
func TestSetLoadBalance_GroupNotInTenant(t *testing.T) {
	lb := &stubLB{
		setFn: func(_ context.Context, _ authapp.SetLoadBalanceInput) (*authdomain.LoadBalanceConfig, error) {
			return nil, authapp.ErrGroupNotInTenant
		},
	}
	h := newTestHandler(lb)

	body, _ := json.Marshal(map[string]interface{}{"algorithm": "round_robin", "active": true})
	c, w := ginCtxWithTenant("tenant-1", "group-999", http.MethodPut, body)
	h.SetLoadBalance(c)

	assert.Equal(t, http.StatusNotFound, w.Code)

	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, "group not found", resp["error"])
}
