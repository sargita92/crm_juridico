package domain

import "context"

type ProductRepository interface {
	Create(ctx context.Context, product *Product) error
	FindByID(ctx context.Context, id string) (*Product, error)
	Update(ctx context.Context, product *Product) error
	FindByTenantID(ctx context.Context, tenantID string, activeOnly bool) ([]Product, error)
	FindActiveByTenantID(ctx context.Context, tenantID string) ([]Product, error)
}

type FunnelProductRepository interface {
	Create(ctx context.Context, fp *FunnelProduct) error
	Delete(ctx context.Context, funnelID, productID string) error
	FindByProductID(ctx context.Context, productID string) ([]FunnelProduct, error)
	FindByFunnelID(ctx context.Context, funnelID string) ([]FunnelProduct, error)
	FindTopPriorityFunnel(ctx context.Context, productID string) (*FunnelProduct, error)
	UpdatePriority(ctx context.Context, funnelID, productID string, priority int) error
}

type PhoneNumberRepository interface {
	Create(ctx context.Context, pn *ProductPhoneNumber) error
	Delete(ctx context.Context, id string) error
	FindByPhoneNumber(ctx context.Context, phoneNumber string) (*ProductPhoneNumber, error)
	FindByProductID(ctx context.Context, productID string) ([]ProductPhoneNumber, error)
}
