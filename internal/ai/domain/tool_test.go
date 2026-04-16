package domain

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewToolDefinition_Valid(t *testing.T) {
	td, err := NewToolDefinition("search_leads", "Search leads by query", ToolCategoryDataQuery, map[string]ParameterDef{
		"query": {Type: "string", Description: "Search term", Required: true},
	})
	require.NoError(t, err)
	assert.Equal(t, "search_leads", td.Name)
	assert.Equal(t, ToolCategoryDataQuery, td.Category)
	assert.Len(t, td.Parameters, 1)
}

func TestNewToolDefinition_EmptyName(t *testing.T) {
	_, err := NewToolDefinition("", "desc", ToolCategoryDataQuery, nil)
	assert.ErrorIs(t, err, ErrToolNameRequired)
}

func TestNewToolDefinition_EmptyDescription(t *testing.T) {
	_, err := NewToolDefinition("name", "", ToolCategoryDataQuery, nil)
	assert.ErrorIs(t, err, ErrToolDescriptionRequired)
}

func TestNewToolDefinition_InvalidCategory(t *testing.T) {
	_, err := NewToolDefinition("name", "desc", ToolCategory("invalid"), nil)
	assert.ErrorIs(t, err, ErrToolCategoryInvalid)
}

func TestNewToolResult_Valid(t *testing.T) {
	r := NewToolResult("call-1", "result text", false)
	assert.Equal(t, "call-1", r.ToolCallID)
	assert.Equal(t, "result text", r.Content)
	assert.False(t, r.IsError)
}

func TestNewToolResult_Error(t *testing.T) {
	r := NewToolResult("call-1", "something failed", true)
	assert.True(t, r.IsError)
}

func TestNewToolResult_Truncate(t *testing.T) {
	long := make([]byte, 5000)
	for i := range long {
		long[i] = 'a'
	}
	r := NewToolResultWithLimit("call-1", string(long), false, 4000)
	assert.Len(t, r.Content, 4000)
}
