package application

import (
	"context"

	"github.com/sasrgita/crm-juridico/internal/specialist/domain"
)

// DetachGuardrailUseCase removes the link between a specialist and a library
// guardrail. The guardrail itself stays in the library. Detach is idempotent.
type DetachGuardrailUseCase struct {
	guardrailRepo domain.GuardrailRepository
}

func NewDetachGuardrailUseCase(guardrailRepo domain.GuardrailRepository) *DetachGuardrailUseCase {
	return &DetachGuardrailUseCase{guardrailRepo: guardrailRepo}
}

func (uc *DetachGuardrailUseCase) Execute(ctx context.Context, specialistID, guardrailID string) error {
	return uc.guardrailRepo.Detach(ctx, specialistID, guardrailID)
}
