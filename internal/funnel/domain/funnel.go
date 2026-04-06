package domain

import "time"

const MaxFunnelNameLength = 255

type Funnel struct {
	ID          string
	TenantID    string
	Name        string
	Description string
	Active      bool
	IsDefault   bool
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

func NewFunnel(id, tenantID, name, description string) (*Funnel, error) {
	if tenantID == "" {
		return nil, ErrTenantIDRequired
	}
	if name == "" {
		return nil, ErrFunnelNameRequired
	}
	if len(name) > MaxFunnelNameLength {
		return nil, ErrFunnelNameTooLong
	}
	now := time.Now()
	return &Funnel{
		ID: id, TenantID: tenantID, Name: name,
		Description: description, Active: true, IsDefault: false,
		CreatedAt: now, UpdatedAt: now,
	}, nil
}

func (f *Funnel) Update(name, description string) error {
	if name == "" {
		return ErrFunnelNameRequired
	}
	if len(name) > MaxFunnelNameLength {
		return ErrFunnelNameTooLong
	}
	f.Name = name
	f.Description = description
	f.UpdatedAt = time.Now()
	return nil
}

func (f *Funnel) Activate() {
	f.Active = true
	f.UpdatedAt = time.Now()
}

func (f *Funnel) Deactivate() {
	f.Active = false
	f.UpdatedAt = time.Now()
}

func (f *Funnel) SetDefault() {
	f.IsDefault = true
	f.UpdatedAt = time.Now()
}
