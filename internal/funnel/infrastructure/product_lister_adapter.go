package infrastructure

import (
	"context"

	funneldomain "github.com/sasrgita/crm-juridico/internal/funnel/domain"
	productinfra "github.com/sasrgita/crm-juridico/internal/product/infrastructure"
)

// ProductListerAdapter adapts the product repository to the funnel domain's ProductLister interface.
type ProductListerAdapter struct {
	productRepo *productinfra.GormProductRepository
}

func NewProductListerAdapter(productRepo *productinfra.GormProductRepository) *ProductListerAdapter {
	return &ProductListerAdapter{productRepo: productRepo}
}

func (a *ProductListerAdapter) ListActiveProducts(ctx context.Context, tenantID string) ([]funneldomain.ProductInfo, error) {
	products, err := a.productRepo.FindActiveByTenantID(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	result := make([]funneldomain.ProductInfo, len(products))
	for i, p := range products {
		result[i] = funneldomain.ProductInfo{ID: p.ID, Name: p.Name}
	}
	return result, nil
}
