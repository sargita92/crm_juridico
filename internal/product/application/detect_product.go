package application

import (
	"context"

	"github.com/sasrgita/crm-juridico/internal/product/domain"
)

type DetectProductUseCase struct {
	productRepo       domain.ProductRepository
	tenantProductRepo domain.TenantProductRepository
}

func NewDetectProductUseCase(productRepo domain.ProductRepository, tenantProductRepo domain.TenantProductRepository) *DetectProductUseCase {
	return &DetectProductUseCase{productRepo: productRepo, tenantProductRepo: tenantProductRepo}
}

// Execute loads active products associated to the tenant and checks each one with MatchesText.
// Returns (productID, found, error). If no product matches, returns ("", false, nil).
func (uc *DetectProductUseCase) Execute(ctx context.Context, tenantID, text string) (string, bool, error) {
	tenantProducts, err := uc.tenantProductRepo.FindByTenantID(ctx, tenantID)
	if err != nil {
		return "", false, err
	}

	if len(tenantProducts) == 0 {
		return "", false, nil
	}

	ids := make([]string, len(tenantProducts))
	for i, tp := range tenantProducts {
		ids[i] = tp.ProductID
	}

	products, err := uc.productRepo.FindActiveByIDs(ctx, ids)
	if err != nil {
		return "", false, err
	}

	for _, p := range products {
		if p.MatchesText(text) {
			return p.ID, true, nil
		}
	}

	return "", false, nil
}
