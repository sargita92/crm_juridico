package application

import (
	"context"

	"github.com/sasrgita/crm-juridico/internal/product/domain"
)

type ToggleProductInput struct {
	ProductID string
}

type ToggleProductUseCase struct {
	productRepo domain.ProductRepository
}

func NewToggleProductUseCase(productRepo domain.ProductRepository) *ToggleProductUseCase {
	return &ToggleProductUseCase{productRepo: productRepo}
}

// Execute toggles the product active/inactive and returns the new active state.
func (uc *ToggleProductUseCase) Execute(ctx context.Context, input ToggleProductInput) (bool, error) {
	product, err := uc.productRepo.FindByID(ctx, input.ProductID)
	if err != nil {
		return false, err
	}

	if product.Active {
		product.Deactivate()
	} else {
		product.Activate()
	}

	if err := uc.productRepo.Update(ctx, product); err != nil {
		return false, err
	}

	return product.Active, nil
}
