package domain

import "time"

type SpecialistMcp struct {
	SpecialistID string
	McpID        string
	CreatedAt    time.Time
}
