package http

import (
	"context"
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

// --- Minimal mocks ---

type owaspProductRepo struct{ products map[string]*domain.Product }

func (m *owaspProductRepo) Create(_ context.Context, p *domain.Product) error {
	m.products[p.ID] = p
	return nil
}
func (m *owaspProductRepo) FindByID(_ context.Context, id string) (*domain.Product, error) {
	if p, ok := m.products[id]; ok {
		return p, nil
	}
	return nil, domain.ErrProductNotFound
}
func (m *owaspProductRepo) Update(_ context.Context, p *domain.Product) error {
	if _, ok := m.products[p.ID]; !ok {
		return domain.ErrProductNotFound
	}
	m.products[p.ID] = p
	return nil
}
func (m *owaspProductRepo) Delete(_ context.Context, id string) error {
	if _, ok := m.products[id]; !ok {
		return domain.ErrProductNotFound
	}
	delete(m.products, id)
	return nil
}
func (m *owaspProductRepo) FindAll(_ context.Context, activeOnly bool) ([]domain.Product, error) {
	return nil, nil
}
func (m *owaspProductRepo) FindActiveByIDs(_ context.Context, ids []string) ([]domain.Product, error) {
	return nil, nil
}

type owaspFPRepo struct{}

func (m *owaspFPRepo) Create(_ context.Context, fp *domain.FunnelProduct) error { return nil }
func (m *owaspFPRepo) Delete(_ context.Context, funnelID, productID string) error { return nil }
func (m *owaspFPRepo) FindByProductID(_ context.Context, productID string) ([]domain.FunnelProduct, error) {
	return nil, nil
}
func (m *owaspFPRepo) FindByFunnelID(_ context.Context, funnelID string) ([]domain.FunnelProduct, error) {
	return nil, nil
}
func (m *owaspFPRepo) FindTopPriorityFunnel(_ context.Context, productID string) (*domain.FunnelProduct, error) {
	return nil, domain.ErrFunnelProductNotFound
}
func (m *owaspFPRepo) UpdatePriority(_ context.Context, funnelID, productID string, priority int) error {
	return nil
}

type owaspTPRepo struct{}

func (m *owaspTPRepo) Create(_ context.Context, tp *domain.TenantProduct) error { return nil }
func (m *owaspTPRepo) Delete(_ context.Context, tenantID, productID string) error { return nil }
func (m *owaspTPRepo) FindByTenantID(_ context.Context, tenantID string) ([]domain.TenantProduct, error) {
	return nil, nil
}
func (m *owaspTPRepo) FindByProductID(_ context.Context, productID string) ([]domain.TenantProduct, error) {
	return nil, nil
}
func (m *owaspTPRepo) Exists(_ context.Context, tenantID, productID string) (bool, error) {
	return false, nil
}

// --- Test environment ---

type owaspProductEnv struct {
	router      *gin.Engine
	provider    *authinfra.JWTProvider
	productRepo *owaspProductRepo
}

func setupOwaspProductEnv() *owaspProductEnv {
	gin.SetMode(gin.TestMode)

	productRepo := &owaspProductRepo{products: make(map[string]*domain.Product)}
	fpRepo := &owaspFPRepo{}
	tpRepo := &owaspTPRepo{}

	createProductUC := application.NewCreateProductUseCase(productRepo)
	updateProductUC := application.NewUpdateProductUseCase(productRepo)
	listProductsUC := application.NewListProductsUseCase(productRepo, fpRepo)
	listTenantProdUC := application.NewListTenantProductsUseCase(productRepo, tpRepo, fpRepo)
	toggleProductUC := application.NewToggleProductUseCase(productRepo)
	deleteProductUC := application.NewDeleteProductUseCase(productRepo)
	manageFPUC := application.NewManageFunnelProductsUseCase(fpRepo)
	manageTPUC := application.NewManageTenantProductsUseCase(tpRepo)

	testLog, _ := zap.NewDevelopment()
	handler := NewHandler(
		createProductUC,
		updateProductUC,
		listProductsUC,
		listTenantProdUC,
		toggleProductUC,
		deleteProductUC,
		manageFPUC,
		manageTPUC,
		productRepo,
		fpRepo,
		testLog,
	)

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

	jwtProvider := authinfra.NewJWTProvider("test-secret-owasp-product", 24*time.Hour)
	authMw := middleware.Auth(jwtProvider)
	tenantMw := middleware.RequireTenant()
	adminMw := middleware.RequireAdmin()
	handler.RegisterRoutes(router, authMw, tenantMw, adminMw)

	return &owaspProductEnv{
		router:      router,
		provider:    jwtProvider,
		productRepo: productRepo,
	}
}

func (e *owaspProductEnv) adminToken(t *testing.T) string {
	t.Helper()
	token, err := e.provider.Generate(authdomain.TokenClaims{
		UserID: uuid.New().String(), Role: authdomain.UserRoleAdmin,
	})
	require.NoError(t, err)
	return token
}

func (e *owaspProductEnv) tenantToken(t *testing.T, tenantID string) string {
	t.Helper()
	token, err := e.provider.Generate(authdomain.TokenClaims{
		UserID: uuid.New().String(), Role: authdomain.UserRoleUser, TenantID: tenantID,
	})
	require.NoError(t, err)
	return token
}

func (e *owaspProductEnv) tokenWithoutTenant(t *testing.T) string {
	t.Helper()
	token, err := e.provider.Generate(authdomain.TokenClaims{
		UserID: uuid.New().String(), Role: authdomain.UserRoleUser,
	})
	require.NoError(t, err)
	return token
}

func owaspProductCookie(token string) *http.Cookie {
	return &http.Cookie{Name: "token", Value: token}
}

// --- Tests ---

// TestOWASP_Product_A01_NoToken_Returns401 verifies that all product endpoints
// reject requests with no authentication token with 401.
func TestOWASP_Product_A01_NoToken_Returns401(t *testing.T) {
	env := setupOwaspProductEnv()

	routes := []struct{ method, path string }{
		// Admin routes
		{http.MethodGet, "/admin/products"},
		{http.MethodPost, "/admin/products"},
		{http.MethodPut, "/admin/products/some-id"},
		{http.MethodPost, "/admin/products/some-id/toggle"},
		{http.MethodDelete, "/admin/products/some-id"},
		// Tenant routes
		{http.MethodGet, "/tenant/products"},
	}

	for _, r := range routes {
		t.Run(r.method+" "+r.path, func(t *testing.T) {
			w := httptest.NewRecorder()
			req := httptest.NewRequest(r.method, r.path, nil)
			env.router.ServeHTTP(w, req)
			assert.Equal(t, http.StatusUnauthorized, w.Code)
		})
	}
}

// TestOWASP_Product_A01_NoTenant_Returns403 verifies that a valid JWT without a
// tenant claim is rejected with 403 on tenant-scoped product endpoints.
func TestOWASP_Product_A01_NoTenant_Returns403(t *testing.T) {
	env := setupOwaspProductEnv()
	token := env.tokenWithoutTenant(t)

	routes := []struct{ method, path string }{
		{http.MethodGet, "/tenant/products"},
	}

	for _, r := range routes {
		t.Run(r.method+" "+r.path, func(t *testing.T) {
			w := httptest.NewRecorder()
			req := httptest.NewRequest(r.method, r.path, nil)
			req.AddCookie(owaspProductCookie(token))
			env.router.ServeHTTP(w, req)
			assert.Equal(t, http.StatusForbidden, w.Code)
		})
	}
}

// TestOWASP_Product_A01_NonAdmin_CannotAccessAdminRoutes verifies that a regular
// user (non-admin) cannot access admin product CRUD endpoints.
func TestOWASP_Product_A01_NonAdmin_CannotAccessAdminRoutes(t *testing.T) {
	env := setupOwaspProductEnv()
	token := env.tenantToken(t, "tenant-a")

	routes := []struct{ method, path string }{
		{http.MethodGet, "/admin/products"},
		{http.MethodPost, "/admin/products"},
		{http.MethodPut, "/admin/products/some-id"},
		{http.MethodPost, "/admin/products/some-id/toggle"},
		{http.MethodDelete, "/admin/products/some-id"},
	}

	for _, r := range routes {
		t.Run(r.method+" "+r.path, func(t *testing.T) {
			w := httptest.NewRecorder()
			var body *strings.Reader
			if r.method == http.MethodPost || r.method == http.MethodPut {
				body = strings.NewReader("name=Test&description=desc&keywords=kw")
			}
			var req *http.Request
			if body != nil {
				req = httptest.NewRequest(r.method, r.path, body)
				req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			} else {
				req = httptest.NewRequest(r.method, r.path, nil)
			}
			req.AddCookie(owaspProductCookie(token))
			env.router.ServeHTTP(w, req)
			assert.Equal(t, http.StatusForbidden, w.Code)
		})
	}
}

// TestOWASP_Product_A03_SQLInjection_AdminCreateProduct verifies that SQL injection
// payloads in product fields are handled safely (the ORM parameterises queries).
func TestOWASP_Product_A03_SQLInjection_AdminCreateProduct(t *testing.T) {
	env := setupOwaspProductEnv()
	token := env.adminToken(t)

	injections := []string{
		"'; DROP TABLE products; --",
		`" OR "1"="1`,
		"1; SELECT * FROM users",
		`<script>alert(1)</script>`,
	}

	for _, payload := range injections {
		t.Run("payload: "+payload, func(t *testing.T) {
			body := strings.NewReader("name=" + payload + "&description=desc&keywords=kw")
			w := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodPost, "/admin/products", body)
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			req.AddCookie(owaspProductCookie(token))
			env.router.ServeHTTP(w, req)

			// Must not panic or return 5xx — any 2xx/3xx/4xx is acceptable
			assert.Less(t, w.Code, http.StatusInternalServerError,
				"server must not return 5xx for injection payload: %q", payload)
		})
	}
}

// TestOWASP_Product_A03_SQLInjection_AdminUpdateProduct verifies that injection
// payloads in the update form are handled safely.
func TestOWASP_Product_A03_SQLInjection_AdminUpdateProduct(t *testing.T) {
	env := setupOwaspProductEnv()
	token := env.adminToken(t)

	// Create a legitimate product first
	product, err := domain.NewProduct(uuid.New().String(), "Clean Product", "desc", []string{})
	require.NoError(t, err)
	require.NoError(t, env.productRepo.Create(context.Background(), product))

	injections := []string{
		"'; DROP TABLE products; --",
		`" OR "1"="1`,
		"UNION SELECT password FROM users",
	}

	for _, payload := range injections {
		t.Run("payload: "+payload, func(t *testing.T) {
			body := strings.NewReader("name=" + payload + "&description=safe&keywords=safe")
			w := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodPut, "/admin/products/"+product.ID, body)
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			req.AddCookie(owaspProductCookie(token))
			env.router.ServeHTTP(w, req)

			assert.Less(t, w.Code, http.StatusInternalServerError,
				"server must not return 5xx for injection payload: %q", payload)
		})
	}
}

// TestOWASP_Product_AdminDeleteProduct verifies that admin can delete a product.
func TestOWASP_Product_AdminDeleteProduct(t *testing.T) {
	env := setupOwaspProductEnv()
	token := env.adminToken(t)

	product, err := domain.NewProduct(uuid.New().String(), "To Delete", "desc", []string{})
	require.NoError(t, err)
	require.NoError(t, env.productRepo.Create(context.Background(), product))

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodDelete, "/admin/products/"+product.ID, nil)
	req.AddCookie(owaspProductCookie(token))
	env.router.ServeHTTP(w, req)

	// Verify product was deleted
	_, exists := env.productRepo.products[product.ID]
	assert.False(t, exists, "product should be deleted")
}
