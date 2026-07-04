package application

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/sasrgita/crm-juridico/internal/specialist/domain"
)

func TestCreateGuardrailUseCase_Success(t *testing.T) {
	specRepo := newMockSpecialistRepo()
	guardrailRepo := newMockGuardrailRepo()
	uc := NewCreateGuardrailUseCase(specRepo, guardrailRepo)

	spec, _ := domain.NewSpecialist("spec-1", "Especialista", "desc", "prompt")
	specRepo.addSpecialist(spec)

	output, err := uc.Execute(context.Background(), CreateGuardrailInput{
		Name: "nome-teste", SpecialistID: "spec-1",
		Type:    "forbidden_topics",
		Rule:    "Nao falar sobre precos",
		Message: "Nao posso informar precos",
	})

	require.NoError(t, err)
	assert.NotEmpty(t, output.ID)
	assert.Equal(t, "forbidden_topics", output.Type)
	assert.Equal(t, "Nao falar sobre precos", output.Rule)
	assert.True(t, output.Active)
}

func TestCreateGuardrailUseCase_SpecialistNotFound(t *testing.T) {
	specRepo := newMockSpecialistRepo()
	guardrailRepo := newMockGuardrailRepo()
	uc := NewCreateGuardrailUseCase(specRepo, guardrailRepo)

	_, err := uc.Execute(context.Background(), CreateGuardrailInput{
		Name: "nome-teste", SpecialistID: "nonexistent", Type: "forbidden_topics", Rule: "regra",
	})

	assert.ErrorIs(t, err, domain.ErrSpecialistNotFound)
}

func TestCreateGuardrailUseCase_SpecialistInactive(t *testing.T) {
	specRepo := newMockSpecialistRepo()
	guardrailRepo := newMockGuardrailRepo()
	uc := NewCreateGuardrailUseCase(specRepo, guardrailRepo)

	spec, _ := domain.NewSpecialist("spec-1", "Especialista", "desc", "prompt")
	_ = spec.Deactivate()
	specRepo.addSpecialist(spec)

	_, err := uc.Execute(context.Background(), CreateGuardrailInput{
		Name: "nome-teste", SpecialistID: "spec-1", Type: "forbidden_topics", Rule: "regra",
	})

	assert.ErrorIs(t, err, domain.ErrSpecialistInactive)
}

func TestCreateGuardrailUseCase_EmptyRule(t *testing.T) {
	specRepo := newMockSpecialistRepo()
	guardrailRepo := newMockGuardrailRepo()
	uc := NewCreateGuardrailUseCase(specRepo, guardrailRepo)

	spec, _ := domain.NewSpecialist("spec-1", "Especialista", "desc", "prompt")
	specRepo.addSpecialist(spec)

	_, err := uc.Execute(context.Background(), CreateGuardrailInput{
		Name: "nome-teste", SpecialistID: "spec-1", Type: "forbidden_topics", Rule: "",
	})

	assert.ErrorIs(t, err, domain.ErrGuardrailRuleRequired)
}

func TestCreateGuardrailUseCase_InvalidType(t *testing.T) {
	specRepo := newMockSpecialistRepo()
	guardrailRepo := newMockGuardrailRepo()
	uc := NewCreateGuardrailUseCase(specRepo, guardrailRepo)

	spec, _ := domain.NewSpecialist("spec-1", "Especialista", "desc", "prompt")
	specRepo.addSpecialist(spec)

	_, err := uc.Execute(context.Background(), CreateGuardrailInput{
		Name: "nome-teste", SpecialistID: "spec-1", Type: "invalid", Rule: "regra",
	})

	assert.ErrorIs(t, err, domain.ErrGuardrailTypeInvalid)
}

func TestUpdateGuardrailUseCase_Success(t *testing.T) {
	guardrailRepo := newMockGuardrailRepo()
	uc := NewUpdateGuardrailUseCase(guardrailRepo)

	g, _ := domain.NewGuardrail("g-1", "spec-1", "nome-teste", domain.GuardrailTypeForbiddenTopics, "regra antiga", "msg antiga")
	guardrailRepo.addGuardrail(g)

	output, err := uc.Execute(context.Background(), UpdateGuardrailInput{
		Name: "nome-teste", ID: "g-1", Type: "scope_limit", Rule: "regra nova", Message: "msg nova",
	})

	require.NoError(t, err)
	assert.Equal(t, "scope_limit", output.Type)
	assert.Equal(t, "regra nova", output.Rule)
}

func TestUpdateGuardrailUseCase_NotFound(t *testing.T) {
	guardrailRepo := newMockGuardrailRepo()
	uc := NewUpdateGuardrailUseCase(guardrailRepo)

	_, err := uc.Execute(context.Background(), UpdateGuardrailInput{
		Name: "nome-teste", ID: "nonexistent", Type: "forbidden_topics", Rule: "regra",
	})

	assert.ErrorIs(t, err, domain.ErrGuardrailNotFound)
}

func TestToggleGuardrailUseCase_Success(t *testing.T) {
	guardrailRepo := newMockGuardrailRepo()
	uc := NewToggleGuardrailUseCase(guardrailRepo)

	g, _ := domain.NewGuardrail("g-1", "spec-1", "nome-teste", domain.GuardrailTypeForbiddenTopics, "regra", "msg")
	guardrailRepo.addGuardrail(g)
	assert.True(t, g.Active)

	err := uc.Execute(context.Background(), "g-1")
	require.NoError(t, err)
	assert.False(t, guardrailRepo.guardrails["g-1"].Active)

	err = uc.Execute(context.Background(), "g-1")
	require.NoError(t, err)
	assert.True(t, guardrailRepo.guardrails["g-1"].Active)
}

func TestToggleGuardrailUseCase_NotFound(t *testing.T) {
	guardrailRepo := newMockGuardrailRepo()
	uc := NewToggleGuardrailUseCase(guardrailRepo)

	err := uc.Execute(context.Background(), "nonexistent")
	assert.ErrorIs(t, err, domain.ErrGuardrailNotFound)
}

func TestDeleteGuardrailUseCase_Success(t *testing.T) {
	guardrailRepo := newMockGuardrailRepo()
	uc := NewDeleteGuardrailUseCase(guardrailRepo)

	g, _ := domain.NewGuardrail("g-1", "spec-1", "nome-teste", domain.GuardrailTypeForbiddenTopics, "regra", "msg")
	guardrailRepo.addGuardrail(g)

	err := uc.Execute(context.Background(), "g-1")
	require.NoError(t, err)
	assert.Empty(t, guardrailRepo.guardrails)
}

func TestDeleteGuardrailUseCase_NotFound(t *testing.T) {
	guardrailRepo := newMockGuardrailRepo()
	uc := NewDeleteGuardrailUseCase(guardrailRepo)

	err := uc.Execute(context.Background(), "nonexistent")
	assert.ErrorIs(t, err, domain.ErrGuardrailNotFound)
}

func TestListGuardrailsUseCase_Success(t *testing.T) {
	guardrailRepo := newMockGuardrailRepo()
	uc := NewListGuardrailsUseCase(guardrailRepo)

	g1, _ := domain.NewGuardrail("g-1", "spec-1", "nome-teste", domain.GuardrailTypeForbiddenTopics, "regra 1", "msg 1")
	g2, _ := domain.NewGuardrail("g-2", "spec-1", "nome-teste", domain.GuardrailTypeScopeLimit, "regra 2", "msg 2")
	guardrailRepo.addGuardrail(g1)
	guardrailRepo.addGuardrail(g2)

	items, err := uc.Execute(context.Background(), "spec-1")

	require.NoError(t, err)
	assert.Len(t, items, 2)
}

func TestListGuardrailsUseCase_Empty(t *testing.T) {
	guardrailRepo := newMockGuardrailRepo()
	uc := NewListGuardrailsUseCase(guardrailRepo)

	items, err := uc.Execute(context.Background(), "spec-1")

	require.NoError(t, err)
	assert.Empty(t, items)
}
