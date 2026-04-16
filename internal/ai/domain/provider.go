package domain

import (
	"context"
	"errors"
)

// MessageRole represents the role of a message in a conversation.
type MessageRole string

const (
	RoleSystem    MessageRole = "system"
	RoleUser      MessageRole = "user"
	RoleAssistant MessageRole = "assistant"
)

var (
	ErrMessageContentEmpty  = errors.New("message content is required")
	ErrMessageRoleInvalid   = errors.New("message role is invalid")
	ErrMessagesEmpty        = errors.New("messages cannot be empty")
	ErrModelRequired        = errors.New("model is required")
	ErrTemperatureInvalid   = errors.New("temperature must be between 0 and 1")
	ErrProviderNotFound     = errors.New("AI provider not found")
)

// AIMessage represents a single message in an AI conversation.
type AIMessage struct {
	Role    MessageRole
	Content string
}

// NewAIMessage constructs an AIMessage with validation.
func NewAIMessage(role MessageRole, content string) (AIMessage, error) {
	if role != RoleSystem && role != RoleUser && role != RoleAssistant {
		return AIMessage{}, ErrMessageRoleInvalid
	}
	if content == "" {
		return AIMessage{}, ErrMessageContentEmpty
	}
	return AIMessage{Role: role, Content: content}, nil
}

// AIRequest represents a request sent to an AI provider.
type AIRequest struct {
	SystemPrompt string
	Messages     []AIMessage
	Provider     string
	Model        string
	Temperature  float64
	MaxTokens    int
	Tools        []ToolDefinition // optional: tools the LLM may call
	ToolResults  []ToolResult     // optional: results from previous tool calls
}

// NewAIRequest constructs an AIRequest with validation.
func NewAIRequest(systemPrompt string, messages []AIMessage, provider, model string, temperature float64, maxTokens int) (*AIRequest, error) {
	if len(messages) == 0 {
		return nil, ErrMessagesEmpty
	}
	if model == "" {
		return nil, ErrModelRequired
	}
	if temperature < 0 || temperature > 1 {
		return nil, ErrTemperatureInvalid
	}
	return &AIRequest{
		SystemPrompt: systemPrompt,
		Messages:     messages,
		Provider:     provider,
		Model:        model,
		Temperature:  temperature,
		MaxTokens:    maxTokens,
	}, nil
}

// AIResponse represents the response from an AI provider.
type AIResponse struct {
	Content          string
	FinishReason     string
	PromptTokens     int
	CompletionTokens int
	ToolCalls        []ToolCall // populated when the LLM requests tool invocations
}

// AIProvider is the interface that AI provider implementations must satisfy.
type AIProvider interface {
	GenerateResponse(ctx context.Context, req *AIRequest) (*AIResponse, error)
	Name() string
}

// ProviderRegistry holds registered AI providers.
type ProviderRegistry struct {
	providers map[string]AIProvider
}

// NewProviderRegistry creates an empty ProviderRegistry.
func NewProviderRegistry() *ProviderRegistry {
	return &ProviderRegistry{
		providers: make(map[string]AIProvider),
	}
}

// Register adds an AIProvider to the registry.
func (r *ProviderRegistry) Register(p AIProvider) {
	r.providers[p.Name()] = p
}

// Get retrieves an AIProvider by name, returning ErrProviderNotFound if absent.
func (r *ProviderRegistry) Get(name string) (AIProvider, error) {
	p, ok := r.providers[name]
	if !ok {
		return nil, ErrProviderNotFound
	}
	return p, nil
}
