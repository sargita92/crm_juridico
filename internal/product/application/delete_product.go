package application

import (
	"context"

	"github.com/sasrgita/crm-juridico/internal/product/domain"
)

type DeleteProductInput struct {
	ProductID string
}

type DeleteProductUseCase struct {
	productRepo domain.ProductRepository
}

func NewDeleteProductUseCase(productRepo domain.ProductRepository) *DeleteProductUseCase {
	return &DeleteProductUseCase{productRepo: productRepo}
}

func (uc *DeleteProductUseCase) Execute(ctx context.Context, input DeleteProductInput) error {
	return uc.productRepo.Delete(ctx, input.ProductID)
}
