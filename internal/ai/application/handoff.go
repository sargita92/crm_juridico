package application

import (
	"context"
	"errors"
	"fmt"

	domain "github.com/sasrgita/crm-juridico/internal/ai/domain"
	"go.uber.org/zap"
)

var (
	// ErrHandoffAlreadyActive is returned when trying to activate a handoff that is already active.
	ErrHandoffAlreadyActive = errors.New("handoff already active")
	// ErrHandoffNotActive is returned when trying to deactivate a handoff that is not active.
	ErrHandoffNotActive = errors.New("handoff not active")
)

// ActivateHandoffUseCase activates a human-agent handoff for a conversation.
type ActivateHandoffUseCase struct {
	stateRepo domain.ConversationStateRepository
	log       *zap.Logger
}

// NewActivateHandoffUseCase creates a new ActivateHandoffUseCase.
func NewActivateHandoffUseCase(stateRepo domain.ConversationStateRepository, log *zap.Logger) *ActivateHandoffUseCase {
	return &ActivateHandoffUseCase{stateRepo: stateRepo, log: log}
}

// Execute activates the handoff for the given conversation.
func (uc *ActivateHandoffUseCase) Execute(ctx context.Context, tenantID, conversationID string) error {
	state, err := uc.stateRepo.FindByConversationID(ctx, conversationID)
	if err != nil {
		return fmt.Errorf("activate_handoff: find state: %w", err)
	}

	if state.HandoffActive {
		return ErrHandoffAlreadyActive
	}

	state.ActivateHandoff()

	if err := uc.stateRepo.Update(ctx, state); err != nil {
		return fmt.Errorf("activate_handoff: update state: %w", err)
	}

	aiHandoffsTotal.WithLabelValues(tenantID, "to_human").Inc()

	uc.log.Info("handoff activated",
		zap.String("tenant_id", tenantID),
		zap.String("conversation_id", conversationID),
	)

	return nil
}

// DeactivateHandoffUseCase deactivates an active human-agent handoff, resuming AI.
type DeactivateHandoffUseCase struct {
	stateRepo domain.ConversationStateRepository
	log       *zap.Logger
}

// NewDeactivateHandoffUseCase creates a new DeactivateHandoffUseCase.
func NewDeactivateHandoffUseCase(stateRepo domain.ConversationStateRepository, log *zap.Logger) *DeactivateHandoffUseCase {
	return &DeactivateHandoffUseCase{stateRepo: stateRepo, log: log}
}

// Execute deactivates the handoff for the given conversation.
func (uc *DeactivateHandoffUseCase) Execute(ctx context.Context, tenantID, conversationID string) error {
	state, err := uc.stateRepo.FindByConversationID(ctx, conversationID)
	if err != nil {
		return fmt.Errorf("deactivate_handoff: find state: %w", err)
	}

	if !state.HandoffActive {
		return ErrHandoffNotActive
	}

	state.DeactivateHandoff()

	if err := uc.stateRepo.Update(ctx, state); err != nil {
		return fmt.Errorf("deactivate_handoff: update state: %w", err)
	}

	aiHandoffsTotal.WithLabelValues(tenantID, "to_ai").Inc()

	uc.log.Info("handoff deactivated",
		zap.String("tenant_id", tenantID),
		zap.String("conversation_id", conversationID),
	)

	return nil
}
