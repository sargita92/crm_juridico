package infrastructure

import (
	"context"
	"errors"

	"github.com/sasrgita/crm-juridico/internal/funnel/application"
	"github.com/sasrgita/crm-juridico/internal/funnel/domain"
	whatsappdomain "github.com/sasrgita/crm-juridico/internal/whatsapp/domain"
)

// WhatsAppNotesAdapter implements whatsapp/domain.LeadNotesService over the funnel
// lead/note repositories, operating on the current lead of a conversation.
type WhatsAppNotesAdapter struct {
	leadRepo  domain.LeadRepository
	noteRepo  domain.LeadNoteRepository
	userNames domain.UserNameProvider
	createUC  *application.CreateLeadNoteUseCase
}

// NewWhatsAppNotesAdapter builds the adapter from the funnel repositories and the
// existing create-note use case.
func NewWhatsAppNotesAdapter(
	leadRepo domain.LeadRepository,
	noteRepo domain.LeadNoteRepository,
	userNames domain.UserNameProvider,
	createUC *application.CreateLeadNoteUseCase,
) *WhatsAppNotesAdapter {
	return &WhatsAppNotesAdapter{leadRepo: leadRepo, noteRepo: noteRepo, userNames: userNames, createUC: createUC}
}

// NotesForConversation resolves the current lead of the conversation and lists its
// notes. When the conversation has no lead in this tenant, returns hasLead=false and no
// error so the caller can render an empty state.
func (a *WhatsAppNotesAdapter) NotesForConversation(ctx context.Context, tenantID, conversationID string) (bool, []whatsappdomain.ConversationNote, error) {
	lead, err := a.leadRepo.FindCurrentByConversationID(ctx, tenantID, conversationID)
	if err != nil {
		if errors.Is(err, domain.ErrLeadNotFound) {
			return false, nil, nil
		}
		return false, nil, err
	}
	notes, err := a.listNotes(ctx, lead.ID)
	if err != nil {
		return false, nil, err
	}
	return true, notes, nil
}

// AddNote creates a note on the current lead of the conversation and returns the
// refreshed list. Returns ErrLeadNotFound when the conversation has no lead.
func (a *WhatsAppNotesAdapter) AddNote(ctx context.Context, tenantID, conversationID, content, createdBy string) ([]whatsappdomain.ConversationNote, error) {
	lead, err := a.leadRepo.FindCurrentByConversationID(ctx, tenantID, conversationID)
	if err != nil {
		return nil, err
	}
	if _, err := a.createUC.Execute(ctx, application.CreateLeadNoteInput{
		TenantID:  tenantID,
		LeadID:    lead.ID,
		Content:   content,
		CreatedBy: createdBy,
	}); err != nil {
		return nil, err
	}
	return a.listNotes(ctx, lead.ID)
}

func (a *WhatsAppNotesAdapter) listNotes(ctx context.Context, leadID string) ([]whatsappdomain.ConversationNote, error) {
	notes, err := a.noteRepo.FindByLeadID(ctx, leadID)
	if err != nil {
		return nil, err
	}
	out := make([]whatsappdomain.ConversationNote, 0, len(notes))
	for _, n := range notes {
		name, _ := a.userNames.FindNameByID(ctx, n.CreatedBy)
		out = append(out, whatsappdomain.ConversationNote{
			ID:         n.ID,
			Content:    n.Content,
			AuthorName: name,
			CreatedAt:  n.CreatedAt,
		})
	}
	return out, nil
}
