package application

import (
	"context"
	"time"

	"github.com/sasrgita/crm-juridico/internal/specialist/domain"
)

// UpdateCrossSellRuleInput holds the fields that may be updated.
type UpdateCrossSellRuleInput struct {
	ID              string
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
