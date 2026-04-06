package domain

import "time"

type LeadStatus string

const (
	LeadStatusOpen LeadStatus = "open"
	LeadStatusWon  LeadStatus = "won"
	LeadStatusLost LeadStatus = "lost"
)

type Lead struct {
	ID              string
	TenantID        string
	FunnelID        string
	ColumnID        string
	ContactID       string
	ConversationID  string
	Score           int
	Status          LeadStatus
	ColumnEnteredAt time.Time
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

func NewLead(id, tenantID, funnelID, columnID, contactID, conversationID string) (*Lead, error) {
	if tenantID == "" {
		return nil, ErrTenantIDRequired
	}
	if funnelID == "" {
		return nil, ErrFunnelIDRequired
	}
	if columnID == "" {
		return nil, ErrColumnIDRequired
	}
	if contactID == "" {
		return nil, ErrContactIDRequired
	}
	if conversationID == "" {
		return nil, ErrConversationIDRequired
	}
	now := time.Now()
	return &Lead{
		ID: id, TenantID: tenantID, FunnelID: funnelID,
		ColumnID: columnID, ContactID: contactID,
		ConversationID: conversationID, Score: 0,
		Status: LeadStatusOpen, ColumnEnteredAt: now,
		CreatedAt: now, UpdatedAt: now,
	}, nil
}

func (l *Lead) MoveTo(columnID string, colType ColumnType) {
	l.ColumnID = columnID
	l.ColumnEnteredAt = time.Now()
	l.UpdatedAt = time.Now()
	switch colType {
	case ColumnTypeWon:
		l.Status = LeadStatusWon
	case ColumnTypeLost:
		l.Status = LeadStatusLost
	default:
		if l.Status != LeadStatusOpen {
			l.Status = LeadStatusOpen
		}
	}
}

func (l *Lead) UpdateScore(score int) {
	l.Score = score
	l.UpdatedAt = time.Now()
}
