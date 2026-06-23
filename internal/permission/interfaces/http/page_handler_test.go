package http

import (
	"context"
	"errors"
	"fmt"
	"html/template"
	"net/http"
	"net/http/httptest"
	"net/url"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	authapp "github.com/sasrgita/crm-juridico/internal/auth/application"
	authdomain "github.com/sasrgita/crm-juridico/internal/auth/domain"
	funnelapp "github.com/sasrgita/crm-juridico/internal/funnel/application"
	funneldomain "github.com/sasrgita/crm-juridico/internal/funnel/domain"
	"github.com/sasrgita/crm-juridico/internal/permission/application"
	"github.com/sasrgita/crm-juridico/internal/shared/middleware"
	"github.com/sasrgita/crm-juridico/internal/shared/ui"
)

// ---- helpers ----------------------------------------------------------------

// templateRoot returns the absolute path to web/templates relative to this file.
func templateRoot() string {
	_, filename, _, _ := runtime.Caller(0)
	// file is at internal/permission/interfaces/http/page_handler_test.go
	// web/templates is 5 levels up + web/templates
	base := filepath.Join(filepath.Dir(filename), "..", "..", "..", "..", "web", "templates")
	return filepath.Join(base, "**", "*.html")
}

// newPageRouter builds a gin.Engine with all templates loaded and the PageHandler registered.
func newPageRouter(ph *PageHandler) *gin.Engine {
	r := gin.New()

	funcMap := template.FuncMap{
		"dict": func(values ...interface{}) map[string]interface{} {
			m := make(map[string]interface{})
			for i := 0; i+1 < len(values); i += 2 {
				key, _ := values[i].(string)
				m[key] = values[i+1]
			}
			return m
		},
		"add":                     func(a, b int) int { return a + b },
		"sub":                     func(a, b int) int { return a - b },
		"aiPlaygroundEnabled":     func() bool { return false },
		"formatFileSize":          func(size int64) string { return "0 B" },
		"typeIcon":                func(t string) string { return "🔔" },
		"typeLabel":               func(t string) string { return "" },
		"columnTypeLabel":         ui.ColumnTypeLabel,
		"permissionActionLabel":   ui.PermissionActionLabel,
		"permissionResourceLabel": ui.PermissionResourceLabel,
		"relativeTime":            func(t time.Time) string { return "" },
		"formatValor": func(c *int64) string {
			if c == nil {
				return ""
			}
			return fmt.Sprintf("%.2f", float64(*c)/100.0)
		},
		"uint8Or": func(p *uint8, fallback uint8) uint8 {
			if p == nil {
				return fallback
			}
			return *p
		},
		"dateOr": func(p *time.Time) string {
			if p == nil {
				return ""
			}
			return p.Format("2006-01-02")
		},
	}

	tmpl := template.Must(template.New("").Funcs(funcMap).ParseGlob(templateRoot()))
	r.SetHTMLTemplate(tmpl)

	// Inject tenantID middleware
	r.Use(func(c *gin.Context) {
		ctx := middleware.SetTenantIDForTest(c.Request.Context(), "tenant-1")
		c.Request = c.Request.WithContext(ctx)
		c.Next()
	})

	r.GET("/team/groups", ph.GroupsPage)
	r.GET("/team/groups/table", ph.GroupsTable)
	r.GET("/team/groups/new-modal", ph.GroupNewModal)
	r.POST("/team/groups", ph.CreateGroupHTML)
	r.GET("/team/groups/:id", ph.GroupDetail)
	r.GET("/team/groups/:id/section/:name", ph.GroupSection)
	r.POST("/team/groups/:id/permissions", ph.SetGroupPermissionsHTML)
	r.POST("/team/groups/:id/funnels", ph.SetGroupFunnelsHTML)
	r.POST("/team/groups/:id/view-profiles/:fid", ph.SetViewProfileHTML)
	r.POST("/team/groups/:id/load-balance", ph.SetLoadBalanceHTML)

	return r
}

// nopLogger returns a no-op zap logger for tests.
func nopLogger() *zap.Logger {
	log, _ := zap.NewDevelopment()
	return log
}

// ---- stub types -------------------------------------------------------------

type stubCreateGroup struct {
	fn func(ctx context.Context, in application.CreateGroupInput) (*application.GroupOutput, error)
}

func (s *stubCreateGroup) Execute(ctx context.Context, in application.CreateGroupInput) (*application.GroupOutput, error) {
	return s.fn(ctx, in)
}

type stubListGroups struct {
	fn func(ctx context.Context, tenantID string) ([]application.GroupOutput, error)
}

func (s *stubListGroups) Execute(ctx context.Context, tenantID string) ([]application.GroupOutput, error) {
	return s.fn(ctx, tenantID)
}

type stubGetGroup struct {
	fn func(ctx context.Context, tenantID, groupID string) (*application.GroupOutput, error)
}

func (s *stubGetGroup) Execute(ctx context.Context, tenantID, groupID string) (*application.GroupOutput, error) {
	return s.fn(ctx, tenantID, groupID)
}

type stubUpdateGroup struct{}

func (s *stubUpdateGroup) Execute(_ context.Context, _ application.UpdateGroupInput) (*application.GroupOutput, error) {
	return nil, nil
}

type stubDeleteGroup struct{}

func (s *stubDeleteGroup) Execute(_ context.Context, _, _ string) error { return nil }

type stubManageMembers struct {
	fn func(ctx context.Context, groupID string) ([]application.MemberOutput, error)
}

func (s *stubManageMembers) ListMembers(ctx context.Context, groupID string) ([]application.MemberOutput, error) {
	return s.fn(ctx, groupID)
}

type stubManagePerms struct {
	getFn func(ctx context.Context, groupID string) ([]application.PermissionOutput, error)
	setFn func(ctx context.Context, tenantID, groupID string, inputs []application.PermissionInput) error
}

func (s *stubManagePerms) GetGroupPermissions(ctx context.Context, groupID string) ([]application.PermissionOutput, error) {
	return s.getFn(ctx, groupID)
}
func (s *stubManagePerms) SetGroupPermissions(ctx context.Context, tenantID, groupID string, inputs []application.PermissionInput) error {
	return s.setFn(ctx, tenantID, groupID, inputs)
}

type stubManageVP struct {
	listFn func(ctx context.Context, groupID string) ([]application.ViewProfileOutput, error)
	setFn  func(ctx context.Context, in application.ViewProfileInput) error
}

func (s *stubManageVP) ListByGroup(ctx context.Context, groupID string) ([]application.ViewProfileOutput, error) {
	return s.listFn(ctx, groupID)
}
func (s *stubManageVP) SetViewProfile(ctx context.Context, in application.ViewProfileInput) error {
	return s.setFn(ctx, in)
}

type stubManageGF struct {
	listFn func(ctx context.Context, groupID string) ([]application.GroupFunnelOutput, error)
	setFn  func(ctx context.Context, in application.GroupFunnelInput) error
}

func (s *stubManageGF) ListByGroup(ctx context.Context, groupID string) ([]application.GroupFunnelOutput, error) {
	return s.listFn(ctx, groupID)
}
func (s *stubManageGF) SetGroupFunnel(ctx context.Context, in application.GroupFunnelInput) error {
	return s.setFn(ctx, in)
}

type stubPageLB struct {
	getFn func(ctx context.Context, tenantID, groupID string) (*authdomain.LoadBalanceConfig, error)
	setFn func(ctx context.Context, in authapp.SetLoadBalanceInput) (*authdomain.LoadBalanceConfig, error)
}

func (s *stubPageLB) GetByGroup(ctx context.Context, tenantID, groupID string) (*authdomain.LoadBalanceConfig, error) {
	return s.getFn(ctx, tenantID, groupID)
}
func (s *stubPageLB) SetByGroup(ctx context.Context, in authapp.SetLoadBalanceInput) (*authdomain.LoadBalanceConfig, error) {
	return s.setFn(ctx, in)
}

type stubListFunnels struct {
	fn func(ctx context.Context, tenantID string) ([]funnelapp.FunnelItem, error)
}

func (s *stubListFunnels) Execute(ctx context.Context, tenantID string) ([]funnelapp.FunnelItem, error) {
	return s.fn(ctx, tenantID)
}

type stubUsersList struct {
	fn func(ctx context.Context, tenantID string) ([]authapp.UserOutput, error)
}

func (s *stubUsersList) ListTenantUsers(ctx context.Context, tenantID string) ([]authapp.UserOutput, error) {
	return s.fn(ctx, tenantID)
}

type stubColumnRepo struct {
	fn func(ctx context.Context, funnelID string) ([]funneldomain.Column, error)
}

func (s *stubColumnRepo) Create(_ context.Context, _ *funneldomain.Column) error { return nil }
func (s *stubColumnRepo) FindByID(_ context.Context, _ string) (*funneldomain.Column, error) {
	return nil, nil
}
func (s *stubColumnRepo) Update(_ context.Context, _ *funneldomain.Column) error { return nil }
func (s *stubColumnRepo) Delete(_ context.Context, _ string) error               { return nil }
func (s *stubColumnRepo) FindByFunnelID(ctx context.Context, funnelID string) ([]funneldomain.Column, error) {
	return s.fn(ctx, funnelID)
}
func (s *stubColumnRepo) FindEntryByFunnelID(_ context.Context, _ string) (*funneldomain.Column, error) {
	return nil, nil
}
func (s *stubColumnRepo) CountByFunnelID(_ context.Context, _ string) (int, error)  { return 0, nil }
func (s *stubColumnRepo) GetMaxOrderIndex(_ context.Context, _ string) (int, error) { return 0, nil }
func (s *stubColumnRepo) SwapOrder(_ context.Context, _ string, _ int, _ string, _ int) error {
	return nil
}

// ---- minimal builder --------------------------------------------------------

type pageHandlerOpts struct {
	createGroup   pageCreateGroupUC
	listGroups    pageListGroupsUC
	getGroup      pageGetGroupUC
	updateGroup   pageUpdateGroupUC
	deleteGroup   pageDeleteGroupUC
	manageMembers pageManageMembersUC
	managePerms   pageManagePermsUC
	manageVP      pageManageVPUC
	manageGF      pageManageGFUC
	loadBalance   pageLoadBalanceUC
	listFunnels   pageListFunnelsUC
	columnRepo    funneldomain.ColumnRepository
	usersList     pageUsersListUC
}

func defaultPageHandlerOpts() pageHandlerOpts {
	return pageHandlerOpts{
		createGroup: &stubCreateGroup{fn: func(_ context.Context, in application.CreateGroupInput) (*application.GroupOutput, error) {
			return &application.GroupOutput{ID: "g-1", Name: in.Name}, nil
		}},
		listGroups: &stubListGroups{fn: func(_ context.Context, _ string) ([]application.GroupOutput, error) {
			return []application.GroupOutput{{ID: "g-1", Name: "Alpha"}}, nil
		}},
		getGroup: &stubGetGroup{fn: func(_ context.Context, _, groupID string) (*application.GroupOutput, error) {
			return &application.GroupOutput{ID: groupID, Name: "Alpha"}, nil
		}},
		updateGroup: &stubUpdateGroup{},
		deleteGroup: &stubDeleteGroup{},
		manageMembers: &stubManageMembers{fn: func(_ context.Context, _ string) ([]application.MemberOutput, error) {
			return []application.MemberOutput{}, nil
		}},
		managePerms: &stubManagePerms{
			getFn: func(_ context.Context, _ string) ([]application.PermissionOutput, error) {
				return []application.PermissionOutput{{Resource: "leads", Action: "read"}}, nil
			},
			setFn: func(_ context.Context, _, _ string, _ []application.PermissionInput) error {
				return nil
			},
		},
		manageVP: &stubManageVP{
			listFn: func(_ context.Context, _ string) ([]application.ViewProfileOutput, error) {
				return []application.ViewProfileOutput{}, nil
			},
			setFn: func(_ context.Context, _ application.ViewProfileInput) error { return nil },
		},
		manageGF: &stubManageGF{
			listFn: func(_ context.Context, _ string) ([]application.GroupFunnelOutput, error) {
				return []application.GroupFunnelOutput{}, nil
			},
			setFn: func(_ context.Context, _ application.GroupFunnelInput) error { return nil },
		},
		loadBalance: &stubPageLB{
			getFn: func(_ context.Context, _, _ string) (*authdomain.LoadBalanceConfig, error) {
				return nil, authdomain.ErrLoadBalanceNotFound
			},
			setFn: func(_ context.Context, _ authapp.SetLoadBalanceInput) (*authdomain.LoadBalanceConfig, error) {
				return &authdomain.LoadBalanceConfig{Algorithm: authdomain.AlgorithmRoundRobin}, nil
			},
		},
		listFunnels: &stubListFunnels{fn: func(_ context.Context, _ string) ([]funnelapp.FunnelItem, error) {
			return []funnelapp.FunnelItem{{ID: "f-1", Name: "Funil A"}}, nil
		}},
		columnRepo: &stubColumnRepo{fn: func(_ context.Context, _ string) ([]funneldomain.Column, error) {
			return []funneldomain.Column{}, nil
		}},
		usersList: &stubUsersList{fn: func(_ context.Context, _ string) ([]authapp.UserOutput, error) {
			return []authapp.UserOutput{}, nil
		}},
	}
}

func buildPageHandler(opts pageHandlerOpts) *PageHandler {
	return NewPageHandler(
		opts.createGroup,
		opts.listGroups,
		opts.getGroup,
		opts.updateGroup,
		opts.deleteGroup,
		opts.manageMembers,
		opts.managePerms,
		opts.manageVP,
		opts.manageGF,
		opts.loadBalance,
		opts.listFunnels,
		opts.columnRepo,
		opts.usersList,
		nopLogger(),
	)
}

// ---- tests ------------------------------------------------------------------

func TestGroupsPage_200(t *testing.T) {
	ph := buildPageHandler(defaultPageHandlerOpts())
	r := newPageRouter(ph)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/team/groups", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "Equipe")
}

func TestGroupsTable_200(t *testing.T) {
	ph := buildPageHandler(defaultPageHandlerOpts())
	r := newPageRouter(ph)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/team/groups/table", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "Alpha")
}

func TestGroupNewModal_200(t *testing.T) {
	ph := buildPageHandler(defaultPageHandlerOpts())
	r := newPageRouter(ph)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/team/groups/new-modal", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "form")
}

func TestCreateGroupHTML_Success(t *testing.T) {
	ph := buildPageHandler(defaultPageHandlerOpts())
	r := newPageRouter(ph)

	form := url.Values{}
	form.Set("name", "Vendas")
	form.Set("description", "Time de vendas")

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/team/groups", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Header().Get("HX-Redirect"), "/tenant/team/groups/")
}

func TestCreateGroupHTML_Failure_422(t *testing.T) {
	opts := defaultPageHandlerOpts()
	opts.createGroup = &stubCreateGroup{fn: func(_ context.Context, _ application.CreateGroupInput) (*application.GroupOutput, error) {
		return nil, errors.New("validation failed")
	}}
	ph := buildPageHandler(opts)
	r := newPageRouter(ph)

	form := url.Values{}
	form.Set("name", "")

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/team/groups", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
}

func TestGroupDetail_200(t *testing.T) {
	ph := buildPageHandler(defaultPageHandlerOpts())
	r := newPageRouter(ph)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/team/groups/g-1", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "Alpha")
}

func TestGroupDetail_404(t *testing.T) {
	opts := defaultPageHandlerOpts()
	opts.getGroup = &stubGetGroup{fn: func(_ context.Context, _, _ string) (*application.GroupOutput, error) {
		return nil, errors.New("not found")
	}}
	ph := buildPageHandler(opts)
	r := newPageRouter(ph)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/team/groups/unknown", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestGroupSection_UnknownName_404(t *testing.T) {
	ph := buildPageHandler(defaultPageHandlerOpts())
	r := newPageRouter(ph)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/team/groups/g-1/section/nonexistent", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestSectionMembers_200(t *testing.T) {
	ph := buildPageHandler(defaultPageHandlerOpts())
	r := newPageRouter(ph)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/team/groups/g-1/section/members", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "Membros")
}

func TestSectionPermissions_200(t *testing.T) {
	ph := buildPageHandler(defaultPageHandlerOpts())
	r := newPageRouter(ph)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/team/groups/g-1/section/permissions", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	body := w.Body.String()
	assert.Contains(t, body, "leads")
}

func TestSectionFunnels_200(t *testing.T) {
	ph := buildPageHandler(defaultPageHandlerOpts())
	r := newPageRouter(ph)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/team/groups/g-1/section/funnels", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "Funis")
}

func TestSectionViewProfiles_EmptyFunnels(t *testing.T) {
	opts := defaultPageHandlerOpts()
	// No funnels assigned to the group -> empty funnels list
	opts.manageGF = &stubManageGF{
		listFn: func(_ context.Context, _ string) ([]application.GroupFunnelOutput, error) {
			return []application.GroupFunnelOutput{}, nil
		},
		setFn: func(_ context.Context, _ application.GroupFunnelInput) error { return nil },
	}
	ph := buildPageHandler(opts)
	r := newPageRouter(ph)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/team/groups/g-1/section/view-profiles", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	// Template shows message when no funnels assigned
	assert.Contains(t, w.Body.String(), "Atribua funis")
}

func TestSectionViewProfiles_WithAssignedFunnelsAndProfiles(t *testing.T) {
	opts := defaultPageHandlerOpts()
	// One funnel assigned to the group
	opts.manageGF = &stubManageGF{
		listFn: func(_ context.Context, _ string) ([]application.GroupFunnelOutput, error) {
			return []application.GroupFunnelOutput{{ID: "gf-1", GroupID: "g-1", FunnelID: "f-1"}}, nil
		},
		setFn: func(_ context.Context, _ application.GroupFunnelInput) error { return nil },
	}
	// View profile exists for that funnel with specific visible columns
	opts.manageVP = &stubManageVP{
		listFn: func(_ context.Context, _ string) ([]application.ViewProfileOutput, error) {
			return []application.ViewProfileOutput{{
				ID:             "vp-1",
				GroupID:        "g-1",
				FunnelID:       "f-1",
				VisibleColumns: []string{"col-1"},
			}}, nil
		},
		setFn: func(_ context.Context, _ application.ViewProfileInput) error { return nil },
	}
	// columnRepo returns one column
	opts.columnRepo = &stubColumnRepo{fn: func(_ context.Context, _ string) ([]funneldomain.Column, error) {
		return []funneldomain.Column{{ID: "col-1", Name: "Novo Lead"}}, nil
	}}
	ph := buildPageHandler(opts)
	r := newPageRouter(ph)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/team/groups/g-1/section/view-profiles", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "Funil A")
}

func TestSectionLoadBalance_DefaultsWhenMissing(t *testing.T) {
	ph := buildPageHandler(defaultPageHandlerOpts())
	r := newPageRouter(ph)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/team/groups/g-1/section/load-balance", nil)
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "round_robin")
}

func TestSetGroupPermissionsHTML_200(t *testing.T) {
	ph := buildPageHandler(defaultPageHandlerOpts())
	r := newPageRouter(ph)

	form := url.Values{}
	form.Add("perms", "leads:read")
	form.Add("perms", "users:update")

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/team/groups/g-1/permissions", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestSetGroupFunnelsHTML_200(t *testing.T) {
	ph := buildPageHandler(defaultPageHandlerOpts())
	r := newPageRouter(ph)

	form := url.Values{}
	form.Add("funnel_ids", "f-1")

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/team/groups/g-1/funnels", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestSetViewProfileHTML_200(t *testing.T) {
	ph := buildPageHandler(defaultPageHandlerOpts())
	r := newPageRouter(ph)

	form := url.Values{}
	form.Add("visible_columns", "col-1")

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/team/groups/g-1/view-profiles/f-1", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestSetLoadBalanceHTML_InvalidAlgorithm_400(t *testing.T) {
	opts := defaultPageHandlerOpts()
	opts.loadBalance = &stubPageLB{
		getFn: func(_ context.Context, _, _ string) (*authdomain.LoadBalanceConfig, error) {
			return nil, authdomain.ErrLoadBalanceNotFound
		},
		setFn: func(_ context.Context, _ authapp.SetLoadBalanceInput) (*authdomain.LoadBalanceConfig, error) {
			return nil, authdomain.ErrInvalidAlgorithm
		},
	}
	ph := buildPageHandler(opts)
	r := newPageRouter(ph)

	form := url.Values{}
	form.Set("algorithm", "foobar")
	form.Set("active", "false")

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/team/groups/g-1/load-balance", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestSetLoadBalanceHTML_GroupNotInTenant_404(t *testing.T) {
	opts := defaultPageHandlerOpts()
	opts.loadBalance = &stubPageLB{
		getFn: func(_ context.Context, _, _ string) (*authdomain.LoadBalanceConfig, error) {
			return nil, authdomain.ErrLoadBalanceNotFound
		},
		setFn: func(_ context.Context, _ authapp.SetLoadBalanceInput) (*authdomain.LoadBalanceConfig, error) {
			return nil, authapp.ErrGroupNotInTenant
		},
	}
	ph := buildPageHandler(opts)
	r := newPageRouter(ph)

	form := url.Values{}
	form.Set("algorithm", "round_robin")
	form.Set("active", "true")

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/team/groups/g-999/load-balance", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

// ---- additional error-path tests for coverage ----

func TestGroupsPage_500_OnRepoError(t *testing.T) {
	opts := defaultPageHandlerOpts()
	opts.listGroups = &stubListGroups{fn: func(_ context.Context, _ string) ([]application.GroupOutput, error) {
		return nil, errors.New("db error")
	}}
	ph := buildPageHandler(opts)
	r := newPageRouter(ph)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/team/groups", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestGroupsTable_500_OnRepoError(t *testing.T) {
	opts := defaultPageHandlerOpts()
	opts.listGroups = &stubListGroups{fn: func(_ context.Context, _ string) ([]application.GroupOutput, error) {
		return nil, errors.New("db error")
	}}
	ph := buildPageHandler(opts)
	r := newPageRouter(ph)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/team/groups/table", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestSectionMembers_500_OnMembersError(t *testing.T) {
	opts := defaultPageHandlerOpts()
	opts.manageMembers = &stubManageMembers{fn: func(_ context.Context, _ string) ([]application.MemberOutput, error) {
		return nil, errors.New("db error")
	}}
	ph := buildPageHandler(opts)
	r := newPageRouter(ph)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/team/groups/g-1/section/members", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestSectionMembers_500_OnUsersError(t *testing.T) {
	opts := defaultPageHandlerOpts()
	opts.usersList = &stubUsersList{fn: func(_ context.Context, _ string) ([]authapp.UserOutput, error) {
		return nil, errors.New("db error")
	}}
	ph := buildPageHandler(opts)
	r := newPageRouter(ph)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/team/groups/g-1/section/members", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestSectionPermissions_500_OnRepoError(t *testing.T) {
	opts := defaultPageHandlerOpts()
	opts.managePerms = &stubManagePerms{
		getFn: func(_ context.Context, _ string) ([]application.PermissionOutput, error) {
			return nil, errors.New("db error")
		},
		setFn: func(_ context.Context, _, _ string, _ []application.PermissionInput) error { return nil },
	}
	ph := buildPageHandler(opts)
	r := newPageRouter(ph)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/team/groups/g-1/section/permissions", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestSectionFunnels_500_OnListFunnelsError(t *testing.T) {
	opts := defaultPageHandlerOpts()
	opts.listFunnels = &stubListFunnels{fn: func(_ context.Context, _ string) ([]funnelapp.FunnelItem, error) {
		return nil, errors.New("db error")
	}}
	ph := buildPageHandler(opts)
	r := newPageRouter(ph)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/team/groups/g-1/section/funnels", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestSectionFunnels_500_OnGroupFunnelsError(t *testing.T) {
	opts := defaultPageHandlerOpts()
	opts.manageGF = &stubManageGF{
		listFn: func(_ context.Context, _ string) ([]application.GroupFunnelOutput, error) {
			return nil, errors.New("db error")
		},
		setFn: func(_ context.Context, _ application.GroupFunnelInput) error { return nil },
	}
	ph := buildPageHandler(opts)
	r := newPageRouter(ph)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/team/groups/g-1/section/funnels", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestSectionViewProfiles_500_OnGroupFunnelsError(t *testing.T) {
	opts := defaultPageHandlerOpts()
	opts.manageGF = &stubManageGF{
		listFn: func(_ context.Context, _ string) ([]application.GroupFunnelOutput, error) {
			return nil, errors.New("db error")
		},
		setFn: func(_ context.Context, _ application.GroupFunnelInput) error { return nil },
	}
	ph := buildPageHandler(opts)
	r := newPageRouter(ph)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/team/groups/g-1/section/view-profiles", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestSectionViewProfiles_500_OnViewProfilesError(t *testing.T) {
	opts := defaultPageHandlerOpts()
	opts.manageVP = &stubManageVP{
		listFn: func(_ context.Context, _ string) ([]application.ViewProfileOutput, error) {
			return nil, errors.New("db error")
		},
		setFn: func(_ context.Context, _ application.ViewProfileInput) error { return nil },
	}
	ph := buildPageHandler(opts)
	r := newPageRouter(ph)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/team/groups/g-1/section/view-profiles", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestSectionViewProfiles_500_OnListFunnelsError(t *testing.T) {
	opts := defaultPageHandlerOpts()
	// Assigned funnels returns a real entry so we get past the first two calls
	opts.manageGF = &stubManageGF{
		listFn: func(_ context.Context, _ string) ([]application.GroupFunnelOutput, error) {
			return []application.GroupFunnelOutput{{ID: "gf-1", GroupID: "g-1", FunnelID: "f-1"}}, nil
		},
		setFn: func(_ context.Context, _ application.GroupFunnelInput) error { return nil },
	}
	opts.listFunnels = &stubListFunnels{fn: func(_ context.Context, _ string) ([]funnelapp.FunnelItem, error) {
		return nil, errors.New("db error")
	}}
	ph := buildPageHandler(opts)
	r := newPageRouter(ph)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/team/groups/g-1/section/view-profiles", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestSectionLoadBalance_500_OnRepoError(t *testing.T) {
	opts := defaultPageHandlerOpts()
	opts.loadBalance = &stubPageLB{
		getFn: func(_ context.Context, _, _ string) (*authdomain.LoadBalanceConfig, error) {
			return nil, errors.New("unexpected db error")
		},
		setFn: func(_ context.Context, _ authapp.SetLoadBalanceInput) (*authdomain.LoadBalanceConfig, error) {
			return nil, nil
		},
	}
	ph := buildPageHandler(opts)
	r := newPageRouter(ph)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/team/groups/g-1/section/load-balance", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestSetGroupPermissionsHTML_422_OnSetError(t *testing.T) {
	opts := defaultPageHandlerOpts()
	opts.managePerms = &stubManagePerms{
		getFn: func(_ context.Context, _ string) ([]application.PermissionOutput, error) {
			return nil, nil
		},
		setFn: func(_ context.Context, _, _ string, _ []application.PermissionInput) error {
			return errors.New("validation error")
		},
	}
	ph := buildPageHandler(opts)
	r := newPageRouter(ph)

	form := url.Values{}
	form.Add("perms", "leads:read")

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/team/groups/g-1/permissions", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
}

func TestSetGroupFunnelsHTML_422_OnSetError(t *testing.T) {
	opts := defaultPageHandlerOpts()
	opts.manageGF = &stubManageGF{
		listFn: func(_ context.Context, _ string) ([]application.GroupFunnelOutput, error) {
			return nil, nil
		},
		setFn: func(_ context.Context, _ application.GroupFunnelInput) error {
			return errors.New("set error")
		},
	}
	ph := buildPageHandler(opts)
	r := newPageRouter(ph)

	form := url.Values{}
	form.Add("funnel_ids", "f-1")

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/team/groups/g-1/funnels", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
}

func TestSetViewProfileHTML_422_OnSetError(t *testing.T) {
	opts := defaultPageHandlerOpts()
	opts.manageVP = &stubManageVP{
		listFn: func(_ context.Context, _ string) ([]application.ViewProfileOutput, error) {
			return nil, nil
		},
		setFn: func(_ context.Context, _ application.ViewProfileInput) error {
			return errors.New("set error")
		},
	}
	ph := buildPageHandler(opts)
	r := newPageRouter(ph)

	form := url.Values{}
	form.Add("visible_columns", "col-1")

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/team/groups/g-1/view-profiles/f-1", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
}

func TestSetLoadBalanceHTML_500_OnUnexpectedError(t *testing.T) {
	opts := defaultPageHandlerOpts()
	opts.loadBalance = &stubPageLB{
		getFn: func(_ context.Context, _, _ string) (*authdomain.LoadBalanceConfig, error) {
			return nil, authdomain.ErrLoadBalanceNotFound
		},
		setFn: func(_ context.Context, _ authapp.SetLoadBalanceInput) (*authdomain.LoadBalanceConfig, error) {
			return nil, errors.New("unexpected db error")
		},
	}
	ph := buildPageHandler(opts)
	r := newPageRouter(ph)

	form := url.Values{}
	form.Set("algorithm", "round_robin")
	form.Set("active", "false")

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/team/groups/g-1/load-balance", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}
