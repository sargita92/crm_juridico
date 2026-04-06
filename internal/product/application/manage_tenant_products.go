package application

import (
	"context"

	"github.com/google/uuid"

	"github.com/sasrgita/crm-juridico/internal/product/domain"
)

type AssociateTenantProductInput struct {
	TenantID  string
	ProductID string
}

type DisassociateTenantProductInput struct {
	TenantID  string
	ProductID string
}

type ManageTenantProductsUseCase struct {
	tenantProductRepo domain.TenantProductRepository
}

func NewManageTenantProductsUseCase(tenantProductRepo domain.TenantProductRepository) *ManageTenantProductsUseCase {
	return &ManageTenantProductsUseCase{tenantProductRepo: tenantProductRepo}
}

// Associate creates a tenant-product association.
func (uc *ManageTenantProductsUseCase) Associate(ctx context.Context, input AssociateTenantProductInput) error {
	tp, err := domain.NewTenantProduct(uuid.New().String(), input.TenantID, input.ProductID)
	if err != nil {
		return err
	}
	return uc.tenantProductRepo.Create(ctx, tp)
}

// Disassociate removes a tenant-product association.
func (uc *ManageTenantProductsUseCase) Disassociate(ctx context.Context, input DisassociateTenantProductInput) error {
	return uc.tenantProductRepo.Delete(ctx, input.TenantID, input.ProductID)
}

// TenantProductInfo contains tenant info for a product association listing.
type TenantProductInfo struct {
	TenantID string
}

// ListProductTenants returns all tenants associated with a given product.
func (uc *ManageTenantProductsUseCase) ListProductTenants(ctx context.Context, productID string) ([]TenantProductInfo, error) {
	tps, err := uc.tenantProductRepo.FindByProductID(ctx, productID)
	if err != nil {
		return nil, err
	}
	result := make([]TenantProductInfo, len(tps))
	for i, tp := range tps {
		result[i] = TenantProductInfo{TenantID: tp.TenantID}
	}
	return result, nil
}
