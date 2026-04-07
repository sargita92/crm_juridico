package domain

import "time"

type ScoringConfig struct {
	ID                   string
	SpecialistID         string
	Threshold            int
	QualifiedColumnID    string
	DisqualifiedColumnID string
	CreatedAt            time.Time
	UpdatedAt            time.Time
}

func NewScoringConfig(id, specialistID string, threshold int) (*ScoringConfig, error) {
	if specialistID == "" {
		return nil, ErrSpecialistIDRequired
	}
	if threshold <= 0 {
		return nil, ErrScoringThresholdInvalid
	}

	now := time.Now()
	return &ScoringConfig{
		ID: id, SpecialistID: specialistID, Threshold: threshold,
		CreatedAt: now, UpdatedAt: now,
	}, nil
}

func (sc *ScoringConfig) UpdateThreshold(threshold, totalPossible int) error {
	if threshold <= 0 {
		return ErrScoringThresholdInvalid
	}
	if threshold > totalPossible {
		return ErrScoringThresholdExceedsTotal
	}
	sc.Threshold = threshold
	sc.UpdatedAt = time.Now()
	return nil
}
