package application

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	domain "github.com/sasrgita/crm-juridico/internal/ai/domain"
	specDomain "github.com/sasrgita/crm-juridico/internal/specialist/domain"
)

func buildEngineWithProvider(t *testing.T, provider domain.AIProvider) (*ConversationEngine, *mockMessageSender) {
	t.Helper()

	registry := domain.NewProviderRegistry()
	registry.Register(provider)
	cfg, _ := domain.NewAIConfig("cfg-1", "spec-1", "openai", "gpt-4", 0.7, 1024, 0)
	specialist, _ := specDomain.NewSpecialist("spec-1", "Ana", "Advogada", "Voce e uma advogada.")
	state, _ := domain.NewConversationState("s-1", "conv-1", "spec-1")
	sender := &mockMessageSender{}

	contextBuilder := NewContextBuilder(
		&mockSpecialistFinder{specialist: specialist},
		&mockStepFinder{},
		&mockGuardrailFinder{},
		&mockDocumentFetcher{},
		&mockProductInfoFinder{},
		&mockMessageHistoryFinder{},
		nil,
	)

	engine := NewConversationEngine(
		registry, &mockConfigResolver{cfg: cfg}, &mockConvStateRepo{state: state}, contextBuilder,
		NewStepEvaluator(), NewGuardrailChecker(), sender, &mockLeadUpdater{},
		nil, false, nil, 0, 5, nil, nil,
		nil, nil, nil,
		zap.NewNop(),
	)

	return engine, sender
}

// A provider timeout or outage used to end the turn with a log line and nothing
// else: from the client's side on WhatsApp the specialist simply went quiet
// mid-conversation. Say something instead of vanishing.
func TestConversationEngine_SendsFallbackWhenProviderFails(t *testing.T) {
	engine, sender := buildEngineWithProvider(t, &mockAIProvider{
		name: "openai",
		err:  errors.New("context deadline exceeded"),
	})

	err := engine.HandleMessages(context.Background(), "tenant-1", "conv-1", "spec-1", "", []string{"sim, tenho"})

	require.Error(t, err, "the failure must still surface to logs and metrics")
	assert.True(t, sender.sent, "client was left in silence after the provider failed")
	assert.Equal(t, aiUnavailableFallbackMessage, sender.content)
}

// The fallback must not fire on a healthy turn.
func TestConversationEngine_NoFallbackOnSuccess(t *testing.T) {
	engine, sender := buildEngineWithProvider(t, &mockAIProvider{
		name: "openai",
		resp: &domain.AIResponse{Content: "Perfeito, vou registrar."},
	})

	require.NoError(t, engine.HandleMessages(context.Background(), "tenant-1", "conv-1", "spec-1", "", []string{"sim, tenho"}))
	assert.Equal(t, "Perfeito, vou registrar.", sender.content)
}
