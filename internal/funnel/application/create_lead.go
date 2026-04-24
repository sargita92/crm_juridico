package application

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel/attribute"

	"github.com/sasrgita/crm-juridico/internal/funnel/domain"
	"github.com/sasrgita/crm-juridico/internal/shared/events"
	"github.com/sasrgita/crm-juridico/internal/shared/observability"
)

// ErrPickerNotConfigured indicates the ResponsiblePicker was never wired into
// the use case. This is a deployment/wiring bug — production flows must always
// have a picker. Returning it here instead of panicking gives callers a
// chance to surface the misconfiguration as a structured error.
var ErrPickerNotConfigured = errors.New("responsible picker not configured")

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
	eventBus            events.EventBus
	automationTrigger   domain.AutomationTrigger
	picker              domain.ResponsiblePicker
}

func NewCreateLeadUseCase(
	funnelRepo domain.FunnelRepository,
	columnRepo domain.ColumnRepository,
	leadRepo domain.LeadRepository,
	movementRepo domain.LeadMovementRepository,
	productDetector domain.ProductDetector,
	funnelProductRouter domain.FunnelProductRouter,
	eventBus events.EventBus,
	picker domain.ResponsiblePicker,
) *CreateLeadUseCase {
	return &CreateLeadUseCase{
		funnelRepo:          funnelRepo,
		columnRepo:          columnRepo,
		leadRepo:            leadRepo,
		movementRepo:        movementRepo,
		productDetector:     productDetector,
		funnelProductRouter: funnelProductRouter,
		eventBus:            eventBus,
		picker:              picker,
	}
}

// SetAutomationTrigger sets an optional automation trigger that will be called
// after a lead is created successfully.
func (uc *CreateLeadUseCase) SetAutomationTrigger(t domain.AutomationTrigger) {
	uc.automationTrigger = t
}

// SetPicker injects the ResponsiblePicker. Used for late-binding when the
// picker depends on cross-module repositories that are constructed after the
// funnel module (see main.go module wiring order).
func (uc *CreateLeadUseCase) SetPicker(p domain.ResponsiblePicker) {
	uc.picker = p
}

func (uc *CreateLeadUseCase) Execute(ctx context.Context, input CreateLeadInput) error {
	ctx, span := observability.StartSpan(ctx, "funnel.usecase.create_lead",
		attribute.String("tenant.id", input.TenantID),
		attribute.String("contact.id", input.ContactID),
		attribute.String("conversation.id", input.ConversationID),
	)
	defer span.End()

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
		funnelID, routeErr := uc.funnelProductRouter.FindTopPriorityFunnelID(ctx, input.TenantID, detectedProductID)
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

	// Pick the responsible user before persisting the lead. If the picker
	// fails (including ErrNoResponsibleAvailable), we abort lead creation
	// entirely — no lead, no movement, no events.
	if uc.picker == nil {
		return ErrPickerNotConfigured
	}
	pick, err := uc.picker.PickForFunnel(ctx, input.TenantID, funnel.ID, entryCol.ID)
	if err != nil {
		return err
	}
	lead.AssignResponsible(pick.UserID)

	if err := uc.leadRepo.Create(ctx, lead); err != nil {
		return err
	}

	// Create initial movement
	movement := domain.NewLeadMovement(uuid.New().String(), lead.ID, "", entryCol.ID)
	if err := uc.movementRepo.Create(ctx, movement); err != nil {
		return err
	}

	if uc.eventBus != nil {
		uc.eventBus.Publish(events.Event{
			Type:     events.EventLeadCreated,
			TenantID: lead.TenantID,
			Payload: map[string]string{
				"lead_id":             lead.ID,
				"funnel_id":           lead.FunnelID,
				"column_id":           lead.ColumnID,
				"contact_id":          lead.ContactID,
				"responsible_user_id": lead.ResponsibleUserID,
			},
		})
		uc.eventBus.Publish(events.Event{
			Type:     events.EventLeadResponsibleAssigned,
			TenantID: lead.TenantID,
			Payload: events.ResponsibleAssignedPayload{
				LeadID:            lead.ID,
				ResponsibleUserID: lead.ResponsibleUserID,
				Reason:            "created",
				Outcome:           string(pick.Outcome),
				Algorithm:         pick.Algorithm,
			},
		})
	}

	if uc.automationTrigger != nil {
		_ = uc.automationTrigger.OnLeadEvent(ctx, lead.TenantID, lead.ID, lead.ColumnID)
	}

	return nil
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
