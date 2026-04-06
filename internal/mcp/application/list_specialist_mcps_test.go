package application

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/sasrgita/crm-juridico/internal/mcp/domain"
)

func TestListSpecialistMcpsUseCase_Success(t *testing.T) {
	mcpRepo := newMockMcpServerRepo()
	specMcpRepo := newMockSpecialistMcpRepo()
	uc := NewListSpecialistMcpsUseCase(specMcpRepo, mcpRepo)

	mcp1, _ := domain.NewMcpServer("mcp-1", "Servidor 1", "https://mcp1.example.com", "", "")
	mcp2, _ := domain.NewMcpServer("mcp-2", "Servidor 2", "https://mcp2.example.com", "", "")
	mcpRepo.addMcp(mcp1)
	mcpRepo.addMcp(mcp2)
	_ = specMcpRepo.Associate(context.Background(), "spec-1", "mcp-1")
	_ = specMcpRepo.Associate(context.Background(), "spec-1", "mcp-2")

	items, err := uc.Execute(context.Background(), "spec-1")

	require.NoError(t, err)
	assert.Len(t, items, 2)
}

func TestListSpecialistMcpsUseCase_Empty(t *testing.T) {
	mcpRepo := newMockMcpServerRepo()
	specMcpRepo := newMockSpecialistMcpRepo()
	uc := NewListSpecialistMcpsUseCase(specMcpRepo, mcpRepo)

	items, err := uc.Execute(context.Background(), "spec-1")

	require.NoError(t, err)
	assert.Empty(t, items)
}
