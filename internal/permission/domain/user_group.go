package domain

import "time"

type UserGroup struct {
	ID        string
	UserID    string
	GroupID   string
	TenantID  string
	CreatedAt time.Time
}

func NewUserGroup(id, userID, groupID, tenantID string) (*UserGroup, error) {
	if userID == "" {
		return nil, ErrUserIDRequired
	}
	if groupID == "" {
		return nil, ErrGroupIDRequired
	}
	if tenantID == "" {
		return nil, ErrTenantIDRequired
	}
	return &UserGroup{
		ID:        id,
		UserID:    userID,
		GroupID:   groupID,
		TenantID:  tenantID,
		CreatedAt: time.Now(),
	}, nil
}
