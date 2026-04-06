package application

import (
	"context"

	"github.com/sasrgita/crm-juridico/internal/mcp/domain"
)

type DeleteMcpUseCase struct {
	repo        domain.McpServerRepository
	specMcpRepo domain.SpecialistMcpRepository
}

func NewDeleteMcpUseCase(repo domain.McpServerRepository, specMcpRepo domain.SpecialistMcpRepository) *DeleteMcpUseCase {
	return &DeleteMcpUseCase{repo: repo, specMcpRepo: specMcpRepo}
}

func (uc *DeleteMcpUseCase) Execute(ctx context.Context, id string) error {
	if _, err := uc.repo.FindByID(ctx, id); err != nil {
		return err
	}

	if err := uc.specMcpRepo.DissociateAllByMcpID(ctx, id); err != nil {
		return err
	}

	return uc.repo.Delete(ctx, id)
}
