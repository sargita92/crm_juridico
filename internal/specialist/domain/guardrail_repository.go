package domain

import "context"

type GuardrailRepository interface {
	Create(ctx context.Context, guardrail *Guardrail) error
	FindByID(ctx context.Context, id string) (*Guardrail, error)
	Update(ctx context.Context, guardrail *Guardrail) error
	Delete(ctx context.Context, id string) error
	FindBySpecialistID(ctx context.Context, specialistID string) ([]Guardrail, error)
}
