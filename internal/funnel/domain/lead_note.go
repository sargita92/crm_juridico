package domain

import "time"

const MaxNoteContentLength = 2000

type LeadNote struct {
	ID        string
	LeadID    string
	TenantID  string
	Content   string
	CreatedBy string
	CreatedAt time.Time
}

func NewLeadNote(id, leadID, tenantID, content, createdBy string) (*LeadNote, error) {
	if leadID == "" {
		return nil, ErrLeadNotFound
	}
	if tenantID == "" {
		return nil, ErrTenantIDRequired
	}
	if content == "" {
		return nil, ErrNoteContentRequired
	}
	if len(content) > MaxNoteContentLength {
		return nil, ErrNoteContentTooLong
	}
	if createdBy == "" {
		return nil, ErrNoteCreatedByRequired
	}
	return &LeadNote{
		ID:        id,
		LeadID:    leadID,
		TenantID:  tenantID,
		Content:   content,
		CreatedBy: createdBy,
		CreatedAt: time.Now(),
	}, nil
}
