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
	MessageText    string
}

type CreateLeadUseCase struct {
	funnelRepo          domain.FunnelRepository
	columnRepo          domain.ColumnRepository
	leadRepo            domain.LeadRepository
	movementRepo        domain.LeadMovementRepository
	productDetector     domain.ProductDetector
	funnelProductRouter domain.FunnelProductRouter
}

func NewCreateLeadUseCase(
	funnelRepo domain.FunnelRepository,
	columnRepo domain.ColumnRepository,
	leadRepo domain.LeadRepository,
	movementRepo domain.LeadMovementRepository,
	productDetector domain.ProductDetector,
	funnelProductRouter domain.FunnelProductRouter,
) *CreateLeadUseCase {
	return &CreateLeadUseCase{
		funnelRepo:          funnelRepo,
		columnRepo:          columnRepo,
		leadRepo:            leadRepo,
		movementRepo:        movementRepo,
		productDetector:     productDetector,
		funnelProductRouter: funnelProductRouter,
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

	// Product detection: try to detect a product from the message text
	var detectedProductID string
	if uc.productDetector != nil && input.MessageText != "" {
		productID, found, detectErr := uc.productDetector.DetectFromMessage(ctx, input.TenantID, input.MessageText)
		if detectErr == nil && found {
			detectedProductID = productID
		}
	}

	// Funnel routing: if a product was detected, try to find the best funnel for it
	var funnel *domain.Funnel
	if detectedProductID != "" && uc.funnelProductRouter != nil {
		funnelID, routeErr := uc.funnelProductRouter.FindTopPriorityFunnelID(ctx, detectedProductID)
		if routeErr == nil && funnelID != "" {
			if candidate, findErr := uc.funnelRepo.FindByID(ctx, funnelID); findErr == nil && candidate.Active {
				funnel = candidate
			}
		}
	}

	// Fall back to default funnel if no product-routed funnel was found
	if funnel == nil {
		funnel, err = uc.funnelRepo.FindDefaultByTenantID(ctx, input.TenantID)
		if err != nil {
			return err
		}
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
	if detectedProductID != "" {
		lead.SetProduct(detectedProductID)
	}
	if err := uc.leadRepo.Create(ctx, lead); err != nil {
		return err
	}

	// Create initial movement
	movement := domain.NewLeadMovement(uuid.New().String(), lead.ID, "", entryCol.ID)
	return uc.movementRepo.Create(ctx, movement)
}

// CreateFromConversation implements the whatsapp domain.LeadCreator interface.
func (uc *CreateLeadUseCase) CreateFromConversation(ctx context.Context, tenantID, contactID, conversationID, messageText string) error {
	return uc.Execute(ctx, CreateLeadInput{
		TenantID:       tenantID,
		ContactID:      contactID,
		ConversationID: conversationID,
		MessageText:    messageText,
	})
}
