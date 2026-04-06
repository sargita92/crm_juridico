package application

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/sasrgita/crm-juridico/internal/product/domain"
)

func setupDetectTest(t *testing.T) (*DetectProductUseCase, *mockProductRepo, *mockTenantProductRepo) {
	t.Helper()
	productRepo := newMockProductRepo()
	tpRepo := newMockTenantProductRepo()
	uc := NewDetectProductUseCase(productRepo, tpRepo)
	return uc, productRepo, tpRepo
}

func addProductToTenant(t *testing.T, productRepo *mockProductRepo, tpRepo *mockTenantProductRepo, tenantID string, product *domain.Product) {
	t.Helper()
	require.NoError(t, productRepo.Create(context.Background(), product))
	tp, err := domain.NewTenantProduct("tp-"+product.ID, tenantID, product.ID)
	require.NoError(t, err)
	require.NoError(t, tpRepo.Create(context.Background(), tp))
}

func TestDetectProduct_MatchesKeyword(t *testing.T) {
	uc, productRepo, tpRepo := setupDetectTest(t)

	product, err := domain.NewProduct("prod-1", "Consultoria Trabalhista", "", []string{"trabalhista", "CLT"})
	require.NoError(t, err)
	addProductToTenant(t, productRepo, tpRepo, "tenant-1", product)

	productID, found, err := uc.Execute(context.Background(), "tenant-1", "preciso de ajuda com processo trabalhista")

	require.NoError(t, err)
	assert.True(t, found)
	assert.Equal(t, "prod-1", productID)
}

func TestDetectProduct_NoMatch(t *testing.T) {
	uc, productRepo, tpRepo := setupDetectTest(t)

	product, err := domain.NewProduct("prod-1", "Consultoria Trabalhista", "", []string{"trabalhista", "CLT"})
	require.NoError(t, err)
	addProductToTenant(t, productRepo, tpRepo, "tenant-1", product)

	productID, found, err := uc.Execute(context.Background(), "tenant-1", "preciso de um advogado de familia")

	require.NoError(t, err)
	assert.False(t, found)
	assert.Empty(t, productID)
}

func TestDetectProduct_CaseInsensitive(t *testing.T) {
	uc, productRepo, tpRepo := setupDetectTest(t)

	product, err := domain.NewProduct("prod-1", "Consultoria", "", []string{"CLT"})
	require.NoError(t, err)
	addProductToTenant(t, productRepo, tpRepo, "tenant-1", product)

	productID, found, err := uc.Execute(context.Background(), "tenant-1", "tenho duvidas sobre clt")

	require.NoError(t, err)
	assert.True(t, found)
	assert.Equal(t, "prod-1", productID)
}

func TestDetectProduct_InactiveProductIgnored(t *testing.T) {
	uc, productRepo, tpRepo := setupDetectTest(t)

	product, err := domain.NewProduct("prod-1", "Consultoria Trabalhista", "", []string{"trabalhista"})
	require.NoError(t, err)
	product.Deactivate()
	addProductToTenant(t, productRepo, tpRepo, "tenant-1", product)

	productID, found, err := uc.Execute(context.Background(), "tenant-1", "preciso de ajuda trabalhista")

	require.NoError(t, err)
	assert.False(t, found)
	assert.Empty(t, productID)
}

func TestDetectProduct_NoTenantProducts(t *testing.T) {
	uc, _, _ := setupDetectTest(t)

	productID, found, err := uc.Execute(context.Background(), "tenant-no-products", "trabalhista")

	require.NoError(t, err)
	assert.False(t, found)
	assert.Empty(t, productID)
}
