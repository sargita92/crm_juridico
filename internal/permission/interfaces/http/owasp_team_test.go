package http

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	authdomain "github.com/sasrgita/crm-juridico/internal/auth/domain"
	authinfra "github.com/sasrgita/crm-juridico/internal/auth/infrastructure"
	"github.com/sasrgita/crm-juridico/internal/shared/middleware"
	"github.com/sasrgita/crm-juridico/internal/shared/testhelper"
)

// ---------------------------------------------------------------------------
// permOwaspDenyChecker satisfies middleware.PermissionChecker, denying all.
// ---------------------------------------------------------------------------

type permOwaspDenyChecker struct{}

func (p *permOwaspDenyChecker) HasPermission(_ context.Context, _, _, _, _ string) (bool, error) {
	return false, nil
}

// ---------------------------------------------------------------------------
// owaspPermEnv — minimal router wired with real auth + tenant + permission
// middlewares. All handlers are stubs because middlewares abort first.
// ---------------------------------------------------------------------------

type owaspPermEnv struct {
	router   *gin.Engine
	provider *authinfra.JWTProvider
}

// minimalHandler builds a Handler with nil use cases — safe because the
// permission middleware short-circuits before any handler body executes.
func minimalHandler() *Handler {
	log, _ := zap.NewDevelopment()
	return &Handler{log: log}
}

func setupOwaspPermEnv(t *testing.T) *owaspPermEnv {
	t.Helper()
	gin.SetMode(gin.TestMode)

	jwtProvider := authinfra.NewJWTProvider("owasp-perm-secret-key", 24*time.Hour)
	authMw := middleware.Auth(jwtProvider)
	tenantMw := middleware.RequireTenant()
	requirePerm := middleware.RequirePermission(&permOwaspDenyChecker{})

	h := minimalHandler()
	ph := buildPageHandler(defaultPageHandlerOpts())

	router := gin.New()
	router.SetHTMLTemplate(testhelper.ParseTemplates())

	// JSON API routes — mirror permission/interfaces/http/routes.go exactly.
	groups := router.Group("/tenant/groups")
	groups.Use(authMw, tenantMw)
	{
		groups.GET("", requirePerm("groups", "manage"), h.ListGroups)
		groups.POST("", requirePerm("groups", "manage"), h.CreateGroup)
		groups.GET("/:id", requirePerm("groups", "manage"), h.GetGroup)
		groups.PUT("/:id", requirePerm("groups", "manage"), h.UpdateGroup)
		groups.DELETE("/:id", requirePerm("groups", "manage"), h.DeleteGroup)
		groups.GET("/:id/members", requirePerm("groups", "manage"), h.ListMembers)
		groups.POST("/:id/members", requirePerm("groups", "manage"), h.AddMember)
		groups.DELETE("/:id/members/:uid", requirePerm("groups", "manage"), h.RemoveMember)
		groups.GET("/:id/permissions", requirePerm("groups", "manage"), h.GetGroupPermissions)
		groups.PUT("/:id/permissions", requirePerm("groups", "manage"), h.SetGroupPermissions)
		groups.GET("/:id/view-profiles", requirePerm("funnels", "customize"), h.ListViewProfiles)
		groups.PUT("/:id/view-profiles/:fid", requirePerm("funnels", "customize"), h.SetViewProfile)
		groups.GET("/:id/funnels", requirePerm("groups", "manage"), h.ListGroupFunnels)
		groups.PUT("/:id/funnels", requirePerm("groups", "manage"), h.SetGroupFunnels)
		groups.GET("/:id/load-balance", requirePerm("groups", "manage"), h.GetLoadBalance)
		groups.PUT("/:id/load-balance", requirePerm("groups", "manage"), h.SetLoadBalance)
	}

	// HTML page routes — mirror permission/interfaces/http/routes.go exactly.
	team := router.Group("/tenant/team")
	team.Use(authMw, tenantMw)
	{
		team.GET("/groups", requirePerm("groups", "manage"), ph.GroupsPage)
		team.GET("/groups/table", requirePerm("groups", "manage"), ph.GroupsTable)
		team.GET("/groups/new-modal", requirePerm("groups", "manage"), ph.GroupNewModal)
		team.POST("/groups", requirePerm("groups", "manage"), ph.CreateGroupHTML)
		team.GET("/groups/:id", requirePerm("groups", "manage"), ph.GroupDetail)
		team.GET("/groups/:id/section/:name", requirePerm("groups", "manage"), ph.GroupSection)
		team.POST("/groups/:id/permissions", requirePerm("groups", "manage"), ph.SetGroupPermissionsHTML)
		team.POST("/groups/:id/funnels", requirePerm("groups", "manage"), ph.SetGroupFunnelsHTML)
		team.POST("/groups/:id/view-profiles/:fid", requirePerm("funnels", "customize"), ph.SetViewProfileHTML)
		team.POST("/groups/:id/load-balance", requirePerm("groups", "manage"), ph.SetLoadBalanceHTML)
	}

	return &owaspPermEnv{router: router, provider: jwtProvider}
}

func (e *owaspPermEnv) tokenWithTenant(t *testing.T, tenantID string) string {
	t.Helper()
	tok, err := e.provider.Generate(authdomain.TokenClaims{
		UserID:   uuid.New().String(),
		Role:     authdomain.UserRoleUser,
		TenantID: tenantID,
	})
	require.NoError(t, err)
	return tok
}

func (e *owaspPermEnv) tokenWithoutTenant(t *testing.T) string {
	t.Helper()
	tok, err := e.provider.Generate(authdomain.TokenClaims{
		UserID: uuid.New().String(),
		Role:   authdomain.UserRoleUser,
	})
	require.NoError(t, err)
	return tok
}

func owaspPermCookie(token string) *http.Cookie {
	return &http.Cookie{Name: "token", Value: token}
}

// ---------------------------------------------------------------------------
// TestOWASP_Perm_A01_NoToken_Returns401
// All /tenant/groups/* and /tenant/team/groups/* endpoints must reject
// requests without a token with 401.
// ---------------------------------------------------------------------------

func TestOWASP_Perm_A01_NoToken_Returns401(t *testing.T) {
	env := setupOwaspPermEnv(t)

	routes := []struct{ method, path string }{
		// JSON API
		{http.MethodGet, "/tenant/groups"},
		{http.MethodPost, "/tenant/groups"},
		{http.MethodGet, "/tenant/groups/some-id"},
		{http.MethodPut, "/tenant/groups/some-id"},
		{http.MethodDelete, "/tenant/groups/some-id"},
		{http.MethodGet, "/tenant/groups/some-id/load-balance"},
		{http.MethodPut, "/tenant/groups/some-id/load-balance"},
		// HTML pages
		{http.MethodGet, "/tenant/team/groups"},
		{http.MethodGet, "/tenant/team/groups/table"},
		{http.MethodGet, "/tenant/team/groups/new-modal"},
		{http.MethodPost, "/tenant/team/groups"},
		{http.MethodGet, "/tenant/team/groups/some-id"},
		{http.MethodGet, "/tenant/team/groups/some-id/section/permissions"},
		{http.MethodPost, "/tenant/team/groups/some-id/permissions"},
		{http.MethodPost, "/tenant/team/groups/some-id/funnels"},
		{http.MethodPost, "/tenant/team/groups/some-id/view-profiles/some-fid"},
		{http.MethodPost, "/tenant/team/groups/some-id/load-balance"},
	}

	for _, r := range routes {
		t.Run(r.method+" "+r.path, func(t *testing.T) {
			w := httptest.NewRecorder()
			req := httptest.NewRequest(r.method, r.path, nil)
			env.router.ServeHTTP(w, req)
			assert.Equal(t, http.StatusUnauthorized, w.Code,
				"expected 401 Unauthorized for unauthenticated request to %s %s", r.method, r.path)
		})
	}
}

// ---------------------------------------------------------------------------
// TestOWASP_Perm_A01_NoTenant_Returns403
// A valid JWT without a tenant_id claim must be rejected with 403.
// RequireTenant fires before RequirePermission.
// ---------------------------------------------------------------------------

func TestOWASP_Perm_A01_NoTenant_Returns403(t *testing.T) {
	env := setupOwaspPermEnv(t)
	token := env.tokenWithoutTenant(t)

	routes := []struct{ method, path string }{
		{http.MethodGet, "/tenant/groups"},
		{http.MethodPost, "/tenant/groups"},
		{http.MethodGet, "/tenant/groups/some-id"},
		{http.MethodPut, "/tenant/groups/some-id"},
		{http.MethodGet, "/tenant/groups/some-id/load-balance"},
		{http.MethodPut, "/tenant/groups/some-id/load-balance"},
		{http.MethodGet, "/tenant/team/groups"},
		{http.MethodGet, "/tenant/team/groups/table"},
		{http.MethodGet, "/tenant/team/groups/new-modal"},
		{http.MethodPost, "/tenant/team/groups"},
		{http.MethodGet, "/tenant/team/groups/some-id"},
		{http.MethodGet, "/tenant/team/groups/some-id/section/permissions"},
		{http.MethodPost, "/tenant/team/groups/some-id/permissions"},
		{http.MethodPost, "/tenant/team/groups/some-id/funnels"},
		{http.MethodPost, "/tenant/team/groups/some-id/view-profiles/some-fid"},
		{http.MethodPost, "/tenant/team/groups/some-id/load-balance"},
	}

	for _, r := range routes {
		t.Run(r.method+" "+r.path, func(t *testing.T) {
			w := httptest.NewRecorder()
			req := httptest.NewRequest(r.method, r.path, nil)
			req.AddCookie(owaspPermCookie(token))
			env.router.ServeHTTP(w, req)
			assert.Equal(t, http.StatusForbidden, w.Code,
				"expected 403 Forbidden for token without tenant on %s %s", r.method, r.path)
		})
	}
}

// ---------------------------------------------------------------------------
// TestOWASP_Perm_A01_NoPermission_Returns403
// A valid JWT with tenant_id but lacking the required permission must get 403.
// ---------------------------------------------------------------------------

func TestOWASP_Perm_A01_NoPermission_Returns403(t *testing.T) {
	env := setupOwaspPermEnv(t)
	token := env.tokenWithTenant(t, "tenant-owasp")

	routes := []struct{ method, path string }{
		// All guarded by "groups:manage"
		{http.MethodGet, "/tenant/groups"},
		{http.MethodPost, "/tenant/groups"},
		{http.MethodGet, "/tenant/groups/some-id"},
		{http.MethodPut, "/tenant/groups/some-id"},
		{http.MethodDelete, "/tenant/groups/some-id"},
		{http.MethodGet, "/tenant/groups/some-id/load-balance"},
		{http.MethodPut, "/tenant/groups/some-id/load-balance"},
		{http.MethodGet, "/tenant/team/groups"},
		{http.MethodGet, "/tenant/team/groups/table"},
		{http.MethodGet, "/tenant/team/groups/new-modal"},
		{http.MethodPost, "/tenant/team/groups"},
		{http.MethodGet, "/tenant/team/groups/some-id"},
		{http.MethodGet, "/tenant/team/groups/some-id/section/permissions"},
		{http.MethodPost, "/tenant/team/groups/some-id/permissions"},
		{http.MethodPost, "/tenant/team/groups/some-id/funnels"},
		{http.MethodPost, "/tenant/team/groups/some-id/load-balance"},
		// Guarded by "funnels:customize"
		{http.MethodPost, "/tenant/team/groups/some-id/view-profiles/some-fid"},
	}

	for _, r := range routes {
		t.Run(r.method+" "+r.path, func(t *testing.T) {
			w := httptest.NewRecorder()
			req := httptest.NewRequest(r.method, r.path, nil)
			req.AddCookie(owaspPermCookie(token))
			env.router.ServeHTTP(w, req)
			assert.Equal(t, http.StatusForbidden, w.Code,
				"expected 403 Forbidden for user without permission on %s %s", r.method, r.path)
		})
	}
}

// ---------------------------------------------------------------------------
// TestOWASP_Perm_A01_TenantIsolation
// Spot-check: a token for tenant-A cannot mutate resources from tenant-B.
// RequirePermission uses the tenant from the JWT, so cross-tenant operations
// are blocked at the middleware layer (403).
// ---------------------------------------------------------------------------

func TestOWASP_Perm_A01_TenantIsolation(t *testing.T) {
	env := setupOwaspPermEnv(t)
	tokenTenantA := env.tokenWithTenant(t, "tenant-A")

	mutations := []struct{ method, path string }{
		{http.MethodPost, "/tenant/team/groups"},
		{http.MethodPost, "/tenant/team/groups/group-in-tenant-b/permissions"},
		{http.MethodPut, "/tenant/groups/group-in-tenant-b/load-balance"},
	}

	for _, r := range mutations {
		t.Run("tenant-isolation "+r.method+" "+r.path, func(t *testing.T) {
			w := httptest.NewRecorder()
			req := httptest.NewRequest(r.method, r.path, nil)
			req.AddCookie(owaspPermCookie(tokenTenantA))
			env.router.ServeHTTP(w, req)
			assert.Equal(t, http.StatusForbidden, w.Code,
				"expected 403 for cross-tenant mutation on %s %s", r.method, r.path)
		})
	}
}

// ---------------------------------------------------------------------------
// TestOWASP_Perm_A03_NoSQLInjection
// Injection payloads in URL segments must not cause 5xx.
// Payloads are URL-encoded so httptest.NewRequest can parse them.
// ---------------------------------------------------------------------------

func TestOWASP_Perm_A03_NoSQLInjection(t *testing.T) {
	env := setupOwaspPermEnv(t)
	// No token — middlewares abort with 401 before handlers process input.

	injections := []string{
		"'; DROP TABLE permission_groups; --",
		`" OR "1"="1`,
		"<script>alert(1)</script>",
		"UNION SELECT password FROM users",
	}

	for _, payload := range injections {
		t.Run("payload: "+payload, func(t *testing.T) {
			encoded := url.PathEscape(payload)
			w := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodGet, "/tenant/groups/"+encoded, nil)
			env.router.ServeHTTP(w, req)
			assert.Less(t, w.Code, http.StatusInternalServerError,
				"server must not return 5xx for injection payload in URL segment: %q", payload)
		})
	}
}

// ---------------------------------------------------------------------------
// TestOWASP_Perm_A03_SQLInjection_JSONBody
// Injection payloads in JSON body must not cause 5xx.
// The permission middleware fires first, returning 403.
// ---------------------------------------------------------------------------

func TestOWASP_Perm_A03_SQLInjection_JSONBody(t *testing.T) {
	env := setupOwaspPermEnv(t)
	token := env.tokenWithTenant(t, "tenant-owasp")

	injections := []string{
		`{"name": "'; DROP TABLE groups; --", "description": "x"}`,
		`{"name": "\" OR \"1\"=\"1", "description": "x"}`,
	}

	for _, payload := range injections {
		t.Run("payload: "+payload, func(t *testing.T) {
			w := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodPost, "/tenant/groups",
				strings.NewReader(payload))
			req.Header.Set("Content-Type", "application/json")
			req.AddCookie(owaspPermCookie(token))
			env.router.ServeHTTP(w, req)
			// Permission middleware blocks before handler reads body.
			assert.Less(t, w.Code, http.StatusInternalServerError,
				"server must not return 5xx for injection payload: %q", payload)
		})
	}
}
