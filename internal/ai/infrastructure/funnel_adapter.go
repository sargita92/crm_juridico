package infrastructure

import (
	"context"
	"fmt"

	funnelApp "github.com/sasrgita/crm-juridico/internal/funnel/application"
	funnelDomain "github.com/sasrgita/crm-juridico/internal/funnel/domain"
)

// LeadUpdaterAdapter satisfies application.LeadUpdater.
type LeadUpdaterAdapter struct {
	leadRepo   funnelDomain.LeadRepository
	moveLeadUC *funnelApp.MoveLeadUseCase
}

func NewLeadUpdaterAdapter(
	leadRepo funnelDomain.LeadRepository,
	moveLeadUC *funnelApp.MoveLeadUseCase,
) *LeadUpdaterAdapter {
	return &LeadUpdaterAdapter{
		leadRepo:   leadRepo,
		moveLeadUC: moveLeadUC,
	}
}

// UpdateLeadScore looks up the lead by conversation ID and updates its score.
func (a *LeadUpdaterAdapter) UpdateLeadScore(ctx context.Context, conversationID string, score int) error {
	lead, err := a.leadRepo.FindByConversationID(ctx, conversationID)
	if err != nil {
		return fmt.Errorf("lead_updater_adapter: find lead: %w", err)
	}
	lead.UpdateScore(score)
	if err := a.leadRepo.Update(ctx, lead); err != nil {
		return fmt.Errorf("lead_updater_adapter: update lead score: %w", err)
	}
	return nil
}

// MoveLeadToColumn looks up the lead by conversation ID and moves it to the
// specified column using the MoveLeadUseCase.
func (a *LeadUpdaterAdapter) MoveLeadToColumn(ctx context.Context, conversationID, columnID string) error {
	lead, err := a.leadRepo.FindByConversationID(ctx, conversationID)
	if err != nil {
		return fmt.Errorf("lead_updater_adapter: find lead: %w", err)
	}
	if err := a.moveLeadUC.Execute(ctx, funnelApp.MoveLeadInput{
		TenantID: lead.TenantID,
		LeadID:   lead.ID,
		ColumnID: columnID,
	}); err != nil {
		return fmt.Errorf("lead_updater_adapter: move lead: %w", err)
	}
	return nil
}

// SetOutcome looks up the lead by conversation ID and persists the qualification outcome.
func (a *LeadUpdaterAdapter) SetOutcome(ctx context.Context, conversationID string, outcome string) error {
	lead, err := a.leadRepo.FindByConversationID(ctx, conversationID)
	if err != nil {
		return fmt.Errorf("lead_updater_adapter: find lead: %w", err)
	}
	lead.SetQualificationOutcome(funnelDomain.QualificationOutcome(outcome))
	if err := a.leadRepo.Update(ctx, lead); err != nil {
		return fmt.Errorf("lead_updater_adapter: set outcome: %w", err)
	}
	return nil
}

// GetLeadIDByConversation resolves the lead.ID for a given conversation ID.
// Required to pass the correct originLeadID to CrossSellExecutor.
func (a *LeadUpdaterAdapter) GetLeadIDByConversation(ctx context.Context, conversationID string) (string, error) {
	lead, err := a.leadRepo.FindByConversationID(ctx, conversationID)
	if err != nil {
		return "", fmt.Errorf("lead_updater_adapter: find lead for id lookup: %w", err)
	}
	return lead.ID, nil
}
