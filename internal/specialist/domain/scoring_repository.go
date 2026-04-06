package domain

import "context"

type ScoringConfigRepository interface {
	CreateOrUpdate(ctx context.Context, config *ScoringConfig) error
	FindBySpecialistID(ctx context.Context, specialistID string) (*ScoringConfig, error)
}
