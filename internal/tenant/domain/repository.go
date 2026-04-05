package domain

import "context"

type TenantFilter struct {
	Search string
	Status TenantStatus
	Type   TenantType
	Page   int
	Limit  int
}

type TenantList struct {
	Tenants []Tenant
	Total   int64
	Page    int
	Limit   int
}

type TenantRepository interface {
	Create(ctx context.Context, tenant *Tenant) error
	FindByID(ctx context.Context, id string) (*Tenant, error)
	FindByIDs(ctx context.Context, ids []string) ([]Tenant, error)
	FindAll(ctx context.Context) ([]Tenant, error)
	Update(ctx context.Context, tenant *Tenant) error
	FindWithFilter(ctx context.Context, filter TenantFilter) (*TenantList, error)
	FindByDocument(ctx context.Context, document string) (*Tenant, error)
}
