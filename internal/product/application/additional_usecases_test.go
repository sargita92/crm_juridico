package application

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/sasrgita/crm-juridico/internal/product/domain"
)

// --- DeleteProductUseCase ---

func TestDeleteProduct_Success(t *testing.T) {
	repo := newMockProductRepo()
	p, _ := domain.NewProduct("p1", "Consultoria", "desc", nil)
	_ = repo.Create(context.Background(), p)

	uc := NewDeleteProductUseCase(repo)
	err := uc.Execute(context.Background(), DeleteProductInput{ProductID: "p1"})
	require.NoError(t, err)
	assert.NotContains(t, repo.products, "p1")
}

func TestDeleteProduct_NotFound(t *testing.T) {
	repo := newMockProductRepo()
	uc := NewDeleteProductUseCase(repo)
	err := uc.Execute(context.Background(), DeleteProductInput{ProductID: "missing"})
	assert.ErrorIs(t, err, domain.ErrProductNotFound)
}

// --- UpdateProductUseCase ---

func TestUpdateProduct_Success(t *testing.T) {
	repo := newMockProductRepo()
	p, _ := domain.NewProduct("p1", "Antigo", "desc", []string{"kw1"})
	_ = repo.Create(context.Background(), p)

	uc := NewUpdateProductUseCase(repo)
	out, err := uc.Execute(context.Background(), UpdateProductInput{
		ProductID:   "p1",
		Name:        "Consultoria Jurídica",
		Description: "Ação trabalhista e previdenciária",
		Keywords:    []string{"trabalhista", "previdência"},
	})
	require.NoError(t, err)
	assert.Equal(t, "Consultoria Jurídica", out.Name)
	assert.Equal(t, "Ação trabalhista e previdenciária", out.Description)
	assert.Equal(t, []string{"trabalhista", "previdência"}, out.Keywords)
}

func TestUpdateProduct_NotFound(t *testing.T) {
	repo := newMockProductRepo()
	uc := NewUpdateProductUseCase(repo)
	_, err := uc.Execute(context.Background(), UpdateProductInput{ProductID: "nope", Name: "x"})
	assert.ErrorIs(t, err, domain.ErrProductNotFound)
}

func TestUpdateProduct_DomainValidationError(t *testing.T) {
	repo := newMockProductRepo()
	p, _ := domain.NewProduct("p1", "ok", "", nil)
	_ = repo.Create(context.Background(), p)

	uc := NewUpdateProductUseCase(repo)
	_, err := uc.Execute(context.Background(), UpdateProductInput{ProductID: "p1", Name: ""})
	assert.ErrorIs(t, err, domain.ErrProductNameRequired)
}

// --- ToggleProductUseCase ---

func TestToggleProduct_ActiveBecomesInactive(t *testing.T) {
	repo := newMockProductRepo()
	p, _ := domain.NewProduct("p1", "Test", "", nil)
	_ = repo.Create(context.Background(), p)

	uc := NewToggleProductUseCase(repo)
	active, err := uc.Execute(context.Background(), ToggleProductInput{ProductID: "p1"})
	require.NoError(t, err)
	assert.False(t, active)
	assert.False(t, repo.products["p1"].Active)
}

func TestToggleProduct_InactiveBecomesActive(t *testing.T) {
	repo := newMockProductRepo()
	p, _ := domain.NewProduct("p1", "Test", "", nil)
	p.Deactivate()
	_ = repo.Create(context.Background(), p)

	uc := NewToggleProductUseCase(repo)
	active, err := uc.Execute(context.Background(), ToggleProductInput{ProductID: "p1"})
	require.NoError(t, err)
	assert.True(t, active)
}

func TestToggleProduct_NotFound(t *testing.T) {
	repo := newMockProductRepo()
	uc := NewToggleProductUseCase(repo)
	_, err := uc.Execute(context.Background(), ToggleProductInput{ProductID: "nope"})
	assert.ErrorIs(t, err, domain.ErrProductNotFound)
}

// --- ListProductsUseCase ---

func TestListProducts_Empty(t *testing.T) {
	uc := NewListProductsUseCase(newMockProductRepo(), newMockFunnelProductRepo())
	out, err := uc.Execute(context.Background(), false)
	require.NoError(t, err)
	assert.Empty(t, out)
}

func TestListProducts_WithFunnels(t *testing.T) {
	productRepo := newMockProductRepo()
	funnelProdRepo := newMockFunnelProductRepo()

	p, _ := domain.NewProduct("p1", "Família", "Ação de guarda e pensão", []string{"família"})
	_ = productRepo.Create(context.Background(), p)

	fp := &domain.FunnelProduct{ID: "fp1", FunnelID: "fn1", ProductID: "p1", Priority: 5}
	_ = funnelProdRepo.Create(context.Background(), fp)

	uc := NewListProductsUseCase(productRepo, funnelProdRepo)
	out, err := uc.Execute(context.Background(), false)
	require.NoError(t, err)
	require.Len(t, out, 1)
	assert.Equal(t, "Família", out[0].Name)
	require.Len(t, out[0].Funnels, 1)
	assert.Equal(t, "fn1", out[0].Funnels[0].FunnelID)
	assert.Equal(t, 5, out[0].Funnels[0].Priority)
}

func TestListProducts_ActiveOnlyFilter(t *testing.T) {
	productRepo := newMockProductRepo()
	active, _ := domain.NewProduct("p1", "A", "", nil)
	inactive, _ := domain.NewProduct("p2", "B", "", nil)
	inactive.Deactivate()
	_ = productRepo.Create(context.Background(), active)
	_ = productRepo.Create(context.Background(), inactive)

	uc := NewListProductsUseCase(productRepo, newMockFunnelProductRepo())
	out, err := uc.Execute(context.Background(), true)
	require.NoError(t, err)
	require.Len(t, out, 1)
	assert.Equal(t, "A", out[0].Name)
}

// --- ListTenantProductsUseCase ---

func TestListTenantProducts_Empty(t *testing.T) {
	uc := NewListTenantProductsUseCase(
		newMockProductRepo(),
		newMockTenantProductRepo(),
		newMockFunnelProductRepo(),
	)
	out, err := uc.Execute(context.Background(), "tenant-1")
	require.NoError(t, err)
	assert.Empty(t, out)
}

func TestListTenantProducts_WithAssociations(t *testing.T) {
	productRepo := newMockProductRepo()
	tenantRepo := newMockTenantProductRepo()
	funnelRepo := newMockFunnelProductRepo()

	p, _ := domain.NewProduct("p1", "São João Advocacia", "ação", []string{"advocacia"})
	_ = productRepo.Create(context.Background(), p)

	tp, _ := domain.NewTenantProduct("tp1", "tenant-1", "p1")
	_ = tenantRepo.Create(context.Background(), tp)

	fp := &domain.FunnelProduct{ID: "fp1", FunnelID: "fn1", ProductID: "p1", Priority: 1}
	_ = funnelRepo.Create(context.Background(), fp)

	uc := NewListTenantProductsUseCase(productRepo, tenantRepo, funnelRepo)
	out, err := uc.Execute(context.Background(), "tenant-1")
	require.NoError(t, err)
	require.Len(t, out, 1)
	assert.Equal(t, "São João Advocacia", out[0].Name)
	require.Len(t, out[0].Funnels, 1)
}

// --- ManageTenantProductsUseCase ---

func TestManageTenantProducts_Associate_Success(t *testing.T) {
	repo := newMockTenantProductRepo()
	uc := NewManageTenantProductsUseCase(repo)

	err := uc.Associate(context.Background(), AssociateTenantProductInput{
		TenantID: "tenant-1", ProductID: "p1",
	})
	require.NoError(t, err)
	exists, _ := repo.Exists(context.Background(), "tenant-1", "p1")
	assert.True(t, exists)
}

func TestManageTenantProducts_Associate_ValidationError(t *testing.T) {
	uc := NewManageTenantProductsUseCase(newMockTenantProductRepo())
	err := uc.Associate(context.Background(), AssociateTenantProductInput{
		TenantID: "", ProductID: "p1",
	})
	assert.ErrorIs(t, err, domain.ErrTenantIDRequired)
}

func TestManageTenantProducts_Associate_Duplicate(t *testing.T) {
	repo := newMockTenantProductRepo()
	uc := NewManageTenantProductsUseCase(repo)
	_ = uc.Associate(context.Background(), AssociateTenantProductInput{
		TenantID: "t1", ProductID: "p1",
	})
	err := uc.Associate(context.Background(), AssociateTenantProductInput{
		TenantID: "t1", ProductID: "p1",
	})
	assert.ErrorIs(t, err, domain.ErrTenantProductAlreadyExists)
}

func TestManageTenantProducts_Disassociate_Success(t *testing.T) {
	repo := newMockTenantProductRepo()
	uc := NewManageTenantProductsUseCase(repo)
	_ = uc.Associate(context.Background(), AssociateTenantProductInput{
		TenantID: "t1", ProductID: "p1",
	})

	err := uc.Disassociate(context.Background(), DisassociateTenantProductInput{
		TenantID: "t1", ProductID: "p1",
	})
	require.NoError(t, err)
	exists, _ := repo.Exists(context.Background(), "t1", "p1")
	assert.False(t, exists)
}

func TestManageTenantProducts_Disassociate_NotFound(t *testing.T) {
	uc := NewManageTenantProductsUseCase(newMockTenantProductRepo())
	err := uc.Disassociate(context.Background(), DisassociateTenantProductInput{
		TenantID: "t1", ProductID: "missing",
	})
	assert.ErrorIs(t, err, domain.ErrTenantProductNotFound)
}

func TestManageTenantProducts_ListProductTenants(t *testing.T) {
	repo := newMockTenantProductRepo()
	uc := NewManageTenantProductsUseCase(repo)
	_ = uc.Associate(context.Background(), AssociateTenantProductInput{TenantID: "t1", ProductID: "p1"})
	_ = uc.Associate(context.Background(), AssociateTenantProductInput{TenantID: "t2", ProductID: "p1"})

	out, err := uc.ListProductTenants(context.Background(), "p1")
	require.NoError(t, err)
	assert.Len(t, out, 2)
}

func TestManageTenantProducts_ListProductTenants_Empty(t *testing.T) {
	uc := NewManageTenantProductsUseCase(newMockTenantProductRepo())
	out, err := uc.ListProductTenants(context.Background(), "missing")
	require.NoError(t, err)
	assert.Empty(t, out)
}
