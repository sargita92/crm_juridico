package application

import (
	"context"

	"github.com/sasrgita/crm-juridico/internal/product/domain"
)

type ProductFunnelInfo struct {
	FunnelID string
	Priority int
}

type ProductListItem struct {
	ID          string
	Name        string
	Description string
	Keywords    []string
	Active      bool
	Funnels     []ProductFunnelInfo
}

type ListProductsUseCase struct {
	productRepo       domain.ProductRepository
	funnelProductRepo domain.FunnelProductRepository
}

func NewListProductsUseCase(productRepo domain.ProductRepository, funnelProductRepo domain.FunnelProductRepository) *ListProductsUseCase {
	return &ListProductsUseCase{productRepo: productRepo, funnelProductRepo: funnelProductRepo}
}

// Execute lists all products (admin view). activeOnly filters inactive products.
func (uc *ListProductsUseCase) Execute(ctx context.Context, activeOnly bool) ([]ProductListItem, error) {
	products, err := uc.productRepo.FindAll(ctx, activeOnly)
	if err != nil {
		return nil, err
	}

	items := make([]ProductListItem, len(products))
	for i, p := range products {
		funnelLinks, err := uc.funnelProductRepo.FindByProductID(ctx, p.ID)
		if err != nil {
			return nil, err
		}

		funnelInfos := make([]ProductFunnelInfo, len(funnelLinks))
		for j, fl := range funnelLinks {
			funnelInfos[j] = ProductFunnelInfo{
				FunnelID: fl.FunnelID,
				Priority: fl.Priority,
			}
		}

		items[i] = ProductListItem{
			ID:          p.ID,
			Name:        p.Name,
			Description: p.Description,
			Keywords:    p.Keywords,
			Active:      p.Active,
			Funnels:     funnelInfos,
		}
	}

	return items, nil
}

// ListTenantProductsUseCase lists products associated to a specific tenant.
type ListTenantProductsUseCase struct {
	productRepo       domain.ProductRepository
	tenantProductRepo domain.TenantProductRepository
}

func NewListTenantProductsUseCase(productRepo domain.ProductRepository, tenantProductRepo domain.TenantProductRepository) *ListTenantProductsUseCase {
	return &ListTenantProductsUseCase{productRepo: productRepo, tenantProductRepo: tenantProductRepo}
}

func (uc *ListTenantProductsUseCase) Execute(ctx context.Context, tenantID string) ([]ProductListItem, error) {
	tenantProducts, err := uc.tenantProductRepo.FindByTenantID(ctx, tenantID)
	if err != nil {
		return nil, err
	}

	if len(tenantProducts) == 0 {
		return []ProductListItem{}, nil
	}

	ids := make([]string, len(tenantProducts))
	for i, tp := range tenantProducts {
		ids[i] = tp.ProductID
	}

	products, err := uc.productRepo.FindActiveByIDs(ctx, ids)
	if err != nil {
		return nil, err
	}

	items := make([]ProductListItem, len(products))
	for i, p := range products {
		items[i] = ProductListItem{
			ID:          p.ID,
			Name:        p.Name,
			Description: p.Description,
			Keywords:    p.Keywords,
			Active:      p.Active,
		}
	}

	return items, nil
}
