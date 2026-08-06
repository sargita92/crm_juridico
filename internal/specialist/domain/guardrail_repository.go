package domain

import "context"

// GuardrailRepository manages the guardrail library and its many-to-many links
// to specialists (the specialist_guardrails join table).
type GuardrailRepository interface {
	// Library CRUD.
	Create(ctx context.Context, guardrail *Guardrail) error
	FindByID(ctx context.Context, id string) (*Guardrail, error)
	Update(ctx context.Context, guardrail *Guardrail) error
	Delete(ctx context.Context, id string) error
	FindAll(ctx context.Context) ([]Guardrail, error)

	// Attachment (join table).
	FindBySpecialistID(ctx context.Context, specialistID string) ([]Guardrail, error)
	Attach(ctx context.Context, specialistID, guardrailID string) error
	Detach(ctx context.Context, specialistID, guardrailID string) error
	CountSpecialistsByGuardrailID(ctx context.Context, guardrailID string) (int, error)
}
