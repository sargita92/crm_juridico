package application

import (
	"context"
	"fmt"
	"strings"
	"time"

	domain "github.com/sasrgita/crm-juridico/internal/ai/domain"
	specDomain "github.com/sasrgita/crm-juridico/internal/specialist/domain"
)

// SpecialistFinder retrieves a specialist by its ID.
type SpecialistFinder interface {
	FindByID(ctx context.Context, id string) (*specDomain.Specialist, error)
}

// StepFinder retrieves the steps configured for a specialist.
type StepFinder interface {
	FindBySpecialistID(ctx context.Context, id string) ([]specDomain.Step, error)
}

// GuardrailFinder retrieves the guardrails configured for a specialist.
type GuardrailFinder interface {
	FindBySpecialistID(ctx context.Context, id string) ([]specDomain.Guardrail, error)
}

// DocumentFetcher retrieves the contents of reference documents for a specialist.
type DocumentFetcher interface {
	FetchDocumentContents(ctx context.Context, specialistID string) ([]string, error)
}

// ProductInfoFinder retrieves human-readable product information.
type ProductInfoFinder interface {
	FindProductInfo(ctx context.Context, productID string) (name, description string, err error)
}

// HistoryMessage represents a single message from the conversation history.
type HistoryMessage struct {
	Role      domain.MessageRole
	Content   string
	Timestamp time.Time
}

// MessageHistoryFinder retrieves recent conversation history.
type MessageHistoryFinder interface {
	FindHistory(ctx context.Context, conversationID string, limit int) ([]HistoryMessage, error)
}

// ContextBuilder assembles a complete AIRequest from a ConversationState.
type ContextBuilder struct {
	SpecialistFinder     SpecialistFinder
	StepFinder           StepFinder
	GuardrailFinder      GuardrailFinder
	DocumentFetcher      DocumentFetcher
	ProductInfoFinder    ProductInfoFinder
	MessageHistoryFinder MessageHistoryFinder
	ToolResolver         *ToolResolver
}

// NewContextBuilder creates a ContextBuilder with all required dependencies.
func NewContextBuilder(
	specialistFinder SpecialistFinder,
	stepFinder StepFinder,
	guardrailFinder GuardrailFinder,
	documentFetcher DocumentFetcher,
	productInfoFinder ProductInfoFinder,
	messageHistoryFinder MessageHistoryFinder,
	toolResolver *ToolResolver,
) *ContextBuilder {
	return &ContextBuilder{
		SpecialistFinder:     specialistFinder,
		StepFinder:           stepFinder,
		GuardrailFinder:      guardrailFinder,
		DocumentFetcher:      documentFetcher,
		ProductInfoFinder:    productInfoFinder,
		MessageHistoryFinder: messageHistoryFinder,
		ToolResolver:         toolResolver,
	}
}

// Build constructs an AIRequest for the given conversation state.
func (b *ContextBuilder) Build(ctx context.Context, state *domain.ConversationState, productID string, historyLimit int) (*domain.AIRequest, error) {
	// 1. Fetch specialist and build base prompt.
	specialist, err := b.SpecialistFinder.FindByID(ctx, state.SpecialistID)
	if err != nil {
		return nil, fmt.Errorf("context_builder: fetch specialist: %w", err)
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Voce e %s. %s", specialist.Name, specialist.Prompt))

	// 2. Product information.
	if productID != "" {
		name, desc, pErr := b.ProductInfoFinder.FindProductInfo(ctx, productID)
		if pErr == nil && name != "" {
			sb.WriteString(fmt.Sprintf("\n\n## Produto\nO cliente esta interessado em: %s. %s", name, desc))
		}
	}

	// 3. Reference documents.
	docs, _ := b.DocumentFetcher.FetchDocumentContents(ctx, state.SpecialistID)
	if len(docs) > 0 {
		sb.WriteString("\n\n## Documentos de referencia")
		for _, doc := range docs {
			sb.WriteString("\n")
			sb.WriteString(doc)
		}
	}

	// 4. Active guardrails.
	guardrails, _ := b.GuardrailFinder.FindBySpecialistID(ctx, state.SpecialistID)
	var activeGuardrails []specDomain.Guardrail
	for _, g := range guardrails {
		if g.Active {
			activeGuardrails = append(activeGuardrails, g)
		}
	}
	if len(activeGuardrails) > 0 {
		sb.WriteString("\n\n## Regras obrigatorias")
		for _, g := range activeGuardrails {
			sb.WriteString(fmt.Sprintf("\n- Nunca fale sobre: %s", g.Rule))
		}
	}

	// 5 & 6. Current step.
	steps, _ := b.StepFinder.FindBySpecialistID(ctx, state.SpecialistID)
	if len(steps) > 0 && state.CurrentStepIndex < len(steps) {
		currentStep := steps[state.CurrentStepIndex]
		sb.WriteString(fmt.Sprintf("\n\n## Etapa atual do atendimento\n%s", currentStep.Text))

		// 6. Instruction for LLM to emit STEP_META for free_text steps.
		if currentStep.DataType == specDomain.StepDataTypeFreeText {
			sb.WriteString("\n\nAo finalizar esta etapa, inclua no final da sua resposta o seguinte comentario (sem exibi-lo ao usuario):\n<!--STEP_META:{\"step_completed\": true, \"collected_data\": \"<dado coletado>\", \"score\": " + fmt.Sprintf("%d", currentStep.Score) + "}-->")
		}
	}

	systemPrompt := sb.String()

	// 7. Message history.
	history, _ := b.MessageHistoryFinder.FindHistory(ctx, state.ConversationID, historyLimit)

	var messages []domain.AIMessage
	for _, h := range history {
		if !state.IsAfterResume(h.Timestamp) {
			continue
		}
		msg, mErr := domain.NewAIMessage(h.Role, h.Content)
		if mErr != nil {
			continue
		}
		messages = append(messages, msg)
	}

	// 8. Fallback message if history is empty.
	if len(messages) == 0 {
		fallback, _ := domain.NewAIMessage(domain.RoleUser, "oi")
		messages = append(messages, fallback)
	}

	// 9. Resolve tools for this specialist and current step (if resolver is available).
	var toolDefs []domain.ToolDefinition
	if b.ToolResolver != nil {
		var currentStep *specDomain.Step
		if len(steps) > 0 && state.CurrentStepIndex < len(steps) {
			currentStep = &steps[state.CurrentStepIndex]
		}
		toolDefs, _ = b.ToolResolver.ResolveDefinitions(ctx, state.SpecialistID, currentStep)
	}

	return &domain.AIRequest{
		SystemPrompt: systemPrompt,
		Messages:     messages,
		Tools:        toolDefs,
	}, nil
}
