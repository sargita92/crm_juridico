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
	Threshold            int
	ThresholdHumanoMin   int
	TotalPossible        int
	Percentage           float64
	Steps                []StepScoreItem
	QualifiedColumnID    string
	HumanColumnID        string
	DisqualifiedColumnID string
	CrossSellColumnID    string
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
	thresholdHumanoMin := 0
	qualifiedColumnID := ""
	humanColumnID := ""
	disqualifiedColumnID := ""
	crossSellColumnID := ""
	if config != nil {
		threshold = config.Threshold
		thresholdHumanoMin = config.ThresholdHumanoMin
		qualifiedColumnID = config.QualifiedColumnID
		humanColumnID = config.HumanColumnID
		disqualifiedColumnID = config.DisqualifiedColumnID
		crossSellColumnID = config.CrossSellColumnID
	}

	var percentage float64
	if total > 0 {
		percentage = float64(threshold) / float64(total) * 100
	}

	return &ScoringOutput{
		Threshold:            threshold,
		ThresholdHumanoMin:   thresholdHumanoMin,
		TotalPossible:        total,
		Percentage:           percentage,
		Steps:                stepItems,
		QualifiedColumnID:    qualifiedColumnID,
		HumanColumnID:        humanColumnID,
		DisqualifiedColumnID: disqualifiedColumnID,
		CrossSellColumnID:    crossSellColumnID,
	}, nil
}
