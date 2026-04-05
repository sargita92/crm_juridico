package application

import (
	"context"
	"time"

	"github.com/sasrgita/crm-juridico/internal/specialist/domain"
)

type ListSpecialistsInput struct {
	Search string
	Status string
	Page   int
	Limit  int
}

type SpecialistItem struct {
	ID          string
	Name        string
	Description string
	Status      string
	TenantCount int
	CreatedAt   time.Time
}

type ListSpecialistsOutput struct {
	Specialists []SpecialistItem
	Total       int64
	Page        int
	Limit       int
}

type ListSpecialistsUseCase struct {
	repo   domain.SpecialistRepository
	stRepo domain.SpecialistTenantRepository
}

func NewListSpecialistsUseCase(repo domain.SpecialistRepository, stRepo domain.SpecialistTenantRepository) *ListSpecialistsUseCase {
	return &ListSpecialistsUseCase{repo: repo, stRepo: stRepo}
}

func (uc *ListSpecialistsUseCase) Execute(ctx context.Context, input ListSpecialistsInput) (*ListSpecialistsOutput, error) {
	filter := domain.SpecialistFilter{
		Search: input.Search,
		Status: domain.SpecialistStatus(input.Status),
		Page:   input.Page,
		Limit:  input.Limit,
	}

	list, err := uc.repo.FindWithFilter(ctx, filter)
	if err != nil {
		return nil, err
	}

	items := make([]SpecialistItem, len(list.Specialists))
	for i, s := range list.Specialists {
		tenantIDs, err := uc.stRepo.FindTenantIDsBySpecialistID(ctx, s.ID)
		if err != nil {
			return nil, err
		}
		items[i] = SpecialistItem{
			ID:          s.ID,
			Name:        s.Name,
			Description: s.Description,
			Status:      string(s.Status),
			TenantCount: len(tenantIDs),
			CreatedAt:   s.CreatedAt,
		}
	}

	return &ListSpecialistsOutput{
		Specialists: items,
		Total:       list.Total,
		Page:        list.Page,
		Limit:       list.Limit,
	}, nil
}
