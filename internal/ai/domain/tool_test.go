package domain

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewToolDefinition_WithValidData_ReturnsDefinition(t *testing.T) {
	td, err := NewToolDefinition("search_leads", "Search leads by query", ToolCategoryDataQuery, map[string]ParameterDef{
		"query": {Type: "string", Description: "Search term", Required: true},
	})
	require.NoError(t, err)
	assert.Equal(t, "search_leads", td.Name)
	assert.Equal(t, ToolCategoryDataQuery, td.Category)
	assert.Len(t, td.Parameters, 1)
}

func TestNewToolDefinition_EmptyName_ReturnsError(t *testing.T) {
	_, err := NewToolDefinition("", "desc", ToolCategoryDataQuery, nil)
	assert.ErrorIs(t, err, ErrToolNameRequired)
}

func TestNewToolDefinition_EmptyDescription_ReturnsError(t *testing.T) {
	_, err := NewToolDefinition("name", "", ToolCategoryDataQuery, nil)
	assert.ErrorIs(t, err, ErrToolDescriptionRequired)
}

func TestNewToolDefinition_InvalidCategory_ReturnsError(t *testing.T) {
	_, err := NewToolDefinition("name", "desc", ToolCategory("invalid"), nil)
	assert.ErrorIs(t, err, ErrToolCategoryInvalid)
}

func TestNewToolDefinition_NilParams_ReturnsDefinitionWithEmptyMap(t *testing.T) {
	td, err := NewToolDefinition("tool", "desc", ToolCategoryDataQuery, nil)
	require.NoError(t, err)
	assert.NotNil(t, td.Parameters)
	assert.Empty(t, td.Parameters)
}

func TestNewToolResult_WithValidData_ReturnsResult(t *testing.T) {
	r := NewToolResult("call-1", "result text", false)
	assert.Equal(t, "call-1", r.ToolCallID)
	assert.Equal(t, "result text", r.Content)
	assert.False(t, r.IsError)
}

func TestNewToolResult_WithIsError_ReturnsErrorResult(t *testing.T) {
	r := NewToolResult("call-1", "something failed", true)
	assert.True(t, r.IsError)
}

func TestNewToolResultWithLimit_ContentExceedsMax_TruncatesContent(t *testing.T) {
	long := make([]byte, 5000)
	for i := range long {
		long[i] = 'a'
	}
	r := NewToolResultWithLimit("call-1", string(long), false, 4000)
	assert.Len(t, r.Content, 4000)
}
