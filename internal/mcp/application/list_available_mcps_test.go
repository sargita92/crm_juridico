package application

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/sasrgita/crm-juridico/internal/mcp/domain"
)

func TestListAvailableMcpsUseCase_FiltersAssociated(t *testing.T) {
	mcpRepo := newMockMcpServerRepo()
	specMcpRepo := newMockSpecialistMcpRepo()
	uc := NewListAvailableMcpsUseCase(specMcpRepo, mcpRepo)

	mcp1, _ := domain.NewMcpServer("mcp-1", "Servidor Associado", "https://mcp1.example.com", "", "")
	mcp2, _ := domain.NewMcpServer("mcp-2", "Servidor Disponivel", "https://mcp2.example.com", "", "")
	mcp3, _ := domain.NewMcpServer("mcp-3", "Servidor Disponivel 2", "https://mcp3.example.com", "", "")
	mcpRepo.addMcp(mcp1)
	mcpRepo.addMcp(mcp2)
	mcpRepo.addMcp(mcp3)
	_ = specMcpRepo.Associate(context.Background(), "spec-1", "mcp-1")

	items, err := uc.Execute(context.Background(), "spec-1", "")

	require.NoError(t, err)
	assert.Len(t, items, 2)
	for _, item := range items {
		assert.NotEqual(t, "mcp-1", item.ID)
	}
}

func TestListAvailableMcpsUseCase_WithSearch(t *testing.T) {
	mcpRepo := newMockMcpServerRepo()
	specMcpRepo := newMockSpecialistMcpRepo()
	uc := NewListAvailableMcpsUseCase(specMcpRepo, mcpRepo)

	mcp1, _ := domain.NewMcpServer("mcp-1", "Servidor Jurídico", "https://mcp1.example.com", "", "")
	mcp2, _ := domain.NewMcpServer("mcp-2", "Servidor Trabalhista", "https://mcp2.example.com", "", "")
	mcpRepo.addMcp(mcp1)
	mcpRepo.addMcp(mcp2)

	items, err := uc.Execute(context.Background(), "spec-1", "Jurídico")

	require.NoError(t, err)
	assert.Len(t, items, 1)
	assert.Equal(t, "Servidor Jurídico", items[0].Name)
}

func TestListAvailableMcpsUseCase_AllAssociated(t *testing.T) {
	mcpRepo := newMockMcpServerRepo()
	specMcpRepo := newMockSpecialistMcpRepo()
	uc := NewListAvailableMcpsUseCase(specMcpRepo, mcpRepo)

	mcp1, _ := domain.NewMcpServer("mcp-1", "Servidor 1", "https://mcp.example.com", "", "")
	mcpRepo.addMcp(mcp1)
	_ = specMcpRepo.Associate(context.Background(), "spec-1", "mcp-1")

	items, err := uc.Execute(context.Background(), "spec-1", "")

	require.NoError(t, err)
	assert.Empty(t, items)
}

func TestListAvailableMcpsUseCase_NoMcps(t *testing.T) {
	mcpRepo := newMockMcpServerRepo()
	specMcpRepo := newMockSpecialistMcpRepo()
	uc := NewListAvailableMcpsUseCase(specMcpRepo, mcpRepo)

	items, err := uc.Execute(context.Background(), "spec-1", "")

	require.NoError(t, err)
	assert.Empty(t, items)
}
