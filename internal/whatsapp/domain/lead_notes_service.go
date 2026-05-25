package domain

import (
	"context"
	"time"
)

// ConversationNote is a note shown in the WhatsApp chat notes drawer.
type ConversationNote struct {
	ID         string
	Content    string
	AuthorName string
	CreatedAt  time.Time
}

// LeadNotesService gives the WhatsApp screen access to the notes of the lead that a
// conversation is currently on. Implemented by the funnel module.
type LeadNotesService interface {
	// NotesForConversation resolves the current lead of the conversation and lists its
	// notes. hasLead is false (with no error) when the conversation has no lead in this
	// tenant.
	NotesForConversation(ctx context.Context, tenantID, conversationID string) (hasLead bool, notes []ConversationNote, err error)
	// AddNote creates a note on the current lead of the conversation and returns the
	// refreshed list.
	AddNote(ctx context.Context, tenantID, conversationID, content, createdBy string) (notes []ConversationNote, err error)
}
