package application

import (
	"context"
	"time"

	"github.com/sasrgita/crm-juridico/internal/mcp/domain"
)

type McpDetailOutput struct {
	ID              string
	Name            string
	URL             string
	Config          string
	Headers         string
	SpecialistCount int
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

type GetMcpUseCase struct {
	repo        domain.McpServerRepository
	specMcpRepo domain.SpecialistMcpRepository
}

func NewGetMcpUseCase(repo domain.McpServerRepository, specMcpRepo domain.SpecialistMcpRepository) *GetMcpUseCase {
	return &GetMcpUseCase{repo: repo, specMcpRepo: specMcpRepo}
}

func (uc *GetMcpUseCase) Execute(ctx context.Context, id string) (*McpDetailOutput, error) {
	mcp, err := uc.repo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}

	count, err := uc.specMcpRepo.CountByMcpID(ctx, mcp.ID)
	if err != nil {
		return nil, err
	}

	return &McpDetailOutput{
		ID:              mcp.ID,
		Name:            mcp.Name,
		URL:             mcp.URL,
		Config:          mcp.Config,
		Headers:         mcp.Headers,
		SpecialistCount: count,
		CreatedAt:       mcp.CreatedAt,
		UpdatedAt:       mcp.UpdatedAt,
	}, nil
}
