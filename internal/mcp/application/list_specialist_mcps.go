package application

import (
	"context"

	"github.com/sasrgita/crm-juridico/internal/mcp/domain"
)

type ListSpecialistMcpsUseCase struct {
	specMcpRepo domain.SpecialistMcpRepository
	mcpRepo     domain.McpServerRepository
}

func NewListSpecialistMcpsUseCase(specMcpRepo domain.SpecialistMcpRepository, mcpRepo domain.McpServerRepository) *ListSpecialistMcpsUseCase {
	return &ListSpecialistMcpsUseCase{specMcpRepo: specMcpRepo, mcpRepo: mcpRepo}
}

func (uc *ListSpecialistMcpsUseCase) Execute(ctx context.Context, specialistID string) ([]McpItem, error) {
	mcpIDs, err := uc.specMcpRepo.FindMcpIDsBySpecialistID(ctx, specialistID)
	if err != nil {
		return nil, err
	}

	if len(mcpIDs) == 0 {
		return []McpItem{}, nil
	}

	mcps, err := uc.mcpRepo.FindByIDs(ctx, mcpIDs)
	if err != nil {
		return nil, err
	}

	items := make([]McpItem, len(mcps))
	for i, mcp := range mcps {
		items[i] = McpItem{
			ID:   mcp.ID,
			Name: mcp.Name,
			URL:  mcp.URL,
		}
	}
	return items, nil
}
