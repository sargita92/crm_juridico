package application

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/sasrgita/crm-juridico/internal/mcp/domain"
)

func TestCreateMcpUseCase_Success(t *testing.T) {
	repo := newMockMcpServerRepo()
	uc := NewCreateMcpUseCase(repo)

	output, err := uc.Execute(context.Background(), CreateMcpInput{
		Name:    "Servidor MCP Jurídico",
		URL:     "https://mcp.example.com",
		Config:  `{"timeout": 30}`,
		Headers: `{"Authorization": "Bearer token"}`,
	})

	require.NoError(t, err)
	assert.NotEmpty(t, output.ID)
	assert.Equal(t, "Servidor MCP Jurídico", output.Name)
	assert.Equal(t, "https://mcp.example.com", output.URL)
	assert.Len(t, repo.mcps, 1)
}

func TestCreateMcpUseCase_EmptyName_ReturnsError(t *testing.T) {
	repo := newMockMcpServerRepo()
	uc := NewCreateMcpUseCase(repo)

	_, err := uc.Execute(context.Background(), CreateMcpInput{
		Name: "",
		URL:  "https://mcp.example.com",
	})

	assert.ErrorIs(t, err, domain.ErrMcpNameRequired)
}

func TestCreateMcpUseCase_EmptyURL_ReturnsError(t *testing.T) {
	repo := newMockMcpServerRepo()
	uc := NewCreateMcpUseCase(repo)

	_, err := uc.Execute(context.Background(), CreateMcpInput{
		Name: "Servidor MCP",
		URL:  "",
	})

	assert.ErrorIs(t, err, domain.ErrMcpURLRequired)
}

func TestCreateMcpUseCase_InvalidURL_ReturnsError(t *testing.T) {
	repo := newMockMcpServerRepo()
	uc := NewCreateMcpUseCase(repo)

	_, err := uc.Execute(context.Background(), CreateMcpInput{
		Name: "Servidor MCP",
		URL:  "not-a-valid-url",
	})

	assert.ErrorIs(t, err, domain.ErrMcpURLInvalid)
}
