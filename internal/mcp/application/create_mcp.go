package application

import (
	"context"

	"github.com/google/uuid"

	"github.com/sasrgita/crm-juridico/internal/mcp/domain"
)

type CreateMcpInput struct {
	Name    string
	URL     string
	Config  string
	Headers string
}

type CreateMcpOutput struct {
	ID   string
	Name string
	URL  string
}

type CreateMcpUseCase struct {
	repo domain.McpServerRepository
}

func NewCreateMcpUseCase(repo domain.McpServerRepository) *CreateMcpUseCase {
	return &CreateMcpUseCase{repo: repo}
}

func (uc *CreateMcpUseCase) Execute(ctx context.Context, input CreateMcpInput) (*CreateMcpOutput, error) {
	mcp, err := domain.NewMcpServer(uuid.New().String(), input.Name, input.URL, input.Config, input.Headers)
	if err != nil {
		return nil, err
	}

	if err := uc.repo.Create(ctx, mcp); err != nil {
		return nil, err
	}

	return &CreateMcpOutput{
		ID:   mcp.ID,
		Name: mcp.Name,
		URL:  mcp.URL,
	}, nil
}
