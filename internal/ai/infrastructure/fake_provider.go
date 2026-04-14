package infrastructure

import (
	"context"
	"errors"

	"github.com/sasrgita/crm-juridico/internal/ai/domain"
)

// FakeProvider is a deterministic AIProvider used in dev and integration tests.
// It produces short canned responses. Step evaluation in the engine is
// rule-based, so the LLM reply quality does not affect flow progression.
type FakeProvider struct{}

func NewFakeProvider() *FakeProvider { return &FakeProvider{} }

func (p *FakeProvider) Name() string { return "fake" }

func (p *FakeProvider) GenerateResponse(ctx context.Context, req *domain.AIRequest) (*domain.AIResponse, error) {
	if req == nil || len(req.Messages) == 0 {
		return nil, errors.New("fake_provider: empty request")
	}
	last := req.Messages[len(req.Messages)-1].Content
	reply := "Entendi: \"" + last + "\". Pode me contar mais?"
	return &domain.AIResponse{
		Content:          reply,
		FinishReason:     "stop",
		PromptTokens:     len(req.SystemPrompt) + len(last),
		CompletionTokens: len(reply),
	}, nil
}
