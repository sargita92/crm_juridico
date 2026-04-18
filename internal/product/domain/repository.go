package domain

import "context"

type ProductRepository interface {
	Create(ctx context.Context, product *Product) error
	FindByID(ctx context.Context, id string) (*Product, error)
	Update(ctx context.Context, product *Product) error
	Delete(ctx context.Context, id string) error
	FindAll(ctx context.Context, activeOnly bool) ([]Product, error)
	FindActiveByIDs(ctx context.Context, ids []string) ([]Product, error)
}

type TenantProductRepository interface {
	Create(ctx context.Context, tp *TenantProduct) error
	Delete(ctx context.Context, tenantID, productID string) error
	FindByTenantID(ctx context.Context, tenantID string) ([]TenantProduct, error)
	FindByProductID(ctx context.Context, productID string) ([]TenantProduct, error)
	Exists(ctx context.Context, tenantID, productID string) (bool, error)
}

type FunnelProductRepository interface {
	Create(ctx context.Context, fp *FunnelProduct) error
	Delete(ctx context.Context, funnelID, productID string) error
	FindByProductID(ctx context.Context, productID string) ([]FunnelProduct, error)
	FindByFunnelID(ctx context.Context, funnelID string) ([]FunnelProduct, error)
	// FindTopPriorityFunnel returns the funnel-product association with the
	// highest priority for this product, restricted to funnels owned by the
	// given tenant. Tenant-scoped to prevent cross-tenant leakage.
	FindTopPriorityFunnel(ctx context.Context, tenantID, productID string) (*FunnelProduct, error)
	UpdatePriority(ctx context.Context, funnelID, productID string, priority int) error
}

type PhoneNumberRepository interface {
	Create(ctx context.Context, pn *ProductPhoneNumber) error
	Delete(ctx context.Context, id string) error
	FindByPhoneNumber(ctx context.Context, phoneNumber string) (*ProductPhoneNumber, error)
	FindByProductID(ctx context.Context, productID string) ([]ProductPhoneNumber, error)
}
