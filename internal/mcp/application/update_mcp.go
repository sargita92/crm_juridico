package application

import (
	"context"

	"github.com/sasrgita/crm-juridico/internal/mcp/domain"
)

type UpdateMcpInput struct {
	ID      string
	Name    string
	URL     string
	Config  string
	Headers string
}

type UpdateMcpUseCase struct {
	repo domain.McpServerRepository
}

func NewUpdateMcpUseCase(repo domain.McpServerRepository) *UpdateMcpUseCase {
	return &UpdateMcpUseCase{repo: repo}
}

func (uc *UpdateMcpUseCase) Execute(ctx context.Context, input UpdateMcpInput) error {
	mcp, err := uc.repo.FindByID(ctx, input.ID)
	if err != nil {
		return err
	}

	if err := mcp.Update(input.Name, input.URL, input.Config, input.Headers); err != nil {
		return err
	}

	return uc.repo.Update(ctx, mcp)
}
