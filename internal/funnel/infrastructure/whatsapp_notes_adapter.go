package infrastructure

import (
	"context"
	"errors"
	"fmt"

	"github.com/sasrgita/crm-juridico/internal/funnel/application"
	"github.com/sasrgita/crm-juridico/internal/funnel/domain"
	whatsappdomain "github.com/sasrgita/crm-juridico/internal/whatsapp/domain"
)

// ConversationFinder resolves which contact a conversation belongs to. Satisfied
// by the whatsapp module's ConversationRepository.
type ConversationFinder interface {
	FindByID(ctx context.Context, id string) (*whatsappdomain.Conversation, error)
}

// LeadCreator promotes a conversation into a lead. Satisfied by the funnel's
// CreateLeadUseCase, the same one that runs on every inbound message.
type LeadCreator interface {
	CreateFromConversation(ctx context.Context, tenantID, contactID, conversationID, messageText string) error
}

// WhatsAppNotesAdapter implements whatsapp/domain.LeadNotesService over the funnel
// lead/note repositories, operating on the current lead of a conversation.
type WhatsAppNotesAdapter struct {
	leadRepo      domain.LeadRepository
	noteRepo      domain.LeadNoteRepository
	userNames     domain.UserNameProvider
	createUC      *application.CreateLeadNoteUseCase
	conversations ConversationFinder
	leadCreator   LeadCreator
}

// NewWhatsAppNotesAdapter builds the adapter from the funnel repositories and the
// existing create-note use case. conversations and leadCreator let AddNote promote
// a lead-less conversation on the spot, so the chat always accepts a note.
func NewWhatsAppNotesAdapter(
	leadRepo domain.LeadRepository,
	noteRepo domain.LeadNoteRepository,
	userNames domain.UserNameProvider,
	createUC *application.CreateLeadNoteUseCase,
	conversations ConversationFinder,
	leadCreator LeadCreator,
) *WhatsAppNotesAdapter {
	return &WhatsAppNotesAdapter{
		leadRepo: leadRepo, noteRepo: noteRepo, userNames: userNames,
		createUC: createUC, conversations: conversations, leadCreator: leadCreator,
	}
}

// NotesForConversation lists the notes of the lead behind the conversation. A
// conversation with no lead simply has no notes — that is not an error, and no
// lead is created here: rendering a panel must not mutate the funnel.
func (a *WhatsAppNotesAdapter) NotesForConversation(ctx context.Context, tenantID, conversationID string) ([]whatsappdomain.ConversationNote, error) {
	lead, err := a.findLead(ctx, tenantID, conversationID)
	if err != nil {
		if errors.Is(err, domain.ErrLeadNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return a.listNotes(ctx, lead.ID)
}

// AddNote creates a note on the lead behind the conversation, promoting the
// conversation to a lead first when it does not have one yet.
func (a *WhatsAppNotesAdapter) AddNote(ctx context.Context, tenantID, conversationID, content, createdBy string) ([]whatsappdomain.ConversationNote, error) {
	lead, err := a.findLead(ctx, tenantID, conversationID)
	if err != nil {
		if !errors.Is(err, domain.ErrLeadNotFound) {
			return nil, err
		}
		lead, err = a.promoteToLead(ctx, tenantID, conversationID)
		if err != nil {
			return nil, err
		}
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

// findLead resolves the lead a note belongs to, without creating anything.
//
// The conversation is the primary key, but a contact can carry a lead created on
// an earlier conversation (CreateLead deduplicates by contact, so it never
// re-points an existing lead at a new conversation). Falling back to the
// contact's current lead keeps reads and writes on the same row — otherwise a
// note would be saved somewhere the panel does not read back from.
func (a *WhatsAppNotesAdapter) findLead(ctx context.Context, tenantID, conversationID string) (*domain.Lead, error) {
	lead, err := a.leadRepo.FindCurrentByConversationID(ctx, tenantID, conversationID)
	if err == nil {
		return lead, nil
	}
	if !errors.Is(err, domain.ErrLeadNotFound) {
		return nil, err
	}

	conv, convErr := a.conversation(ctx, tenantID, conversationID)
	if convErr != nil {
		return nil, convErr
	}
	return a.leadRepo.FindByContactAndTenant(ctx, tenantID, conv.ContactID)
}

// promoteToLead runs the same lead creation used on every inbound message. It
// fails loudly when the tenant has no default funnel or entry column — that
// misconfiguration currently only reaches the logs, and surfacing it on the note
// the operator just tried to save is strictly better than silence.
func (a *WhatsAppNotesAdapter) promoteToLead(ctx context.Context, tenantID, conversationID string) (*domain.Lead, error) {
	conv, err := a.conversation(ctx, tenantID, conversationID)
	if err != nil {
		return nil, err
	}
	if err := a.leadCreator.CreateFromConversation(ctx, tenantID, conv.ContactID, conversationID, ""); err != nil {
		return nil, fmt.Errorf("whatsapp_notes_adapter: promote conversation to lead: %w", err)
	}
	return a.findLead(ctx, tenantID, conversationID)
}

// conversation loads a conversation and enforces tenant ownership. Without the
// check, a conversation ID from another tenant would resolve that tenant's
// contact and attach the note to their lead.
func (a *WhatsAppNotesAdapter) conversation(ctx context.Context, tenantID, conversationID string) (*whatsappdomain.Conversation, error) {
	if a.conversations == nil {
		return nil, domain.ErrLeadNotFound
	}
	conv, err := a.conversations.FindByID(ctx, conversationID)
	if err != nil {
		if errors.Is(err, whatsappdomain.ErrConversationNotFound) {
			return nil, domain.ErrLeadNotFound
		}
		return nil, err
	}
	if conv.TenantID != tenantID {
		return nil, domain.ErrLeadNotFound
	}
	return conv, nil
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
