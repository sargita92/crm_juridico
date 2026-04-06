package application

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/sasrgita/crm-juridico/internal/mcp/domain"
)

func TestDeleteMcpUseCase_Success(t *testing.T) {
	mcpRepo := newMockMcpServerRepo()
	specMcpRepo := newMockSpecialistMcpRepo()
	uc := NewDeleteMcpUseCase(mcpRepo, specMcpRepo)

	mcp, _ := domain.NewMcpServer("mcp-1", "Servidor para excluir", "https://mcp.example.com", "", "")
	mcpRepo.addMcp(mcp)

	err := uc.Execute(context.Background(), "mcp-1")

	require.NoError(t, err)
	assert.Empty(t, mcpRepo.mcps)
}

func TestDeleteMcpUseCase_NotFound(t *testing.T) {
	mcpRepo := newMockMcpServerRepo()
	specMcpRepo := newMockSpecialistMcpRepo()
	uc := NewDeleteMcpUseCase(mcpRepo, specMcpRepo)

	err := uc.Execute(context.Background(), "nonexistent")

	assert.ErrorIs(t, err, domain.ErrMcpNotFound)
}

func TestDeleteMcpUseCase_DissociatesBeforeDelete(t *testing.T) {
	mcpRepo := newMockMcpServerRepo()
	specMcpRepo := newMockSpecialistMcpRepo()
	uc := NewDeleteMcpUseCase(mcpRepo, specMcpRepo)

	mcp, _ := domain.NewMcpServer("mcp-1", "Servidor Compartilhado", "https://mcp.example.com", "", "")
	mcpRepo.addMcp(mcp)
	_ = specMcpRepo.Associate(context.Background(), "spec-1", "mcp-1")
	_ = specMcpRepo.Associate(context.Background(), "spec-2", "mcp-1")

	err := uc.Execute(context.Background(), "mcp-1")

	require.NoError(t, err)
	assert.Empty(t, mcpRepo.mcps)

	ids, _ := specMcpRepo.FindSpecialistIDsByMcpID(context.Background(), "mcp-1")
	assert.Empty(t, ids)
}
