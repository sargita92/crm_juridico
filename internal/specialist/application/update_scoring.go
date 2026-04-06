package application

import (
	"context"
	"errors"

	"github.com/google/uuid"

	"github.com/sasrgita/crm-juridico/internal/specialist/domain"
)

type UpdateScoringInput struct {
	SpecialistID string
	Threshold    int
}

type UpdateScoringUseCase struct {
	stepRepo    domain.StepRepository
	scoringRepo domain.ScoringConfigRepository
}

func NewUpdateScoringUseCase(stepRepo domain.StepRepository, scoringRepo domain.ScoringConfigRepository) *UpdateScoringUseCase {
	return &UpdateScoringUseCase{stepRepo: stepRepo, scoringRepo: scoringRepo}
}

func (uc *UpdateScoringUseCase) Execute(ctx context.Context, input UpdateScoringInput) (*ScoringOutput, error) {
	steps, err := uc.stepRepo.FindBySpecialistID(ctx, input.SpecialistID)
	if err != nil {
		return nil, err
	}

	total := 0
	stepItems := make([]StepScoreItem, len(steps))
	for i, s := range steps {
		total += s.Score
		stepItems[i] = StepScoreItem{StepID: s.ID, Text: s.Text, Score: s.Score}
	}

	config, err := uc.scoringRepo.FindBySpecialistID(ctx, input.SpecialistID)
	if err != nil && !errors.Is(err, domain.ErrScoringConfigNotFound) {
		return nil, err
	}

	if config == nil {
		config, err = domain.NewScoringConfig(uuid.New().String(), input.SpecialistID, input.Threshold)
		if err != nil {
			return nil, err
		}
	} else {
		if err := config.UpdateThreshold(input.Threshold, total); err != nil {
			return nil, err
		}
	}

	if err := uc.scoringRepo.CreateOrUpdate(ctx, config); err != nil {
		return nil, err
	}

	var percentage float64
	if total > 0 {
		percentage = float64(config.Threshold) / float64(total) * 100
	}

	return &ScoringOutput{
		Threshold:     config.Threshold,
		TotalPossible: total,
		Percentage:    percentage,
		Steps:         stepItems,
	}, nil
}
