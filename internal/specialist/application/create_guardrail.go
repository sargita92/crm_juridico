package application

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/sasrgita/crm-juridico/internal/specialist/domain"
)

type CreateGuardrailInput struct {
	// SpecialistID is optional. When empty, the guardrail is created in the
	// library unattached; when set, it is created and immediately attached to
	// that specialist (the "create new" flow on the specialist detail page).
	SpecialistID string
	Name         string
	Type         string
	Rule         string
	Message      string
}

type GuardrailOutput struct {
	ID        string
	Name      string
	Type      string
	Rule      string
	Message   string
	Active    bool
	CreatedAt time.Time
	UpdatedAt time.Time
}

func newGuardrailOutput(g *domain.Guardrail) GuardrailOutput {
	return GuardrailOutput{
		ID: g.ID, Name: g.Name, Type: string(g.Type),
		Rule: g.Rule, Message: g.Message, Active: g.Active,
		CreatedAt: g.CreatedAt, UpdatedAt: g.UpdatedAt,
	}
}

type CreateGuardrailUseCase struct {
	specRepo      domain.SpecialistRepository
	guardrailRepo domain.GuardrailRepository
}

func NewCreateGuardrailUseCase(specRepo domain.SpecialistRepository, guardrailRepo domain.GuardrailRepository) *CreateGuardrailUseCase {
	return &CreateGuardrailUseCase{specRepo: specRepo, guardrailRepo: guardrailRepo}
}

func (uc *CreateGuardrailUseCase) Execute(ctx context.Context, input CreateGuardrailInput) (*GuardrailOutput, error) {
	// When attaching on creation, the target specialist must exist and be active.
	if input.SpecialistID != "" {
		spec, err := uc.specRepo.FindByID(ctx, input.SpecialistID)
		if err != nil {
			return nil, err
		}
		if !spec.IsActive() {
			return nil, domain.ErrSpecialistInactive
		}
	}

	g, err := domain.NewGuardrail(uuid.New().String(), input.Name, domain.GuardrailType(input.Type), input.Rule, input.Message)
	if err != nil {
		return nil, err
	}

	if err := uc.guardrailRepo.Create(ctx, g); err != nil {
		return nil, err
	}

	if input.SpecialistID != "" {
		if err := uc.guardrailRepo.Attach(ctx, input.SpecialistID, g.ID); err != nil {
			return nil, err
		}
	}

	out := newGuardrailOutput(g)
	return &out, nil
}
