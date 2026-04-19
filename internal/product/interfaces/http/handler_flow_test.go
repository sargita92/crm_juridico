package http

import (
	"context"
	"errors"
	"html/template"
	"net/http"
	"net/http/httptest"
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
	"github.com/sasrgita/crm-juridico/internal/product/application"
	"github.com/sasrgita/crm-juridico/internal/product/domain"
	"github.com/sasrgita/crm-juridico/internal/shared/middleware"
)

// --- Flow mocks: richer than owasp mocks so happy-path handlers can render data. ---

type flowProductRepo struct {
	products  map[string]*domain.Product
	findAllErr error
}

func newFlowProductRepo() *flowProductRepo {
	return &flowProductRepo{products: make(map[string]*domain.Product)}
}

func (m *flowProductRepo) Create(_ context.Context, p *domain.Product) error {
	m.products[p.ID] = p
	return nil
}
func (m *flowProductRepo) FindByID(_ context.Context, id string) (*domain.Product, error) {
	if p, ok := m.products[id]; ok {
		return p, nil
	}
	return nil, domain.ErrProductNotFound
}
func (m *flowProductRepo) Update(_ context.Context, p *domain.Product) error {
	if _, ok := m.products[p.ID]; !ok {
		return domain.ErrProductNotFound
	}
	m.products[p.ID] = p
	return nil
}
func (m *flowProductRepo) Delete(_ context.Context, id string) error {
	delete(m.products, id)
	return nil
}
func (m *flowProductRepo) FindAll(_ context.Context, _ bool) ([]domain.Product, error) {
	if m.findAllErr != nil {
		return nil, m.findAllErr
	}
	out := make([]domain.Product, 0, len(m.products))
	for _, p := range m.products {
		out = append(out, *p)
	}
	return out, nil
}
func (m *flowProductRepo) FindActiveByIDs(_ context.Context, ids []string) ([]domain.Product, error) {
	out := make([]domain.Product, 0, len(ids))
	for _, id := range ids {
		if p, ok := m.products[id]; ok && p.Active {
			out = append(out, *p)
		}
	}
	return out, nil
}

type flowFPRepo struct {
	byProduct map[string][]domain.FunnelProduct
	created   []domain.FunnelProduct
	deleted   []struct{ FunnelID, ProductID string }
	updated   []domain.FunnelProduct
}

func newFlowFPRepo() *flowFPRepo {
	return &flowFPRepo{byProduct: make(map[string][]domain.FunnelProduct)}
}

func (m *flowFPRepo) Create(_ context.Context, fp *domain.FunnelProduct) error {
	m.byProduct[fp.ProductID] = append(m.byProduct[fp.ProductID], *fp)
	m.created = append(m.created, *fp)
	return nil
}
func (m *flowFPRepo) Delete(_ context.Context, funnelID, productID string) error {
	m.deleted = append(m.deleted, struct{ FunnelID, ProductID string }{funnelID, productID})
	return nil
}
func (m *flowFPRepo) FindByProductID(_ context.Context, productID string) ([]domain.FunnelProduct, error) {
	return m.byProduct[productID], nil
}
func (m *flowFPRepo) FindByFunnelID(_ context.Context, _ string) ([]domain.FunnelProduct, error) {
	return nil, nil
}
func (m *flowFPRepo) FindTopPriorityFunnel(_ context.Context, _ string, _ string) (*domain.FunnelProduct, error) {
	return nil, domain.ErrFunnelProductNotFound
}
func (m *flowFPRepo) UpdatePriority(_ context.Context, funnelID, productID string, priority int) error {
	m.updated = append(m.updated, domain.FunnelProduct{FunnelID: funnelID, ProductID: productID, Priority: priority})
	return nil
}

type flowTPRepo struct {
	byProduct map[string][]domain.TenantProduct
	byTenant  map[string][]domain.TenantProduct
	created   []domain.TenantProduct
	deleted   []struct{ TenantID, ProductID string }
}

func newFlowTPRepo() *flowTPRepo {
	return &flowTPRepo{
		byProduct: make(map[string][]domain.TenantProduct),
		byTenant:  make(map[string][]domain.TenantProduct),
	}
}

func (m *flowTPRepo) Create(_ context.Context, tp *domain.TenantProduct) error {
	m.byProduct[tp.ProductID] = append(m.byProduct[tp.ProductID], *tp)
	m.byTenant[tp.TenantID] = append(m.byTenant[tp.TenantID], *tp)
	m.created = append(m.created, *tp)
	return nil
}
func (m *flowTPRepo) Delete(_ context.Context, tenantID, productID string) error {
	m.deleted = append(m.deleted, struct{ TenantID, ProductID string }{tenantID, productID})
	return nil
}
func (m *flowTPRepo) FindByTenantID(_ context.Context, tenantID string) ([]domain.TenantProduct, error) {
	return m.byTenant[tenantID], nil
}
func (m *flowTPRepo) FindByProductID(_ context.Context, productID string) ([]domain.TenantProduct, error) {
	return m.byProduct[productID], nil
}
func (m *flowTPRepo) Exists(_ context.Context, _, _ string) (bool, error) { return false, nil }

type flowFunnelLister struct {
	funnels []FunnelInfo
	err     error
}

func (l *flowFunnelLister) ListFunnels(_ *gin.Context, _ string) ([]FunnelInfo, error) {
	return l.funnels, l.err
}

type flowTenantLister struct {
	tenants []TenantInfo
	err     error
}

func (l *flowTenantLister) ListTenants(_ *gin.Context) ([]TenantInfo, error) {
	return l.tenants, l.err
}

// --- Environment ---

type flowEnv struct {
	router   *gin.Engine
	provider *authinfra.JWTProvider
	handler  *Handler
	products *flowProductRepo
	fp       *flowFPRepo
	tp       *flowTPRepo
	funnels  *flowFunnelLister
	tenants  *flowTenantLister
}

func setupFlowEnv() *flowEnv {
	gin.SetMode(gin.TestMode)

	products := newFlowProductRepo()
	fp := newFlowFPRepo()
	tp := newFlowTPRepo()

	createProductUC := application.NewCreateProductUseCase(products)
	updateProductUC := application.NewUpdateProductUseCase(products)
	listProductsUC := application.NewListProductsUseCase(products, fp)
	listTenantProdUC := application.NewListTenantProductsUseCase(products, tp, fp)
	toggleProductUC := application.NewToggleProductUseCase(products)
	deleteProductUC := application.NewDeleteProductUseCase(products)
	manageFPUC := application.NewManageFunnelProductsUseCase(fp)
	manageTPUC := application.NewManageTenantProductsUseCase(tp)

	testLog := zap.NewNop()
	handler := NewHandler(
		createProductUC, updateProductUC, listProductsUC, listTenantProdUC,
		toggleProductUC, deleteProductUC, manageFPUC, manageTPUC,
		products, fp, testLog,
	)

	funnels := &flowFunnelLister{funnels: []FunnelInfo{{ID: "f-1", Name: "Funil Principal"}}}
	tenants := &flowTenantLister{tenants: []TenantInfo{{ID: "t-1", Name: "Tenant A"}}}
	handler.SetFunnelLister(funnels)
	handler.SetTenantLister(tenants)

	router := gin.New()
	tmpl := template.New("")
	for _, name := range []string{
		"product/product_list.html",
		"product/admin_product_list.html",
		"product/admin_product_form.html",
	} {
		template.Must(tmpl.New(name).Parse("ok"))
	}
	router.SetHTMLTemplate(tmpl)

	jwtProvider := authinfra.NewJWTProvider("flow-secret", 24*time.Hour)
	authMw := middleware.Auth(jwtProvider)
	tenantMw := middleware.RequireTenant()
	adminMw := middleware.RequireAdmin()
	handler.RegisterRoutes(router, authMw, tenantMw, adminMw)

	return &flowEnv{
		router: router, provider: jwtProvider, handler: handler,
		products: products, fp: fp, tp: tp,
		funnels: funnels, tenants: tenants,
	}
}

func (e *flowEnv) admin(t *testing.T) string {
	t.Helper()
	token, err := e.provider.Generate(authdomain.TokenClaims{
		UserID: uuid.New().String(), Role: authdomain.UserRoleAdmin,
	})
	require.NoError(t, err)
	return token
}

func (e *flowEnv) tenant(t *testing.T, tenantID string) string {
	t.Helper()
	token, err := e.provider.Generate(authdomain.TokenClaims{
		UserID: uuid.New().String(), Role: authdomain.UserRoleUser, TenantID: tenantID,
	})
	require.NoError(t, err)
	return token
}

func cookie(token string) *http.Cookie {
	return &http.Cookie{Name: "token", Value: token}
}

func (e *flowEnv) do(t *testing.T, method, path, token, body string) *httptest.ResponseRecorder {
	t.Helper()
	var req *http.Request
	if body != "" {
		req = httptest.NewRequest(method, path, strings.NewReader(body))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	} else {
		req = httptest.NewRequest(method, path, nil)
	}
	if token != "" {
		req.AddCookie(cookie(token))
	}
	w := httptest.NewRecorder()
	e.router.ServeHTTP(w, req)
	return w
}

// --- Admin handlers ---

func TestFlow_RenderAdminProductList_Success(t *testing.T) {
	env := setupFlowEnv()
	p, err := domain.NewProduct(uuid.New().String(), "Produto A", "desc", []string{"kw"})
	require.NoError(t, err)
	require.NoError(t, env.products.Create(context.Background(), p))

	w := env.do(t, http.MethodGet, "/admin/products", env.admin(t), "")
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestFlow_RenderAdminProductList_RepoError(t *testing.T) {
	env := setupFlowEnv()
	env.products.findAllErr = errors.New("boom")

	w := env.do(t, http.MethodGet, "/admin/products", env.admin(t), "")
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestFlow_RenderAdminProductForm_New(t *testing.T) {
	env := setupFlowEnv()
	w := env.do(t, http.MethodGet, "/admin/products/new", env.admin(t), "")
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestFlow_RenderAdminProductForm_Edit_WithLinks(t *testing.T) {
	env := setupFlowEnv()
	product, err := domain.NewProduct(uuid.New().String(), "Prod", "d", []string{"a", "b"})
	require.NoError(t, err)
	require.NoError(t, env.products.Create(context.Background(), product))

	// seed funnel-product link
	fp, err := domain.NewFunnelProduct(uuid.New().String(), "f-1", product.ID, 2)
	require.NoError(t, err)
	require.NoError(t, env.fp.Create(context.Background(), fp))

	// seed tenant-product association
	tp, err := domain.NewTenantProduct(uuid.New().String(), "t-1", product.ID)
	require.NoError(t, err)
	require.NoError(t, env.tp.Create(context.Background(), tp))

	w := env.do(t, http.MethodGet, "/admin/products/"+product.ID+"/edit", env.admin(t), "")
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestFlow_RenderAdminProductForm_Edit_ListerErrors(t *testing.T) {
	env := setupFlowEnv()
	product, err := domain.NewProduct(uuid.New().String(), "Prod", "d", nil)
	require.NoError(t, err)
	require.NoError(t, env.products.Create(context.Background(), product))

	fp, err := domain.NewFunnelProduct(uuid.New().String(), "f-xx", product.ID, 1)
	require.NoError(t, err)
	require.NoError(t, env.fp.Create(context.Background(), fp))

	tp, err := domain.NewTenantProduct(uuid.New().String(), "t-xx", product.ID)
	require.NoError(t, err)
	require.NoError(t, env.tp.Create(context.Background(), tp))

	// listers fail — handler must still render form (fallback to ID as name).
	env.funnels.err = errors.New("funnel lister down")
	env.tenants.err = errors.New("tenant lister down")

	w := env.do(t, http.MethodGet, "/admin/products/"+product.ID+"/edit", env.admin(t), "")
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestFlow_RenderAdminProductForm_NotFound(t *testing.T) {
	env := setupFlowEnv()
	w := env.do(t, http.MethodGet, "/admin/products/missing/edit", env.admin(t), "")
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestFlow_HandleAdminToggle_Success(t *testing.T) {
	env := setupFlowEnv()
	p, err := domain.NewProduct(uuid.New().String(), "P", "d", nil)
	require.NoError(t, err)
	require.NoError(t, env.products.Create(context.Background(), p))

	w := env.do(t, http.MethodPost, "/admin/products/"+p.ID+"/toggle", env.admin(t), "")
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "/admin/products", w.Header().Get("HX-Redirect"))
	assert.False(t, env.products.products[p.ID].Active)
}

func TestFlow_HandleAdminLinkFunnel_Success(t *testing.T) {
	env := setupFlowEnv()
	p, err := domain.NewProduct(uuid.New().String(), "P", "d", nil)
	require.NoError(t, err)
	require.NoError(t, env.products.Create(context.Background(), p))

	w := env.do(t, http.MethodPost, "/admin/products/"+p.ID+"/funnels", env.admin(t),
		"funnel_id=f-99&priority=3")
	assert.Equal(t, http.StatusOK, w.Code)
	require.Len(t, env.fp.created, 1)
	assert.Equal(t, 3, env.fp.created[0].Priority)
}

func TestFlow_HandleAdminLinkFunnel_InvalidPriorityFallsBackToOne(t *testing.T) {
	env := setupFlowEnv()
	p, err := domain.NewProduct(uuid.New().String(), "P", "d", nil)
	require.NoError(t, err)
	require.NoError(t, env.products.Create(context.Background(), p))

	w := env.do(t, http.MethodPost, "/admin/products/"+p.ID+"/funnels", env.admin(t),
		"funnel_id=f-1&priority=not-a-number")
	assert.Equal(t, http.StatusOK, w.Code)
	require.Len(t, env.fp.created, 1)
	assert.Equal(t, 1, env.fp.created[0].Priority)
}

func TestFlow_HandleAdminUnlinkFunnel_Success(t *testing.T) {
	env := setupFlowEnv()

	w := env.do(t, http.MethodDelete, "/admin/products/p1/funnels/f1", env.admin(t), "")
	assert.Equal(t, http.StatusOK, w.Code)
	require.Len(t, env.fp.deleted, 1)
	assert.Equal(t, "f1", env.fp.deleted[0].FunnelID)
	assert.Equal(t, "p1", env.fp.deleted[0].ProductID)
}

func TestFlow_HandleAdminUpdatePriority_Success(t *testing.T) {
	env := setupFlowEnv()

	w := env.do(t, http.MethodPut, "/admin/products/p1/funnels/f1/priority", env.admin(t), "priority=7")
	assert.Equal(t, http.StatusOK, w.Code)
	require.Len(t, env.fp.updated, 1)
	assert.Equal(t, 7, env.fp.updated[0].Priority)
}

func TestFlow_HandleAdminUpdatePriority_InvalidFallsBackToOne(t *testing.T) {
	env := setupFlowEnv()

	w := env.do(t, http.MethodPut, "/admin/products/p1/funnels/f1/priority", env.admin(t), "priority=0")
	assert.Equal(t, http.StatusOK, w.Code)
	require.Len(t, env.fp.updated, 1)
	assert.Equal(t, 1, env.fp.updated[0].Priority)
}

func TestFlow_HandleAdminAssociateTenant_Success(t *testing.T) {
	env := setupFlowEnv()

	w := env.do(t, http.MethodPost, "/admin/products/p1/tenants", env.admin(t), "tenant_id=t-1")
	assert.Equal(t, http.StatusOK, w.Code)
	require.Len(t, env.tp.created, 1)
	assert.Equal(t, "t-1", env.tp.created[0].TenantID)
}

func TestFlow_HandleAdminDisassociateTenant_Success(t *testing.T) {
	env := setupFlowEnv()

	w := env.do(t, http.MethodDelete, "/admin/products/p1/tenants/t-1", env.admin(t), "")
	assert.Equal(t, http.StatusOK, w.Code)
	require.Len(t, env.tp.deleted, 1)
	assert.Equal(t, "t-1", env.tp.deleted[0].TenantID)
}

// --- Tenant handlers ---

func TestFlow_RenderTenantProductList_Success(t *testing.T) {
	env := setupFlowEnv()
	p, err := domain.NewProduct(uuid.New().String(), "Prod", "d", []string{"kw"})
	require.NoError(t, err)
	require.NoError(t, env.products.Create(context.Background(), p))

	tp, err := domain.NewTenantProduct(uuid.New().String(), "t-a", p.ID)
	require.NoError(t, err)
	require.NoError(t, env.tp.Create(context.Background(), tp))

	fp, err := domain.NewFunnelProduct(uuid.New().String(), "f-1", p.ID, 1)
	require.NoError(t, err)
	require.NoError(t, env.fp.Create(context.Background(), fp))

	w := env.do(t, http.MethodGet, "/tenant/products", env.tenant(t, "t-a"), "")
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestFlow_HandleTenantLinkFunnel_Success(t *testing.T) {
	env := setupFlowEnv()

	w := env.do(t, http.MethodPost, "/tenant/products/p1/funnels", env.tenant(t, "t-a"),
		"funnel_id=f-1&priority=2")
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "/tenant/products", w.Header().Get("HX-Redirect"))
	require.Len(t, env.fp.created, 1)
	assert.Equal(t, 2, env.fp.created[0].Priority)
}

func TestFlow_HandleTenantLinkFunnel_InvalidPriorityFallsBackToOne(t *testing.T) {
	env := setupFlowEnv()

	w := env.do(t, http.MethodPost, "/tenant/products/p1/funnels", env.tenant(t, "t-a"),
		"funnel_id=f-1&priority=abc")
	assert.Equal(t, http.StatusOK, w.Code)
	require.Len(t, env.fp.created, 1)
	assert.Equal(t, 1, env.fp.created[0].Priority)
}

func TestFlow_HandleTenantUnlinkFunnel_Success(t *testing.T) {
	env := setupFlowEnv()

	w := env.do(t, http.MethodDelete, "/tenant/products/p1/funnels/f1", env.tenant(t, "t-a"), "")
	assert.Equal(t, http.StatusOK, w.Code)
	require.Len(t, env.fp.deleted, 1)
}

func TestFlow_HandleTenantUpdatePriority_Success(t *testing.T) {
	env := setupFlowEnv()

	w := env.do(t, http.MethodPut, "/tenant/products/p1/funnels/f1/priority",
		env.tenant(t, "t-a"), "priority=5")
	assert.Equal(t, http.StatusOK, w.Code)
	require.Len(t, env.fp.updated, 1)
	assert.Equal(t, 5, env.fp.updated[0].Priority)
}

func TestFlow_HandleTenantUpdatePriority_InvalidFallsBackToOne(t *testing.T) {
	env := setupFlowEnv()

	w := env.do(t, http.MethodPut, "/tenant/products/p1/funnels/f1/priority",
		env.tenant(t, "t-a"), "priority=")
	assert.Equal(t, http.StatusOK, w.Code)
	require.Len(t, env.fp.updated, 1)
	assert.Equal(t, 1, env.fp.updated[0].Priority)
}

// --- Parse helpers ---

func TestFlow_LegacyParseKeywords(t *testing.T) {
	got := legacyParseKeywords(", one ,two,  ,three, ")
	assert.Equal(t, []string{"one", "two", "three"}, got)
}

func TestFlow_NormalizeQuotedCSV_EscapedQuotes(t *testing.T) {
	// Ensures the escaped-quote branch (""") inside a quoted field is preserved.
	got := parseKeywords(`"a ""b"" c", d`)
	assert.Equal(t, []string{`a "b" c`, "d"}, got)
}
