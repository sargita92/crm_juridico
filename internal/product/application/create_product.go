package application

import (
	"context"

	"github.com/google/uuid"

	"github.com/sasrgita/crm-juridico/internal/product/domain"
)

type CreateProductInput struct {
	Name        string
	Description string
	Keywords    []string
}

type ProductOutput struct {
	ID          string
	Name        string
	Description string
	Keywords    []string
	Active      bool
}

type CreateProductUseCase struct {
	productRepo domain.ProductRepository
}

func NewCreateProductUseCase(productRepo domain.ProductRepository) *CreateProductUseCase {
	return &CreateProductUseCase{productRepo: productRepo}
}

func (uc *CreateProductUseCase) Execute(ctx context.Context, input CreateProductInput) (*ProductOutput, error) {
	product, err := domain.NewProduct(uuid.New().String(), input.Name, input.Description, input.Keywords)
	if err != nil {
		return nil, err
	}

	if err := uc.productRepo.Create(ctx, product); err != nil {
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
