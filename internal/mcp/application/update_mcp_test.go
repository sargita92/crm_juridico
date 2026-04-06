package application

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/sasrgita/crm-juridico/internal/mcp/domain"
)

func TestUpdateMcpUseCase_Success(t *testing.T) {
	repo := newMockMcpServerRepo()
	uc := NewUpdateMcpUseCase(repo)

	mcp, _ := domain.NewMcpServer("mcp-1", "Nome Original", "https://old.example.com", "", "")
	repo.addMcp(mcp)

	err := uc.Execute(context.Background(), UpdateMcpInput{
		ID:      "mcp-1",
		Name:    "Nome Atualizado",
		URL:     "https://new.example.com",
		Config:  `{"timeout":60}`,
		Headers: `{"X-Key":"new"}`,
	})

	require.NoError(t, err)
	updated := repo.mcps["mcp-1"]
	assert.Equal(t, "Nome Atualizado", updated.Name)
	assert.Equal(t, "https://new.example.com", updated.URL)
	assert.Equal(t, `{"timeout":60}`, updated.Config)
	assert.Equal(t, `{"X-Key":"new"}`, updated.Headers)
}

func TestUpdateMcpUseCase_NotFound(t *testing.T) {
	repo := newMockMcpServerRepo()
	uc := NewUpdateMcpUseCase(repo)

	err := uc.Execute(context.Background(), UpdateMcpInput{
		ID:   "nonexistent",
		Name: "Nome",
		URL:  "https://mcp.example.com",
	})

	assert.ErrorIs(t, err, domain.ErrMcpNotFound)
}

func TestUpdateMcpUseCase_EmptyName_ReturnsError(t *testing.T) {
	repo := newMockMcpServerRepo()
	uc := NewUpdateMcpUseCase(repo)

	mcp, _ := domain.NewMcpServer("mcp-1", "Nome Original", "https://mcp.example.com", "", "")
	repo.addMcp(mcp)

	err := uc.Execute(context.Background(), UpdateMcpInput{
		ID:   "mcp-1",
		Name: "",
		URL:  "https://mcp.example.com",
	})

	assert.ErrorIs(t, err, domain.ErrMcpNameRequired)
}

func TestUpdateMcpUseCase_InvalidURL_ReturnsError(t *testing.T) {
	repo := newMockMcpServerRepo()
	uc := NewUpdateMcpUseCase(repo)

	mcp, _ := domain.NewMcpServer("mcp-1", "Nome Original", "https://mcp.example.com", "", "")
	repo.addMcp(mcp)

	err := uc.Execute(context.Background(), UpdateMcpInput{
		ID:   "mcp-1",
		Name: "Nome",
		URL:  "not-valid",
	})

	assert.ErrorIs(t, err, domain.ErrMcpURLInvalid)
}
