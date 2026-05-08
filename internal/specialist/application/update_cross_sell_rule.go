package application

import (
	"context"
	"time"

	"github.com/sasrgita/crm-juridico/internal/specialist/domain"
)

// UpdateCrossSellRuleInput holds the fields that may be updated.
type UpdateCrossSellRuleInput struct {
	ID              string
	SpecialistID    string // used to assert ownership; required for tenant isolation
	TriggerType     string
	TriggerConfig   any
	TargetProductID string
	Ativo           bool
}

// UpdateCrossSellRuleUseCase updates an existing cross-sell rule.
type UpdateCrossSellRuleUseCase struct {
	repo domain.CrossSellRuleRepository
}

func NewUpdateCrossSellRuleUseCase(repo domain.CrossSellRuleRepository) *UpdateCrossSellRuleUseCase {
	return &UpdateCrossSellRuleUseCase{repo: repo}
}

func (uc *UpdateCrossSellRuleUseCase) Execute(ctx context.Context, input UpdateCrossSellRuleInput) (*CrossSellRuleOutput, error) {
	rule, err := uc.repo.FindByID(ctx, input.ID)
	if err != nil {
		return nil, err
	}

	// Tenant isolation: ensure the rule belongs to the specialist in the request context.
	if input.SpecialistID != "" && rule.SpecialistID != input.SpecialistID {
		return nil, domain.ErrCrossSellRuleNotOwnedBySpecialist
	}

	triggerType := domain.CrossSellTriggerType(input.TriggerType)
	rule.TriggerType = triggerType
	rule.TriggerConfig = input.TriggerConfig
	rule.TargetProductID = input.TargetProductID
	rule.Ativo = input.Ativo
	rule.UpdatedAt = time.Now()

	// Re-validate trigger config
	if err := domain.ValidateTriggerConfig(triggerType, input.TriggerConfig); err != nil {
		return nil, err
	}
	if input.TargetProductID == "" {
		return nil, domain.ErrTargetProductRequired
	}

	if err := uc.repo.Save(ctx, rule); err != nil {
		return nil, err
	}

	out := crossSellRuleToOutput(rule)
	return &out, nil
}
