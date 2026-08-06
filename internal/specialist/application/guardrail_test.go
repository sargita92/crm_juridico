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

	// Creating with a SpecialistID must also attach it to that specialist.
	attached, _ := guardrailRepo.FindBySpecialistID(context.Background(), "spec-1")
	assert.Len(t, attached, 1)
}

// A library-only creation (no SpecialistID) succeeds and stays unattached.
func TestCreateGuardrailUseCase_LibraryOnly_NoSpecialist(t *testing.T) {
	specRepo := newMockSpecialistRepo()
	guardrailRepo := newMockGuardrailRepo()
	uc := NewCreateGuardrailUseCase(specRepo, guardrailRepo)

	output, err := uc.Execute(context.Background(), CreateGuardrailInput{
		Name: "biblioteca", Type: "forbidden_topics", Rule: "regra",
	})

	require.NoError(t, err)
	count, _ := guardrailRepo.CountSpecialistsByGuardrailID(context.Background(), output.ID)
	assert.Equal(t, 0, count, "guardrail de biblioteca não deve nascer anexado")
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

	g, _ := domain.NewGuardrail("g-1", "nome-teste", domain.GuardrailTypeForbiddenTopics, "regra antiga", "msg antiga")
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

	g, _ := domain.NewGuardrail("g-1", "nome-teste", domain.GuardrailTypeForbiddenTopics, "regra", "msg")
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

	g, _ := domain.NewGuardrail("g-1", "nome-teste", domain.GuardrailTypeForbiddenTopics, "regra", "msg")
	guardrailRepo.addGuardrail(g)

	err := uc.Execute(context.Background(), "g-1")
	require.NoError(t, err)
	assert.Empty(t, guardrailRepo.guardrails)
}

// A guardrail attached to any specialist must not be deletable from the library.
func TestDeleteGuardrailUseCase_InUse_Blocked(t *testing.T) {
	guardrailRepo := newMockGuardrailRepo()
	uc := NewDeleteGuardrailUseCase(guardrailRepo)

	g, _ := domain.NewGuardrail("g-1", "nome-teste", domain.GuardrailTypeForbiddenTopics, "regra", "msg")
	guardrailRepo.addGuardrail(g)
	require.NoError(t, guardrailRepo.Attach(context.Background(), "spec-1", "g-1"))

	err := uc.Execute(context.Background(), "g-1")
	assert.ErrorIs(t, err, domain.ErrGuardrailInUse)
	assert.NotEmpty(t, guardrailRepo.guardrails, "guardrail em uso não deve ser removido")
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

	g1, _ := domain.NewGuardrail("g-1", "nome-teste", domain.GuardrailTypeForbiddenTopics, "regra 1", "msg 1")
	g2, _ := domain.NewGuardrail("g-2", "nome-teste", domain.GuardrailTypeScopeLimit, "regra 2", "msg 2")
	guardrailRepo.addGuardrail(g1)
	guardrailRepo.addGuardrail(g2)
	require.NoError(t, guardrailRepo.Attach(context.Background(), "spec-1", "g-1"))
	require.NoError(t, guardrailRepo.Attach(context.Background(), "spec-1", "g-2"))

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

// A single library guardrail attached to two specialists is returned for both —
// the core "reuse" behavior.
func TestAttachGuardrailUseCase_SharedAcrossSpecialists(t *testing.T) {
	specRepo := newMockSpecialistRepo()
	guardrailRepo := newMockGuardrailRepo()
	uc := NewAttachGuardrailUseCase(specRepo, guardrailRepo)

	for _, id := range []string{"spec-1", "spec-2"} {
		s, _ := domain.NewSpecialist(id, id, "desc", "prompt")
		specRepo.addSpecialist(s)
	}
	g, _ := domain.NewGuardrail("g-1", "compartilhado", domain.GuardrailTypeForbiddenTopics, "regra", "msg")
	guardrailRepo.addGuardrail(g)

	require.NoError(t, uc.Execute(context.Background(), "spec-1", "g-1"))
	require.NoError(t, uc.Execute(context.Background(), "spec-2", "g-1"))

	l1, _ := guardrailRepo.FindBySpecialistID(context.Background(), "spec-1")
	l2, _ := guardrailRepo.FindBySpecialistID(context.Background(), "spec-2")
	assert.Len(t, l1, 1)
	assert.Len(t, l2, 1)
	assert.Equal(t, l1[0].ID, l2[0].ID, "o mesmo guardrail deve servir os dois especialistas")
}

func TestAttachGuardrailUseCase_GuardrailNotFound(t *testing.T) {
	specRepo := newMockSpecialistRepo()
	guardrailRepo := newMockGuardrailRepo()
	uc := NewAttachGuardrailUseCase(specRepo, guardrailRepo)

	s, _ := domain.NewSpecialist("spec-1", "s", "desc", "prompt")
	specRepo.addSpecialist(s)

	err := uc.Execute(context.Background(), "spec-1", "nonexistent")
	assert.ErrorIs(t, err, domain.ErrGuardrailNotFound)
}

func TestAttachGuardrailUseCase_SpecialistInactive(t *testing.T) {
	specRepo := newMockSpecialistRepo()
	guardrailRepo := newMockGuardrailRepo()
	uc := NewAttachGuardrailUseCase(specRepo, guardrailRepo)

	s, _ := domain.NewSpecialist("spec-1", "s", "desc", "prompt")
	_ = s.Deactivate()
	specRepo.addSpecialist(s)
	g, _ := domain.NewGuardrail("g-1", "x", domain.GuardrailTypeForbiddenTopics, "regra", "msg")
	guardrailRepo.addGuardrail(g)

	err := uc.Execute(context.Background(), "spec-1", "g-1")
	assert.ErrorIs(t, err, domain.ErrSpecialistInactive)
}

// Detach removes only the link; the guardrail remains in the library and
// attached to any other specialists.
func TestDetachGuardrailUseCase_KeepsLibraryAndOtherLinks(t *testing.T) {
	guardrailRepo := newMockGuardrailRepo()
	uc := NewDetachGuardrailUseCase(guardrailRepo)

	g, _ := domain.NewGuardrail("g-1", "x", domain.GuardrailTypeForbiddenTopics, "regra", "msg")
	guardrailRepo.addGuardrail(g)
	require.NoError(t, guardrailRepo.Attach(context.Background(), "spec-1", "g-1"))
	require.NoError(t, guardrailRepo.Attach(context.Background(), "spec-2", "g-1"))

	require.NoError(t, uc.Execute(context.Background(), "spec-1", "g-1"))

	l1, _ := guardrailRepo.FindBySpecialistID(context.Background(), "spec-1")
	l2, _ := guardrailRepo.FindBySpecialistID(context.Background(), "spec-2")
	assert.Empty(t, l1)
	assert.Len(t, l2, 1)
	assert.NotEmpty(t, guardrailRepo.guardrails, "guardrail continua na biblioteca após detach")
}

func TestListAvailableGuardrailsUseCase_ExcludesAttached(t *testing.T) {
	guardrailRepo := newMockGuardrailRepo()
	uc := NewListAvailableGuardrailsUseCase(guardrailRepo)

	g1, _ := domain.NewGuardrail("g-1", "anexado", domain.GuardrailTypeForbiddenTopics, "r", "m")
	g2, _ := domain.NewGuardrail("g-2", "livre", domain.GuardrailTypeScopeLimit, "r", "m")
	guardrailRepo.addGuardrail(g1)
	guardrailRepo.addGuardrail(g2)
	require.NoError(t, guardrailRepo.Attach(context.Background(), "spec-1", "g-1"))

	items, err := uc.Execute(context.Background(), "spec-1")

	require.NoError(t, err)
	require.Len(t, items, 1)
	assert.Equal(t, "g-2", items[0].ID)
}

func TestListAllGuardrailsUseCase_ReturnsWholeLibrary(t *testing.T) {
	guardrailRepo := newMockGuardrailRepo()
	uc := NewListAllGuardrailsUseCase(guardrailRepo)

	g1, _ := domain.NewGuardrail("g-1", "a", domain.GuardrailTypeForbiddenTopics, "r", "m")
	g2, _ := domain.NewGuardrail("g-2", "b", domain.GuardrailTypeScopeLimit, "r", "m")
	guardrailRepo.addGuardrail(g1)
	guardrailRepo.addGuardrail(g2)

	items, err := uc.Execute(context.Background())

	require.NoError(t, err)
	assert.Len(t, items, 2)
}
