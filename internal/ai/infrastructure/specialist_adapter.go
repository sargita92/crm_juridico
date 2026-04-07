package infrastructure

import (
	"context"

	specDomain "github.com/sasrgita/crm-juridico/internal/specialist/domain"
)

// SpecialistFinderAdapter satisfies application.SpecialistFinder.
type SpecialistFinderAdapter struct {
	repo specDomain.SpecialistRepository
}

func NewSpecialistFinderAdapter(repo specDomain.SpecialistRepository) *SpecialistFinderAdapter {
	return &SpecialistFinderAdapter{repo: repo}
}

func (a *SpecialistFinderAdapter) FindByID(ctx context.Context, id string) (*specDomain.Specialist, error) {
	return a.repo.FindByID(ctx, id)
}

// StepFinderAdapter satisfies application.StepFinder.
type StepFinderAdapter struct {
	repo specDomain.StepRepository
}

func NewStepFinderAdapter(repo specDomain.StepRepository) *StepFinderAdapter {
	return &StepFinderAdapter{repo: repo}
}

func (a *StepFinderAdapter) FindBySpecialistID(ctx context.Context, id string) ([]specDomain.Step, error) {
	return a.repo.FindBySpecialistID(ctx, id)
}

// GuardrailFinderAdapter satisfies application.GuardrailFinder.
type GuardrailFinderAdapter struct {
	repo specDomain.GuardrailRepository
}

func NewGuardrailFinderAdapter(repo specDomain.GuardrailRepository) *GuardrailFinderAdapter {
	return &GuardrailFinderAdapter{repo: repo}
}

func (a *GuardrailFinderAdapter) FindBySpecialistID(ctx context.Context, id string) ([]specDomain.Guardrail, error) {
	return a.repo.FindBySpecialistID(ctx, id)
}

// DefaultSpecialistFinderAdapter satisfies application.DefaultSpecialistFinder.
type DefaultSpecialistFinderAdapter struct {
	repo specDomain.SpecialistTenantRepository
}

func NewDefaultSpecialistFinderAdapter(repo specDomain.SpecialistTenantRepository) *DefaultSpecialistFinderAdapter {
	return &DefaultSpecialistFinderAdapter{repo: repo}
}

func (a *DefaultSpecialistFinderAdapter) FindDefaultByTenantID(ctx context.Context, tenantID string) (string, error) {
	return a.repo.FindDefaultByTenantID(ctx, tenantID)
}
