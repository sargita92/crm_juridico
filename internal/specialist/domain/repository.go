package domain

import "context"

type SpecialistFilter struct {
	Search string
	Status SpecialistStatus
	Page   int
	Limit  int
}

type SpecialistList struct {
	Specialists []Specialist
	Total       int64
	Page        int
	Limit       int
}

type SpecialistRepository interface {
	Create(ctx context.Context, specialist *Specialist) error
	FindByID(ctx context.Context, id string) (*Specialist, error)
	Update(ctx context.Context, specialist *Specialist) error
	FindWithFilter(ctx context.Context, filter SpecialistFilter) (*SpecialistList, error)
}

type SpecialistTenantRepository interface {
	Associate(ctx context.Context, specialistID, tenantID string) error
	Dissociate(ctx context.Context, specialistID, tenantID string) error
	FindTenantIDsBySpecialistID(ctx context.Context, specialistID string) ([]string, error)
	FindSpecialistIDsByTenantID(ctx context.Context, tenantID string) ([]string, error)
	Exists(ctx context.Context, specialistID, tenantID string) (bool, error)
	FindDefaultByTenantID(ctx context.Context, tenantID string) (string, error)
	SetDefault(ctx context.Context, specialistID, tenantID string) error
}
