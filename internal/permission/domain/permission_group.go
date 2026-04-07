package domain

import "time"

const (
	MaxGroupNameLength = 100
	MaxGroupDescLength = 500
)

type PermissionGroup struct {
	ID          string
	TenantID    string
	Name        string
	Description string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

func NewPermissionGroup(id, tenantID, name, description string) (*PermissionGroup, error) {
	if tenantID == "" {
		return nil, ErrTenantIDRequired
	}
	if name == "" {
		return nil, ErrGroupNameRequired
	}
	if len(name) > MaxGroupNameLength {
		return nil, ErrGroupNameTooLong
	}
	if len(description) > MaxGroupDescLength {
		return nil, ErrGroupDescTooLong
	}
	now := time.Now()
	return &PermissionGroup{
		ID:          id,
		TenantID:    tenantID,
		Name:        name,
		Description: description,
		CreatedAt:   now,
		UpdatedAt:   now,
	}, nil
}

func (g *PermissionGroup) Update(name, description string) error {
	if name == "" {
		return ErrGroupNameRequired
	}
	if len(name) > MaxGroupNameLength {
		return ErrGroupNameTooLong
	}
	if len(description) > MaxGroupDescLength {
		return ErrGroupDescTooLong
	}
	g.Name = name
	g.Description = description
	g.UpdatedAt = time.Now()
	return nil
}
