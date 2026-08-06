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

	// A guardrail shared by specialists must not be deleted out from under them:
	// detach it from every specialist first. This makes deletion an explicit,
	// deliberate action rather than a silent cascade.
	count, err := uc.guardrailRepo.CountSpecialistsByGuardrailID(ctx, id)
	if err != nil {
		return err
	}
	if count > 0 {
		return domain.ErrGuardrailInUse
	}

	return uc.guardrailRepo.Delete(ctx, id)
}
