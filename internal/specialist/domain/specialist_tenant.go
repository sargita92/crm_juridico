package domain

import "time"

type SpecialistTenant struct {
	SpecialistID string
	TenantID     string
	CreatedAt    time.Time
}
