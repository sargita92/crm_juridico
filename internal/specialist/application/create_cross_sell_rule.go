package application

import (
	"context"

	"github.com/google/uuid"

	"github.com/sasrgita/crm-juridico/internal/specialist/domain"
)

// CreateCrossSellRuleInput holds the data required to create a new rule.
type CreateCrossSellRuleInput struct {
	SpecialistID    string
	TriggerType     string
	TriggerConfig   any
	TargetProductID string
}

// CreateCrossSellRuleUseCase creates a cross-sell rule for a specialist.
// It validates that the specialist exists and belongs to the given context.
type CreateCrossSellRuleUseCase struct {
	specRepo domain.SpecialistRepository
	repo     domain.CrossSellRuleRepository
}

func NewCreateCrossSellRuleUseCase(
	specRepo domain.SpecialistRepository,
	repo domain.CrossSellRuleRepository,
) *CreateCrossSellRuleUseCase {
	return &CreateCrossSellRuleUseCase{specRepo: specRepo, repo: repo}
}

func (uc *CreateCrossSellRuleUseCase) Execute(ctx context.Context, input CreateCrossSellRuleInput) (*CrossSellRuleOutput, error) {
	// Validate specialist exists
	if _, err := uc.specRepo.FindByID(ctx, input.SpecialistID); err != nil {
		return nil, err
	}

	// Determine next Ordem
	existing, err := uc.repo.ListBySpecialistID(ctx, input.SpecialistID)
	if err != nil {
		return nil, err
	}
	ordem := len(existing) + 1

	rule, err := domain.NewCrossSellRule(
		uuid.New().String(),
		input.SpecialistID,
		ordem,
		domain.CrossSellTriggerType(input.TriggerType),
		input.TriggerConfig,
		input.TargetProductID,
	)
	if err != nil {
		return nil, err
	}

	if err := uc.repo.Save(ctx, rule); err != nil {
		return nil, err
	}

	out := crossSellRuleToOutput(rule)
	return &out, nil
}
