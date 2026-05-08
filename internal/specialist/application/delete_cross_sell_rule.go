package application

import (
	"context"

	"github.com/sasrgita/crm-juridico/internal/specialist/domain"
)

// DeleteCrossSellRuleInput holds the data required to delete a rule.
type DeleteCrossSellRuleInput struct {
	ID           string
	SpecialistID string // used to assert ownership; required for tenant isolation
}

// DeleteCrossSellRuleUseCase removes a cross-sell rule by ID.
type DeleteCrossSellRuleUseCase struct {
	repo domain.CrossSellRuleRepository
}

func NewDeleteCrossSellRuleUseCase(repo domain.CrossSellRuleRepository) *DeleteCrossSellRuleUseCase {
	return &DeleteCrossSellRuleUseCase{repo: repo}
}

func (uc *DeleteCrossSellRuleUseCase) Execute(ctx context.Context, input DeleteCrossSellRuleInput) error {
	rule, err := uc.repo.FindByID(ctx, input.ID)
	if err != nil {
		return err
	}

	// Tenant isolation: ensure the rule belongs to the specialist in the request context.
	if input.SpecialistID != "" && rule.SpecialistID != input.SpecialistID {
		return domain.ErrCrossSellRuleNotOwnedBySpecialist
	}

	return uc.repo.Delete(ctx, input.ID)
}
