package domain

import "time"

type SpecialistTenant struct {
	SpecialistID string
	TenantID     string
	IsDefault    bool
	CreatedAt    time.Time
}
