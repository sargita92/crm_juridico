package application

import (
	"context"
	"fmt"
	"strings"
	"time"

	domain "github.com/sasrgita/crm-juridico/internal/ai/domain"
	specDomain "github.com/sasrgita/crm-juridico/internal/specialist/domain"
	"go.uber.org/zap"
)

// ConfigResolver resolves the AI configuration for a given specialist.
type ConfigResolver interface {
	Resolve(ctx context.Context, specialistID string) *domain.AIConfig
}

// MessageSender sends an AI-generated response back to the conversation.
type MessageSender interface {
	SendAIResponse(ctx context.Context, tenantID, conversationID, content string) error
}

// LeadUpdater updates lead scoring and column position.
type LeadUpdater interface {
	UpdateLeadScore(ctx context.Context, conversationID string, score int) error
	MoveLeadToColumn(ctx context.Context, conversationID, columnID string) error
}

// ConversationEngine orchestrates the AI conversation flow for a lead.
type ConversationEngine struct {
	providerRegistry *domain.ProviderRegistry
	configResolver   ConfigResolver
	stateRepo        domain.ConversationStateRepository
	contextBuilder   *ContextBuilder
	stepEvaluator    *StepEvaluator
	guardrailChecker *GuardrailChecker
	messageSender    MessageSender
	leadUpdater      LeadUpdater
	log              *zap.Logger
}

// NewConversationEngine creates a ConversationEngine with all required dependencies.
func NewConversationEngine(
	providerRegistry *domain.ProviderRegistry,
	configResolver ConfigResolver,
	stateRepo domain.ConversationStateRepository,
	contextBuilder *ContextBuilder,
	stepEvaluator *StepEvaluator,
	guardrailChecker *GuardrailChecker,
	messageSender MessageSender,
	leadUpdater LeadUpdater,
	log *zap.Logger,
) *ConversationEngine {
	return &ConversationEngine{
		providerRegistry: providerRegistry,
		configResolver:   configResolver,
		stateRepo:        stateRepo,
		contextBuilder:   contextBuilder,
		stepEvaluator:    stepEvaluator,
		guardrailChecker: guardrailChecker,
		messageSender:    messageSender,
		leadUpdater:      leadUpdater,
		log:              log,
	}
}

// HandleMessages processes one or more debounced messages for a conversation.
func (e *ConversationEngine) HandleMessages(
	ctx context.Context,
	tenantID, conversationID, specialistID, productID string,
	messages []string,
) error {
	// 1. Get or create ConversationState.
	state, err := e.stateRepo.FindByConversationID(ctx, conversationID)
	if err != nil {
		if err != domain.ErrConversationStateNotFound {
			return fmt.Errorf("conversation_engine: find state: %w", err)
		}
		// Create new state.
		state, err = domain.NewConversationState("", conversationID, specialistID)
		if err != nil {
			return fmt.Errorf("conversation_engine: new state: %w", err)
		}
		if createErr := e.stateRepo.Create(ctx, state); createErr != nil {
			return fmt.Errorf("conversation_engine: create state: %w", createErr)
		}
	}

	// 2. If handoff is active, skip AI processing.
	if state.HandoffActive {
		return nil
	}

	// 3. Resolve config.
	cfg := e.configResolver.Resolve(ctx, specialistID)

	// 4. Build AI request context.
	req, err := e.contextBuilder.Build(ctx, state, productID, 20)
	if err != nil {
		return fmt.Errorf("conversation_engine: build context: %w", err)
	}

	// 5. Set provider/model/temperature/maxTokens from config.
	req.Provider = cfg.Provider
	req.Model = cfg.Model
	req.Temperature = cfg.Temperature
	req.MaxTokens = cfg.MaxTokens

	// 6. Get provider and generate response.
	provider, err := e.providerRegistry.Get(cfg.Provider)
	if err != nil {
		return fmt.Errorf("conversation_engine: get provider: %w", err)
	}

	start := time.Now()
	resp, err := provider.GenerateResponse(ctx, req)
	elapsed := time.Since(start).Seconds()

	status := "success"
	if err != nil {
		status = "error"
		aiRequestsTotal.WithLabelValues(tenantID, specialistID, cfg.Provider, cfg.Model, status).Inc()
		return fmt.Errorf("conversation_engine: generate response: %w", err)
	}

	// 7. Record metrics.
	aiRequestsTotal.WithLabelValues(tenantID, specialistID, cfg.Provider, cfg.Model, status).Inc()
	aiRequestDuration.WithLabelValues(tenantID, cfg.Provider, cfg.Model).Observe(elapsed)
	aiTokensPromptTotal.WithLabelValues(tenantID, specialistID, cfg.Provider).Add(float64(resp.PromptTokens))
	aiTokensCompletionTotal.WithLabelValues(tenantID, specialistID, cfg.Provider).Add(float64(resp.CompletionTokens))

	// 8. Parse metadata from response.
	meta, cleanedContent := e.stepEvaluator.ParseMetadata(resp.Content)

	// 9. Evaluate steps.
	steps, _ := e.contextBuilder.StepFinder.FindBySpecialistID(ctx, specialistID)
	combinedMessage := strings.Join(messages, " ")

	var stepCompleted bool
	var collectedData string
	var stepScore int
	var targetColumnID string

	if len(steps) > 0 && state.CurrentStepIndex < len(steps) {
		currentStep := steps[state.CurrentStepIndex]

		// Try rule-based evaluation first.
		evalResult := e.stepEvaluator.EvaluateByRule(&currentStep, combinedMessage)
		if evalResult.Completed {
			stepCompleted = true
			collectedData = evalResult.CollectedData
			stepScore = currentStep.Score
			targetColumnID = currentStep.TargetColumnID
		} else if meta.StepCompleted {
			// Fallback to LLM metadata.
			stepCompleted = true
			collectedData = meta.CollectedData
			stepScore = meta.Score
			targetColumnID = currentStep.TargetColumnID
		}

		// 10. If step completed → advance step, update lead score, move to column.
		if stepCompleted {
			state.AdvanceStep(fmt.Sprintf("step_%d", state.CurrentStepIndex), collectedData, stepScore)
			aiStepsCompletedTotal.WithLabelValues(tenantID, specialistID).Inc()

			if e.leadUpdater != nil {
				if updateErr := e.leadUpdater.UpdateLeadScore(ctx, conversationID, state.AccumulatedScore); updateErr != nil {
					e.log.Warn("conversation_engine: update lead score failed",
						zap.String("conversation_id", conversationID),
						zap.Error(updateErr),
					)
				}
				if targetColumnID != "" {
					if moveErr := e.leadUpdater.MoveLeadToColumn(ctx, conversationID, targetColumnID); moveErr != nil {
						e.log.Warn("conversation_engine: move lead to column failed",
							zap.String("conversation_id", conversationID),
							zap.String("column_id", targetColumnID),
							zap.Error(moveErr),
						)
					}
				}
			}

			// 11. If all steps completed → log qualification.
			if state.CurrentStepIndex >= len(steps) {
				aiLeadsQualifiedTotal.WithLabelValues(tenantID, specialistID).Inc()
				e.log.Info("conversation_engine: all steps completed, lead qualified",
					zap.String("tenant_id", tenantID),
					zap.String("conversation_id", conversationID),
					zap.String("specialist_id", specialistID),
					zap.Int("accumulated_score", state.AccumulatedScore),
				)
			}
		}
	}

	// 12. Check guardrails.
	responseContent := cleanedContent
	guardrails, _ := e.contextBuilder.GuardrailFinder.FindBySpecialistID(ctx, specialistID)
	if violated, fallback := e.guardrailChecker.Check(responseContent, guardrails); violated {
		aiGuardrailViolationsTotal.WithLabelValues(tenantID, specialistID, string(specDomain.GuardrailTypeForbiddenTopics)).Inc()
		e.log.Warn("conversation_engine: guardrail violated, using fallback message",
			zap.String("tenant_id", tenantID),
			zap.String("conversation_id", conversationID),
		)
		responseContent = fallback
	}

	// 13. Send AI response.
	if sendErr := e.messageSender.SendAIResponse(ctx, tenantID, conversationID, responseContent); sendErr != nil {
		return fmt.Errorf("conversation_engine: send response: %w", sendErr)
	}

	// 14. Persist state.
	if updateErr := e.stateRepo.Update(ctx, state); updateErr != nil {
		return fmt.Errorf("conversation_engine: update state: %w", updateErr)
	}

	return nil
}
