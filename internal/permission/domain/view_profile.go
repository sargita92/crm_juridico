package domain

import "time"

type ViewProfile struct {
	ID             string
	GroupID        string
	FunnelID       string
	VisibleColumns []string
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

func NewViewProfile(id, groupID, funnelID string, visibleColumns []string) (*ViewProfile, error) {
	if groupID == "" {
		return nil, ErrGroupIDRequired
	}
	if funnelID == "" {
		return nil, ErrFunnelIDRequired
	}
	if visibleColumns == nil {
		visibleColumns = []string{}
	}
	now := time.Now()
	return &ViewProfile{
		ID:             id,
		GroupID:        groupID,
		FunnelID:       funnelID,
		VisibleColumns: visibleColumns,
		CreatedAt:      now,
		UpdatedAt:      now,
	}, nil
}

func (vp *ViewProfile) UpdateColumns(columns []string) {
	if columns == nil {
		columns = []string{}
	}
	vp.VisibleColumns = columns
	vp.UpdatedAt = time.Now()
}
