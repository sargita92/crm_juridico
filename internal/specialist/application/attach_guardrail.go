package application

import (
	"context"

	"github.com/sasrgita/crm-juridico/internal/specialist/domain"
)

// AttachGuardrailUseCase links an existing library guardrail to a specialist.
type AttachGuardrailUseCase struct {
	specRepo      domain.SpecialistRepository
	guardrailRepo domain.GuardrailRepository
}

func NewAttachGuardrailUseCase(specRepo domain.SpecialistRepository, guardrailRepo domain.GuardrailRepository) *AttachGuardrailUseCase {
	return &AttachGuardrailUseCase{specRepo: specRepo, guardrailRepo: guardrailRepo}
}

func (uc *AttachGuardrailUseCase) Execute(ctx context.Context, specialistID, guardrailID string) error {
	spec, err := uc.specRepo.FindByID(ctx, specialistID)
	if err != nil {
		return err
	}
	if !spec.IsActive() {
		return domain.ErrSpecialistInactive
	}
	if _, err := uc.guardrailRepo.FindByID(ctx, guardrailID); err != nil {
		return err
	}
	return uc.guardrailRepo.Attach(ctx, specialistID, guardrailID)
}
