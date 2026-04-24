package application

import (
	"context"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel/attribute"
	"go.uber.org/zap"

	"github.com/sasrgita/crm-juridico/internal/funnel/domain"
	"github.com/sasrgita/crm-juridico/internal/shared/events"
	"github.com/sasrgita/crm-juridico/internal/shared/observability"
)

type MoveLeadInput struct {
	TenantID string
	LeadID   string
	ColumnID string
	FunnelID string // optional: if different from current funnel
}

type MoveLeadUseCase struct {
	funnelRepo        domain.FunnelRepository
	columnRepo        domain.ColumnRepository
	leadRepo          domain.LeadRepository
	movementRepo      domain.LeadMovementRepository
	eventBus          events.EventBus
	automationTrigger domain.AutomationTrigger
	auditLog          *zap.Logger
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

// SetAutomationTrigger sets an optional automation trigger that will be called
// after a lead is moved successfully.
func (uc *MoveLeadUseCase) SetAutomationTrigger(t domain.AutomationTrigger) {
	uc.automationTrigger = t
}

// SetAuditLogger sets an optional structured logger for audit events
// (successful movements and cross-tenant access denials).
func (uc *MoveLeadUseCase) SetAuditLogger(l *zap.Logger) {
	uc.auditLog = l
}

func (uc *MoveLeadUseCase) audit() *zap.Logger {
	if uc.auditLog == nil {
		return zap.NewNop()
	}
	return uc.auditLog
}

func (uc *MoveLeadUseCase) Execute(ctx context.Context, input MoveLeadInput) error {
	ctx, span := observability.StartSpan(ctx, "funnel.usecase.move_lead",
		attribute.String("tenant.id", input.TenantID),
		attribute.String("lead.id", input.LeadID),
		attribute.String("funnel.column.id", input.ColumnID),
	)
	defer span.End()

	lead, err := uc.leadRepo.FindByID(ctx, input.LeadID)
	if err != nil {
		return err
	}
	if lead.TenantID != input.TenantID {
		uc.audit().Warn("cross-tenant lead access denied",
			zap.String("tenant_id", input.TenantID),
			zap.String("lead_id", input.LeadID),
			zap.String("operation", "move_lead"),
		)
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

	if uc.automationTrigger != nil {
		_ = uc.automationTrigger.OnLeadEvent(ctx, lead.TenantID, lead.ID, lead.ColumnID)
	}

	uc.audit().Info("lead moved",
		zap.String("tenant_id", lead.TenantID),
		zap.String("lead_id", lead.ID),
		zap.String("funnel_id", lead.FunnelID),
		zap.String("from_column_id", fromColumnID),
		zap.String("to_column_id", col.ID),
	)

	return nil
}
