package application

import (
	"context"
	"time"

	"github.com/sasrgita/crm-juridico/internal/tenant/domain"
)

type ListTenantsInput struct {
	Search string
	Status string
	Type   string
	Page   int
	Limit  int
}

type TenantItem struct {
	ID        string
	Name      string
	Type      string
	Document  string
	Status    string
	CreatedAt time.Time
}

type ListTenantsOutput struct {
	Tenants []TenantItem
	Total   int64
	Page    int
	Limit   int
}

type ListTenantsUseCase struct {
	repo domain.TenantRepository
}

func NewListTenantsUseCase(repo domain.TenantRepository) *ListTenantsUseCase {
	return &ListTenantsUseCase{repo: repo}
}

func (uc *ListTenantsUseCase) Execute(ctx context.Context, input ListTenantsInput) (*ListTenantsOutput, error) {
	filter := domain.TenantFilter{
		Search: input.Search,
		Status: domain.TenantStatus(input.Status),
		Type:   domain.TenantType(input.Type),
		Page:   input.Page,
		Limit:  input.Limit,
	}

	list, err := uc.repo.FindWithFilter(ctx, filter)
	if err != nil {
		return nil, err
	}

	items := make([]TenantItem, len(list.Tenants))
	for i, t := range list.Tenants {
		items[i] = TenantItem{
			ID:        t.ID,
			Name:      t.Name,
			Type:      string(t.Type),
			Document:  t.Document,
			Status:    string(t.Status),
			CreatedAt: t.CreatedAt,
		}
	}

	return &ListTenantsOutput{
		Tenants: items,
		Total:   list.Total,
		Page:    list.Page,
		Limit:   list.Limit,
	}, nil
}
