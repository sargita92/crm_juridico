package application

import (
	"context"

	"github.com/sasrgita/crm-juridico/internal/product/domain"
)

type DetectProductUseCase struct {
	productRepo domain.ProductRepository
}

func NewDetectProductUseCase(productRepo domain.ProductRepository) *DetectProductUseCase {
	return &DetectProductUseCase{productRepo: productRepo}
}

// Execute loads active products for the tenant and checks each one with MatchesText.
// Returns (productID, found, error). If no product matches, returns ("", false, nil).
func (uc *DetectProductUseCase) Execute(ctx context.Context, tenantID, text string) (string, bool, error) {
	products, err := uc.productRepo.FindActiveByTenantID(ctx, tenantID)
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
