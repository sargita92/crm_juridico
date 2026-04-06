package domain

import "time"

type LeadMovement struct {
	ID           string
	LeadID       string
	FromColumnID string
	ToColumnID   string
	MovedAt      time.Time
}

func NewLeadMovement(id, leadID, fromColumnID, toColumnID string) *LeadMovement {
	return &LeadMovement{
		ID:           id,
		LeadID:       leadID,
		FromColumnID: fromColumnID,
		ToColumnID:   toColumnID,
		MovedAt:      time.Now(),
	}
}
