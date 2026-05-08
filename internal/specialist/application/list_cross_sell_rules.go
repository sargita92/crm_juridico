package application

import (
	"context"
	"time"

	"github.com/sasrgita/crm-juridico/internal/specialist/domain"
)

// CrossSellRuleOutput is the shared output DTO for a cross-sell rule.
type CrossSellRuleOutput struct {
	ID              string
	SpecialistID    string
	Ordem           int
	TriggerType     string
	TriggerConfig   any
	TargetProductID string
	Ativo           bool
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

func crossSellRuleToOutput(r *domain.CrossSellRule) CrossSellRuleOutput {
	return CrossSellRuleOutput{
		ID:              r.ID,
		SpecialistID:    r.SpecialistID,
		Ordem:           r.Ordem,
		TriggerType:     string(r.TriggerType),
		TriggerConfig:   r.TriggerConfig,
		TargetProductID: r.TargetProductID,
		Ativo:           r.Ativo,
		CreatedAt:       r.CreatedAt,
		UpdatedAt:       r.UpdatedAt,
	}
}

// ListCrossSellRulesUseCase lists all cross-sell rules for a specialist.
type ListCrossSellRulesUseCase struct {
	repo domain.CrossSellRuleRepository
}

func NewListCrossSellRulesUseCase(repo domain.CrossSellRuleRepository) *ListCrossSellRulesUseCase {
	return &ListCrossSellRulesUseCase{repo: repo}
}

func (uc *ListCrossSellRulesUseCase) Execute(ctx context.Context, specialistID string) ([]CrossSellRuleOutput, error) {
	rules, err := uc.repo.ListBySpecialistID(ctx, specialistID)
	if err != nil {
		return nil, err
	}

	out := make([]CrossSellRuleOutput, len(rules))
	for i, r := range rules {
		out[i] = crossSellRuleToOutput(r)
	}
	return out, nil
}
