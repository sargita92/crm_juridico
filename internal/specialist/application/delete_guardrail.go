package application

import (
	"context"

	"github.com/sasrgita/crm-juridico/internal/specialist/domain"
)

type DeleteGuardrailUseCase struct {
	guardrailRepo domain.GuardrailRepository
}

func NewDeleteGuardrailUseCase(guardrailRepo domain.GuardrailRepository) *DeleteGuardrailUseCase {
	return &DeleteGuardrailUseCase{guardrailRepo: guardrailRepo}
}

func (uc *DeleteGuardrailUseCase) Execute(ctx context.Context, id string) error {
	if _, err := uc.guardrailRepo.FindByID(ctx, id); err != nil {
		return err
	}
	return uc.guardrailRepo.Delete(ctx, id)
}
