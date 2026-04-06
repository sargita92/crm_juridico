package application

import (
	"context"
	"errors"

	"github.com/sasrgita/crm-juridico/internal/specialist/domain"
)

type StepScoreItem struct {
	StepID string
	Text   string
	Score  int
}

type ScoringOutput struct {
	Threshold     int
	TotalPossible int
	Percentage    float64
	Steps         []StepScoreItem
}

type GetScoringUseCase struct {
	stepRepo    domain.StepRepository
	scoringRepo domain.ScoringConfigRepository
}

func NewGetScoringUseCase(stepRepo domain.StepRepository, scoringRepo domain.ScoringConfigRepository) *GetScoringUseCase {
	return &GetScoringUseCase{stepRepo: stepRepo, scoringRepo: scoringRepo}
}

func (uc *GetScoringUseCase) Execute(ctx context.Context, specialistID string) (*ScoringOutput, error) {
	steps, err := uc.stepRepo.FindBySpecialistID(ctx, specialistID)
	if err != nil {
		return nil, err
	}

	total := 0
	stepItems := make([]StepScoreItem, len(steps))
	for i, s := range steps {
		total += s.Score
		stepItems[i] = StepScoreItem{StepID: s.ID, Text: s.Text, Score: s.Score}
	}

	config, err := uc.scoringRepo.FindBySpecialistID(ctx, specialistID)
	if err != nil && !errors.Is(err, domain.ErrScoringConfigNotFound) {
		return nil, err
	}

	threshold := 0
	if config != nil {
		threshold = config.Threshold
	}

	var percentage float64
	if total > 0 {
		percentage = float64(threshold) / float64(total) * 100
	}

	return &ScoringOutput{
		Threshold:     threshold,
		TotalPossible: total,
		Percentage:    percentage,
		Steps:         stepItems,
	}, nil
}
