package application

import (
	"context"
	"time"

	"github.com/sasrgita/crm-juridico/internal/mcp/domain"
)

type ListMcpsInput struct {
	Search string
	Page   int
	Limit  int
}

type McpItem struct {
	ID              string
	Name            string
	URL             string
	SpecialistCount int
	CreatedAt       time.Time
}

type ListMcpsOutput struct {
	McpServers []McpItem
	Total      int64
	Page       int
	Limit      int
}

type ListMcpsUseCase struct {
	repo        domain.McpServerRepository
	specMcpRepo domain.SpecialistMcpRepository
}

func NewListMcpsUseCase(repo domain.McpServerRepository, specMcpRepo domain.SpecialistMcpRepository) *ListMcpsUseCase {
	return &ListMcpsUseCase{repo: repo, specMcpRepo: specMcpRepo}
}

func (uc *ListMcpsUseCase) Execute(ctx context.Context, input ListMcpsInput) (*ListMcpsOutput, error) {
	filter := domain.McpFilter{
		Search: input.Search,
		Page:   input.Page,
		Limit:  input.Limit,
	}

	list, err := uc.repo.FindWithFilter(ctx, filter)
	if err != nil {
		return nil, err
	}

	items := make([]McpItem, len(list.McpServers))
	for i, mcp := range list.McpServers {
		count, err := uc.specMcpRepo.CountByMcpID(ctx, mcp.ID)
		if err != nil {
			return nil, err
		}
		items[i] = McpItem{
			ID:              mcp.ID,
			Name:            mcp.Name,
			URL:             mcp.URL,
			SpecialistCount: count,
			CreatedAt:       mcp.CreatedAt,
		}
	}

	return &ListMcpsOutput{
		McpServers: items,
		Total:      list.Total,
		Page:       list.Page,
		Limit:      list.Limit,
	}, nil
}
