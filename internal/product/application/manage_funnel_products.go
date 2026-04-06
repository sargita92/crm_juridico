package application

import (
	"context"

	"github.com/google/uuid"

	"github.com/sasrgita/crm-juridico/internal/product/domain"
)

type LinkFunnelProductInput struct {
	FunnelID  string
	ProductID string
	Priority  int
}

type UnlinkFunnelProductInput struct {
	FunnelID  string
	ProductID string
}

type UpdateFunnelProductPriorityInput struct {
	FunnelID  string
	ProductID string
	Priority  int
}

type ManageFunnelProductsUseCase struct {
	funnelProductRepo domain.FunnelProductRepository
}

func NewManageFunnelProductsUseCase(funnelProductRepo domain.FunnelProductRepository) *ManageFunnelProductsUseCase {
	return &ManageFunnelProductsUseCase{funnelProductRepo: funnelProductRepo}
}

// Link creates a new funnel-product association.
func (uc *ManageFunnelProductsUseCase) Link(ctx context.Context, input LinkFunnelProductInput) error {
	fp, err := domain.NewFunnelProduct(uuid.New().String(), input.FunnelID, input.ProductID, input.Priority)
	if err != nil {
		return err
	}
	return uc.funnelProductRepo.Create(ctx, fp)
}

// Unlink removes a funnel-product association.
func (uc *ManageFunnelProductsUseCase) Unlink(ctx context.Context, input UnlinkFunnelProductInput) error {
	return uc.funnelProductRepo.Delete(ctx, input.FunnelID, input.ProductID)
}

// UpdatePriority updates the priority of an existing funnel-product link.
func (uc *ManageFunnelProductsUseCase) UpdatePriority(ctx context.Context, input UpdateFunnelProductPriorityInput) error {
	return uc.funnelProductRepo.UpdatePriority(ctx, input.FunnelID, input.ProductID, input.Priority)
}
