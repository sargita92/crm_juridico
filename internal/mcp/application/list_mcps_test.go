package application

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/sasrgita/crm-juridico/internal/mcp/domain"
)

func TestListMcpsUseCase_Success(t *testing.T) {
	mcpRepo := newMockMcpServerRepo()
	specMcpRepo := newMockSpecialistMcpRepo()
	uc := NewListMcpsUseCase(mcpRepo, specMcpRepo)

	mcp1, _ := domain.NewMcpServer("mcp-1", "Servidor Jurídico", "https://mcp1.example.com", "", "")
	mcp2, _ := domain.NewMcpServer("mcp-2", "Servidor Trabalhista", "https://mcp2.example.com", "", "")
	mcpRepo.addMcp(mcp1)
	mcpRepo.addMcp(mcp2)

	output, err := uc.Execute(context.Background(), ListMcpsInput{Page: 1, Limit: 20})

	require.NoError(t, err)
	assert.Equal(t, int64(2), output.Total)
	assert.Len(t, output.McpServers, 2)
}

func TestListMcpsUseCase_WithSearch(t *testing.T) {
	mcpRepo := newMockMcpServerRepo()
	specMcpRepo := newMockSpecialistMcpRepo()
	uc := NewListMcpsUseCase(mcpRepo, specMcpRepo)

	mcp1, _ := domain.NewMcpServer("mcp-1", "Servidor Jurídico", "https://mcp1.example.com", "", "")
	mcp2, _ := domain.NewMcpServer("mcp-2", "Servidor Trabalhista", "https://mcp2.example.com", "", "")
	mcpRepo.addMcp(mcp1)
	mcpRepo.addMcp(mcp2)

	output, err := uc.Execute(context.Background(), ListMcpsInput{Search: "Jurídico", Page: 1, Limit: 20})

	require.NoError(t, err)
	assert.Equal(t, int64(1), output.Total)
	assert.Equal(t, "Servidor Jurídico", output.McpServers[0].Name)
}

func TestListMcpsUseCase_Empty(t *testing.T) {
	mcpRepo := newMockMcpServerRepo()
	specMcpRepo := newMockSpecialistMcpRepo()
	uc := NewListMcpsUseCase(mcpRepo, specMcpRepo)

	output, err := uc.Execute(context.Background(), ListMcpsInput{Page: 1, Limit: 20})

	require.NoError(t, err)
	assert.Equal(t, int64(0), output.Total)
	assert.Empty(t, output.McpServers)
}

func TestListMcpsUseCase_WithSpecialistCount(t *testing.T) {
	mcpRepo := newMockMcpServerRepo()
	specMcpRepo := newMockSpecialistMcpRepo()
	uc := NewListMcpsUseCase(mcpRepo, specMcpRepo)

	mcp, _ := domain.NewMcpServer("mcp-1", "Servidor Compartilhado", "https://mcp.example.com", "", "")
	mcpRepo.addMcp(mcp)
	_ = specMcpRepo.Associate(context.Background(), "spec-1", "mcp-1")
	_ = specMcpRepo.Associate(context.Background(), "spec-2", "mcp-1")

	output, err := uc.Execute(context.Background(), ListMcpsInput{Page: 1, Limit: 20})

	require.NoError(t, err)
	assert.Equal(t, 2, output.McpServers[0].SpecialistCount)
}
