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
func (m *owaspProductRepo) FindByTenantID(_ context.Context, tenantID string, activeOnly bool) ([]domain.Product, error) {
	return nil, nil
}
func (m *owaspProductRepo) FindActiveByTenantID(_ context.Context, tenantID string) ([]domain.Product, error) {
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

	createProductUC := application.NewCreateProductUseCase(productRepo)
	updateProductUC := application.NewUpdateProductUseCase(productRepo)
	listProductsUC := application.NewListProductsUseCase(productRepo, fpRepo)
	toggleProductUC := application.NewToggleProductUseCase(productRepo)
	manageFPUC := application.NewManageFunnelProductsUseCase(fpRepo)

	testLog, _ := zap.NewDevelopment()
	handler := NewHandler(
		createProductUC,
		updateProductUC,
		listProductsUC,
		toggleProductUC,
		manageFPUC,
		productRepo,
		fpRepo,
		testLog,
	)

	router := gin.New()

	tmpl := template.New("")
	for _, name := range []string{
		"product/product_list.html",
		"product/product_form.html",
	} {
		template.Must(tmpl.New(name).Parse("ok"))
	}
	router.SetHTMLTemplate(tmpl)

	jwtProvider := authinfra.NewJWTProvider("test-secret-owasp-product", 24*time.Hour)
	authMw := middleware.Auth(jwtProvider)
	tenantMw := middleware.RequireTenant()
	handler.RegisterRoutes(router, authMw, tenantMw)

	return &owaspProductEnv{
		router:      router,
		provider:    jwtProvider,
		productRepo: productRepo,
	}
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
		{http.MethodGet, "/tenant/products"},
		{http.MethodPost, "/tenant/products"},
		{http.MethodPut, "/tenant/products/some-id"},
		{http.MethodPost, "/tenant/products/some-id/toggle"},
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
		{http.MethodPost, "/tenant/products"},
		{http.MethodPut, "/tenant/products/some-id"},
		{http.MethodPost, "/tenant/products/some-id/toggle"},
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

// TestOWASP_Product_A01_TenantIsolation_UpdateDenied verifies that a user from
// tenant-B cannot update a product belonging to tenant-A.
func TestOWASP_Product_A01_TenantIsolation_UpdateDenied(t *testing.T) {
	env := setupOwaspProductEnv()

	// Create a product for tenant-A
	product, err := domain.NewProduct(uuid.New().String(), "tenant-a", "Product A", "desc", []string{"kw1"})
	require.NoError(t, err)
	require.NoError(t, env.productRepo.Create(context.Background(), product))

	// Attempt to update it as tenant-B
	tokenB := env.tenantToken(t, "tenant-b")
	body := strings.NewReader("name=Hacked&description=hacked&keywords=hacked")
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPut, "/tenant/products/"+product.ID, body)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(owaspProductCookie(tokenB))
	env.router.ServeHTTP(w, req)

	// The update use case enforces tenant isolation — must not succeed
	assert.NotEqual(t, http.StatusOK, w.Code)

	// Confirm the product was NOT modified
	stored := env.productRepo.products[product.ID]
	assert.Equal(t, "Product A", stored.Name)
}

// TestOWASP_Product_A01_TenantIsolation_ToggleDenied verifies that a user from
// tenant-B cannot toggle a product belonging to tenant-A.
func TestOWASP_Product_A01_TenantIsolation_ToggleDenied(t *testing.T) {
	env := setupOwaspProductEnv()

	// Create an active product for tenant-A
	product, err := domain.NewProduct(uuid.New().String(), "tenant-a", "Product A", "desc", []string{})
	require.NoError(t, err)
	require.NoError(t, env.productRepo.Create(context.Background(), product))
	originalActive := product.Active

	// Attempt to toggle it as tenant-B
	tokenB := env.tenantToken(t, "tenant-b")
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/tenant/products/"+product.ID+"/toggle", nil)
	req.AddCookie(owaspProductCookie(tokenB))
	env.router.ServeHTTP(w, req)

	// The active state must remain unchanged
	stored := env.productRepo.products[product.ID]
	assert.Equal(t, originalActive, stored.Active, "product active state must not change when toggled by wrong tenant")
}

// TestOWASP_Product_A03_SQLInjection_CreateProduct verifies that SQL injection
// payloads in product fields are handled safely (the ORM parameterises queries).
func TestOWASP_Product_A03_SQLInjection_CreateProduct(t *testing.T) {
	env := setupOwaspProductEnv()
	token := env.tenantToken(t, "tenant-a")

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
			req := httptest.NewRequest(http.MethodPost, "/tenant/products", body)
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			req.AddCookie(owaspProductCookie(token))
			env.router.ServeHTTP(w, req)

			// Must not panic or return 5xx — any 2xx/3xx/4xx is acceptable
			assert.Less(t, w.Code, http.StatusInternalServerError,
				"server must not return 5xx for injection payload: %q", payload)
		})
	}
}

// TestOWASP_Product_A03_SQLInjection_UpdateProduct verifies that injection
// payloads in the update form are handled safely.
func TestOWASP_Product_A03_SQLInjection_UpdateProduct(t *testing.T) {
	env := setupOwaspProductEnv()
	token := env.tenantToken(t, "tenant-a")

	// Create a legitimate product first
	product, err := domain.NewProduct(uuid.New().String(), "tenant-a", "Clean Product", "desc", []string{})
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
			req := httptest.NewRequest(http.MethodPut, "/tenant/products/"+product.ID, body)
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			req.AddCookie(owaspProductCookie(token))
			env.router.ServeHTTP(w, req)

			assert.Less(t, w.Code, http.StatusInternalServerError,
				"server must not return 5xx for injection payload: %q", payload)
		})
	}
}
