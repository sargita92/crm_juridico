package infrastructure

import (
	"context"
	"fmt"

	productApp "github.com/sasrgita/crm-juridico/internal/product/application"
	productDomain "github.com/sasrgita/crm-juridico/internal/product/domain"
)

// ProductInfoFinderAdapter satisfies application.ProductInfoFinder.
type ProductInfoFinderAdapter struct {
	repo productDomain.ProductRepository
}

func NewProductInfoFinderAdapter(repo productDomain.ProductRepository) *ProductInfoFinderAdapter {
	return &ProductInfoFinderAdapter{repo: repo}
}

func (a *ProductInfoFinderAdapter) FindProductInfo(ctx context.Context, productID string) (name, description string, err error) {
	p, err := a.repo.FindByID(ctx, productID)
	if err != nil {
		return "", "", fmt.Errorf("product_info_finder_adapter: find product: %w", err)
	}
	return p.Name, p.Description, nil
}

// PhoneNumberFinderAdapter satisfies application.PhoneNumberFinder.
type PhoneNumberFinderAdapter struct {
	repo productDomain.PhoneNumberRepository
}

func NewPhoneNumberFinderAdapter(repo productDomain.PhoneNumberRepository) *PhoneNumberFinderAdapter {
	return &PhoneNumberFinderAdapter{repo: repo}
}

func (a *PhoneNumberFinderAdapter) FindProductIDByPhone(ctx context.Context, phone string) (string, error) {
	pn, err := a.repo.FindByPhoneNumber(ctx, phone)
	if err != nil {
		return "", fmt.Errorf("phone_number_finder_adapter: find by phone: %w", err)
	}
	if pn == nil {
		return "", fmt.Errorf("phone_number_finder_adapter: no product for phone %s", phone)
	}
	return pn.ProductID, nil
}

// ProductDetectorAdapter satisfies application.ProductDetectorForRouter.
type ProductDetectorAdapter struct {
	uc *productApp.DetectProductUseCase
}

func NewProductDetectorAdapter(uc *productApp.DetectProductUseCase) *ProductDetectorAdapter {
	return &ProductDetectorAdapter{uc: uc}
}

func (a *ProductDetectorAdapter) DetectFromMessage(ctx context.Context, tenantID, text string) (string, bool, error) {
	return a.uc.Execute(ctx, tenantID, text)
}
