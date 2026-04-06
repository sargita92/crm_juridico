package domain

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewMcpServer_WithValidData_ReturnsServer(t *testing.T) {
	m, err := NewMcpServer("uuid-1", "API Processos", "https://api.processos.com/mcp", `{"timeout": 30}`, `{"Authorization": "Bearer token"}`)

	require.NoError(t, err)
	assert.Equal(t, "uuid-1", m.ID)
	assert.Equal(t, "API Processos", m.Name)
	assert.Equal(t, "https://api.processos.com/mcp", m.URL)
	assert.Equal(t, `{"timeout": 30}`, m.Config)
	assert.Equal(t, `{"Authorization": "Bearer token"}`, m.Headers)
	assert.False(t, m.CreatedAt.IsZero())
	assert.False(t, m.UpdatedAt.IsZero())
}

func TestNewMcpServer_EmptyName_ReturnsError(t *testing.T) {
	_, err := NewMcpServer("uuid-1", "", "https://example.com", "", "")
	assert.ErrorIs(t, err, ErrMcpNameRequired)
}

func TestNewMcpServer_NameTooLong_ReturnsError(t *testing.T) {
	longName := strings.Repeat("a", MaxMcpNameLength+1)
	_, err := NewMcpServer("uuid-1", longName, "https://example.com", "", "")
	assert.ErrorIs(t, err, ErrMcpNameTooLong)
}

func TestNewMcpServer_NameAtMaxLength_ReturnsServer(t *testing.T) {
	name := strings.Repeat("a", MaxMcpNameLength)
	m, err := NewMcpServer("uuid-1", name, "https://example.com", "", "")
	require.NoError(t, err)
	assert.Equal(t, name, m.Name)
}

func TestNewMcpServer_EmptyURL_ReturnsError(t *testing.T) {
	_, err := NewMcpServer("uuid-1", "MCP", "", "", "")
	assert.ErrorIs(t, err, ErrMcpURLRequired)
}

func TestNewMcpServer_InvalidURL_ReturnsError(t *testing.T) {
	_, err := NewMcpServer("uuid-1", "MCP", "nao-e-url", "", "")
	assert.ErrorIs(t, err, ErrMcpURLInvalid)
}

func TestNewMcpServer_ValidURLSchemes(t *testing.T) {
	for _, url := range []string{"https://example.com", "http://localhost:8080"} {
		t.Run(url, func(t *testing.T) {
			m, err := NewMcpServer("uuid-1", "MCP", url, "", "")
			require.NoError(t, err)
			assert.Equal(t, url, m.URL)
		})
	}
}

func TestNewMcpServer_EmptyConfigAndHeaders_ReturnsServer(t *testing.T) {
	m, err := NewMcpServer("uuid-1", "MCP", "https://example.com", "", "")
	require.NoError(t, err)
	assert.Empty(t, m.Config)
	assert.Empty(t, m.Headers)
}

func TestNewMcpServer_ConfigTooLong_ReturnsError(t *testing.T) {
	longConfig := strings.Repeat("a", MaxMcpConfigLength+1)
	_, err := NewMcpServer("uuid-1", "MCP", "https://example.com", longConfig, "")
	assert.ErrorIs(t, err, ErrMcpConfigTooLong)
}

func TestNewMcpServer_HeadersTooLong_ReturnsError(t *testing.T) {
	longHeaders := strings.Repeat("a", MaxMcpHeadersLength+1)
	_, err := NewMcpServer("uuid-1", "MCP", "https://example.com", "", longHeaders)
	assert.ErrorIs(t, err, ErrMcpHeadersTooLong)
}

func TestMcpServer_Update_Success(t *testing.T) {
	m, _ := NewMcpServer("uuid-1", "Antigo", "https://old.com", "", "")

	err := m.Update("Novo", "https://new.com", `{"key": "value"}`, `{"Auth": "Bearer x"}`)

	require.NoError(t, err)
	assert.Equal(t, "Novo", m.Name)
	assert.Equal(t, "https://new.com", m.URL)
	assert.Equal(t, `{"key": "value"}`, m.Config)
	assert.Equal(t, `{"Auth": "Bearer x"}`, m.Headers)
}

func TestMcpServer_Update_EmptyName_ReturnsError(t *testing.T) {
	m, _ := NewMcpServer("uuid-1", "MCP", "https://example.com", "", "")
	err := m.Update("", "https://example.com", "", "")
	assert.ErrorIs(t, err, ErrMcpNameRequired)
	assert.Equal(t, "MCP", m.Name)
}

func TestMcpServer_Update_EmptyURL_ReturnsError(t *testing.T) {
	m, _ := NewMcpServer("uuid-1", "MCP", "https://example.com", "", "")
	err := m.Update("MCP", "", "", "")
	assert.ErrorIs(t, err, ErrMcpURLRequired)
}

func TestMcpServer_Update_InvalidURL_ReturnsError(t *testing.T) {
	m, _ := NewMcpServer("uuid-1", "MCP", "https://example.com", "", "")
	err := m.Update("MCP", "invalido", "", "")
	assert.ErrorIs(t, err, ErrMcpURLInvalid)
}

func TestIsValidURL_ValidURLs(t *testing.T) {
	assert.True(t, IsValidURL("https://example.com"))
	assert.True(t, IsValidURL("http://localhost:8080"))
	assert.True(t, IsValidURL("https://api.example.com/v1/mcp"))
}

func TestIsValidURL_InvalidURLs(t *testing.T) {
	assert.False(t, IsValidURL(""))
	assert.False(t, IsValidURL("nao-e-url"))
	assert.False(t, IsValidURL("ftp://example.com"))
	assert.False(t, IsValidURL("javascript:alert(1)"))
}
