package application

import (
	"context"
	"time"

	"github.com/sasrgita/crm-juridico/internal/funnel/domain"
)

type GetLeadDetailInput struct {
	TenantID string
	LeadID   string
}

type LeadMovementOutput struct {
	ID           string
	FromColumnID string
	ToColumnID   string
	MovedAt      time.Time
}

type LeadDetailOutput struct {
	ID              string
	TenantID        string
	FunnelID        string
	ColumnID        string
	ContactID       string
	ConversationID  string
	Score           int
	Status          string
	ColumnEnteredAt time.Time
	CreatedAt       time.Time
	Movements       []LeadMovementOutput
}

type GetLeadDetailUseCase struct {
	leadRepo     domain.LeadRepository
	movementRepo domain.LeadMovementRepository
}

func NewGetLeadDetailUseCase(leadRepo domain.LeadRepository, movementRepo domain.LeadMovementRepository) *GetLeadDetailUseCase {
	return &GetLeadDetailUseCase{leadRepo: leadRepo, movementRepo: movementRepo}
}

func (uc *GetLeadDetailUseCase) Execute(ctx context.Context, input GetLeadDetailInput) (*LeadDetailOutput, error) {
	lead, err := uc.leadRepo.FindByID(ctx, input.LeadID)
	if err != nil {
		return nil, err
	}
	if lead.TenantID != input.TenantID {
		return nil, domain.ErrLeadNotFound
	}

	movements, err := uc.movementRepo.FindByLeadID(ctx, lead.ID)
	if err != nil {
		return nil, err
	}

	mvOutputs := make([]LeadMovementOutput, len(movements))
	for i, mv := range movements {
		mvOutputs[i] = LeadMovementOutput{
			ID:           mv.ID,
			FromColumnID: mv.FromColumnID,
			ToColumnID:   mv.ToColumnID,
			MovedAt:      mv.MovedAt,
		}
	}

	return &LeadDetailOutput{
		ID:              lead.ID,
		TenantID:        lead.TenantID,
		FunnelID:        lead.FunnelID,
		ColumnID:        lead.ColumnID,
		ContactID:       lead.ContactID,
		ConversationID:  lead.ConversationID,
		Score:           lead.Score,
		Status:          string(lead.Status),
		ColumnEnteredAt: lead.ColumnEnteredAt,
		CreatedAt:       lead.CreatedAt,
		Movements:       mvOutputs,
	}, nil
}
