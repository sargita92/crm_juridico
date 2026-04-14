package application

import (
	"context"
	"errors"
	"fmt"

	domain "github.com/sasrgita/crm-juridico/internal/ai/domain"
	"go.uber.org/zap"
)

// EntryColumnFinder returns the entry-column id for the tenant's default funnel.
// Returns empty string if the tenant has no default funnel or the funnel has
// no entry column — the reset proceeds anyway, only the lead move is skipped.
type EntryColumnFinder interface {
	FindEntryColumnID(ctx context.Context, tenantID string) (string, error)
}

// LeadResetter finds a lead by conversation id and moves it back to a column,
// zeroing its score. Implementations wrap the funnel repositories.
type LeadResetter interface {
	ResetLeadForConversation(ctx context.Context, conversationID, entryColumnID string) error
}

// ResetConversationMessage is the confirmation sent back after reset.
const ResetConversationMessage = "Conversa reiniciada. Pode começar de novo quando quiser."

// ResetConversationUseCase zeroes the conversation state and moves the lead
// back to the entry column of the tenant's default funnel.
type ResetConversationUseCase struct {
	stateRepo   domain.ConversationStateRepository
	entryFinder EntryColumnFinder
	leadReset   LeadResetter
	sender      MessageSender
	log         *zap.Logger
}

func NewResetConversationUseCase(
	stateRepo domain.ConversationStateRepository,
	entryFinder EntryColumnFinder,
	leadReset LeadResetter,
	sender MessageSender,
	log *zap.Logger,
) *ResetConversationUseCase {
	return &ResetConversationUseCase{
		stateRepo:   stateRepo,
		entryFinder: entryFinder,
		leadReset:   leadReset,
		sender:      sender,
		log:         log,
	}
}

// Execute resets the conversation. source is either "command" or "playground"
// and is used as a metric label.
func (uc *ResetConversationUseCase) Execute(ctx context.Context, tenantID, conversationID, source string) error {
	state, err := uc.stateRepo.FindByConversationID(ctx, conversationID)
	if err != nil {
		if errors.Is(err, domain.ErrConversationStateNotFound) {
			// Nothing to zero, but still emit confirmation + metric.
			return uc.finish(ctx, tenantID, conversationID, "", source)
		}
		return fmt.Errorf("reset_conversation: find state: %w", err)
	}

	specialistID := state.SpecialistID
	state.Reset()
	if err := uc.stateRepo.Update(ctx, state); err != nil {
		return fmt.Errorf("reset_conversation: update state: %w", err)
	}

	return uc.finish(ctx, tenantID, conversationID, specialistID, source)
}

func (uc *ResetConversationUseCase) finish(ctx context.Context, tenantID, conversationID, specialistID, source string) error {
	entryColID, err := uc.entryFinder.FindEntryColumnID(ctx, tenantID)
	if err != nil {
		uc.log.Warn("reset_conversation: find entry column failed",
			zap.String("tenant_id", tenantID),
			zap.Error(err),
		)
	}
	if entryColID != "" {
		if resErr := uc.leadReset.ResetLeadForConversation(ctx, conversationID, entryColID); resErr != nil {
			uc.log.Warn("reset_conversation: lead reset failed",
				zap.String("conversation_id", conversationID),
				zap.Error(resErr),
			)
		}
	}

	if sendErr := uc.sender.SendAIResponse(ctx, tenantID, conversationID, ResetConversationMessage); sendErr != nil {
		uc.log.Warn("reset_conversation: confirmation send failed",
			zap.String("conversation_id", conversationID),
			zap.Error(sendErr),
		)
	}

	aiResetCommandsTotal.WithLabelValues(tenantID, specialistID, source).Inc()

	uc.log.Info("reset_conversation: executed",
		zap.String("tenant_id", tenantID),
		zap.String("conversation_id", conversationID),
		zap.String("source", source),
	)
	return nil
}
