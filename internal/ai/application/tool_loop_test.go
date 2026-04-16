package application

import (
	"context"
	"fmt"
	"testing"

	"github.com/sasrgita/crm-juridico/internal/ai/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// fakeToolProvider simulates an AI provider returning a sequence of responses.
type fakeToolProvider struct {
	callCount int
	responses []*domain.AIResponse
}

func (f *fakeToolProvider) Name() string { return "fake" }
func (f *fakeToolProvider) GenerateResponse(_ context.Context, _ *domain.AIRequest) (*domain.AIResponse, error) {
	if f.callCount >= len(f.responses) {
		return nil, fmt.Errorf("unexpected call %d", f.callCount)
	}
	resp := f.responses[f.callCount]
	f.callCount++
	return resp, nil
}

func TestExecuteToolLoop_NoToolCalls_ReturnsResponse(t *testing.T) {
	provider := &fakeToolProvider{
		responses: []*domain.AIResponse{
			{Content: "Hello!", FinishReason: "stop"},
		},
	}
	registry := NewToolRegistry()

	engine := &ConversationEngine{
		toolRegistry: registry,
		log:          zap.NewNop(),
	}

	req := &domain.AIRequest{Messages: []domain.AIMessage{{Role: domain.RoleUser, Content: "hi"}}}
	resp, err := engine.executeToolLoop(context.Background(), provider, req, "tenant-1", "spec-1", 5, 4000)
	require.NoError(t, err)
	assert.Equal(t, "Hello!", resp.Content)
	assert.Equal(t, 1, provider.callCount)
}

func TestExecuteToolLoop_WithToolCall_ExecutesToolAndContinues(t *testing.T) {
	provider := &fakeToolProvider{
		responses: []*domain.AIResponse{
			{
				FinishReason: "tool_calls",
				ToolCalls: []domain.ToolCall{
					{ID: "call-1", ToolName: "fake_tool", Arguments: map[string]interface{}{"q": "test"}},
				},
			},
			{Content: "Found results!", FinishReason: "stop"},
		},
	}

	registry := NewToolRegistry()
	registry.Register(newFakeTool("fake_tool", domain.ToolCategoryDataQuery))

	engine := &ConversationEngine{
		toolRegistry: registry,
		log:          zap.NewNop(),
	}

	req := &domain.AIRequest{Messages: []domain.AIMessage{{Role: domain.RoleUser, Content: "search"}}}
	resp, err := engine.executeToolLoop(context.Background(), provider, req, "tenant-1", "spec-1", 5, 4000)
	require.NoError(t, err)
	assert.Equal(t, "Found results!", resp.Content)
	assert.Equal(t, 2, provider.callCount)
}

func TestExecuteToolLoop_MaxIterations_ReturnsError(t *testing.T) {
	provider := &fakeToolProvider{
		responses: []*domain.AIResponse{
			{FinishReason: "tool_calls", ToolCalls: []domain.ToolCall{{ID: "c1", ToolName: "fake_tool"}}},
			{FinishReason: "tool_calls", ToolCalls: []domain.ToolCall{{ID: "c2", ToolName: "fake_tool"}}},
			{FinishReason: "tool_calls", ToolCalls: []domain.ToolCall{{ID: "c3", ToolName: "fake_tool"}}},
		},
	}

	registry := NewToolRegistry()
	registry.Register(newFakeTool("fake_tool", domain.ToolCategoryDataQuery))

	engine := &ConversationEngine{
		toolRegistry: registry,
		log:          zap.NewNop(),
	}

	req := &domain.AIRequest{Messages: []domain.AIMessage{{Role: domain.RoleUser, Content: "loop"}}}
	_, err := engine.executeToolLoop(context.Background(), provider, req, "tenant-1", "spec-1", 3, 4000)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "max iterations")
}

func TestExecuteToolLoop_ToolNotFound_ReturnsErrorResultToLLM(t *testing.T) {
	provider := &fakeToolProvider{
		responses: []*domain.AIResponse{
			{
				FinishReason: "tool_calls",
				ToolCalls:    []domain.ToolCall{{ID: "c1", ToolName: "nonexistent"}},
			},
			{Content: "OK I couldn't find that tool", FinishReason: "stop"},
		},
	}

	registry := NewToolRegistry()

	engine := &ConversationEngine{
		toolRegistry: registry,
		log:          zap.NewNop(),
	}

	req := &domain.AIRequest{Messages: []domain.AIMessage{{Role: domain.RoleUser, Content: "test"}}}
	resp, err := engine.executeToolLoop(context.Background(), provider, req, "tenant-1", "spec-1", 5, 4000)
	require.NoError(t, err)
	assert.Equal(t, "OK I couldn't find that tool", resp.Content)
	// Verify the error was injected into req.ToolResults
	require.Len(t, req.ToolResults, 1)
	assert.True(t, req.ToolResults[0].IsError)
	assert.Contains(t, req.ToolResults[0].Content, "tool not found")
}

func TestExecuteToolLoop_ResultExceedsMaxLength_Truncates(t *testing.T) {
	// Make a tool whose Execute returns a long result
	longResult := make([]byte, 5000)
	for i := range longResult {
		longResult[i] = 'a'
	}

	longTool := &fakeToolWithContent{
		def:     mustNewToolDef("long_tool", domain.ToolCategoryDataQuery),
		content: string(longResult),
	}

	provider := &fakeToolProvider{
		responses: []*domain.AIResponse{
			{FinishReason: "tool_calls", ToolCalls: []domain.ToolCall{{ID: "c1", ToolName: "long_tool"}}},
			{Content: "done", FinishReason: "stop"},
		},
	}

	registry := NewToolRegistry()
	registry.Register(longTool)

	engine := &ConversationEngine{
		toolRegistry: registry,
		log:          zap.NewNop(),
	}

	req := &domain.AIRequest{Messages: []domain.AIMessage{{Role: domain.RoleUser, Content: "test"}}}
	_, err := engine.executeToolLoop(context.Background(), provider, req, "tenant-1", "spec-1", 5, 4000)
	require.NoError(t, err)
	require.Len(t, req.ToolResults, 1)
	assert.Len(t, req.ToolResults[0].Content, 4000)
}

// --- test helpers (fakeToolWithContent for truncation test) ---
type fakeToolWithContent struct {
	def     domain.ToolDefinition
	content string
}

func (f *fakeToolWithContent) Definition() domain.ToolDefinition { return f.def }
func (f *fakeToolWithContent) Execute(_ context.Context, _ string, _ map[string]interface{}) (*domain.ToolResult, error) {
	r := domain.NewToolResult("", f.content, false)
	return &r, nil
}

func mustNewToolDef(name string, cat domain.ToolCategory) domain.ToolDefinition {
	def, _ := domain.NewToolDefinition(name, "desc for "+name, cat, nil)
	return def
}
