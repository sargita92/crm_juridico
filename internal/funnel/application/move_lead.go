package application

import (
	"context"

	"github.com/google/uuid"

	"github.com/sasrgita/crm-juridico/internal/funnel/domain"
	"github.com/sasrgita/crm-juridico/internal/shared/events"
)

type MoveLeadInput struct {
	TenantID string
	LeadID   string
	ColumnID string
	FunnelID string // optional: if different from current funnel
}

type MoveLeadUseCase struct {
	funnelRepo   domain.FunnelRepository
	columnRepo   domain.ColumnRepository
	leadRepo     domain.LeadRepository
	movementRepo domain.LeadMovementRepository
	eventBus     events.EventBus
}

func NewMoveLeadUseCase(
	funnelRepo domain.FunnelRepository,
	columnRepo domain.ColumnRepository,
	leadRepo domain.LeadRepository,
	movementRepo domain.LeadMovementRepository,
	eventBus events.EventBus,
) *MoveLeadUseCase {
	return &MoveLeadUseCase{
		funnelRepo:   funnelRepo,
		columnRepo:   columnRepo,
		leadRepo:     leadRepo,
		movementRepo: movementRepo,
		eventBus:     eventBus,
	}
}

func (uc *MoveLeadUseCase) Execute(ctx context.Context, input MoveLeadInput) error {
	lead, err := uc.leadRepo.FindByID(ctx, input.LeadID)
	if err != nil {
		return err
	}
	if lead.TenantID != input.TenantID {
		return domain.ErrLeadNotFound
	}

	col, err := uc.columnRepo.FindByID(ctx, input.ColumnID)
	if err != nil {
		return err
	}

	fromColumnID := lead.ColumnID

	// If moving to a different funnel
	if input.FunnelID != "" && input.FunnelID != lead.FunnelID {
		funnel, err := uc.funnelRepo.FindByID(ctx, input.FunnelID)
		if err != nil {
			return err
		}
		if funnel.TenantID != input.TenantID {
			return domain.ErrFunnelNotFound
		}
		lead.FunnelID = input.FunnelID
	}

	lead.MoveTo(col.ID, col.Type)

	if err := uc.leadRepo.Update(ctx, lead); err != nil {
		return err
	}

	movement := domain.NewLeadMovement(uuid.New().String(), lead.ID, fromColumnID, col.ID)
	if err := uc.movementRepo.Create(ctx, movement); err != nil {
		return err
	}

	if uc.eventBus != nil {
		uc.eventBus.Publish(events.Event{
			Type:     events.EventLeadMoved,
			TenantID: lead.TenantID,
			Payload: map[string]string{
				"lead_id":        lead.ID,
				"funnel_id":      lead.FunnelID,
				"from_column_id": fromColumnID,
				"to_column_id":   input.ColumnID,
			},
		})
	}

	return nil
}
