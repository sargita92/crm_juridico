package domain

import "time"

// GroupFunnel associates a permission group with a funnel, optionally
// restricting access to specific columns. When ColumnIDs is empty the group
// has access to the entire funnel.
type GroupFunnel struct {
	ID        string
	GroupID   string
	FunnelID  string
	ColumnIDs []string
	CreatedAt time.Time
}

func NewGroupFunnel(id, groupID, funnelID string, columnIDs []string) (*GroupFunnel, error) {
	if groupID == "" {
		return nil, ErrGroupIDRequired
	}
	if funnelID == "" {
		return nil, ErrFunnelIDRequired
	}
	if columnIDs == nil {
		columnIDs = []string{}
	}
	return &GroupFunnel{
		ID:        id,
		GroupID:   groupID,
		FunnelID:  funnelID,
		ColumnIDs: columnIDs,
		CreatedAt: time.Now(),
	}, nil
}

// CoversColumn reports whether the group has access to the given column.
// An empty ColumnIDs slice means the group covers the entire funnel.
func (gf *GroupFunnel) CoversColumn(columnID string) bool {
	if len(gf.ColumnIDs) == 0 {
		return true
	}
	for _, id := range gf.ColumnIDs {
		if id == columnID {
			return true
		}
	}
	return false
}
