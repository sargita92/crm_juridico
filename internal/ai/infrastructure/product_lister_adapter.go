package infrastructure

import (
	"context"
	"fmt"

	aihttp "github.com/sasrgita/crm-juridico/internal/ai/interfaces/http"
	productDomain "github.com/sasrgita/crm-juridico/internal/product/domain"
)

// ProductListerAdapter wraps productDomain.ProductRepository to satisfy the
// ai/interfaces/http.ProductLister interface.
type ProductListerAdapter struct {
	repo productDomain.ProductRepository
}

// NewProductListerAdapter creates a new ProductListerAdapter.
func NewProductListerAdapter(repo productDomain.ProductRepository) *ProductListerAdapter {
	return &ProductListerAdapter{repo: repo}
}

// ListAllProducts returns all active products as ProductInfo slices.
func (a *ProductListerAdapter) ListAllProducts(ctx context.Context) ([]aihttp.ProductInfo, error) {
	products, err := a.repo.FindAll(ctx, true)
	if err != nil {
		return nil, fmt.Errorf("product_lister_adapter: find all: %w", err)
	}

	result := make([]aihttp.ProductInfo, len(products))
	for i, p := range products {
		result[i] = aihttp.ProductInfo{
			ID:   p.ID,
			Name: p.Name,
		}
	}
	return result, nil
}
