package application

import (
	"context"
	"errors"

	"github.com/google/uuid"

	"github.com/sasrgita/crm-juridico/internal/funnel/domain"
)

type CreateLeadInput struct {
	TenantID       string
	ContactID      string
	ConversationID string
}

type CreateLeadUseCase struct {
	funnelRepo   domain.FunnelRepository
	columnRepo   domain.ColumnRepository
	leadRepo     domain.LeadRepository
	movementRepo domain.LeadMovementRepository
}

func NewCreateLeadUseCase(
	funnelRepo domain.FunnelRepository,
	columnRepo domain.ColumnRepository,
	leadRepo domain.LeadRepository,
	movementRepo domain.LeadMovementRepository,
) *CreateLeadUseCase {
	return &CreateLeadUseCase{
		funnelRepo:   funnelRepo,
		columnRepo:   columnRepo,
		leadRepo:     leadRepo,
		movementRepo: movementRepo,
	}
}

func (uc *CreateLeadUseCase) Execute(ctx context.Context, input CreateLeadInput) error {
	// Check if lead already exists for this contact+tenant
	_, err := uc.leadRepo.FindByContactAndTenant(ctx, input.TenantID, input.ContactID)
	if err == nil {
		return nil // lead already exists, nothing to do
	}
	if !errors.Is(err, domain.ErrLeadNotFound) {
		return err
	}

	// Find default funnel
	funnel, err := uc.funnelRepo.FindDefaultByTenantID(ctx, input.TenantID)
	if err != nil {
		return err
	}

	// Find entry column
	entryCol, err := uc.columnRepo.FindEntryByFunnelID(ctx, funnel.ID)
	if err != nil {
		return err
	}

	// Create lead
	lead, err := domain.NewLead(uuid.New().String(), input.TenantID, funnel.ID, entryCol.ID, input.ContactID, input.ConversationID)
	if err != nil {
		return err
	}
	if err := uc.leadRepo.Create(ctx, lead); err != nil {
		return err
	}

	// Create initial movement
	movement := domain.NewLeadMovement(uuid.New().String(), lead.ID, "", entryCol.ID)
	return uc.movementRepo.Create(ctx, movement)
}

// CreateFromConversation implements the whatsapp domain.LeadCreator interface.
func (uc *CreateLeadUseCase) CreateFromConversation(ctx context.Context, tenantID, contactID, conversationID string) error {
	return uc.Execute(ctx, CreateLeadInput{
		TenantID:       tenantID,
		ContactID:      contactID,
		ConversationID: conversationID,
	})
}
