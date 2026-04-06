package domain

import "time"

type SpecialistDocument struct {
	SpecialistID string
	DocumentID   string
	CreatedAt    time.Time
}
