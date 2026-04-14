package infrastructure

import (
	"context"
	"testing"

	"github.com/sasrgita/crm-juridico/internal/ai/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFakeProvider_NameIsFake(t *testing.T) {
	p := NewFakeProvider()
	assert.Equal(t, "fake", p.Name())
}

func TestFakeProvider_GenerateResponse_Deterministic(t *testing.T) {
	p := NewFakeProvider()
	msg, err := domain.NewAIMessage(domain.RoleUser, "oi")
	require.NoError(t, err)
	req := &domain.AIRequest{
		SystemPrompt: "sys",
		Messages:     []domain.AIMessage{msg},
		Provider:     "fake",
		Model:        "fake-v1",
		Temperature:  0.5,
		MaxTokens:    128,
	}
	resp, err := p.GenerateResponse(context.Background(), req)
	require.NoError(t, err)
	assert.NotEmpty(t, resp.Content)
	assert.Equal(t, "stop", resp.FinishReason)
	assert.Greater(t, resp.PromptTokens, 0)
	assert.Greater(t, resp.CompletionTokens, 0)
}

func TestFakeProvider_GenerateResponse_EmptyMessages(t *testing.T) {
	p := NewFakeProvider()
	_, err := p.GenerateResponse(context.Background(), &domain.AIRequest{})
	assert.Error(t, err)
}
