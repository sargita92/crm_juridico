package infrastructure

import (
	"context"

	productapp "github.com/sasrgita/crm-juridico/internal/product/application"
	productinfra "github.com/sasrgita/crm-juridico/internal/product/infrastructure"
)

type ProductDetectorAdapter struct {
	detectUC *productapp.DetectProductUseCase
}

func NewProductDetectorAdapter(detectUC *productapp.DetectProductUseCase) *ProductDetectorAdapter {
	return &ProductDetectorAdapter{detectUC: detectUC}
}

func (a *ProductDetectorAdapter) DetectFromMessage(ctx context.Context, tenantID, messageText string) (string, bool, error) {
	return a.detectUC.Execute(ctx, tenantID, messageText)
}

type ProductProviderAdapter struct {
	productRepo *productinfra.GormProductRepository
}

func NewProductProviderAdapter(productRepo *productinfra.GormProductRepository) *ProductProviderAdapter {
	return &ProductProviderAdapter{productRepo: productRepo}
}

func (a *ProductProviderAdapter) FindProductNameByID(ctx context.Context, id string) (string, error) {
	p, err := a.productRepo.FindByID(ctx, id)
	if err != nil {
		return "", err
	}
	return p.Name, nil
}

type FunnelProductRouterAdapter struct {
	fpRepo *productinfra.GormFunnelProductRepository
}

func NewFunnelProductRouterAdapter(fpRepo *productinfra.GormFunnelProductRepository) *FunnelProductRouterAdapter {
	return &FunnelProductRouterAdapter{fpRepo: fpRepo}
}

func (a *FunnelProductRouterAdapter) FindTopPriorityFunnelID(ctx context.Context, tenantID, productID string) (string, error) {
	fp, err := a.fpRepo.FindTopPriorityFunnel(ctx, tenantID, productID)
	if err != nil {
		return "", err
	}
	return fp.FunnelID, nil
}
