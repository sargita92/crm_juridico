package application

import (
	"context"

	"github.com/sasrgita/crm-juridico/internal/files/domain"
)

type GetFileUseCase struct {
	repo domain.FileRepository
}

func NewGetFileUseCase(repo domain.FileRepository) *GetFileUseCase {
	return &GetFileUseCase{repo: repo}
}

func (uc *GetFileUseCase) Execute(ctx context.Context, tenantID, id string) (*domain.File, error) {
	if tenantID == "" {
		return nil, domain.ErrTenantIDRequired
	}
	return uc.repo.FindByID(ctx, tenantID, id)
}
