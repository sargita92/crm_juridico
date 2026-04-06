package application

import (
	"context"

	"github.com/sasrgita/crm-juridico/internal/product/domain"
)

type UpdateProductInput struct {
	ProductID   string
	Name        string
	Description string
	Keywords    []string
}

type UpdateProductUseCase struct {
	productRepo domain.ProductRepository
}

func NewUpdateProductUseCase(productRepo domain.ProductRepository) *UpdateProductUseCase {
	return &UpdateProductUseCase{productRepo: productRepo}
}

func (uc *UpdateProductUseCase) Execute(ctx context.Context, input UpdateProductInput) (*ProductOutput, error) {
	product, err := uc.productRepo.FindByID(ctx, input.ProductID)
	if err != nil {
		return nil, err
	}

	if err := product.Update(input.Name, input.Description, input.Keywords); err != nil {
		return nil, err
	}

	if err := uc.productRepo.Update(ctx, product); err != nil {
		return nil, err
	}

	return &ProductOutput{
		ID:          product.ID,
		Name:        product.Name,
		Description: product.Description,
		Keywords:    product.Keywords,
		Active:      product.Active,
	}, nil
}
