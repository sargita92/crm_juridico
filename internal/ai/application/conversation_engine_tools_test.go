package application

import (
	"context"
	"testing"

	"github.com/sasrgita/crm-juridico/internal/ai/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// fakeProviderWithToolReplay records a snapshot of each AIRequest at call time
// so tests can assert that ToolResults are correctly threaded back to the provider.
type fakeProviderWithToolReplay struct {
	calls     []*domain.AIRequest // snapshots of each req at call time
	responses []*domain.AIResponse
	callIdx   int
}

func (f *fakeProviderWithToolReplay) Name() string { return "fake-replay" }
func (f *fakeProviderWithToolReplay) GenerateResponse(_ context.Context, req *domain.AIRequest) (*domain.AIResponse, error) {
	// Deep-copy ToolResults so subsequent mutations to req don't affect our snapshot.
	snapshot := &domain.AIRequest{
		SystemPrompt: req.SystemPrompt,
		Messages:     append([]domain.AIMessage(nil), req.Messages...),
		Provider:     req.Provider,
		Model:        req.Model,
		Temperature:  req.Temperature,
		MaxTokens:    req.MaxTokens,
		Tools:        append([]domain.ToolDefinition(nil), req.Tools...),
	}
	if req.ToolResults != nil {
		snapshot.ToolResults = append([]domain.ToolResult(nil), req.ToolResults...)
	}
	f.calls = append(f.calls, snapshot)

	if f.callIdx >= len(f.responses) {
		return nil, assert.AnError
	}
	resp := f.responses[f.callIdx]
	f.callIdx++
	return resp, nil
}

// searchLeadsEchoTool echoes the query back so tests can assert content propagation.
type searchLeadsEchoTool struct{}

func (s *searchLeadsEchoTool) Definition() domain.ToolDefinition {
	def, _ := domain.NewToolDefinition(
		"search_leads",
		"Search for leads by query",
		domain.ToolCategoryDataQuery,
		map[string]domain.ParameterDef{
			"query": {Type: "string", Required: true, Description: "Search query"},
		},
	)
	return def
}

func (s *searchLeadsEchoTool) Execute(_ context.Context, _ string, args map[string]interface{}) (*domain.ToolResult, error) {
	query, _ := args["query"].(string)
	r := domain.NewToolResult("", "Lead found: "+query, false)
	return &r, nil
}

// getLeadDetailTool returns canned detail for a lead ID.
type getLeadDetailTool struct{}

func (g *getLeadDetailTool) Definition() domain.ToolDefinition {
	def, _ := domain.NewToolDefinition(
		"get_lead_detail",
		"Get detail for a lead by ID",
		domain.ToolCategoryDataQuery,
		map[string]domain.ParameterDef{
			"lead_id": {Type: "string", Required: true, Description: "Lead ID"},
		},
	)
	return def
}

func (g *getLeadDetailTool) Execute(_ context.Context, _ string, args map[string]interface{}) (*domain.ToolResult, error) {
	leadID, _ := args["lead_id"].(string)
	r := domain.NewToolResult("", "Detail for lead "+leadID+": score=85, status=qualified", false)
	return &r, nil
}

// TestExecuteToolLoop_EndToEnd_PassesToolResultBackToProvider verifies that:
//   - The tool is executed with the correct arguments
//   - The tool result is threaded back to the provider in the second call
//   - The final response is returned correctly
func TestExecuteToolLoop_EndToEnd_PassesToolResultBackToProvider(t *testing.T) {
	provider := &fakeProviderWithToolReplay{
		responses: []*domain.AIResponse{
			// First call: request a tool invocation
			{
				FinishReason: "tool_calls",
				ToolCalls: []domain.ToolCall{
					{ID: "call-1", ToolName: "search_leads", Arguments: map[string]interface{}{"query": "Joao"}},
				},
			},
			// Second call: text response referencing the tool result
			{Content: "Found Joao's lead successfully", FinishReason: "stop"},
		},
	}

	registry := NewToolRegistry()
	registry.Register(&searchLeadsEchoTool{})

	engine := &ConversationEngine{
		toolRegistry: registry,
		log:          zap.NewNop(),
	}

	req := &domain.AIRequest{
		Messages: []domain.AIMessage{{Role: domain.RoleUser, Content: "Find Joao"}},
	}

	resp, err := engine.executeToolLoop(context.Background(), provider, req, "tenant-1", "spec-1", 5, 4000)
	require.NoError(t, err)
	require.NotNil(t, resp)

	assert.Equal(t, "Found Joao's lead successfully", resp.Content)
	assert.Equal(t, 2, provider.callIdx, "provider should have been called exactly 2 times")

	// Verify the first call had no tool results (clean start).
	require.Len(t, provider.calls, 2)
	firstCall := provider.calls[0]
	assert.Empty(t, firstCall.ToolResults, "first call should have no tool results")

	// Verify the second call received the tool result from the first iteration.
	secondCall := provider.calls[1]
	require.Len(t, secondCall.ToolResults, 1, "second call must include the tool result")
	assert.Equal(t, "call-1", secondCall.ToolResults[0].ToolCallID)
	assert.Contains(t, secondCall.ToolResults[0].Content, "Joao", "tool result content must contain echo of query")
	assert.False(t, secondCall.ToolResults[0].IsError)
}

// TestExecuteToolLoop_TwoToolsInSequence verifies that when two different tools
// are called in successive iterations, each iteration's results are passed to the
// next provider call — and only the most recent results are in ToolResults.
func TestExecuteToolLoop_TwoToolsInSequence(t *testing.T) {
	provider := &fakeProviderWithToolReplay{
		responses: []*domain.AIResponse{
			// First call: request search_leads
			{
				FinishReason: "tool_calls",
				ToolCalls: []domain.ToolCall{
					{ID: "call-1", ToolName: "search_leads", Arguments: map[string]interface{}{"query": "Silva"}},
				},
			},
			// Second call: request get_lead_detail (uses search result)
			{
				FinishReason: "tool_calls",
				ToolCalls: []domain.ToolCall{
					{ID: "call-2", ToolName: "get_lead_detail", Arguments: map[string]interface{}{"lead_id": "lead-42"}},
				},
			},
			// Third call: final text response
			{Content: "Lead Silva has score 85, status qualified", FinishReason: "stop"},
		},
	}

	registry := NewToolRegistry()
	registry.Register(&searchLeadsEchoTool{})
	registry.Register(&getLeadDetailTool{})

	engine := &ConversationEngine{
		toolRegistry: registry,
		log:          zap.NewNop(),
	}

	req := &domain.AIRequest{
		Messages: []domain.AIMessage{{Role: domain.RoleUser, Content: "Tell me about Silva"}},
	}

	resp, err := engine.executeToolLoop(context.Background(), provider, req, "tenant-1", "spec-1", 5, 4000)
	require.NoError(t, err)
	require.NotNil(t, resp)

	assert.Equal(t, "Lead Silva has score 85, status qualified", resp.Content)
	assert.Equal(t, 3, provider.callIdx, "provider should have been called exactly 3 times")
	require.Len(t, provider.calls, 3)

	// First call: no tool results yet.
	assert.Empty(t, provider.calls[0].ToolResults)

	// Second call: result from search_leads (call-1).
	secondCallResults := provider.calls[1].ToolResults
	require.Len(t, secondCallResults, 1)
	assert.Equal(t, "call-1", secondCallResults[0].ToolCallID)
	assert.Contains(t, secondCallResults[0].Content, "Silva")
	assert.False(t, secondCallResults[0].IsError)

	// Third call: result from get_lead_detail (call-2).
	thirdCallResults := provider.calls[2].ToolResults
	require.Len(t, thirdCallResults, 1)
	assert.Equal(t, "call-2", thirdCallResults[0].ToolCallID)
	assert.Contains(t, thirdCallResults[0].Content, "lead-42")
	assert.Contains(t, thirdCallResults[0].Content, "score=85")
	assert.False(t, thirdCallResults[0].IsError)
}

// TestExecuteToolLoop_NoToolsPath_SingleIteration verifies that a provider
// returning a plain text response (no tool calls) exits in exactly one iteration.
func TestExecuteToolLoop_NoToolsPath_SingleIteration(t *testing.T) {
	provider := &fakeProviderWithToolReplay{
		responses: []*domain.AIResponse{
			{Content: "I can help you with that.", FinishReason: "stop"},
		},
	}

	registry := NewToolRegistry() // empty registry

	engine := &ConversationEngine{
		toolRegistry: registry,
		log:          zap.NewNop(),
	}

	req := &domain.AIRequest{
		Messages: []domain.AIMessage{{Role: domain.RoleUser, Content: "Hello"}},
	}

	resp, err := engine.executeToolLoop(context.Background(), provider, req, "tenant-1", "spec-1", 5, 4000)
	require.NoError(t, err)
	require.NotNil(t, resp)

	assert.Equal(t, "I can help you with that.", resp.Content)
	assert.Equal(t, 1, provider.callIdx, "loop must exit after a single iteration when there are no tool calls")
	assert.Empty(t, req.ToolResults, "no tool results should be set when no tools were called")
}
