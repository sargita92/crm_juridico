package application

import (
	"context"
	"testing"

	"github.com/sasrgita/crm-juridico/internal/ai/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fakeTool implements domain.Tool for testing.
type fakeTool struct {
	def domain.ToolDefinition
}

func (f *fakeTool) Definition() domain.ToolDefinition { return f.def }
func (f *fakeTool) Execute(_ context.Context, _ string, _ map[string]interface{}) (*domain.ToolResult, error) {
	r := domain.NewToolResult("", "ok", false)
	return &r, nil
}

func newFakeTool(name string, cat domain.ToolCategory) *fakeTool {
	def, _ := domain.NewToolDefinition(name, "desc for "+name, cat, nil)
	return &fakeTool{def: def}
}

func TestToolRegistry_RegisterAndGet_ReturnsTool(t *testing.T) {
	reg := NewToolRegistry()
	tool := newFakeTool("search_leads", domain.ToolCategoryDataQuery)

	reg.Register(tool)

	got, err := reg.Get("search_leads")
	require.NoError(t, err)
	assert.Equal(t, "search_leads", got.Definition().Name)
}

func TestToolRegistry_Get_NotFound_ReturnsError(t *testing.T) {
	reg := NewToolRegistry()
	_, err := reg.Get("nonexistent")
	assert.ErrorIs(t, err, domain.ErrToolNotFound)
}

func TestToolRegistry_All_ReturnsAllTools(t *testing.T) {
	reg := NewToolRegistry()
	reg.Register(newFakeTool("a", domain.ToolCategoryDataQuery))
	reg.Register(newFakeTool("b", domain.ToolCategoryCRMAction))

	all := reg.All()
	assert.Len(t, all, 2)
}

func TestToolRegistry_Definitions_ReturnsAllDefinitions(t *testing.T) {
	reg := NewToolRegistry()
	reg.Register(newFakeTool("a", domain.ToolCategoryDataQuery))
	reg.Register(newFakeTool("b", domain.ToolCategoryCRMAction))

	defs := reg.Definitions()
	assert.Len(t, defs, 2)
}
