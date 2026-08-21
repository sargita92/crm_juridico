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
	// NotesForConversation lists the notes of the lead behind the conversation. A
	// conversation with no lead yet simply has no notes — empty, not an error.
	//
	// It used to also report whether a lead existed, and the panel hid the note
	// form when it did not. That coupling is gone on purpose: the chat always
	// offers the field, and AddNote creates the lead if it is missing.
	NotesForConversation(ctx context.Context, tenantID, conversationID string) (notes []ConversationNote, err error)
	// AddNote creates a note on the lead behind the conversation, promoting the
	// conversation to a lead first when it does not have one, and returns the
	// refreshed list.
	AddNote(ctx context.Context, tenantID, conversationID, content, createdBy string) (notes []ConversationNote, err error)
}
