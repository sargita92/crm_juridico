package application

import (
	"context"
	"time"

	"github.com/sasrgita/crm-juridico/internal/tenant/domain"
)

type GetTenantOutput struct {
	ID          string
	Name        string
	Type        string
	Document    string
	Status      string
	BlockReason string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

type GetTenantUseCase struct {
	repo domain.TenantRepository
}

func NewGetTenantUseCase(repo domain.TenantRepository) *GetTenantUseCase {
	return &GetTenantUseCase{repo: repo}
}

func (uc *GetTenantUseCase) Execute(ctx context.Context, id string) (*GetTenantOutput, error) {
	tenant, err := uc.repo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}

	return &GetTenantOutput{
		ID:          tenant.ID,
		Name:        tenant.Name,
		Type:        string(tenant.Type),
		Document:    tenant.Document,
		Status:      string(tenant.Status),
		BlockReason: tenant.BlockReason,
		CreatedAt:   tenant.CreatedAt,
		UpdatedAt:   tenant.UpdatedAt,
	}, nil
}
