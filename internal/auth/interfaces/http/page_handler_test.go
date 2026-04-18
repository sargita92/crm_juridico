package http

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"github.com/sasrgita/crm-juridico/internal/auth/application"
	authdomain "github.com/sasrgita/crm-juridico/internal/auth/domain"
	permapp "github.com/sasrgita/crm-juridico/internal/permission/application"
	"github.com/sasrgita/crm-juridico/internal/shared/middleware"
	"github.com/sasrgita/crm-juridico/internal/shared/testhelper"
)

// ---------------------------------------------------------------------------
// Stubs
// ---------------------------------------------------------------------------

type stubInviteUC struct {
	listFn func(ctx context.Context, tenantID string) ([]application.InviteOutput, error)
	genFn  func(ctx context.Context, tenantID, createdBy string, groupIDs []string, days int) (*application.InviteOutput, error)
}

func (s *stubInviteUC) ListInvites(ctx context.Context, tenantID string) ([]application.InviteOutput, error) {
	if s.listFn == nil {
		return nil, nil
	}
	return s.listFn(ctx, tenantID)
}

func (s *stubInviteUC) GenerateInvite(ctx context.Context, tenantID, createdBy string, groupIDs []string, days int) (*application.InviteOutput, error) {
	if s.genFn == nil {
		return nil, nil
	}
	return s.genFn(ctx, tenantID, createdBy, groupIDs, days)
}

type stubManageUsers struct {
	listFn func(ctx context.Context, tenantID string) ([]application.UserOutput, error)
	setFn  func(ctx context.Context, userID, tenantID, whatsAppID string) error
}

func (s *stubManageUsers) ListTenantUsers(ctx context.Context, tenantID string) ([]application.UserOutput, error) {
	if s.listFn == nil {
		return nil, nil
	}
	return s.listFn(ctx, tenantID)
}

func (s *stubManageUsers) SetWhatsAppID(ctx context.Context, userID, tenantID, whatsAppID string) error {
	if s.setFn == nil {
		return nil
	}
	return s.setFn(ctx, userID, tenantID, whatsAppID)
}

type stubListGroups struct {
	execFn func(ctx context.Context, tenantID string) ([]permapp.GroupOutput, error)
}

func (s *stubListGroups) Execute(ctx context.Context, tenantID string) ([]permapp.GroupOutput, error) {
	if s.execFn == nil {
		return nil, nil
	}
	return s.execFn(ctx, tenantID)
}

type stubManageUserPerms struct {
	getFn func(ctx context.Context, tenantID, userID string) ([]permapp.PermissionOutput, error)
	setFn func(ctx context.Context, tenantID, userID string, inputs []permapp.PermissionInput) error
}

func (s *stubManageUserPerms) GetUserPermissions(ctx context.Context, tenantID, userID string) ([]permapp.PermissionOutput, error) {
	if s.getFn == nil {
		return nil, nil
	}
	return s.getFn(ctx, tenantID, userID)
}

func (s *stubManageUserPerms) SetUserPermissions(ctx context.Context, tenantID, userID string, inputs []permapp.PermissionInput) error {
	if s.setFn == nil {
		return nil
	}
	return s.setFn(ctx, tenantID, userID, inputs)
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func newTestPageHandler(
	invite inviteUsecase,
	users manageUsersUsecase,
	groups listGroupsUsecase,
	perms manageUserPermsUsecase,
) *PageHandler {
	log, _ := zap.NewDevelopment()
	return &PageHandler{
		inviteUC:        invite,
		manageUsers:     users,
		listGroupsUC:    groups,
		manageUserPerms: perms,
		resolverUC:      nil,
		log:             log,
	}
}

const pageTestTenantID = "tenant-page-test"

// buildPageRouter wires a PageHandler into a gin router and injects tenant + optional claims.
func buildPageRouter(h *PageHandler, claims *authdomain.TokenClaims) *gin.Engine {
	tmpl := testhelper.ParseTemplates()
	r := gin.New()
	r.SetHTMLTemplate(tmpl)

	// Inject tenant ID and (optionally) claims via middleware.
	r.Use(func(c *gin.Context) {
		ctx := middleware.SetTenantIDForTest(c.Request.Context(), pageTestTenantID)
		if claims != nil {
			ctx = middleware.SetClaimsForTest(ctx, claims)
		}
		c.Request = c.Request.WithContext(ctx)
		c.Next()
	})

	r.GET("/team", h.RedirectToUsers)
	r.GET("/team/users", h.UsersPage)
	r.GET("/team/users/table", h.UsersTable)
	r.GET("/team/invites/new-modal", h.InviteNewModal)
	r.POST("/team/invites", h.CreateInvite)
	r.GET("/team/users/:id/permissions-modal", h.UserPermissionsModal)
	r.POST("/team/users/:id/permissions", h.SetUserPermissionsHTML)
	r.GET("/team/users/:id/whatsapp-modal", h.UserWhatsAppModal)
	r.POST("/team/users/:id/whatsapp", h.SetUserWhatsApp)

	return r
}

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

func TestPageHandler_RedirectToUsers(t *testing.T) {
	h := newTestPageHandler(
		&stubInviteUC{},
		&stubManageUsers{},
		&stubListGroups{},
		&stubManageUserPerms{},
	)
	r := buildPageRouter(h, nil)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/team", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusFound, w.Code)
	assert.Equal(t, "/tenant/team/users", w.Header().Get("Location"))
}

func TestPageHandler_UsersPage_Renders(t *testing.T) {
	h := newTestPageHandler(
		&stubInviteUC{},
		&stubManageUsers{
			listFn: func(_ context.Context, _ string) ([]application.UserOutput, error) {
				return []application.UserOutput{{ID: "u1", Name: "Alice", Email: "alice@test.com"}}, nil
			},
		},
		&stubListGroups{},
		&stubManageUserPerms{},
	)
	r := buildPageRouter(h, nil)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/team/users", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	body := w.Body.String()
	assert.True(t,
		strings.Contains(body, "Equipe") || strings.Contains(body, "Usu"),
		"expected 'Equipe' or 'Usuários' in body, got: %s", body)
}

func TestPageHandler_UsersTable_Renders(t *testing.T) {
	h := newTestPageHandler(
		&stubInviteUC{},
		&stubManageUsers{
			listFn: func(_ context.Context, _ string) ([]application.UserOutput, error) {
				return []application.UserOutput{{ID: "u2", Name: "Bob", Email: "bob@test.com"}}, nil
			},
		},
		&stubListGroups{},
		&stubManageUserPerms{},
	)
	r := buildPageRouter(h, nil)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/team/users/table", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestPageHandler_InviteNewModal_Renders(t *testing.T) {
	h := newTestPageHandler(
		&stubInviteUC{},
		&stubManageUsers{},
		&stubListGroups{
			execFn: func(_ context.Context, _ string) ([]permapp.GroupOutput, error) {
				return []permapp.GroupOutput{
					{ID: "g1", Name: "Advogados"},
					{ID: "g2", Name: "Assistentes"},
				}, nil
			},
		},
		&stubManageUserPerms{},
	)
	r := buildPageRouter(h, nil)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/team/invites/new-modal", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	body := w.Body.String()
	assert.Contains(t, body, "Advogados")
}

func TestPageHandler_CreateInvite_Success(t *testing.T) {
	h := newTestPageHandler(
		&stubInviteUC{
			genFn: func(_ context.Context, _, _ string, _ []string, _ int) (*application.InviteOutput, error) {
				return &application.InviteOutput{
					ID:    "inv-1",
					Token: "abc123token",
				}, nil
			},
		},
		&stubManageUsers{},
		&stubListGroups{},
		&stubManageUserPerms{},
	)
	claims := &authdomain.TokenClaims{UserID: "user-owner", TenantID: pageTestTenantID}
	r := buildPageRouter(h, claims)

	form := url.Values{
		"expires_in_days": {"7"},
	}
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/team/invites",
		strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "/invite/abc123token")
}

func TestPageHandler_CreateInvite_NoAuth(t *testing.T) {
	h := newTestPageHandler(
		&stubInviteUC{},
		&stubManageUsers{},
		&stubListGroups{},
		&stubManageUserPerms{},
	)
	// Build router without injecting claims.
	r := buildPageRouter(h, nil)

	form := url.Values{"expires_in_days": {"7"}}
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/team/invites",
		strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestPageHandler_UserPermissionsModal_Renders(t *testing.T) {
	h := newTestPageHandler(
		&stubInviteUC{},
		&stubManageUsers{},
		&stubListGroups{},
		&stubManageUserPerms{
			getFn: func(_ context.Context, _, _ string) ([]permapp.PermissionOutput, error) {
				return []permapp.PermissionOutput{
					{ID: "p1", Resource: "leads", Action: "read"},
				}, nil
			},
		},
	)
	r := buildPageRouter(h, nil)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/team/users/user-42/permissions-modal", nil)
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	body := w.Body.String()
	assert.Contains(t, body, "leads")
	assert.Contains(t, body, "read")
}

func TestPageHandler_SetUserPermissionsHTML_Success(t *testing.T) {
	h := newTestPageHandler(
		&stubInviteUC{},
		&stubManageUsers{},
		&stubListGroups{},
		&stubManageUserPerms{
			setFn: func(_ context.Context, _, _ string, _ []permapp.PermissionInput) error {
				return nil
			},
		},
	)
	r := buildPageRouter(h, nil)

	form := url.Values{
		"perms": {"leads:read", "users:update"},
	}
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/team/users/user-42/permissions",
		strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNoContent, w.Code)
	assert.Equal(t, "refreshTeam", w.Header().Get("HX-Trigger"))
}

func TestPageHandler_UserWhatsAppModal_Renders(t *testing.T) {
	const wid = "5511988887777"
	h := newTestPageHandler(
		&stubInviteUC{},
		&stubManageUsers{
			listFn: func(_ context.Context, _ string) ([]application.UserOutput, error) {
				return []application.UserOutput{
					{ID: "user-55", Name: "Carol", WhatsAppID: wid},
				}, nil
			},
		},
		&stubListGroups{},
		&stubManageUserPerms{},
	)
	r := buildPageRouter(h, nil)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/team/users/user-55/whatsapp-modal", nil)
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), wid)
}

func TestPageHandler_SetUserWhatsApp_Success(t *testing.T) {
	h := newTestPageHandler(
		&stubInviteUC{},
		&stubManageUsers{
			setFn: func(_ context.Context, _, _, _ string) error { return nil },
		},
		&stubListGroups{},
		&stubManageUserPerms{},
	)
	r := buildPageRouter(h, nil)

	form := url.Values{"whatsapp_id": {"5511999990000"}}
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/team/users/user-55/whatsapp",
		strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNoContent, w.Code)
	assert.Equal(t, "refreshTeam", w.Header().Get("HX-Trigger"))
}

func TestPageHandler_SetUserWhatsApp_EmptyInput(t *testing.T) {
	h := newTestPageHandler(
		&stubInviteUC{},
		&stubManageUsers{},
		&stubListGroups{},
		&stubManageUserPerms{},
	)
	r := buildPageRouter(h, nil)

	form := url.Values{"whatsapp_id": {""}}
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/team/users/user-55/whatsapp",
		strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// ---------------------------------------------------------------------------
// Error-path tests (cover missing branches)
// ---------------------------------------------------------------------------

func TestPageHandler_UsersPage_ListUsersError(t *testing.T) {
	h := newTestPageHandler(
		&stubInviteUC{},
		&stubManageUsers{
			listFn: func(_ context.Context, _ string) ([]application.UserOutput, error) {
				return nil, assert.AnError
			},
		},
		&stubListGroups{},
		&stubManageUserPerms{},
	)
	r := buildPageRouter(h, nil)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/team/users", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestPageHandler_UsersTable_ListUsersError(t *testing.T) {
	h := newTestPageHandler(
		&stubInviteUC{},
		&stubManageUsers{
			listFn: func(_ context.Context, _ string) ([]application.UserOutput, error) {
				return nil, assert.AnError
			},
		},
		&stubListGroups{},
		&stubManageUserPerms{},
	)
	r := buildPageRouter(h, nil)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/team/users/table", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestPageHandler_InviteNewModal_ListGroupsError(t *testing.T) {
	h := newTestPageHandler(
		&stubInviteUC{},
		&stubManageUsers{},
		&stubListGroups{
			execFn: func(_ context.Context, _ string) ([]permapp.GroupOutput, error) {
				return nil, assert.AnError
			},
		},
		&stubManageUserPerms{},
	)
	r := buildPageRouter(h, nil)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/team/invites/new-modal", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestPageHandler_CreateInvite_GenerateError(t *testing.T) {
	h := newTestPageHandler(
		&stubInviteUC{
			genFn: func(_ context.Context, _, _ string, _ []string, _ int) (*application.InviteOutput, error) {
				return nil, assert.AnError
			},
		},
		&stubManageUsers{},
		&stubListGroups{},
		&stubManageUserPerms{},
	)
	claims := &authdomain.TokenClaims{UserID: "owner", TenantID: pageTestTenantID}
	r := buildPageRouter(h, claims)

	form := url.Values{"expires_in_days": {"7"}}
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/team/invites",
		strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestPageHandler_UserPermissionsModal_GetError(t *testing.T) {
	h := newTestPageHandler(
		&stubInviteUC{},
		&stubManageUsers{},
		&stubListGroups{},
		&stubManageUserPerms{
			getFn: func(_ context.Context, _, _ string) ([]permapp.PermissionOutput, error) {
				return nil, assert.AnError
			},
		},
	)
	r := buildPageRouter(h, nil)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/team/users/u1/permissions-modal", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestPageHandler_SetUserPermissionsHTML_SetError(t *testing.T) {
	h := newTestPageHandler(
		&stubInviteUC{},
		&stubManageUsers{},
		&stubListGroups{},
		&stubManageUserPerms{
			setFn: func(_ context.Context, _, _ string, _ []permapp.PermissionInput) error {
				return assert.AnError
			},
		},
	)
	r := buildPageRouter(h, nil)

	form := url.Values{"perms": {"leads:read"}}
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/team/users/u1/permissions",
		strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
}

func TestPageHandler_UserWhatsAppModal_ListError(t *testing.T) {
	h := newTestPageHandler(
		&stubInviteUC{},
		&stubManageUsers{
			listFn: func(_ context.Context, _ string) ([]application.UserOutput, error) {
				return nil, assert.AnError
			},
		},
		&stubListGroups{},
		&stubManageUserPerms{},
	)
	r := buildPageRouter(h, nil)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/team/users/u1/whatsapp-modal", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestPageHandler_SetUserWhatsApp_SetError(t *testing.T) {
	h := newTestPageHandler(
		&stubInviteUC{},
		&stubManageUsers{
			setFn: func(_ context.Context, _, _, _ string) error { return assert.AnError },
		},
		&stubListGroups{},
		&stubManageUserPerms{},
	)
	r := buildPageRouter(h, nil)

	form := url.Values{"whatsapp_id": {"5511999990000"}}
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/team/users/u1/whatsapp",
		strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
}
