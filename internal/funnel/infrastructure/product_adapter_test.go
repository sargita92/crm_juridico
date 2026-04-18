package infrastructure

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	productapp "github.com/sasrgita/crm-juridico/internal/product/application"
	productdomain "github.com/sasrgita/crm-juridico/internal/product/domain"
	productinfra "github.com/sasrgita/crm-juridico/internal/product/infrastructure"
)

// Integration tests for product/funnel cross-module adapters.
// Uses the shared testcontainer setup (see setupFunnelRepos in gorm_repositories_test.go).

func TestProductProviderAdapter_FindProductNameByID(t *testing.T) {
	_, db := setupFunnelRepos(t)
	productRepo := productinfra.NewGormProductRepository(db)
	ctx := context.Background()

	p, err := productdomain.NewProduct(uuid.New().String(), "Consultoria Tributaria", "", []string{"tribut"})
	require.NoError(t, err)
	require.NoError(t, productRepo.Create(ctx, p))

	adapter := NewProductProviderAdapter(productRepo)
	name, err := adapter.FindProductNameByID(ctx, p.ID)
	require.NoError(t, err)
	assert.Equal(t, "Consultoria Tributaria", name)
}

func TestProductProviderAdapter_FindProductNameByID_NotFound(t *testing.T) {
	_, db := setupFunnelRepos(t)
	productRepo := productinfra.NewGormProductRepository(db)

	adapter := NewProductProviderAdapter(productRepo)
	_, err := adapter.FindProductNameByID(context.Background(), uuid.New().String())
	assert.ErrorIs(t, err, productdomain.ErrProductNotFound)
}

func TestProductDetectorAdapter_DetectFromMessage(t *testing.T) {
	_, db := setupFunnelRepos(t)
	tenantID := createTenant(t, db)
	productRepo := productinfra.NewGormProductRepository(db)
	tpRepo := productinfra.NewGormTenantProductRepository(db)
	ctx := context.Background()

	p, err := productdomain.NewProduct(uuid.New().String(), "Divorcio", "", []string{"divorcio", "separacao"})
	require.NoError(t, err)
	require.NoError(t, productRepo.Create(ctx, p))

	tp, err := productdomain.NewTenantProduct(uuid.New().String(), tenantID, p.ID)
	require.NoError(t, err)
	require.NoError(t, tpRepo.Create(ctx, tp))

	uc := productapp.NewDetectProductUseCase(productRepo, tpRepo)
	adapter := NewProductDetectorAdapter(uc)

	productID, found, err := adapter.DetectFromMessage(ctx, tenantID, "Preciso de ajuda com o divorcio da minha irma")
	require.NoError(t, err)
	assert.True(t, found, "deve detectar produto por keyword")
	assert.Equal(t, p.ID, productID)
}

func TestProductDetectorAdapter_DetectFromMessage_NoMatch(t *testing.T) {
	_, db := setupFunnelRepos(t)
	tenantID := createTenant(t, db)
	productRepo := productinfra.NewGormProductRepository(db)
	tpRepo := productinfra.NewGormTenantProductRepository(db)

	uc := productapp.NewDetectProductUseCase(productRepo, tpRepo)
	adapter := NewProductDetectorAdapter(uc)

	productID, found, err := adapter.DetectFromMessage(context.Background(), tenantID, "mensagem sem palavras-chave")
	require.NoError(t, err)
	assert.False(t, found)
	assert.Equal(t, "", productID)
}

func TestFunnelProductRouterAdapter_FindTopPriorityFunnelID(t *testing.T) {
	repos, db := setupFunnelRepos(t)
	tenantID := createTenant(t, db)
	ctx := context.Background()

	productRepo := productinfra.NewGormProductRepository(db)
	fpRepo := productinfra.NewGormFunnelProductRepository(db)

	funnel := mustCreateFunnel(t, repos.funnels, tenantID, "Funil Divorcio")
	p, err := productdomain.NewProduct(uuid.New().String(), "Divorcio", "", nil)
	require.NoError(t, err)
	require.NoError(t, productRepo.Create(ctx, p))

	fp, err := productdomain.NewFunnelProduct(uuid.New().String(), funnel.ID, p.ID, 10)
	require.NoError(t, err)
	require.NoError(t, fpRepo.Create(ctx, fp))

	adapter := NewFunnelProductRouterAdapter(fpRepo)
	funnelID, err := adapter.FindTopPriorityFunnelID(ctx, tenantID, p.ID)
	require.NoError(t, err)
	assert.Equal(t, funnel.ID, funnelID)
}

func TestProductListerAdapter_ListActiveProducts(t *testing.T) {
	_, db := setupFunnelRepos(t)
	tenantID := createTenant(t, db)
	productRepo := productinfra.NewGormProductRepository(db)
	tpRepo := productinfra.NewGormTenantProductRepository(db)
	ctx := context.Background()

	// Two active products for the tenant
	p1, err := productdomain.NewProduct(uuid.New().String(), "Consultoria A", "", nil)
	require.NoError(t, err)
	require.NoError(t, productRepo.Create(ctx, p1))
	tp1, _ := productdomain.NewTenantProduct(uuid.New().String(), tenantID, p1.ID)
	require.NoError(t, tpRepo.Create(ctx, tp1))

	p2, err := productdomain.NewProduct(uuid.New().String(), "Consultoria B", "", nil)
	require.NoError(t, err)
	require.NoError(t, productRepo.Create(ctx, p2))
	tp2, _ := productdomain.NewTenantProduct(uuid.New().String(), tenantID, p2.ID)
	require.NoError(t, tpRepo.Create(ctx, tp2))

	adapter := NewProductListerAdapter(productRepo, tpRepo)
	products, err := adapter.ListActiveProducts(ctx, tenantID)
	require.NoError(t, err)
	assert.Len(t, products, 2)
}

func TestProductListerAdapter_ListActiveProducts_Empty(t *testing.T) {
	_, db := setupFunnelRepos(t)
	tenantID := createTenant(t, db)
	productRepo := productinfra.NewGormProductRepository(db)
	tpRepo := productinfra.NewGormTenantProductRepository(db)

	adapter := NewProductListerAdapter(productRepo, tpRepo)
	products, err := adapter.ListActiveProducts(context.Background(), tenantID)
	require.NoError(t, err)
	assert.Empty(t, products)
}
