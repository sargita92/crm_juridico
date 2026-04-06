package infrastructure

import (
	"context"

	funneldomain "github.com/sasrgita/crm-juridico/internal/funnel/domain"
	productinfra "github.com/sasrgita/crm-juridico/internal/product/infrastructure"
)

// ProductListerAdapter adapts the product repository to the funnel domain's ProductLister interface.
// It uses tenant_products to find which products belong to a tenant, then fetches the active ones.
type ProductListerAdapter struct {
	productRepo *productinfra.GormProductRepository
	tpRepo      *productinfra.GormTenantProductRepository
}

func NewProductListerAdapter(productRepo *productinfra.GormProductRepository, tpRepo *productinfra.GormTenantProductRepository) *ProductListerAdapter {
	return &ProductListerAdapter{productRepo: productRepo, tpRepo: tpRepo}
}

func (a *ProductListerAdapter) ListActiveProducts(ctx context.Context, tenantID string) ([]funneldomain.ProductInfo, error) {
	tenantProducts, err := a.tpRepo.FindByTenantID(ctx, tenantID)
	if err != nil {
		return nil, err
	}

	if len(tenantProducts) == 0 {
		return []funneldomain.ProductInfo{}, nil
	}

	ids := make([]string, len(tenantProducts))
	for i, tp := range tenantProducts {
		ids[i] = tp.ProductID
	}

	products, err := a.productRepo.FindActiveByIDs(ctx, ids)
	if err != nil {
		return nil, err
	}

	result := make([]funneldomain.ProductInfo, len(products))
	for i, p := range products {
		result[i] = funneldomain.ProductInfo{ID: p.ID, Name: p.Name}
	}
	return result, nil
}
