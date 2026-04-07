package domain

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- AIMessage tests ---

func TestNewAIMessage_WithValidData_ReturnsMessage(t *testing.T) {
	msg, err := NewAIMessage(RoleUser, "Hello")

	require.NoError(t, err)
	assert.Equal(t, RoleUser, msg.Role)
	assert.Equal(t, "Hello", msg.Content)
}

func TestNewAIMessage_SystemRole_ReturnsMessage(t *testing.T) {
	msg, err := NewAIMessage(RoleSystem, "You are an assistant.")

	require.NoError(t, err)
	assert.Equal(t, RoleSystem, msg.Role)
}

func TestNewAIMessage_AssistantRole_ReturnsMessage(t *testing.T) {
	msg, err := NewAIMessage(RoleAssistant, "I can help you.")

	require.NoError(t, err)
	assert.Equal(t, RoleAssistant, msg.Role)
}

func TestNewAIMessage_EmptyContent_ReturnsError(t *testing.T) {
	_, err := NewAIMessage(RoleUser, "")

	assert.ErrorIs(t, err, ErrMessageContentEmpty)
}

func TestNewAIMessage_InvalidRole_ReturnsError(t *testing.T) {
	_, err := NewAIMessage(MessageRole("invalid"), "Hello")

	assert.ErrorIs(t, err, ErrMessageRoleInvalid)
}

// --- AIRequest tests ---

func TestNewAIRequest_WithValidData_ReturnsRequest(t *testing.T) {
	msg, _ := NewAIMessage(RoleUser, "Hello")
	req, err := NewAIRequest("You are helpful", []AIMessage{msg}, "openai", "gpt-4", 0.7, 1000)

	require.NoError(t, err)
	assert.Equal(t, "You are helpful", req.SystemPrompt)
	assert.Len(t, req.Messages, 1)
	assert.Equal(t, "openai", req.Provider)
	assert.Equal(t, "gpt-4", req.Model)
	assert.Equal(t, 0.7, req.Temperature)
	assert.Equal(t, 1000, req.MaxTokens)
}

func TestNewAIRequest_EmptyMessages_ReturnsError(t *testing.T) {
	_, err := NewAIRequest("sys", []AIMessage{}, "openai", "gpt-4", 0.7, 1000)

	assert.ErrorIs(t, err, ErrMessagesEmpty)
}

func TestNewAIRequest_EmptyModel_ReturnsError(t *testing.T) {
	msg, _ := NewAIMessage(RoleUser, "Hello")
	_, err := NewAIRequest("sys", []AIMessage{msg}, "openai", "", 0.7, 1000)

	assert.ErrorIs(t, err, ErrModelRequired)
}

func TestNewAIRequest_TemperatureAboveOne_ReturnsError(t *testing.T) {
	msg, _ := NewAIMessage(RoleUser, "Hello")
	_, err := NewAIRequest("sys", []AIMessage{msg}, "openai", "gpt-4", 1.5, 1000)

	assert.ErrorIs(t, err, ErrTemperatureInvalid)
}

func TestNewAIRequest_TemperatureBelowZero_ReturnsError(t *testing.T) {
	msg, _ := NewAIMessage(RoleUser, "Hello")
	_, err := NewAIRequest("sys", []AIMessage{msg}, "openai", "gpt-4", -0.1, 1000)

	assert.ErrorIs(t, err, ErrTemperatureInvalid)
}

func TestNewAIRequest_TemperatureAtBoundaries_ReturnsRequest(t *testing.T) {
	msg, _ := NewAIMessage(RoleUser, "Hello")

	req0, err0 := NewAIRequest("sys", []AIMessage{msg}, "openai", "gpt-4", 0.0, 1000)
	require.NoError(t, err0)
	assert.Equal(t, 0.0, req0.Temperature)

	req1, err1 := NewAIRequest("sys", []AIMessage{msg}, "openai", "gpt-4", 1.0, 1000)
	require.NoError(t, err1)
	assert.Equal(t, 1.0, req1.Temperature)
}

// --- ProviderRegistry tests ---

type mockProvider struct {
	name string
}

func (m *mockProvider) GenerateResponse(_ context.Context, _ *AIRequest) (*AIResponse, error) {
	return &AIResponse{Content: "mocked"}, nil
}

func (m *mockProvider) Name() string {
	return m.name
}

func TestProviderRegistry_RegisterAndGet_ReturnsProvider(t *testing.T) {
	registry := NewProviderRegistry()
	provider := &mockProvider{name: "openai"}

	registry.Register(provider)
	got, err := registry.Get("openai")

	require.NoError(t, err)
	assert.Equal(t, "openai", got.Name())
}

func TestProviderRegistry_GetNotFound_ReturnsError(t *testing.T) {
	registry := NewProviderRegistry()

	_, err := registry.Get("nonexistent")

	assert.ErrorIs(t, err, ErrProviderNotFound)
}

func TestProviderRegistry_RegisterMultiple_ReturnsCorrectProvider(t *testing.T) {
	registry := NewProviderRegistry()
	registry.Register(&mockProvider{name: "openai"})
	registry.Register(&mockProvider{name: "anthropic"})

	got, err := registry.Get("anthropic")

	require.NoError(t, err)
	assert.Equal(t, "anthropic", got.Name())
}
