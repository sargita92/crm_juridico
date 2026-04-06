package application

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/sasrgita/crm-juridico/internal/mcp/domain"
)

func TestGetMcpUseCase_Success(t *testing.T) {
	mcpRepo := newMockMcpServerRepo()
	specMcpRepo := newMockSpecialistMcpRepo()
	uc := NewGetMcpUseCase(mcpRepo, specMcpRepo)

	mcp, _ := domain.NewMcpServer("mcp-1", "Servidor Jurídico", "https://mcp.example.com", `{"timeout":30}`, `{"X-Key":"val"}`)
	mcpRepo.addMcp(mcp)

	output, err := uc.Execute(context.Background(), "mcp-1")

	require.NoError(t, err)
	assert.Equal(t, "mcp-1", output.ID)
	assert.Equal(t, "Servidor Jurídico", output.Name)
	assert.Equal(t, "https://mcp.example.com", output.URL)
	assert.Equal(t, `{"timeout":30}`, output.Config)
	assert.Equal(t, `{"X-Key":"val"}`, output.Headers)
}

func TestGetMcpUseCase_NotFound(t *testing.T) {
	mcpRepo := newMockMcpServerRepo()
	specMcpRepo := newMockSpecialistMcpRepo()
	uc := NewGetMcpUseCase(mcpRepo, specMcpRepo)

	_, err := uc.Execute(context.Background(), "nonexistent")

	assert.ErrorIs(t, err, domain.ErrMcpNotFound)
}

func TestGetMcpUseCase_WithSpecialistCount(t *testing.T) {
	mcpRepo := newMockMcpServerRepo()
	specMcpRepo := newMockSpecialistMcpRepo()
	uc := NewGetMcpUseCase(mcpRepo, specMcpRepo)

	mcp, _ := domain.NewMcpServer("mcp-1", "Servidor", "https://mcp.example.com", "", "")
	mcpRepo.addMcp(mcp)
	_ = specMcpRepo.Associate(context.Background(), "spec-1", "mcp-1")
	_ = specMcpRepo.Associate(context.Background(), "spec-2", "mcp-1")

	output, err := uc.Execute(context.Background(), "mcp-1")

	require.NoError(t, err)
	assert.Equal(t, 2, output.SpecialistCount)
}
