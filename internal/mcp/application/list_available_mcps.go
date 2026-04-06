package application

import (
	"context"

	"github.com/sasrgita/crm-juridico/internal/mcp/domain"
)

type ListAvailableMcpsUseCase struct {
	specMcpRepo domain.SpecialistMcpRepository
	mcpRepo     domain.McpServerRepository
}

func NewListAvailableMcpsUseCase(specMcpRepo domain.SpecialistMcpRepository, mcpRepo domain.McpServerRepository) *ListAvailableMcpsUseCase {
	return &ListAvailableMcpsUseCase{specMcpRepo: specMcpRepo, mcpRepo: mcpRepo}
}

func (uc *ListAvailableMcpsUseCase) Execute(ctx context.Context, specialistID string, search string) ([]McpItem, error) {
	associatedIDs, err := uc.specMcpRepo.FindMcpIDsBySpecialistID(ctx, specialistID)
	if err != nil {
		return nil, err
	}

	associatedSet := make(map[string]bool, len(associatedIDs))
	for _, id := range associatedIDs {
		associatedSet[id] = true
	}

	list, err := uc.mcpRepo.FindWithFilter(ctx, domain.McpFilter{
		Search: search,
		Page:   1,
		Limit:  100,
	})
	if err != nil {
		return nil, err
	}

	var items []McpItem
	for _, mcp := range list.McpServers {
		if associatedSet[mcp.ID] {
			continue
		}
		items = append(items, McpItem{
			ID:   mcp.ID,
			Name: mcp.Name,
			URL:  mcp.URL,
		})
	}

	if items == nil {
		items = []McpItem{}
	}
	return items, nil
}
