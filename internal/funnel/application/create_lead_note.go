package application

import (
	"context"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel/attribute"

	"github.com/sasrgita/crm-juridico/internal/funnel/domain"
	"github.com/sasrgita/crm-juridico/internal/shared/observability"
)

type CreateLeadNoteInput struct {
	TenantID  string
	LeadID    string
	Content   string
	CreatedBy string
}

type CreateLeadNoteUseCase struct {
	leadRepo domain.LeadRepository
	noteRepo domain.LeadNoteRepository
}

func NewCreateLeadNoteUseCase(leadRepo domain.LeadRepository, noteRepo domain.LeadNoteRepository) *CreateLeadNoteUseCase {
	return &CreateLeadNoteUseCase{leadRepo: leadRepo, noteRepo: noteRepo}
}

func (uc *CreateLeadNoteUseCase) Execute(ctx context.Context, input CreateLeadNoteInput) (*LeadNoteOutput, error) {
	ctx, span := observability.StartSpan(ctx, "funnel.usecase.create_lead_note",
		attribute.String("tenant.id", input.TenantID),
		attribute.String("lead.id", input.LeadID),
	)
	defer span.End()

	lead, err := uc.leadRepo.FindByID(ctx, input.LeadID)
	if err != nil {
		return nil, err
	}
	if lead.TenantID != input.TenantID {
		return nil, domain.ErrLeadNotFound
	}

	note, err := domain.NewLeadNote(uuid.New().String(), lead.ID, lead.TenantID, input.Content, input.CreatedBy)
	if err != nil {
		return nil, err
	}

	if err := uc.noteRepo.Create(ctx, note); err != nil {
		return nil, err
	}

	return &LeadNoteOutput{
		ID:        note.ID,
		Content:   note.Content,
		CreatedBy: note.CreatedBy,
		CreatedAt: note.CreatedAt,
	}, nil
}
