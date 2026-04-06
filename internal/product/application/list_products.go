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
	productRepo      domain.ProductRepository
	funnelProductRepo domain.FunnelProductRepository
}

func NewListProductsUseCase(productRepo domain.ProductRepository, funnelProductRepo domain.FunnelProductRepository) *ListProductsUseCase {
	return &ListProductsUseCase{productRepo: productRepo, funnelProductRepo: funnelProductRepo}
}

func (uc *ListProductsUseCase) Execute(ctx context.Context, tenantID string, activeOnly bool) ([]ProductListItem, error) {
	products, err := uc.productRepo.FindByTenantID(ctx, tenantID, activeOnly)
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
