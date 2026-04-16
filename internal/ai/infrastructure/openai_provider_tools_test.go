package infrastructure

import (
	"testing"

	"github.com/sasrgita/crm-juridico/internal/ai/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuildOpenAITools_WithParameters_ReturnsCorrectStructure(t *testing.T) {
	p := &OpenAIProvider{}
	td, _ := domain.NewToolDefinition("search_leads", "Search leads", domain.ToolCategoryDataQuery, map[string]domain.ParameterDef{
		"query":  {Type: "string", Description: "Search term", Required: true},
		"status": {Type: "string", Description: "Lead status", Required: false, Enum: []string{"open", "won", "lost"}},
	})

	tools := p.buildOpenAITools([]domain.ToolDefinition{td})

	require.Len(t, tools, 1)
	assert.Equal(t, "function", tools[0].Type)
	assert.Equal(t, "search_leads", tools[0].Function.Name)
	assert.Equal(t, "Search leads", tools[0].Function.Description)

	params := tools[0].Function.Parameters
	assert.Equal(t, "object", params["type"])
	props := params["properties"].(map[string]interface{})
	assert.Contains(t, props, "query")
	assert.Contains(t, props, "status")

	statusProp := props["status"].(map[string]interface{})
	assert.Equal(t, []string{"open", "won", "lost"}, statusProp["enum"])

	required := params["required"].([]string)
	assert.Contains(t, required, "query")
	assert.NotContains(t, required, "status")
}

func TestBuildOpenAITools_EmptyTools_ReturnsNil(t *testing.T) {
	p := &OpenAIProvider{}
	tools := p.buildOpenAITools(nil)
	assert.Nil(t, tools)
}

func TestParseToolCalls_ReturnsToolCalls(t *testing.T) {
	p := &OpenAIProvider{}
	choices := []openAIChoice{
		{
			Message: openAIMessage{
				Role:    "assistant",
				Content: "",
				ToolCalls: []openAIToolCall{
					{
						ID:   "call_123",
						Type: "function",
						Function: openAIFunctionCall{
							Name:      "search_leads",
							Arguments: `{"query":"test","status":"open"}`,
						},
					},
				},
			},
			FinishReason: "tool_calls",
		},
	}

	calls := p.parseToolCalls(choices)
	require.Len(t, calls, 1)
	assert.Equal(t, "call_123", calls[0].ID)
	assert.Equal(t, "search_leads", calls[0].ToolName)
	assert.Equal(t, "test", calls[0].Arguments["query"])
	assert.Equal(t, "open", calls[0].Arguments["status"])
}

func TestParseToolCalls_NoToolCalls_ReturnsNil(t *testing.T) {
	p := &OpenAIProvider{}
	choices := []openAIChoice{
		{
			Message:      openAIMessage{Role: "assistant", Content: "hi"},
			FinishReason: "stop",
		},
	}

	calls := p.parseToolCalls(choices)
	assert.Nil(t, calls)
}

func TestBuildToolResultMessages_ReturnsToolMessages(t *testing.T) {
	p := &OpenAIProvider{}
	results := []domain.ToolResult{
		{ToolCallID: "call_123", Content: "found 3 leads", IsError: false},
	}

	msgs := p.buildToolResultMessages(results)
	require.Len(t, msgs, 1)
	assert.Equal(t, "tool", msgs[0].Role)
	assert.Equal(t, "call_123", msgs[0].ToolCallID)
	assert.Equal(t, "found 3 leads", msgs[0].Content)
}
