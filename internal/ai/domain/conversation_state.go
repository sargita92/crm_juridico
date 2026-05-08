package domain

import (
	"context"
	"errors"
	"time"
)

var (
	ErrConversationIDRequired    = errors.New("conversation ID is required")
	ErrConversationStateNotFound = errors.New("conversation state not found")
)

// ConversationState tracks the AI conversation progress for a lead.
type ConversationState struct {
	ID                     string
	ConversationID         string
	SpecialistID           string
	CurrentStepIndex       int
	CollectedData          map[string]string
	AccumulatedScore       int
	HandoffActive          bool
	HandoffAt              *time.Time
	ResumedAt              *time.Time
	PendingCrossSellRuleID *string
	CreatedAt              time.Time
	UpdatedAt              time.Time
}

// SetPendingCrossSellRuleID stores the cross-sell rule ID that awaits confirmation.
func (s *ConversationState) SetPendingCrossSellRuleID(id string) {
	s.PendingCrossSellRuleID = &id
	s.UpdatedAt = time.Now()
}

// ClearPendingCrossSellRuleID removes any pending cross-sell rule.
func (s *ConversationState) ClearPendingCrossSellRuleID() {
	s.PendingCrossSellRuleID = nil
	s.UpdatedAt = time.Now()
}

// NewConversationState constructs a ConversationState with validation.
func NewConversationState(id, conversationID, specialistID string) (*ConversationState, error) {
	if conversationID == "" {
		return nil, ErrConversationIDRequired
	}
	if specialistID == "" {
		return nil, ErrSpecialistIDRequired
	}
	now := time.Now()
	return &ConversationState{
		ID:               id,
		ConversationID:   conversationID,
		SpecialistID:     specialistID,
		CurrentStepIndex: 0,
		CollectedData:    make(map[string]string),
		AccumulatedScore: 0,
		HandoffActive:    false,
		CreatedAt:        now,
		UpdatedAt:        now,
	}, nil
}

// AdvanceStep increments the step index, records the collected data and adds to the score.
func (s *ConversationState) AdvanceStep(key, value string, score int) {
	s.CurrentStepIndex++
	s.CollectedData[key] = value
	s.AccumulatedScore += score
	s.UpdatedAt = time.Now()
}

// ActivateHandoff flags the conversation as handed off to a human agent.
func (s *ConversationState) ActivateHandoff() {
	now := time.Now()
	s.HandoffActive = true
	s.HandoffAt = &now
	s.UpdatedAt = now
}

// DeactivateHandoff resumes AI handling of the conversation.
func (s *ConversationState) DeactivateHandoff() {
	now := time.Now()
	s.HandoffActive = false
	s.ResumedAt = &now
	s.UpdatedAt = now
}

// IsAfterResume returns true if no handoff ever occurred, or if timestamp is after ResumedAt.
func (s *ConversationState) IsAfterResume(timestamp time.Time) bool {
	if s.ResumedAt == nil {
		return true
	}
	return timestamp.After(*s.ResumedAt)
}

// Reset clears the state so the conversation starts from step 0.
// CollectedData is emptied, score and step index are zeroed, and any active
// handoff is cleared. HandoffAt and ResumedAt are cleared as well so a new
// lifecycle can begin.
func (s *ConversationState) Reset() {
	s.CurrentStepIndex = 0
	s.CollectedData = make(map[string]string)
	s.AccumulatedScore = 0
	s.HandoffActive = false
	s.HandoffAt = nil
	s.ResumedAt = nil
	s.UpdatedAt = time.Now()
}

// ConversationStateRepository defines persistence operations for ConversationState.
type ConversationStateRepository interface {
	Create(ctx context.Context, state *ConversationState) error
	FindByConversationID(ctx context.Context, conversationID string) (*ConversationState, error)
	Update(ctx context.Context, state *ConversationState) error
}
