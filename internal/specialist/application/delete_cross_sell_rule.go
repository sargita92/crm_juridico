package application

import (
	"context"

	"github.com/sasrgita/crm-juridico/internal/specialist/domain"
)

// DeleteCrossSellRuleUseCase removes a cross-sell rule by ID.
type DeleteCrossSellRuleUseCase struct {
	repo domain.CrossSellRuleRepository
}

func NewDeleteCrossSellRuleUseCase(repo domain.CrossSellRuleRepository) *DeleteCrossSellRuleUseCase {
	return &DeleteCrossSellRuleUseCase{repo: repo}
}

func (uc *DeleteCrossSellRuleUseCase) Execute(ctx context.Context, id string) error {
	return uc.repo.Delete(ctx, id)
}
